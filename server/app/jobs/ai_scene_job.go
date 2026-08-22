package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"spiritFruit/app/models/async_tasks"
	"spiritFruit/app/models/scenes"
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
	"gorm.io/gorm"
)

// HandleExtractScenes 处理场景提取任务
func HandleExtractScenes(ctx context.Context, t *asynq.Task) error {
	// 1. 解析参数
	var p myAsynq.ExtractScenesPayload
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
	console.Success(fmt.Sprintf("任务[%d] - 开始提取场景", p.AsyncTaskID))

	// 3. 获取剧本章节内容
	var episode scripts.Scripts
	// Preload Projectss 拿到风格 Style
	if err := database.DB.Preload("Projectss").First(&episode, p.ScriptID).Error; err != nil {
		err = fmt.Errorf("episode script not found: %v", err)
		taskModel.MarkAsFailed(err)
		return nil
	}

	if episode.Content == nil || *episode.Content == "" {
		err := fmt.Errorf("script content is empty")
		taskModel.MarkAsFailed(err)
		return nil
	}

	// 获取风格
	dramaStyle := "realistic"
	if episode.Projectss != nil && episode.Projectss.Style != nil {
		dramaStyle = *episode.Projectss.Style
	}

	// [Stage 2] 准备 Prompt 和 AI 配置，进度 -> 20%
	taskModel.UpdateProgress(20)

	promptGen := prompt.NewGenerator()
	systemPrompt := promptGen.GetSceneExtractionPrompt(dramaStyle)
	contentLabel := "【剧本内容】"
	formatInstructions := getSceneFormatInstructions(promptGen.IsEnglish())
	finalPrompt := fmt.Sprintf("%s\n\n%s\n%s\n\n%s", systemPrompt, contentLabel, *episode.Content, formatInstructions)

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

	req := openai.ScriptRequest{
		Messages: []openai.ChatMessage{
			// 对于场景提取，通常只需要一次对话即可，这里为了简单直接发一条 User 消息
			// 也可以拆分为 System 和 User
			{Role: "user", Content: finalPrompt},
		},
		Temperature: 0.7,
	}

	aiResp, err := aiProvider.GenerateScript(req)
	if err != nil {
		console.Error(fmt.Sprintf("AI提取失败: %v", err))
		taskModel.MarkAsFailed(err)
		return err
	}

	// [Stage 4] 解析结果，进度 -> 60%
	taskModel.UpdateProgress(60)

	type BackgroundInfo struct {
		Location   string `json:"location"`
		Time       string `json:"time"`
		Atmosphere string `json:"atmosphere"`
		Prompt     string `json:"prompt"`
	}

	var backgrounds []BackgroundInfo

	// 兼容 Array 和 Object 两种返回格式
	if err := utils.SafeParseAIJSON(aiResp, &backgrounds); err != nil {
		var wrapper struct {
			Backgrounds []BackgroundInfo `json:"backgrounds"`
		}
		if err2 := utils.SafeParseAIJSON(aiResp, &wrapper); err2 != nil {
			err = fmt.Errorf("failed to parse scene JSON: %v", err)
			taskModel.MarkAsFailed(err)
			return nil
		}
		backgrounds = wrapper.Backgrounds
	}

	// [Stage 5] 数据入库，进度 -> 80%
	taskModel.UpdateProgress(80)

	count := 0
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		for _, bg := range backgrounds {
			// 构造场景名
			sceneName := fmt.Sprintf("%s-%s", bg.Location, bg.Time)

			var existCount int64
			// ProjectId 是指针，需要解引用或者直接用
			if episode.ProjectId == nil {
				return fmt.Errorf("project id is nil")
			}
			projID := *episode.ProjectId

			tx.Model(&scenes.Scenes{}).Where("project_id = ? AND name = ?", projID, sceneName).Count(&existCount)

			if existCount == 0 {
				loc := bg.Location
				tm := bg.Time
				atm := bg.Atmosphere
				prt := bg.Prompt
				status := int8(1) // 1-待生成

				newScene := scenes.Scenes{
					ProjectId:    &projID,
					Name:         &sceneName,
					Location:     &loc,
					Time:         &tm,
					Atmosphere:   &atm,
					VisualPrompt: &prt,
					Status:       &status,
				}
				if err := tx.Create(&newScene).Error; err != nil {
					return err
				}
				count++
			}
		}
		return nil
	})

	if err != nil {
		taskModel.MarkAsFailed(fmt.Errorf("database transaction failed: %v", err))
		return err
	}

	// [Stage 6] 全部完成，进度 -> 100%
	resultInfo := fmt.Sprintf(`{"extracted_count": %d, "provider": "%s"}`, count, aiConfig.Provider)
	taskModel.MarkAsSuccess(resultInfo)

	console.Success(fmt.Sprintf("任务[%d] - 场景提取完成，新增 %d 个场景", p.AsyncTaskID, count))
	return nil
}

// 辅助函数保持不变
func getSceneFormatInstructions(isEnglish bool) string {
	if isEnglish {
		return `[Output JSON Format]
{
  "backgrounds": [
    {
      "location": "Location Name",
      "time": "Time Description",
      "atmosphere": "Atmosphere",
      "prompt": "Detailed English image generation prompt..."
    }
  ]
}`
	}
	return `【输出JSON格式】
{
  "backgrounds": [
    {
      "location": "地点名称",
      "time": "时间描述",
      "atmosphere": "氛围",
      "prompt": "详细的中文图片生成提示词，纯背景，无人物..."
    }
  ]
}`
}
