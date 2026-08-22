package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"spiritFruit/app/models/async_tasks"
	"spiritFruit/app/models/characters"
	"spiritFruit/app/models/projects"
	"spiritFruit/app/services"
	myAsynq "spiritFruit/pkg/asynq"
	"spiritFruit/pkg/config"
	"spiritFruit/pkg/console"
	"spiritFruit/pkg/database"
	"spiritFruit/pkg/openai"
	"spiritFruit/pkg/prompt"
	"spiritFruit/pkg/utils"
	"strings"

	"github.com/hibiken/asynq"
)

// HandleGenerateCharacters 处理角色生成任务
func HandleGenerateCharacters(ctx context.Context, t *asynq.Task) error {
	// 1. 解析参数
	var p myAsynq.GenerateCharactersPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json unmarshal failed: %v", err)
	}

	// 2. 获取任务并标记开始
	taskModel := async_tasks.AsyncTask{}
	if err := database.DB.First(&taskModel, p.AsyncTaskID).Error; err != nil {
		console.Error(fmt.Sprintf("Task %d not found in DB", p.AsyncTaskID))
		return nil // 任务不存在，直接结束
	}

	// [Stage 1] 状态变更为 Processing，进度 -> 10%
	taskModel.MarkAsProcessing()
	console.Success(fmt.Sprintf("任务[%d] - 开始生成角色", p.AsyncTaskID))

	// 3. 获取项目信息
	var projectModel projects.Projects
	if err := database.DB.First(&projectModel, p.ProjectID).Error; err != nil {
		err = fmt.Errorf("project not found: %v", err)
		taskModel.MarkAsFailed(err)
		return nil // 项目不存在，无需重试
	}

	// 处理字段默认值
	projectTitle := ""
	if projectModel.Title != nil {
		projectTitle = *projectModel.Title
	}
	projectDesc := ""
	if projectModel.Description != nil {
		projectDesc = *projectModel.Description
	}
	projectGenre := "都市"
	if projectModel.Genre != nil {
		projectGenre = *projectModel.Genre
	}
	projectStyle := "realistic"
	if projectModel.Style != nil {
		projectStyle = *projectModel.Style
	}

	// [Stage 2] 准备 Prompt 和 AI 配置，进度 -> 20%
	taskModel.UpdateProgress(20)

	// 准备 Prompt
	promptGen := prompt.NewGenerator()
	systemPrompt := promptGen.GetCharacterExtractionPrompt(projectStyle)

	outlineText := p.Outline
	if outlineText == "" {
		outlineText = fmt.Sprintf("剧名：%s\n简介：%s\n类型：%s", projectTitle, projectDesc, projectGenre)
	}
	userPrompt := promptGen.FormatUserPrompt("character_request", outlineText, p.Count)

	// ==========================================
	// 🔴 优先查库，后查 .env
	// ==========================================

	// 1. 先用 .env 环境变量进行基础兜底初始化
	aiConfig := openai.Config{
		Provider: config.GetString("ai.provider", "openai"),

		// OpenAI 配置
		OpenAIBaseURL:    config.GetString("ai.openai.base_url"),
		OpenAIKey:        config.GetString("ai.openai.api_key"),
		OpenAIModel:      config.GetString("ai.openai.model"),
		OpenAIImageModel: config.GetString("ai.openai.image_model", "dall-e-3"),

		// GetGoAPI 配置
		GetGoAPIBaseURL:    config.GetString("ai.getgoapi.base_url"),
		GetGoAPIKey:        config.GetString("ai.getgoapi.api_key"),
		GetGoAPIModel:      config.GetString("ai.getgoapi.model"),
		GetGoAPIImageModel: config.GetString("ai.getgoapi.image_model", "gpt-4o-image"),

		// Gemini 配置
		GeminiBaseURL: config.GetString("ai.gemini.base_url"),
		GeminiKey:     config.GetString("ai.gemini.api_key"),
		GeminiModel:   config.GetString("ai.gemini.model"),

		// 豆包 (Volcengine) 配置
		DoubaoBaseURL:    config.GetString("ai.doubao.base_url"),
		DoubaoKey:        config.GetString("ai.doubao.api_key"),
		DoubaoModel:      config.GetString("ai.doubao.model"),
		DoubaoImageModel: config.GetString("ai.doubao.image_model"),

		// Vertex AI 配置
		VertexKey:        config.GetString("ai.vertex.api_key"),
		VertexModel:      config.GetString("ai.vertex.model"),
		VertexImageModel: config.GetString("ai.vertex.image_model"),
	}

	// 2. 尝试从数据库加载优先级最高的 text (文本) 配置
	aiService := new(services.AiConfigService)
	errConfig, dbConfig := aiService.GetActiveConfigByType("text", taskModel.AdminID)

	if errConfig == nil && dbConfig.ID > 0 {
		providerName := strings.ToLower(*dbConfig.Provider)
		baseURL := *dbConfig.BaseUrl
		apiKey := *dbConfig.ApiKey

		// 取 JSON 数组配置的第一个模型作为生文模型
		modelName := ""
		if len(dbConfig.Model) > 0 {
			modelName = dbConfig.Model[0]
		}

		// 动态覆盖兜底配置
		switch providerName {
		case "getgoapi":
			aiConfig.Provider = "getgoapi"
			aiConfig.GetGoAPIBaseURL = baseURL
			aiConfig.GetGoAPIKey = apiKey
			if modelName != "" {
				aiConfig.GetGoAPIModel = modelName
			}

		case "openai":
			aiConfig.Provider = "openai"
			aiConfig.OpenAIBaseURL = baseURL
			aiConfig.OpenAIKey = apiKey
			if modelName != "" {
				aiConfig.OpenAIModel = modelName
			}

		case "atlascloud", "atlas-cloud", "atlas":
			openai.ApplyAtlasCloudTextConfig(&aiConfig, baseURL, apiKey, modelName)

		case "gemini", "google":
			aiConfig.Provider = "gemini"
			aiConfig.GeminiBaseURL = baseURL
			aiConfig.GeminiKey = apiKey
			if modelName != "" {
				aiConfig.GeminiModel = modelName
			}

		case "doubao", "volcengine", "volces":
			aiConfig.Provider = "doubao"
			aiConfig.DoubaoBaseURL = baseURL
			aiConfig.DoubaoKey = apiKey
			if modelName != "" {
				aiConfig.DoubaoModel = modelName
			}

		case "vertex":
			aiConfig.Provider = "vertex"
			aiConfig.VertexKey = apiKey
			if modelName != "" {
				aiConfig.VertexModel = modelName
			}

		default:
			// 兜底当作 OpenAI 协议处理
			aiConfig.Provider = "openai"
			aiConfig.OpenAIBaseURL = baseURL
			aiConfig.OpenAIKey = apiKey
			if modelName != "" {
				aiConfig.OpenAIModel = modelName
			}
		}

		console.Success(fmt.Sprintf("任务[%d] - 成功挂载数据库 AI 文本配置: Provider=%s, Model=%s", p.AsyncTaskID, providerName, modelName))
	} else {
		console.Warning(fmt.Sprintf("任务[%d] - 未命中数据库 AI 文本配置，将降级使用 .env 默认配置", p.AsyncTaskID))
	}

	aiProvider := openai.NewProvider(aiConfig)

	// [Stage 3] 发起 AI 请求，进度 -> 30%
	taskModel.UpdateProgress(30)
	console.Success(fmt.Sprintf("任务[%d] - Sending prompt to AI", p.AsyncTaskID))

	// 构造请求
	req := openai.ScriptRequest{
		Messages: []openai.ChatMessage{
			{Role: "system", Content: systemPrompt}, // 注意：部分模型(如Gemini)可能自动合并system到user
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.7,
	}

	// 调用 AI
	aiResp, err := aiProvider.GenerateScript(req)
	if err != nil {
		console.Error(fmt.Sprintf("AI生成失败: %v", err))
		taskModel.MarkAsFailed(err)
		return err // 返回 err 触发重试
	}
	// [Stage 4] 解析结果，进度 -> 60%
	taskModel.UpdateProgress(60)

	var aiResult []struct {
		Name        string `json:"name"`
		Role        string `json:"role"`
		Description string `json:"description"`
		Personality string `json:"personality"`
		Appearance  string `json:"appearance"`
	}

	if err := utils.SafeParseAIJSON(aiResp, &aiResult); err != nil {
		err = fmt.Errorf("failed to parse AI response: %v", err)
		taskModel.MarkAsFailed(err)
		return nil // 解析失败通常是 AI 输出格式错，重试意义不大
	}

	// [Stage 5] 数据入库，进度 -> 80%
	taskModel.UpdateProgress(80)

	tx := database.DB.Begin()
	count := 0
	projID := p.ProjectID

	for _, char := range aiResult {
		// 查重
		var existCount int64
		tx.Model(&characters.Characters{}).Where("project_id = ? AND name = ?", projID, char.Name).Count(&existCount)
		if existCount > 0 {
			continue
		}

		roleType := char.Role
		pers := char.Personality
		appDesc := char.Appearance
		// 简单的 Visual Prompt 生成逻辑
		visualPrompt := fmt.Sprintf("%s, %s, %s", char.Appearance, projectStyle, "high quality, best quality")

		newChar := characters.Characters{
			ProjectId:      &projID,
			Name:           &char.Name,
			RoleType:       &roleType,
			Personality:    &pers,
			AppearanceDesc: &appDesc,
			VisualPrompt:   &visualPrompt,
		}

		if err := tx.Create(&newChar).Error; err != nil {
			tx.Rollback()
			err = fmt.Errorf("db create failed: %v", err)
			taskModel.MarkAsFailed(err)
			return err
		}
		count++
	}
	tx.Commit()

	// [Stage 6] 全部完成，进度 -> 100%
	resultInfo := fmt.Sprintf(`{"generated_count": %d, "provider": "%s"}`, count, aiConfig.Provider)
	taskModel.MarkAsSuccess(resultInfo)

	console.Success(fmt.Sprintf("任务[%d] - 角色生成完成，新增 %d 个角色", p.AsyncTaskID, count))
	return nil
}
