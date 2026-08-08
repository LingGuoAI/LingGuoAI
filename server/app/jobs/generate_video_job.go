// app/jobs/generate_video_job.go
package jobs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hibiken/asynq"

	"spiritFruit/app/models/async_tasks"
	"spiritFruit/app/models/projects"
	"spiritFruit/app/models/shot_frame_image"
	"spiritFruit/app/models/shots"
	"spiritFruit/app/services"
	myAsynq "spiritFruit/pkg/asynq"
	"spiritFruit/pkg/config"
	"spiritFruit/pkg/console"
	"spiritFruit/pkg/database"
	"spiritFruit/pkg/prompt"
	"spiritFruit/pkg/upload"
	"spiritFruit/pkg/video"
)

// VideoAPIConfig 准备视频 AI 配置结构体
type VideoAPIConfig struct {
	GetGoBaseURL   string
	GetGoKey       string
	VolcesBaseURL  string
	VolcesKey      string
	MinimaxBaseURL string
	MinimaxKey     string
	RunwayBaseURL  string
	RunwayKey      string
	PikaBaseURL    string
	PikaKey        string
	OpenAIBaseURL  string
	OpenAIKey      string
	VertexKey      string
}

// 🔴 helper: 将数据库的 provider 映射为 video.NewClient 底层包识别的名称
func mapDBProvider(dbProvider string, modelName string) string {
	dbProvider = strings.ToLower(dbProvider)
	switch dbProvider {
	case "volcengine", "volces", "doubao":
		return "volces"
	case "openai":
		return "openai"
	case "vertex", "google", "gemini":
		return "vertex"
	case "getgoapi":
		return "getgoapi"
	case "runway":
		return "runway"
	case "pika":
		return "pika"
	case "minimax", "hailuo":
		return "minimax"
	default:
		// 未知提供商时，尽量根据模型名推断，推断不出就兜底中转商
		lowerModel := strings.ToLower(modelName)
		if strings.Contains(lowerModel, "veo") || strings.Contains(lowerModel, "vertex") {
			return "vertex"
		} else if strings.Contains(lowerModel, "doubao") || strings.Contains(lowerModel, "seedance") {
			return "volces"
		} else if strings.Contains(lowerModel, "sora") {
			return "openai"
		}
		return "getgoapi"
	}
}

// 🔴 helper: 优先从数据库动态查询，无配置则根据模型名字兜底 .env
func getProviderConfig(modelName string, adminID *uint64) (provider, baseURL, endpointValue, apiKey string, finalModel string) {
	finalModel = modelName
	var endpoint string
	// 1. 优先查库：获取所有激活的 video 配置
	aiService := new(services.AiConfigService)
	err, configs := aiService.GetAllActiveConfigsByType("video", adminID)

	if err == nil && len(configs) > 0 {
		// 1.1 如果前端传了指定的 modelName，尝试在配置中精确寻找包含该模型的厂商
		if finalModel != "" {
			for _, cfg := range configs {
				for _, m := range cfg.Model {
					if m == finalModel {
						if strings.Contains(finalModel, "doubao") || strings.Contains(finalModel, "seedance") {
							endpoint = "/contents/generations/tasks"
						}
						return mapDBProvider(*cfg.Provider, finalModel), *cfg.BaseUrl, endpoint, *cfg.ApiKey, finalModel
					}
				}
			}
		}

		// 1.2 如果没找到精确匹配，或者没有传 modelName，直接使用优先级最高的第一个配置
		topCfg := configs[0]
		if finalModel == "" && len(topCfg.Model) > 0 {
			finalModel = topCfg.Model[0] // 取数组第一个模型
			if strings.Contains(finalModel, "doubao") || strings.Contains(finalModel, "seedance") {
				endpoint = "/contents/generations/tasks"
			}
		}

		return mapDBProvider(*topCfg.Provider, finalModel), *topCfg.BaseUrl, endpoint, *topCfg.ApiKey, finalModel
	}

	// 2. 降级查 .env
	cfg := VideoAPIConfig{
		GetGoBaseURL:   config.GetString("ai.getgoapi.base_url"),
		GetGoKey:       config.GetString("ai.getgoapi.api_key"),
		VolcesBaseURL:  config.GetString("ai.volces.base_url"),
		VolcesKey:      config.GetString("ai.volces.api_key"),
		MinimaxBaseURL: config.GetString("ai.minimax.base_url"),
		MinimaxKey:     config.GetString("ai.minimax.api_key"),
		RunwayBaseURL:  config.GetString("ai.runway.base_url"),
		RunwayKey:      config.GetString("ai.runway.api_key"),
		PikaBaseURL:    config.GetString("ai.pika.base_url"),
		PikaKey:        config.GetString("ai.pika.api_key"),
		OpenAIBaseURL:  config.GetString("ai.openai.base_url"),
		OpenAIKey:      config.GetString("ai.openai.api_key"),
		VertexKey:      config.GetString("ai.vertex.api_key"),
	}

	lowerModel := strings.ToLower(finalModel)
	if strings.Contains(lowerModel, "veo") || strings.Contains(lowerModel, "vertex") {
		return "vertex", "", endpoint, cfg.VertexKey, finalModel
	} else if strings.Contains(lowerModel, "doubao") || strings.Contains(lowerModel, "seedance") {
		endpoint = "/contents/generations/tasks"
		return "volces", cfg.VolcesBaseURL, endpoint, cfg.VolcesKey, finalModel
	} else if strings.Contains(lowerModel, "sora") {
		return "openai", cfg.OpenAIBaseURL, endpoint, cfg.OpenAIKey, finalModel
	} else if strings.Contains(lowerModel, "runway") {
		return "runway", cfg.RunwayBaseURL, endpoint, cfg.RunwayKey, finalModel
	} else if strings.Contains(lowerModel, "pika") {
		return "pika", cfg.PikaBaseURL, endpoint, cfg.PikaKey, finalModel
	} else if strings.Contains(lowerModel, "minimax") || strings.Contains(lowerModel, "hailuo") {
		return "minimax", cfg.MinimaxBaseURL, endpoint, cfg.MinimaxKey, finalModel
	}

	// 兜底默认使用 getgoapi 中转
	return "getgoapi", cfg.GetGoBaseURL, endpoint, cfg.GetGoKey, finalModel
}

// HandleGenerateVideoTask 处理视频生成任务
func HandleGenerateVideoTask(ctx context.Context, t *asynq.Task) error {
	var p myAsynq.GenerateVideoPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json unmarshal failed: %v", err)
	}

	// 1. 获取并标记任务开始
	taskModel := async_tasks.AsyncTask{}
	if err := database.DB.First(&taskModel, p.AsyncTaskID).Error; err != nil {
		return nil
	}
	taskModel.MarkAsProcessing()
	console.Success(fmt.Sprintf("任务[%d] - 开始生成视频 (ShotID: %d, 请求Model: %s)", p.AsyncTaskID, p.ShotID, p.Model))

	// 2. 初始化客户端
	taskModel.UpdateProgress(5)

	// 🔴 使用重构后的辅助函数获取动态配置
	provider, baseURL, endpoint, apiKey, finalModel := getProviderConfig(p.Model, taskModel.AdminID)
	console.Success(fmt.Sprintf("任务[%d] - 命中视频配置: Provider=%s, 最终Model=%s, endpoint=%s", p.AsyncTaskID, provider, finalModel, endpoint))

	client, err := video.NewClient(provider, baseURL, apiKey, finalModel, endpoint, "")
	if err != nil {
		taskModel.MarkAsFailed(err)
		return err
	}

	// 3. 预加载数据库关联数据：提取文本特征并收集所有角色的定妆照、道具参考图
	taskModel.UpdateProgress(10)
	var shotModel shots.Shots
	var structuredPrompt string
	var autoRefImages []string // 备用的角色/道具图片池
	projectStyle := "realistic cinematic style"
	aspectRatio := "16:9"
	var projectModel projects.Projects
	if err := database.DB.First(&projectModel, p.ProjectID).Error; err == nil {
		if projectModel.Style != nil && strings.TrimSpace(*projectModel.Style) != "" {
			projectStyle = strings.TrimSpace(*projectModel.Style)
		}
		if projectModel.Settings != nil && *projectModel.Settings != "" {
			var settings map[string]interface{}
			if json.Unmarshal([]byte(*projectModel.Settings), &settings) == nil {
				for _, key := range []string{"aspectRatio", "aspect_ratio", "ratio"} {
					if v, ok := settings[key].(string); ok && v != "" {
						aspectRatio = v
						break
					}
				}
			}
		}
	}
	p.Style = projectStyle
	p.AspectRatio = aspectRatio

	if err := database.DB.Preload("Scenes").Preload("Characters").Preload("Props").First(&shotModel, p.ShotID).Error; err == nil {
		var pb strings.Builder

		// 3.1 组装登场角色与定妆照
		if len(shotModel.Characters) > 0 {
			var charDetails []string
			for _, c := range shotModel.Characters {
				if c.Name != nil && *c.Name != "" {
					desc := *c.Name
					var traits []string
					if c.VisualPrompt != nil && *c.VisualPrompt != "" {
						traits = append(traits, *c.VisualPrompt)
					} else if c.AppearanceDesc != nil && *c.AppearanceDesc != "" {
						traits = append(traits, *c.AppearanceDesc)
					}
					if c.Personality != nil && *c.Personality != "" {
						traits = append(traits, "性格:"+*c.Personality)
					}
					if len(traits) > 0 {
						desc += fmt.Sprintf(" (特征: %s)", strings.Join(traits, ", "))
					}
					charDetails = append(charDetails, desc)
				}
				// 收集定妆照
				if c.AvatarUrl != nil && *c.AvatarUrl != "" {
					autoRefImages = append(autoRefImages, *c.AvatarUrl)
				}
			}
			if len(charDetails) > 0 {
				pb.WriteString(fmt.Sprintf("Characters details: %s. \n", strings.Join(charDetails, "; ")))
			}
		}

		// 3.2 组装相关道具与道具图
		if len(shotModel.Props) > 0 {
			var propDetails []string
			for _, prop := range shotModel.Props {
				if prop.Name != nil && *prop.Name != "" {
					desc := *prop.Name
					var traits []string
					if prop.ImagePrompt != nil && *prop.ImagePrompt != "" {
						traits = append(traits, *prop.ImagePrompt)
					} else if prop.Description != nil && *prop.Description != "" {
						traits = append(traits, *prop.Description)
					}
					if len(traits) > 0 {
						desc += fmt.Sprintf(" (外观: %s)", strings.Join(traits, ", "))
					}
					propDetails = append(propDetails, desc)
				}
				// 收集道具图
				if prop.ImageUrl != nil && *prop.ImageUrl != "" {
					autoRefImages = append(autoRefImages, *prop.ImageUrl)
				}
			}
			if len(propDetails) > 0 {
				pb.WriteString(fmt.Sprintf("Key Props: %s. \n", strings.Join(propDetails, "; ")))
			}
		}

		// 3.3 组装场景与时间
		sceneDesc := ""
		if shotModel.Scenes != nil && shotModel.Scenes.Name != nil {
			sceneDesc += *shotModel.Scenes.Name + "·"
		}
		if shotModel.Location != nil && *shotModel.Location != "" {
			sceneDesc += *shotModel.Location
		}
		timeStr := ""
		if shotModel.Time != nil {
			timeStr = *shotModel.Time
		}
		if sceneDesc != "" || timeStr != "" {
			pb.WriteString(fmt.Sprintf("Scene: %s %s. \n", strings.TrimRight(sceneDesc, "·"), timeStr))
		}

		// 3.4 组装其他关键参数
		if shotModel.Action != nil && *shotModel.Action != "" {
			pb.WriteString(fmt.Sprintf("Action: %s. \n", *shotModel.Action))
		}
		if shotModel.Dialogue != nil && *shotModel.Dialogue != "" {
			pb.WriteString(fmt.Sprintf("Dialogue: %s. \n", *shotModel.Dialogue))
		}
		if shotModel.CameraMovement != nil && *shotModel.CameraMovement != "" {
			pb.WriteString(fmt.Sprintf("Camera movement: %s. ", *shotModel.CameraMovement))
		}
		if shotModel.ShotType != nil && *shotModel.ShotType != "" {
			pb.WriteString(fmt.Sprintf("Shot type: %s. ", *shotModel.ShotType))
		}
		if shotModel.Angle != nil && *shotModel.Angle != "" {
			pb.WriteString(fmt.Sprintf("Camera angle: %s. \n", *shotModel.Angle))
		}
		if shotModel.Atmosphere != nil && *shotModel.Atmosphere != "" {
			pb.WriteString(fmt.Sprintf("Atmosphere: %s. \n", *shotModel.Atmosphere))
		}
		if shotModel.VisualDesc != nil && *shotModel.VisualDesc != "" {
			pb.WriteString(fmt.Sprintf("Result: %s. \n", *shotModel.VisualDesc))
		}
		if shotModel.AudioPrompt != nil && *shotModel.AudioPrompt != "" {
			pb.WriteString(fmt.Sprintf("BGM/Sound effects: %s. \n", *shotModel.AudioPrompt))
		}

		structuredPrompt = strings.TrimSpace(pb.String())
	}

	// 🔴 4. 提前推断参考图模式
	referenceMode := p.ReferenceMode
	if referenceMode == "" {
		if p.ImageURL != "" {
			referenceMode = "single"
		} else if p.FirstFrameURL != "" || p.LastFrameURL != "" {
			referenceMode = "first_last"
		} else if len(p.ReferenceImageURLs) > 0 {
			referenceMode = "multiple"
		} else {
			referenceMode = "none"
		}
	}

	// 检查是否为动作序列(九宫格)
	if referenceMode == "single" && p.ImageURL != "" {
		var frameImg shot_frame_image.ShotFrameImages
		err := database.DB.Where("shot_id = ? AND image_url = ?", p.ShotID, p.ImageURL).First(&frameImg).Error
		if err == nil && frameImg.FrameType != nil && *frameImg.FrameType == "action" {
			referenceMode = "action_sequence"
			console.Success(fmt.Sprintf("任务[%d] - 检测到动作序列图，应用物理动态约束", p.AsyncTaskID))
		}
	}

	// 🔴 5. 图片严格管控（解决 R2V 报错的核心）
	if referenceMode == "multiple" {
		// 只有在多图模式下，才把定妆照/道具照喂给大模型
		urlMap := make(map[string]bool)
		var finalRefURLs []string

		for _, u := range p.ReferenceImageURLs {
			if u != "" && !urlMap[u] {
				finalRefURLs = append(finalRefURLs, u)
				urlMap[u] = true
			}
		}
		for _, u := range autoRefImages {
			if u != "" && !urlMap[u] && u != p.ImageURL && u != p.FirstFrameURL {
				finalRefURLs = append(finalRefURLs, u)
				urlMap[u] = true
			}
		}

		// 多数模型多图模式上限为 4 张
		maxSlots := 4
		if len(finalRefURLs) > maxSlots {
			console.Warning(fmt.Sprintf("任务[%d] - 多图超过限制(%d)，自动截断", p.AsyncTaskID, maxSlots))
			finalRefURLs = finalRefURLs[:maxSlots]
		}
		p.ReferenceImageURLs = finalRefURLs
	} else {
		// 🚨 极其重要：如果是单图/首尾帧模式，严格清空附加参考图，防止 API 误以为是 R2V 任务！
		p.ReferenceImageURLs = nil
	}

	// 6. 构建选项并转换图片为 Base64
	taskModel.UpdateProgress(20)
	var opts []video.VideoOption
	opts = append(opts, video.WithDuration(p.Duration))
	opts = append(opts, video.WithStyle(projectStyle), video.WithAspectRatio(aspectRatio))

	appURL := config.GetString("app.url")
	fixURL := func(url string) string {
		if url == "" || strings.HasPrefix(url, "http") || strings.HasPrefix(url, "data:") {
			return url
		}
		cleanPath := strings.TrimPrefix(url, "/")
		fileData, err := os.ReadFile(cleanPath)
		if err == nil {
			ext := filepath.Ext(cleanPath)
			mimeType := mime.TypeByExtension(ext)
			if mimeType == "" {
				mimeType = "image/jpeg"
			}
			base64Str := base64.StdEncoding.EncodeToString(fileData)
			return fmt.Sprintf("data:%s;base64,%s", mimeType, base64Str)
		}
		return strings.TrimRight(appURL, "/") + "/" + cleanPath
	}

	if p.ImageURL != "" {
		opts = append(opts, video.WithImageURL(fixURL(p.ImageURL)))
	}
	if p.FirstFrameURL != "" {
		opts = append(opts, video.WithFirstFrame(fixURL(p.FirstFrameURL)))
	}
	if p.LastFrameURL != "" {
		opts = append(opts, video.WithLastFrame(fixURL(p.LastFrameURL)))
	}
	if len(p.ReferenceImageURLs) > 0 {
		var fixedURLs []string
		for _, u := range p.ReferenceImageURLs {
			fixedURLs = append(fixedURLs, fixURL(u))
		}
		opts = append(opts, video.WithReferenceImages(fixedURLs))
	}

	// 7. 最终 Prompt 组装与系统约束判断
	promptGen := prompt.NewGenerator()

	finalPrompt := structuredPrompt
	if finalPrompt == "" {
		finalPrompt = p.Prompt
	} else if p.Prompt != "" && !strings.Contains(finalPrompt, p.Prompt) {
		finalPrompt += "\n\nAdditional Requirements:\n" + p.Prompt
	}

	constraintPrompt := promptGen.GetVideoConstraintPrompt(referenceMode)
	styleConstraint := fmt.Sprintf(`=== Global Visual Bible (mandatory) ===
Visual style: %s.
Aspect ratio: %s.
Keep the same art direction, color palette, lighting language, rendering technique, character facial identity, hairstyle, costume, body proportions, scene architecture, material texture and lens language as all other shots in this project.
The supplied reference frame is the visual source of truth. Do not redesign characters, costumes, locations or color grading. Do not switch between realistic, anime, 3D, illustration or live-action styles.`, projectStyle, aspectRatio)
	if constraintPrompt != "" {
		finalPrompt = styleConstraint + "\n\n" + constraintPrompt + "\n\n=== Script & Scene Details ===\n" + finalPrompt
	} else {
		finalPrompt = styleConstraint + "\n\n=== Script & Scene Details ===\n" + finalPrompt
	}

	console.Success(fmt.Sprintf("任务[%d] 最终 Prompt: \n%s", p.AsyncTaskID, finalPrompt))

	// 8. 发起生成请求
	taskModel.UpdateProgress(30)
	result, err := client.GenerateVideo(finalPrompt, opts...)
	if err != nil {
		taskModel.MarkAsFailed(err)
		return err
	}

	console.Success(fmt.Sprintf("任务[%d] - 视频请求提交成功，TaskID: %s, 轮询中...", p.AsyncTaskID, result.TaskID))

	// 9. 轮询获取任务结果
	taskModel.UpdateProgress(40)
	maxAttempts := 150
	interval := 10 * time.Second

	for attempt := 0; attempt < maxAttempts; attempt++ {
		time.Sleep(interval)

		var checkTask async_tasks.AsyncTask
		if err := database.DB.First(&checkTask, p.AsyncTaskID).Error; err == nil {
			if checkTask.Status != async_tasks.StatusProcessing {
				return nil
			}
		}

		statusRes, err := client.GetTaskStatus(result.TaskID)
		if err != nil {
			console.Warning(fmt.Sprintf("任务[%d] 轮询异常: %v", p.AsyncTaskID, err))
			continue
		}

		if statusRes.Error != "" {
			errStr := fmt.Errorf("视频生成失败: %s", statusRes.Error)
			taskModel.MarkAsFailed(errStr)
			return errStr
		}

		if statusRes.Completed && statusRes.VideoURL != "" {
			result = statusRes
			break
		}

		prog := 40 + int(float64(attempt)/float64(maxAttempts)*45)
		taskModel.UpdateProgress(uint64(prog))
	}

	if !result.Completed || result.VideoURL == "" {
		errStr := fmt.Errorf("生成超时或未返回下载地址")
		taskModel.MarkAsFailed(errStr)
		return errStr
	}

	console.Success(fmt.Sprintf("任务[%d] - 视频生成完成，开始下载...", p.AsyncTaskID))

	// 10. 下载视频并更新数据库
	taskModel.UpdateProgress(90)

	var localPath string

	// 🔴 判断返回的是网络 URL 还是已经被 Content 接口存好的本地路径
	if strings.HasPrefix(result.VideoURL, "http://") || strings.HasPrefix(result.VideoURL, "https://") {
		// 需要网络下载
		var dlErr error
		localPath, dlErr = upload.DownloadAndSaveVideo(result.VideoURL)
		if dlErr != nil {
			taskModel.MarkAsFailed(fmt.Errorf("下载视频失败: %v", dlErr))
			return dlErr
		}
	} else {
		// Content API 已经把视频存为本地文件并返回了本地路径
		localPath = result.VideoURL
	}

	err = database.DB.Model(&shots.Shots{}).
		Where("id = ?", p.ShotID).
		Updates(map[string]interface{}{
			"video_url":   localPath,
			"duration_ms": p.Duration * 1000,
		}).Error

	if err != nil {
		taskModel.MarkAsFailed(fmt.Errorf("数据入库失败: %v", err))
		return err
	}

	// 11. 完结任务
	taskModel.UpdateProgress(100)
	resData := map[string]interface{}{
		"url":      localPath,
		"shot_id":  p.ShotID,
		"duration": p.Duration,
	}
	resBytes, _ := json.Marshal(resData)
	taskModel.MarkAsSuccess(string(resBytes))

	console.Success(fmt.Sprintf("任务[%d] - 视频已生成保存至: %s", p.AsyncTaskID, localPath))
	return nil
}
