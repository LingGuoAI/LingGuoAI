package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"spiritFruit/app/models/async_tasks"
	"spiritFruit/app/models/projects" // 如果需要项目风格
	"spiritFruit/app/models/props"
	"spiritFruit/app/models/scripts"
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

// HandleExtractPropsTask 处理道具提取任务
func HandleExtractPropsTask(ctx context.Context, t *asynq.Task) error {
	// 1. 解析参数
	var p myAsynq.ExtractPropsPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json unmarshal failed: %v", err)
	}

	// 2. 获取任务并标记开始
	taskModel := async_tasks.AsyncTask{}
	if err := database.DB.First(&taskModel, p.AsyncTaskID).Error; err != nil {
		console.Error(fmt.Sprintf("Task %d not found in DB", p.AsyncTaskID))
		return nil
	}

	// [Stage 1] 状态变更为 Processing，进度 -> 10%
	taskModel.MarkAsProcessing()
	console.Success(fmt.Sprintf("任务[%d] - 开始从剧本提取道具", p.AsyncTaskID))

	// 3. 获取剧本内容
	var scriptModel scripts.Scripts
	if err := database.DB.First(&scriptModel, p.EpisodeID).Error; err != nil {
		err = fmt.Errorf("script not found: %v", err)
		taskModel.MarkAsFailed(err)
		return nil
	}

	// 尝试获取项目风格 (用于 Prompt 优化，可选)
	var projectModel projects.Projects
	projectStyle := "realistic" // 默认风格
	if scriptModel.ProjectId != nil {
		if err := database.DB.First(&projectModel, *scriptModel.ProjectId).Error; err == nil {
			if projectModel.Style != nil {
				projectStyle = *projectModel.Style
			}
		}
	}

	// [Stage 2] 准备 Prompt 和 AI 配置，进度 -> 20%
	taskModel.UpdateProgress(20)

	promptGen := prompt.NewGenerator()
	systemPrompt := promptGen.GetPropExtractionPrompt(projectStyle)

	userPrompt := fmt.Sprintf("剧本内容：\n%s", *scriptModel.Content)

	// ==========================================
	// 🔴 文本任务优先查库，后查 .env
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

	// 2. 尝试从数据库加载优先级最高的 text (脚本处理) 配置
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

	req := openai.ScriptRequest{
		Messages: []openai.ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.5, // 提取任务温度低一点更稳定
	}

	aiResp, err := aiProvider.GenerateScript(req)
	if err != nil {
		console.Error(fmt.Sprintf("AI生成失败: %v", err))
		taskModel.MarkAsFailed(err)
		return err
	}
	// [Stage 4] 解析结果，进度 -> 60%
	taskModel.UpdateProgress(60)

	type ExtractedProp struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Description string `json:"description"`
		ImagePrompt string `json:"image_prompt"`
	}
	var aiResult []ExtractedProp

	if err := utils.SafeParseAIJSON(aiResp, &aiResult); err != nil {
		err = fmt.Errorf("failed to parse AI response: %v", err)
		taskModel.MarkAsFailed(err)
		return nil
	}

	// [Stage 5] 数据入库，进度 -> 80%
	taskModel.UpdateProgress(80)

	tx := database.DB.Begin()
	count := 0
	projID := p.ProjectID

	for _, item := range aiResult {
		// 查重
		var existCount int64
		tx.Model(&props.Props{}).Where("project_id = ? AND name = ?", projID, item.Name).Count(&existCount)
		if existCount > 0 {
			continue
		}

		newProp := props.Props{
			ProjectId:   &projID,
			Name:        &item.Name,
			Type:        &item.Type,
			Description: &item.Description,
			ImagePrompt: &item.ImagePrompt,
		}

		if err := tx.Create(&newProp).Error; err != nil {
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

	console.Success(fmt.Sprintf("任务[%d] - 道具提取完成，新增 %d 个道具", p.AsyncTaskID, count))
	return nil
}
