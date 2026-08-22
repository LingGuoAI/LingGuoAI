package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"spiritFruit/app/services"
	"strings"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"spiritFruit/app/models/async_tasks"
	"spiritFruit/app/models/ai_config"
	"spiritFruit/app/models/projects"
	"spiritFruit/app/models/scenes"
	"spiritFruit/app/models/shot_frame_prompts"
	"spiritFruit/app/models/shots"
	myAsynq "spiritFruit/pkg/asynq"
	"spiritFruit/pkg/config"
	"spiritFruit/pkg/console"
	"spiritFruit/pkg/database"
	"spiritFruit/pkg/openai"
	"spiritFruit/pkg/prompt"
	"spiritFruit/pkg/utils"
)

// HandleExtractFramePromptTask 处理提取分镜帧提示词任务
func HandleExtractFramePromptTask(ctx context.Context, t *asynq.Task) error {
	var p myAsynq.ExtractFramePromptPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json unmarshal failed: %v", err)
	}

	// 1. 获取任务记录
	taskModel := async_tasks.AsyncTask{}
	if err := database.DB.First(&taskModel, p.AsyncTaskID).Error; err != nil {
		return nil
	}
	taskModel.MarkAsProcessing()
	console.Success(fmt.Sprintf("任务[%d] - 开始提取帧提示词 (类型: %s)", p.AsyncTaskID, p.FrameType))
	taskModel.UpdateProgress(10)

	// 2. 准备数据: 获取分镜和场景
	var shot shots.Shots
	if err := database.DB.First(&shot, p.ShotID).Error; err != nil {
		err = fmt.Errorf("shot not found: %v", err)
		taskModel.MarkAsFailed(err)
		return nil
	}

	var scene scenes.Scenes
	if shot.SceneId != nil && *shot.SceneId > 0 {
		database.DB.First(&scene, *shot.SceneId)
	}

	// 获取项目信息以确定画风 (Style)
	var projectInfo projects.Projects
	dramaStyle := "cyberpunk" // 默认给一个风格，以防项目未配置
	if shot.ProjectId != nil {
		if err := database.DB.First(&projectInfo, *shot.ProjectId).Error; err == nil {
			if projectInfo.Style != nil && *projectInfo.Style != "" {
				dramaStyle = *projectInfo.Style
			}
		}
	}

	// 3. 构建 Prompt 上下文
	taskModel.UpdateProgress(30)

	// 初始化提示词生成器
	promptGen := prompt.NewGenerator()

	// 构建分镜信息
	contextInfo := buildShotContextForPrompt(promptGen, shot, scene)

	// 根据帧类型获取 System Prompt 和 User Prompt
	var systemPrompt string
	var userPrompt string

	switch p.FrameType {
	case "first":
		systemPrompt = promptGen.GetFirstFramePrompt(dramaStyle)
		userPrompt = promptGen.FormatUserPrompt("frame_info", contextInfo)
	case "key":
		systemPrompt = promptGen.GetKeyFramePrompt(dramaStyle)
		userPrompt = promptGen.FormatUserPrompt("key_frame_info", contextInfo)
	case "last":
		systemPrompt = promptGen.GetLastFramePrompt(dramaStyle)
		userPrompt = promptGen.FormatUserPrompt("last_frame_info", contextInfo)
	case "action":
		systemPrompt = promptGen.GetActionSequenceFramePrompt(dramaStyle)
		userPrompt = promptGen.FormatUserPrompt("frame_info", contextInfo)
	default:
		// 兜底为首帧
		systemPrompt = promptGen.GetFirstFramePrompt(dramaStyle)
		userPrompt = promptGen.FormatUserPrompt("frame_info", contextInfo)
	}

	// 4. 初始化 AI 客户端并调用
	taskModel.UpdateProgress(40)
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
	// 如果任务指定了模型，尝试使用特定模型
	var errConfig error
	var dbConfig ai_config.AiConfig

	if p.Model != "" {
		errConfig, dbConfig = aiService.GetSpecificModelConfig("text", "", p.Model, taskModel.AdminID)
	}

	if p.Model == "" || errConfig != nil {
		errConfig, dbConfig = aiService.GetActiveConfigByType("text", taskModel.AdminID)
	}

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

	provider := openai.NewProvider(aiConfig)

	aiReq := openai.ScriptRequest{
		Messages: []openai.ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.7,
	}

	aiResp, err := provider.GenerateScript(aiReq)
	if err != nil {
		taskModel.MarkAsFailed(fmt.Errorf("AI生成失败: %v", err))
		return err
	}

	// 5. 解析 AI 返回的 JSON 结果
	taskModel.UpdateProgress(70)

	// 根据 prompt.go 中的定义，返回的是纯 JSON 对象
	type AIFrameResult struct {
		Prompt      string `json:"prompt"`
		Description string `json:"description"`
	}

	var parsedResult AIFrameResult
	if err := utils.SafeParseAIJSON(aiResp, &parsedResult); err != nil {
		taskModel.MarkAsFailed(fmt.Errorf("failed to parse AI JSON: %v, raw response: %s", err, aiResp))
		return nil
	}

	if parsedResult.Prompt == "" {
		taskModel.MarkAsFailed(fmt.Errorf("AI 返回的提示词为空"))
		return nil
	}

	// 6. 入库 (保存到 shot_frame_prompts 表)
	taskModel.UpdateProgress(80)

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// 先删除该镜头下同类型的旧提示词 (覆盖更新逻辑)
		if err := tx.Where("shot_id = ? AND frame_type = ?", p.ShotID, p.FrameType).
			Delete(&shot_frame_prompts.ShotFramePrompts{}).Error; err != nil {
			return err
		}

		// 创建新的帧提示词记录
		shotID := p.ShotID
		frameType := p.FrameType
		promptStr := parsedResult.Prompt
		descStr := parsedResult.Description

		newFramePrompt := shot_frame_prompts.ShotFramePrompts{
			ShotId:      &shotID,
			FrameType:   &frameType,
			Prompt:      &promptStr,
			Description: &descStr,
		}

		if err := tx.Create(&newFramePrompt).Error; err != nil {
			return err
		}

		// 附加逻辑：更新 shots 表中的 image_prompt 字段（如果生成的是首帧，用于列表页封面图的基础提示词）
		if p.FrameType == "first" {
			if err := tx.Model(&shots.Shots{}).Where("id = ?", p.ShotID).
				Update("image_prompt", promptStr).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		taskModel.MarkAsFailed(fmt.Errorf("db transaction failed: %v", err))
		return err
	}

	// 7. 任务完成
	taskModel.UpdateProgress(100)
	resultData := map[string]interface{}{
		"shot_id":    p.ShotID,
		"frame_type": p.FrameType,
		"prompt":     parsedResult.Prompt,
	}
	resBytes, _ := json.Marshal(resultData)
	taskModel.MarkAsSuccess(string(resBytes))

	console.Success(fmt.Sprintf("任务[%d] - 帧提示词提取完成", p.AsyncTaskID))
	return nil
}

// buildShotContextForPrompt 辅助函数：使用 prompt.go 中定义的格式拼接分镜上下文
func buildShotContextForPrompt(gen *prompt.Generator, shot shots.Shots, scene scenes.Scenes) string {
	var parts []string

	// 镜头描述
	if shot.VisualDesc != nil && *shot.VisualDesc != "" {
		parts = append(parts, gen.FormatUserPrompt("shot_description_label", *shot.VisualDesc))
	}

	// 场景信息
	if scene.ID > 0 {
		loc, tm := "", ""
		if scene.Location != nil {
			loc = *scene.Location
		}
		if scene.Time != nil {
			tm = *scene.Time
		}
		parts = append(parts, gen.FormatUserPrompt("scene_label", loc, tm))
	} else if shot.Location != nil && shot.Time != nil {
		parts = append(parts, gen.FormatUserPrompt("scene_label", *shot.Location, *shot.Time))
	}

	// 角色
	// parts = append(parts, gen.FormatUserPrompt("characters_label", "角色名1, 角色名2"))

	// 动作
	if shot.Action != nil && *shot.Action != "" {
		parts = append(parts, gen.FormatUserPrompt("action_label", *shot.Action))
	}

	// 对白
	if shot.Dialogue != nil && *shot.Dialogue != "" {
		parts = append(parts, gen.FormatUserPrompt("dialogue_label", *shot.Dialogue))
	}

	// 氛围
	if shot.Atmosphere != nil && *shot.Atmosphere != "" {
		parts = append(parts, gen.FormatUserPrompt("atmosphere_label", *shot.Atmosphere))
	}

	// 镜头参数
	if shot.ShotType != nil && *shot.ShotType != "" {
		parts = append(parts, gen.FormatUserPrompt("shot_type_label", *shot.ShotType))
	}
	if shot.Angle != nil && *shot.Angle != "" {
		parts = append(parts, gen.FormatUserPrompt("angle_label", *shot.Angle))
	}
	if shot.CameraMovement != nil && *shot.CameraMovement != "" {
		parts = append(parts, gen.FormatUserPrompt("movement_label", *shot.CameraMovement))
	}

	return strings.Join(parts, "\n")
}
