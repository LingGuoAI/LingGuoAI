package v1

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"spiritFruit/app/models/async_tasks"
	"spiritFruit/app/models/characters"
	"spiritFruit/app/models/projects"
	"spiritFruit/app/models/props"
	"spiritFruit/app/models/scenes"
	"spiritFruit/app/models/scripts"
	"spiritFruit/app/models/shots"
	"spiritFruit/app/requests"
	"spiritFruit/app/services"
	"spiritFruit/pkg/asynq"
	"spiritFruit/pkg/console"
	"spiritFruit/pkg/database"
	"spiritFruit/pkg/openai"
	"spiritFruit/pkg/response"
	"spiritFruit/pkg/video"
	"strings"
	"time"
)

type AiController struct {
	BaseADMINController
}

// GenerateCharacters 异步生成角色
func (ctrl *AiController) GenerateCharacters(c *gin.Context) {
	// 1. 验证参数
	request := requests.GenerateCharactersRequest{}
	if ok := requests.Validate(c, &request, requests.ValidateGenerateCharacters); !ok {
		return
	}

	// 2. 调用 TaskService
	taskService := new(services.TaskService)
	adminID := ctrl.GetAdminID(c)
	task, err := taskService.CreateGenerateCharactersTask(adminID, uint64(request.ProjectId), request.Count, request.Outline)

	if err != nil {
		response.Abort500(c, "任务启动失败: "+err.Error())
		return
	}

	// 3. 返回结果 (taskId 是数据库主键 ID)
	response.JSON(c, gin.H{
		"status":  0,
		"message": "角色生成任务已在后台运行",
		"data": map[string]interface{}{
			"task_id": task.ID,
			"status":  task.Status,
		},
	})
}

// ExtractScenes 异步提取场景
func (ctrl *AiController) ExtractScenes(c *gin.Context) {
	request := requests.ExtractScenesRequest{}
	if ok := requests.Validate(c, &request, requests.ValidateExtractScenes); !ok {
		return
	}

	// 查询关联的项目ID
	var scriptsInfo scripts.Scripts
	if err := database.DB.First(&scriptsInfo, request.ScriptId).Error; err != nil {
		response.Abort500(c, "未找到对应章节")
		return
	}
	// 注意：episode.ProjectId 是指针
	var projectID uint64
	if scriptsInfo.ProjectId != nil {
		projectID = *scriptsInfo.ProjectId
	}

	// 2. 调用 TaskService
	taskService := new(services.TaskService)
	adminID := ctrl.GetAdminID(c)
	task, err := taskService.CreateExtractScenesTask(adminID, projectID, uint64(request.ScriptId))

	if err != nil {
		response.Abort500(c, "任务启动失败: "+err.Error())
		return
	}

	// 3. 返回结果
	response.JSON(c, gin.H{
		"status":  0,
		"message": "场景提取任务已在后台运行",
		"data": map[string]interface{}{
			"task_id": task.ID,
			"status":  task.Status,
		},
	})
}

// GenerateCharacterImage 生成角色图片
func (ctrl *AiController) GenerateCharacterImage(c *gin.Context) {
	// 1. 定义请求参数
	type Req struct {
		CharacterID uint64 `json:"characterId" binding:"required"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Abort500(c, "参数错误: "+err.Error())
		return
	}

	// 2. 查询角色信息
	var char characters.Characters
	if err := database.DB.First(&char, req.CharacterID).Error; err != nil {
		response.Abort500(c, "角色不存在")
		return
	}

	// 初始化 ProjectId
	projectID := uint64(0)
	if char.ProjectId != nil {
		projectID = *char.ProjectId
	}

	projectStyle := ""
	// 只有当角色归属于某个项目时，才去查询项目风格
	if projectID > 0 {
		var proj projects.Projects
		// 尝试查询项目信息
		if err := database.DB.Select("style").First(&proj, projectID).Error; err == nil {
			// 如果查询成功且 Style 字段不为 nil，获取其值
			if proj.Style != nil {
				projectStyle = *proj.Style
			}
		} else {
			// 这里可以选择记录日志，或者视作非致命错误继续
			fmt.Printf("Warning: Failed to load project style for character %d: %v\n", req.CharacterID, err)
		}
	}

	// --- 确定基础描述 (VisualPrompt 或 AppearanceDesc) ---
	basePrompt := ""
	if char.VisualPrompt != nil && *char.VisualPrompt != "" {
		basePrompt = *char.VisualPrompt
	} else if char.AppearanceDesc != nil && *char.AppearanceDesc != "" {
		// 如果没有专门的视觉提示词，回退到外观描述
		basePrompt = *char.AppearanceDesc
	}

	// 如果两项描述都为空，则无法生成
	if basePrompt == "" {
		response.Abort500(c, "角色缺少外貌描述或视觉提示词，无法生成")
		return
	}

	// --- 组合最终提示词 ---
	finalPrompt := ""
	if projectStyle != "" {
		// 如果有项目风格，将其作为高质量前缀，用逗号分隔
		finalPrompt = fmt.Sprintf("%s, %s", projectStyle, basePrompt)
	} else {
		finalPrompt = basePrompt
	}

	// 记录一下最终的提示词，方便调试
	fmt.Printf("Final generated prompt for char %d: %s\n", req.CharacterID, finalPrompt)

	// --- 优化部分结束 ---

	// 3. 调用 Service 创建任务
	taskService := new(services.TaskService)
	adminID := ctrl.GetAdminID(c)
	task, err := taskService.CreateImageGenerationTask(adminID, projectID, req.CharacterID, finalPrompt)

	if err != nil {
		response.Abort500(c, "任务启动失败: "+err.Error())
		return
	}

	// 4. 返回结果
	response.JSON(c, gin.H{
		"code":    200,
		"message": "图片生成任务已在后台运行",
		"data": map[string]interface{}{
			"task_id": task.ID,
			"status":  task.Status,
		},
	})
}

// BatchGenerateCharacterImages 批量生成角色图片
func (ctrl *AiController) BatchGenerateCharacterImages(c *gin.Context) {
	// 1. 定义请求参数：接收 ID 数组
	type BatchReq struct {
		CharacterIDs []uint64 `json:"characterIds" binding:"required,min=1,max=10"` // 限制一次最多10个，防止超时
	}
	var req BatchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Abort500(c, "参数错误: "+err.Error())
		return
	}

	// 2. 准备返回结果结构
	type TaskResult struct {
		CharacterID uint64 `json:"character_id"`
		TaskID      uint64 `json:"task_id"`
		Status      int    `json:"status"`
	}
	var results []TaskResult
	taskService := new(services.TaskService)

	// 3. 遍历 ID，逐个创建任务
	// 注意：这里是在 Controller 层循环调用 Service。
	// 虽然不是最高效（比如可以做批量 Insert），但对于 Asynq 任务投递来说，逐个投递更稳健，
	// 且能确保每个角色都有独立的 TaskID 供前端追踪进度。
	for _, charID := range req.CharacterIDs {
		// A. 查询角色信息获取 Prompt
		var char characters.Characters
		if err := database.DB.First(&char, charID).Error; err != nil {
			// 如果某个角色没找到，记录错误或跳过，这里选择跳过
			continue
		}

		prompt := ""
		if char.VisualPrompt != nil {
			prompt = *char.VisualPrompt
		}
		if prompt == "" && char.AppearanceDesc != nil {
			prompt = *char.AppearanceDesc
		}
		if prompt == "" {
			continue // 无描述无法生成
		}

		projectID := uint64(0)
		if char.ProjectId != nil {
			projectID = *char.ProjectId
		}

		// B. 创建任务
		adminID := ctrl.GetAdminID(c)
		task, err := taskService.CreateImageGenerationTask(adminID, projectID, charID, prompt)
		if err == nil {
			results = append(results, TaskResult{
				CharacterID: charID,
				TaskID:      task.ID,
				Status:      task.Status,
			})
		}
	}

	// 4. 返回结果列表
	response.JSON(c, gin.H{
		"status":  200,
		"message": "批量任务已提交",
		"data":    results,
	})
}

// GenerateSceneImage 单个场景生图
func (ctrl *AiController) GenerateSceneImage(c *gin.Context) {
	type Req struct {
		SceneID uint64 `json:"sceneId" binding:"required"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Abort500(c, "参数错误")
		return
	}

	// 1. 查询场景信息
	var scene scenes.Scenes
	if err := database.DB.First(&scene, req.SceneID).Error; err != nil {
		response.Abort500(c, "场景不存在")
		return
	}

	// 2. 准备 Prompt (优先用 VisualPrompt，如果没有则用 Atmosphere)
	// 如果 VisualPrompt 已经是 URL 了 (http开头)，需要处理这种情况
	// 通常这里应该有一个原始 Prompt 字段，或者重新拼接
	prompt := ""
	if scene.VisualPrompt != nil && *scene.VisualPrompt != "" {
		// 简单的判断，如果不是 URL，则认为是 Prompt
		if len(*scene.VisualPrompt) < 4 || (*scene.VisualPrompt)[:4] != "http" && (*scene.VisualPrompt)[0] != '/' {
			prompt = *scene.VisualPrompt
		}
	}
	// 如果 VisualPrompt 是空的或者是 URL，尝试使用 Atmosphere + Location + Time
	if prompt == "" {
		loc := ""
		if scene.Location != nil {
			loc = *scene.Location
		}
		tm := ""
		if scene.Time != nil {
			tm = *scene.Time
		}
		atm := ""
		if scene.Atmosphere != nil {
			atm = *scene.Atmosphere
		}

		prompt = fmt.Sprintf("%s, %s, %s", loc, tm, atm)
	}

	projectID := uint64(0)
	if scene.ProjectId != nil {
		projectID = *scene.ProjectId
	}

	// 3. 创建任务
	taskService := new(services.TaskService)
	adminID := ctrl.GetAdminID(c)
	task, err := taskService.CreateSceneImageGenerationTask(adminID, projectID, req.SceneID, prompt)
	if err != nil {
		response.Abort500(c, "任务启动失败: "+err.Error())
		return
	}

	response.JSON(c, gin.H{
		"code":    0,
		"message": "任务已提交",
		"data": map[string]interface{}{
			"task_id": task.ID,
		},
	})
}

// BatchGenerateSceneImages 批量场景生图
func (ctrl *AiController) BatchGenerateSceneImages(c *gin.Context) {
	type Req struct {
		SceneIDs []uint64 `json:"sceneIds" binding:"required,min=1,max=20"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Abort500(c, "参数错误: "+err.Error())
		return
	}

	type TaskResult struct {
		SceneID uint64 `json:"scene_id"`
		TaskID  uint64 `json:"task_id"`
	}
	var results []TaskResult
	taskService := new(services.TaskService)

	for _, sceneID := range req.SceneIDs {
		var scene scenes.Scenes
		if err := database.DB.First(&scene, sceneID).Error; err != nil {
			continue
		}

		// 构造 Prompt (逻辑同上)
		prompt := ""
		if scene.VisualPrompt != nil && *scene.VisualPrompt != "" && (*scene.VisualPrompt)[0] != '/' && (*scene.VisualPrompt)[:4] != "http" {
			prompt = *scene.VisualPrompt
		}
		if prompt == "" {
			loc := ""
			if scene.Location != nil {
				loc = *scene.Location
			}
			tm := ""
			if scene.Time != nil {
				tm = *scene.Time
			}
			atm := ""
			if scene.Atmosphere != nil {
				atm = *scene.Atmosphere
			}
			prompt = fmt.Sprintf("%s, %s, %s", loc, tm, atm)
		}

		projectID := uint64(0)
		if scene.ProjectId != nil {
			projectID = *scene.ProjectId
		}

		task, err := taskService.CreateSceneImageGenerationTask(ctrl.GetAdminID(c), projectID, sceneID, prompt)
		if err == nil {
			results = append(results, TaskResult{
				SceneID: sceneID,
				TaskID:  task.ID,
			})
		}
	}

	response.JSON(c, gin.H{
		"code":    0,
		"message": "批量任务已提交",
		"data":    results,
	})
}

// GenerateShots 智能拆分分镜
func (ctrl *AiController) GenerateShots(c *gin.Context) {
	type Req struct {
		ScriptID uint64 `json:"scriptId" binding:"required"`
		Model    string `json:"model"` // 可选
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Abort500(c, "参数错误")
		return
	}

	// 1. 校验剧本是否存在
	var script scripts.Scripts
	if err := database.DB.First(&script, req.ScriptID).Error; err != nil {
		response.Abort500(c, "剧本不存在")
		return
	}

	projectID := uint64(0)
	if script.ProjectId != nil {
		projectID = *script.ProjectId
	}

	// 2. 创建任务
	taskService := new(services.TaskService)
	adminID := ctrl.GetAdminID(c)
	task, err := taskService.CreateGenerateShotsTask(adminID, projectID, req.ScriptID, req.Model)
	if err != nil {
		response.Abort500(c, "任务启动失败: "+err.Error())
		return
	}

	response.JSON(c, gin.H{
		"code":    0,
		"message": "分镜拆分任务已提交",
		"data": map[string]interface{}{
			"task_id": task.ID,
		},
	})
}

// ExtractProps 从剧本提取道具
func (ctrl *AiController) ExtractProps(c *gin.Context) {
	type Req struct {
		EpisodeID uint64 `json:"episodeId" binding:"required"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Abort500(c, "参数错误: "+err.Error())
		return
	}

	// 查剧本信息
	var script scripts.Scripts
	if err := database.DB.First(&script, req.EpisodeID).Error; err != nil {
		response.Abort500(c, "剧本不存在")
		return
	}

	taskService := new(services.TaskService)
	projectID := uint64(0)
	if script.ProjectId != nil {
		projectID = *script.ProjectId
	}

	// 创建任务
	adminID := ctrl.GetAdminID(c)
	task, err := taskService.CreateExtractPropsTask(adminID, projectID, req.EpisodeID)
	if err != nil {
		response.Abort500(c, "任务创建失败: "+err.Error())
		return
	}

	response.JSON(c, gin.H{
		"code": 0,
		"data": gin.H{
			"task_id": task.ID,
		},
		"message": "道具提取任务已提交",
	})
}

// GeneratePropImage 单个道具生图
func (ctrl *AiController) GeneratePropImage(c *gin.Context) {
	type Req struct {
		PropID uint64 `json:"propId" binding:"required"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Abort500(c, "参数错误: "+err.Error())
		return
	}

	// 查找道具
	var prop props.Props
	if err := database.DB.First(&prop, req.PropID).Error; err != nil {
		response.Abort500(c, "道具不存在")
		return
	}

	// 构造 Prompt
	prompt := ""
	if prop.ImagePrompt != nil && *prop.ImagePrompt != "" {
		prompt = *prop.ImagePrompt
	} else {
		// 如果没有专门的 ImagePrompt，使用描述或名字兜底
		desc := ""
		if prop.Description != nil {
			desc = *prop.Description
		}
		name := ""
		if prop.Name != nil {
			name = *prop.Name
		}
		prompt = fmt.Sprintf("%s, %s", name, desc)
	}

	taskService := new(services.TaskService)
	projectID := uint64(0)
	if prop.ProjectId != nil {
		projectID = *prop.ProjectId
	}

	adminID := ctrl.GetAdminID(c)
	task, err := taskService.CreatePropImageGenerationTask(adminID, projectID, req.PropID, prompt)
	if err != nil {
		response.Abort500(c, "任务创建失败: "+err.Error())
		return
	}

	response.JSON(c, gin.H{
		"code":    0,
		"data":    gin.H{"task_id": task.ID},
		"message": "生图任务已提交",
	})
}

// BatchGeneratePropImages 批量道具生图
func (ctrl *AiController) BatchGeneratePropImages(c *gin.Context) {
	type Req struct {
		PropIDs []uint64 `json:"propIds" binding:"required,min=1,max=20"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Abort500(c, "参数错误: "+err.Error())
		return
	}

	type TaskResult struct {
		PropID uint64 `json:"prop_id"`
		TaskID uint64 `json:"task_id"`
	}
	var results []TaskResult
	taskService := new(services.TaskService)

	for _, propID := range req.PropIDs {
		var prop props.Props
		if err := database.DB.First(&prop, propID).Error; err != nil {
			continue
		}

		// 构造 Prompt
		prompt := ""
		if prop.ImagePrompt != nil && *prop.ImagePrompt != "" {
			prompt = *prop.ImagePrompt
		} else {
			desc := ""
			if prop.Description != nil {
				desc = *prop.Description
			}
			name := ""
			if prop.Name != nil {
				name = *prop.Name
			}
			prompt = fmt.Sprintf("%s, %s", name, desc)
		}

		projectID := uint64(0)
		if prop.ProjectId != nil {
			projectID = *prop.ProjectId
		}

		task, err := taskService.CreatePropImageGenerationTask(ctrl.GetAdminID(c), projectID, propID, prompt)
		if err == nil {
			results = append(results, TaskResult{
				PropID: propID,
				TaskID: task.ID,
			})
		}
	}

	response.JSON(c, gin.H{
		"code":    0,
		"message": "批量任务已提交",
		"data":    results,
	})
}

// ExtractPrompt 提取分镜图片提示词
func (ctrl *AiController) ExtractPrompt(c *gin.Context) {
	type Req struct {
		ShotID    uint64 `json:"shotId" binding:"required"`
		FrameType string `json:"frameType" binding:"required,oneof=first last key action panel"`
		Model     string `json:"model"` // 可选
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Abort500(c, "参数错误: "+err.Error())
		return
	}

	// 1. 查找分镜信息，验证是否存在
	var shot shots.Shots
	if err := database.DB.First(&shot, req.ShotID).Error; err != nil {
		response.Abort500(c, "分镜镜头不存在")
		return
	}
	projectID := uint64(0)
	if shot.ProjectId != nil {
		projectID = *shot.ProjectId
	}

	// 2. 创建异步任务
	taskService := new(services.TaskService)
	adminID := ctrl.GetAdminID(c)
	task, err := taskService.CreateExtractFramePromptTask(adminID, projectID, req.ShotID, req.FrameType, req.Model)
	if err != nil {
		response.Abort500(c, "任务创建失败: "+err.Error())
		return
	}

	// 3. 返回任务ID供前端轮询
	response.JSON(c, gin.H{
		"code":    0,
		"message": "提示词提取任务已提交",
		"data": map[string]interface{}{
			"task_id": task.ID,
			"status":  task.Status,
		},
	})
}

// GenerateImageByPrompt 根据帧提示词生成图片
func (ctrl *AiController) GenerateImageByPrompt(c *gin.Context) {
	type Req struct {
		ShotID    uint64 `json:"shotId" binding:"required"`
		FrameType string `json:"frameType" binding:"required,oneof=first last action key"`
		Prompt    string `json:"prompt" binding:"required"` // 前端传递要生成的最终提示词
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Abort500(c, "参数错误: "+err.Error())
		return
	}

	// 1. 获取关联项目ID
	var shot shots.Shots
	if err := database.DB.First(&shot, req.ShotID).Error; err != nil {
		response.Abort500(c, "分镜不存在")
		return
	}
	projectID := uint64(0)
	if shot.ProjectId != nil {
		projectID = *shot.ProjectId
	}

	// 2. 调用 Service 创建任务
	taskService := new(services.TaskService)
	adminID := ctrl.GetAdminID(c)
	task, err := taskService.CreateGenerateFrameImageTask(adminID, projectID, req.ShotID, req.FrameType, req.Prompt)
	if err != nil {
		response.Abort500(c, "任务启动失败: "+err.Error())
		return
	}

	// 3. 返回结果给前端进行轮询
	response.JSON(c, gin.H{
		"code":    0,
		"message": "图片生成任务已在后台运行",
		"data": map[string]interface{}{
			"task_id": task.ID,
		},
	})
}

// GenerateVideo 根据参数和提示词生成视频
func (ctrl *AiController) GenerateVideo(c *gin.Context) {
	// 定义对应前端传递的复杂请求体
	type Req struct {
		ProjectID     uint64   `json:"projectId" binding:"required"`
		ShotID        uint64   `json:"shotId" binding:"required"`
		Model         string   `json:"model" binding:"required"`
		Duration      int      `json:"duration"`
		Prompt        string   `json:"prompt" binding:"required"`
		ReferenceMode string   `json:"referenceMode" binding:"required"`
		ImageURL      string   `json:"imageUrl"`
		FirstFrameURL string   `json:"firstFrameUrl"`
		LastFrameURL  string   `json:"lastFrameUrl"`
		ImageURLs     []string `json:"imageUrls"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Abort500(c, "参数错误: "+err.Error())
		return
	}

	// 1. 验证分镜是否存在
	var shot shots.Shots
	if err := database.DB.First(&shot, req.ShotID).Error; err != nil {
		response.Abort500(c, "分镜不存在")
		return
	}
	if shot.ProjectId == nil || *shot.ProjectId != req.ProjectID {
		response.Abort400(c, "分镜不属于当前项目")
		return
	}
	// 镜头属性是时长的唯一数据源，忽略前端可能缓存的独立时长。
	duration := 3
	if shot.DurationMs != nil && *shot.DurationMs > 0 {
		duration = int((*shot.DurationMs + 500) / 1000)
		if duration < 1 {
			duration = 1
		}
	}

	// 2. 组装 Payload
	payload := asynq.GenerateVideoPayload{
		ProjectID:          req.ProjectID,
		ShotID:             req.ShotID,
		Model:              req.Model,
		Duration:           duration,
		Prompt:             req.Prompt,
		ReferenceMode:      req.ReferenceMode,
		ImageURL:           req.ImageURL,
		FirstFrameURL:      req.FirstFrameURL,
		LastFrameURL:       req.LastFrameURL,
		ReferenceImageURLs: req.ImageURLs,
	}

	// 3. 调用 Service 创建并投递任务
	taskService := new(services.TaskService)
	adminID := ctrl.GetAdminID(c)
	task, err := taskService.CreateGenerateVideoTask(adminID, payload)
	if err != nil {
		response.Abort500(c, "视频生成任务启动失败: "+err.Error())
		return
	}

	// 4. 返回 TaskID 给前端进行轮询
	response.JSON(c, gin.H{
		"code":    0,
		"message": "视频生成任务已提交",
		"data": map[string]interface{}{
			"task_id": task.ID,
		},
	})
}

// TestConfigReq 前端传来的配置数据
type TestConfigReq struct {
	BaseURL  string   `json:"base_url" binding:"required"`
	APIKey   string   `json:"api_key" binding:"required"`
	Provider string   `json:"provider" binding:"required"`
	Model    []string `json:"model" binding:"required"`
}

// helper: 构建测试用的生文/生图配置
func buildTestAiConfig(req TestConfigReq, serviceType string) openai.Config {
	aiConfig := openai.Config{
		Provider: strings.ToLower(req.Provider),
	}

	modelName := ""
	if len(req.Model) > 0 {
		modelName = req.Model[0]
	}

	providerName := aiConfig.Provider

	switch providerName {
	case "getgoapi":
		aiConfig.GetGoAPIBaseURL = req.BaseURL
		aiConfig.GetGoAPIKey = req.APIKey
		if serviceType == "image" {
			aiConfig.GetGoAPIImageModel = modelName
		} else {
			aiConfig.GetGoAPIModel = modelName
		}
		// 同时也给 OpenAI 字段赋值，以防万一内部有引用
		aiConfig.OpenAIBaseURL = req.BaseURL
		aiConfig.OpenAIKey = req.APIKey

	case "openai":
		aiConfig.OpenAIBaseURL = req.BaseURL
		aiConfig.OpenAIKey = req.APIKey
		if serviceType == "image" {
			aiConfig.OpenAIImageModel = modelName
		} else {
			aiConfig.OpenAIModel = modelName
		}

	case "gemini", "google":
		aiConfig.GeminiBaseURL = req.BaseURL
		aiConfig.GeminiKey = req.APIKey
		aiConfig.GeminiModel = modelName

	case "doubao", "volcengine", "volces":
		aiConfig.DoubaoBaseURL = req.BaseURL
		aiConfig.DoubaoKey = req.APIKey
		if serviceType == "image" {
			aiConfig.DoubaoImageModel = modelName
		} else {
			aiConfig.DoubaoModel = modelName
		}

	case "vertex":
		aiConfig.VertexKey = req.APIKey
		if serviceType == "image" {
			aiConfig.VertexImageModel = modelName
		} else {
			aiConfig.VertexModel = modelName
		}

	default:
		// 默认走 OpenAI 协议
		aiConfig.Provider = "openai"
		aiConfig.OpenAIBaseURL = req.BaseURL
		aiConfig.OpenAIKey = req.APIKey
		if serviceType == "image" {
			aiConfig.OpenAIImageModel = modelName
		} else {
			aiConfig.OpenAIModel = modelName
		}
	}
	return aiConfig
}

// TestTextConfig 测试文本大模型配置
func (ctrl *AiController) TestTextConfig(c *gin.Context) {
	var req TestConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Abort500(c, "参数错误: "+err.Error())
		return
	}

	aiConfig := buildTestAiConfig(req, "text")
	aiProvider := openai.NewProviderWithTimeout(aiConfig, 20*time.Second)

	// 发起轻量级文本请求
	scriptReq := openai.ScriptRequest{
		Messages: []openai.ChatMessage{
			{Role: "user", Content: "你好，请用一句话回答：大模型连接测试成功。"},
		},
		Temperature: 0.5,
	}

	aiResp, err := aiProvider.GenerateScript(scriptReq)
	if err != nil {
		// 上游 AI 服务不可达/返回错误属于网关错误，不应伪装成应用内部异常。
		c.AbortWithStatusJSON(502, gin.H{"code": 502, "message": "文本模型连接失败: " + err.Error()})
		return
	}

	response.JSON(c, gin.H{
		"code":    0,
		"message": "连接成功",
		"data": map[string]interface{}{
			"reply": aiResp, // 直接返回 AI 的回答
		},
	})
}

// TestImageConfig 测试生图大模型配置
func (ctrl *AiController) TestImageConfig(c *gin.Context) {
	var req TestConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Abort500(c, "参数错误: "+err.Error())
		return
	}
	aiConfig := buildTestAiConfig(req, "image")
	aiProvider := openai.NewProvider(aiConfig)

	// 发起一张简单的低分辨率生图请求以节省时间和成本
	imageReq := openai.ImageRequest{
		Prompt: "A cute little cat, minimalist vector art, clean background",
		N:      1,
		Size:   "1024x1024",
	}
	urls, err := aiProvider.GenerateImage(imageReq)
	if err != nil {
		response.Abort500(c, "图片模型连接失败: "+err.Error())
		return
	}
	if len(urls) == 0 {
		response.Abort500(c, "图片模型连接成功，但未返回图片数据")
		return
	}
	response.JSON(c, gin.H{
		"code":    0,
		"message": "连接成功",
		"data": map[string]interface{}{
			"image_url": urls[0], // 返回生成的图片URL(或Base64)
		},
	})
}

// TestVideoConfig 测试生视频大模型配置
func (ctrl *AiController) TestVideoConfig(c *gin.Context) {
	var req TestConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Abort500(c, "参数错误: "+err.Error())
		return
	}

	// 1. 将 Provider 映射为 video.NewClient 支持的底层驱动名称
	providerName := strings.ToLower(req.Provider)
	endpoint := ""
	switch providerName {
	case "volcengine", "volces", "doubao":
		providerName = "volces"
		endpoint = "/contents/generations/tasks"
	case "chatfire", "getgoapi":
		providerName = "getgoapi"
	case "google", "gemini", "vertex":
		providerName = "vertex"
	case "minimax", "hailuo":
		providerName = "minimax"
	case "openai", "runway", "pika":
		// 这些名字可以直接透传给 video.NewClient，保持原样
		providerName = providerName
	default:
		// 无法识别时兜底为 getgoapi
		providerName = "getgoapi"
	}

	modelName := ""
	if len(req.Model) > 0 {
		modelName = req.Model[0]
	}

	// 2. 初始化视频客户端
	client, err := video.NewClient(providerName, req.BaseURL, req.APIKey, modelName, endpoint, "")
	if err != nil {
		response.Abort500(c, "视频客户端初始化失败: "+err.Error())
		return
	}

	// 智能判断测试时长 (Sora 限制较多，一般测短一点)
	testDuration := 5
	if strings.Contains(strings.ToLower(modelName), "sora") {
		testDuration = 4
	}

	// 3. 在本地数据库创建测试任务记录 (让前端去轮询这个内部ID)
	task := async_tasks.AsyncTask{
		ProjectID:  0, // 测试任务，无实际项目归属
		Type:       "test_video_config",
		Status:     async_tasks.StatusProcessing, // 设置为进行中 (通常是 1 或 2，视你系统定)
		Process:    10,                           // 初始进度
		Payload:    "{}",
		StatusName: async_tasks.StatusNameRunning,
	}
	if err := database.DB.Create(&task).Error; err != nil {
		response.Abort500(c, "创建测试记录失败: "+err.Error())
		return
	}

	// 4. 提交测试任务到厂商
	result, err := client.GenerateVideo("A fast running dog in a green field", video.WithDuration(testDuration))
	if err != nil {
		// 提交被拒，更新数据库状态并返回报错
		task.MarkAsFailed(err) // 假设你的 AsyncTask 模型有这个辅助方法
		response.Abort500(c, "视频任务提交被拒 (请检查URL/Key/模型名): "+err.Error())
		return
	}

	// 5. 开启后台协程，轮询算力厂商的任务进度
	go func(internalID uint64, externalID string) {
		// 🔴 必备：防止协程 Panic 导致整个 Go 服务崩溃
		defer func() {
			if r := recover(); r != nil {
				console.Error(fmt.Sprintf("视频测试轮询协程崩溃: %v", r))
				database.DB.Model(&async_tasks.AsyncTask{}).Where("id = ?", internalID).Updates(map[string]interface{}{
					"status":    async_tasks.StatusFailed, // 通常是 3 或 4
					"error_msg": "内部系统错误 (Panic)",
				})
			}
		}()

		maxAttempts := 150
		interval := 10 * time.Second

		for attempt := 0; attempt < maxAttempts; attempt++ {
			time.Sleep(interval)

			statusRes, checkErr := client.GetTaskStatus(externalID)
			if checkErr != nil {
				continue // 网络波动，忽略本次错误，继续下一次查
			}

			// 如果厂商明确返回失败
			if statusRes.Error != "" {
				database.DB.Model(&async_tasks.AsyncTask{}).Where("id = ?", internalID).Updates(map[string]interface{}{
					"status":      async_tasks.StatusFailed,
					"status_name": async_tasks.StatusNameFailed,
					"error_msg":   statusRes.Error,
					"finished_at": time.Now(),
				})
				return
			}

			// 成功获取到视频地址
			if statusRes.Completed && statusRes.VideoURL != "" {
				resBytes, _ := json.Marshal(map[string]interface{}{
					"video_url": statusRes.VideoURL,
				})
				database.DB.Model(&async_tasks.AsyncTask{}).Where("id = ?", internalID).Updates(map[string]interface{}{
					"status":      async_tasks.StatusSuccess,
					"status_name": async_tasks.StatusNameSucceeded,
					"result":      string(resBytes),
					"process":     100,
					"finished_at": time.Now(),
				})
				return
			}

			// 🔴 修复进度条计算逻辑，将进度平滑控制在 20% ~ 99% 之间
			prog := 20 + int(float64(attempt)/float64(maxAttempts)*79)
			database.DB.Model(&async_tasks.AsyncTask{}).Where("id = ?", internalID).Update("process", prog)
		}

		// 超过轮询次数 (超时)
		database.DB.Model(&async_tasks.AsyncTask{}).Where("id = ?", internalID).Updates(map[string]interface{}{
			"status":      async_tasks.StatusFailed,
			"status_name": async_tasks.StatusNameFailed,
			"error_msg":   "轮询任务超时，未能获取到视频",
			"finished_at": time.Now(),
		})
	}(task.ID, result.TaskID)

	// 6. 给前端返回【本地数据库的TaskID】
	response.JSON(c, gin.H{
		"code":    0,
		"message": "连接成功",
		"data": map[string]interface{}{
			"task_id": task.ID,
			"status":  "任务已提交，正在轮询",
		},
	})
}
