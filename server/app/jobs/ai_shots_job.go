package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"spiritFruit/app/services"
	"strings"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"spiritFruit/app/models/ai_config"
	"spiritFruit/app/models/async_tasks"
	"spiritFruit/app/models/characters"
	"spiritFruit/app/models/props"
	"spiritFruit/app/models/scenes"
	"spiritFruit/app/models/scripts"
	"spiritFruit/app/models/shots"
	myAsynq "spiritFruit/pkg/asynq"
	"spiritFruit/pkg/config"
	"spiritFruit/pkg/console"
	"spiritFruit/pkg/database"
	"spiritFruit/pkg/openai"
	"spiritFruit/pkg/prompt"
	"spiritFruit/pkg/utils"
)

type AIStoryboard struct {
	SequenceNo     int      `json:"sequenceNo"`
	Title          string   `json:"title"`
	ShotType       string   `json:"shotType"`
	Angle          string   `json:"angle"`
	CameraMovement string   `json:"cameraMovement"`
	Time           string   `json:"time"`
	Location       string   `json:"location"`
	SceneID        *uint64  `json:"sceneId"`
	Action         string   `json:"action"`
	Dialogue       string   `json:"dialogue"`
	VisualDesc     string   `json:"visualDesc"`
	Result         string   `json:"result"`
	Atmosphere     string   `json:"atmosphere"`
	AudioPrompt    string   `json:"audioPrompt"`
	DurationSec    int      `json:"durationSec"`  // AI 推理的时长秒数
	CharacterIDs   []uint64 `json:"characterIds"` // 角色关联
	PropIDs        []uint64 `json:"propIds"`      // 道具关联
}

// UnmarshalJSON accepts both the camelCase contract used by the UI and the
// snake_case/legacy names commonly returned by screenplay prompts.
func (s *AIStoryboard) UnmarshalJSON(data []byte) error {
	type storyboardAlias AIStoryboard
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for from, to := range map[string]string{
		"sequence_no": "sequenceNo", "shot_number": "sequenceNo", "shot_type": "shotType",
		"camera_movement": "cameraMovement", "scene_id": "sceneId", "visual_desc": "visualDesc",
		"audio_prompt": "audioPrompt", "duration_sec": "durationSec", "character_ids": "characterIds",
		"prop_ids": "propIds",
	} {
		if v, ok := raw[from]; ok {
			if _, exists := raw[to]; !exists {
				raw[to] = v
			}
		}
	}
	normalized, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	var value storyboardAlias
	if err := json.Unmarshal(normalized, &value); err != nil {
		return err
	}
	*s = AIStoryboard(value)
	return nil
}

// HandleGenerateShots 处理分镜生成任务
func HandleGenerateShots(ctx context.Context, t *asynq.Task) error {
	var p myAsynq.GenerateShotsPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json unmarshal failed: %v", err)
	}

	// 1. 获取任务记录
	taskModel := async_tasks.AsyncTask{}
	if err := database.DB.First(&taskModel, p.AsyncTaskID).Error; err != nil {
		return nil
	}
	taskModel.MarkAsProcessing()
	console.Success(fmt.Sprintf("任务[%d] - 开始拆分分镜", p.AsyncTaskID))

	// 2. 准备数据
	taskModel.UpdateProgress(10)

	// A. 获取剧本内容
	var script scripts.Scripts
	if err := database.DB.First(&script, p.ScriptID).Error; err != nil {
		err = fmt.Errorf("script not found: %v", err)
		taskModel.MarkAsFailed(err)
		return nil
	}
	if script.Content == nil || *script.Content == "" {
		err := fmt.Errorf("script content is empty")
		taskModel.MarkAsFailed(err)
		return nil
	}
	scriptContent := *script.Content

	// B. 获取角色列表 (用于 Prompt 上下文)
	var charList []characters.Characters
	database.DB.Where("project_id = ?", p.ProjectID).Find(&charList)
	charInfoStr := "无角色"
	if len(charList) > 0 {
		var infos []string
		for _, c := range charList {
			cName := ""
			if c.Name != nil {
				cName = *c.Name
			}
			infos = append(infos, fmt.Sprintf(`{"id": %d, "name": "%s"}`, c.ID, cName))
		}
		charInfoStr = fmt.Sprintf("[%s]", strings.Join(infos, ", "))
	}

	// C. 获取场景列表 (用于 Prompt 上下文)
	var sceneList []scenes.Scenes
	database.DB.Where("project_id = ?", p.ProjectID).Find(&sceneList)
	sceneInfoStr := "无场景"
	if len(sceneList) > 0 {
		var infos []string
		for _, s := range sceneList {
			sName := ""
			if s.Name != nil {
				sName = *s.Name
			}
			infos = append(infos, fmt.Sprintf(`{"id": %d, "location": "%s", "time": "%s"}`, s.ID, sName, getString(s.Time)))
		}
		sceneInfoStr = fmt.Sprintf("[%s]", strings.Join(infos, ", "))
	}

	// D. 获取道具列表 (用于 Prompt 上下文，供 AI 绑定 PropIDs)
	var propList []props.Props
	database.DB.Where("project_id = ?", p.ProjectID).Find(&propList)
	propInfoStr := "无道具"
	if len(propList) > 0 {
		var infos []string
		for _, pr := range propList {
			pName := ""
			if pr.Name != nil {
				pName = *pr.Name
			}
			infos = append(infos, fmt.Sprintf(`{"id": %d, "name": "%s"}`, pr.ID, pName))
		}
		propInfoStr = fmt.Sprintf("[%s]", strings.Join(infos, ", "))
	}

	// 3. 构建 Prompt
	taskModel.UpdateProgress(30)
	promptGen := prompt.NewGenerator() // 实例化您封装的提示词工具
	systemPrompt := promptGen.GetStoryboardSystemPrompt()

	// 拼装究极版的用户提示词
	userPrompt := fmt.Sprintf(`
【本剧可用角色列表(JSON)】:
%s
**重要**：在 characterIds 字段中，只能使用上述角色列表中的角色ID（数字），不得自创角色或使用其他ID。无则填 []。

【本剧可用道具列表(JSON)】:
%s
**重要**：在 propIds 字段中，只能使用上述道具列表中的道具ID（数字），不得自创道具或使用其他ID。无则填 []。

【本剧已提取的场景列表(JSON)】:
%s
**重要**：在 sceneId 字段中，必须从上述背景列表中选择最匹配的背景ID（数字）。如果没有合适的背景，则填 null。

【剧本原文】:
%s

【分镜要素】每个镜头聚焦单一动作，描述要详尽具体：
1. **镜头标题(title)**：用3-5个字概括该镜头的核心内容或情绪
   - 例如："噩梦惊醒"、"对视沉思"、"逃离现场"
2. **时间(time)**：[清晨/午后/深夜/具体时分+详细光线描述]
3. **地点(location)**：[场景完整描述+空间布局+环境细节]
4. **镜头设计**：
   - **景别(shotType)**：[大远景/远景/全景/中景/近景/特写/大特写]
   - **镜头角度(angle)**：[平视/仰视/俯视/侧面/背面/鸟瞰]
   - **运镜方式(cameraMovement)**：[固定镜头/推镜/拉镜/摇镜/跟镜/移镜/环绕]
5. **人物行为(action)**：**详细动作描述**，包含[谁+具体怎么做+肢体细节+表情状态]
6. **对话/独白(dialogue)**：提取该镜头中的完整对话或独白内容（如无对话则为空字符串）
7. **动作结果(result)**：仅描述该动作完成后的即时后果或状态变化
8. **画面描述(visualDesc)**：描述完整构图、主体位置、环境及可见细节，不与result混为一项
9. **环境氛围(atmosphere)**：光线质感+色调+声音环境+整体氛围+角色情绪状态
10. **音效配乐(audioPrompt)**：描述该镜头配乐的氛围、节奏、情绪及关键音效

【输出格式】请以JSON格式输出，每个镜头包含以下字段（**所有键名必须严格遵守如下驼峰格式，描述性字段都要详细完整**）：
{
  "storyboards": [
    {
      "sequenceNo": 1,
      "title": "噩梦惊醒",
      "shotType": "全景",
      "angle": "俯视45度角",
      "cameraMovement": "固定镜头",
      "time": "深夜22:30·月光从破窗斜射入仓库，在地面积水中形成银白色反光",
      "location": "废弃码头仓库·锈蚀货架林立，墙角堆放腐朽木箱和渔网",
      "sceneId": 1,
      "action": "陈峥弯腰双手握住撬棍用力撬动保险箱门，手臂青筋暴起，眉头紧锁，汗水滑落",
      "dialogue": "（独白）这么多年了，里面到底藏着什么秘密？",
      "result": "保险箱门弹开，箱内空无一物，陈峥由期待转为震惊",
      "visualDesc": "保险箱门突然弹开发出刺耳金属声，扬起灰尘在手电筒光束中飘散，箱内空无一物，陈峥表情从期待转为震惊",
      "atmosphere": "昏暗冷色调·青灰色为主，只有手电筒光束在黑暗中晃动，整体氛围压抑沉重。情绪：好奇感↑↑转失望↓",
      "audioPrompt": "低沉紧张的弦乐，节奏缓慢。金属碰撞声、灰尘飘散声、海浪拍打声",
      "durationSec": 9,
      "characterIds": [159],
      "propIds": [12]
    }
  ]
}

**dialogue字段说明**：
- 如果有对话，格式为：角色名："台词内容"
- 多人对话用空格分隔：角色A："..." 角色B："..."
- 独白格式为：（独白）内容
- 旁白格式为：（旁白）内容
- 无对话时填写空字符串：""
- **对话内容必须从原剧本中提取，保持原汁原味**

**durationSec时长估算规则（秒）**：
- **所有镜头时长必须在4-12秒范围内**，确保节奏合理流畅
- **综合估算原则**：时长由对话内容、动作复杂度、情绪节奏三方面综合决定
1. 基础时长：纯对话场景4秒；纯动作场景5秒；混合场景6秒
2. 对话调整：短对话+1~2秒；中等对话+2~4秒；长对话+4~6秒
3. 动作调整：简单动作+0~1秒；一般动作+1~2秒；复杂动作+2~4秒
4. **最终时长** = 基础时长 + 对话调整 + 动作调整，确保结果在4-12秒范围内

**特别要求**：
- **【极其重要】必须100%%完整拆解整个剧本，不得省略、跳过、压缩任何剧情内容**
- **从剧本第一个字到最后一个字，逐句逐段转换为分镜**
- **每个对话、每个动作、每个场景转换都必须有对应的分镜**
- 严格按照JSON格式输出，不要包含任何markdown代码块、说明文字或其他内容。
`, charInfoStr, propInfoStr, sceneInfoStr, scriptContent)

	// 4. 调用 AI
	taskModel.UpdateProgress(40)
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

	// 2. 尝试加载 AI 配置
	aiService := new(services.AiConfigService)
	var errConfig error
	var dbConfig ai_config.AiConfig

	// 如果指定了模型，优先寻找对应的配置
	if p.Model != "" {
		errConfig, dbConfig = aiService.GetSpecificModelConfig("text", "", p.Model, taskModel.AdminID)
	}

	// 如果未指定或未找到，使用默认激活配置
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

	// 模拟单次对话
	aiReq := openai.ScriptRequest{
		Messages: []openai.ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
		//MaxTokens:   4096,// openai
		MaxTokens: 16384, // 豆包
	}

	aiResp, err := provider.GenerateScript(aiReq)
	if err != nil {
		taskModel.MarkAsFailed(err)
		return err
	}
	fmt.Println("aiResp", aiResp)
	// 5. 解析结果
	taskModel.UpdateProgress(70)

	var resultWrapper struct {
		Storyboards []AIStoryboard `json:"storyboards"`
	}

	if err := utils.SafeParseAIJSON(aiResp, &resultWrapper); err != nil {
		var arr []AIStoryboard
		if err2 := utils.SafeParseAIJSON(aiResp, &arr); err2 == nil {
			resultWrapper.Storyboards = arr
		} else {
			taskModel.MarkAsFailed(fmt.Errorf("failed to parse JSON: %v", err))
			return nil
		}
	}

	if len(resultWrapper.Storyboards) == 0 {
		taskModel.MarkAsFailed(fmt.Errorf("AI返回的分镜数量为0"))
		return nil
	}

	// 6. 入库 (使用事务)
	taskModel.UpdateProgress(80)

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// A. 清理旧分镜：为了安全，先找出现有分镜ID并删除
		var oldShotIds []uint64
		tx.Model(&shots.Shots{}).Where("script_id = ?", p.ScriptID).Pluck("id", &oldShotIds)

		if len(oldShotIds) > 0 {
			if err := tx.Where("script_id = ?", p.ScriptID).Delete(&shots.Shots{}).Error; err != nil {
				return err
			}
		}

		// B. 插入新分镜
		for i, sb := range resultWrapper.Storyboards {
			projectID := p.ProjectID
			scriptID := p.ScriptID

			// 兜底序号与时长
			sequenceNo := uint64(sb.SequenceNo)
			if sequenceNo == 0 {
				sequenceNo = uint64(i + 1)
			}
			durationSec := sb.DurationSec
			if durationSec < 1 {
				durationSec = 4 // 时长兜底 4 秒
			}
			durationMs := uint64(durationSec * 1000)
			status := int8(0)

			// 预先生成专用的图生视频、文生图提示词
			imgPromptStr := generateImagePrompt(sb)
			vidPromptStr := generateVideoPrompt(sb)

			// 提取各字段副本以用于存入结构体指针
			title := sb.Title
			shotType := sb.ShotType
			cameraMove := sb.CameraMovement
			angle := sb.Angle
			timeDesc := sb.Time
			locDesc := sb.Location
			action := sb.Action
			dialogue := sb.Dialogue
			visualDesc := sb.VisualDesc
			resultText := sb.Result
			if resultText == "" {
				resultText = visualDesc
			}
			atmosphere := sb.Atmosphere
			audioPrompt := sb.AudioPrompt

			// 构造 Shot 模型
			newShot := shots.Shots{
				ProjectId:      &projectID,
				ScriptId:       &scriptID,
				SceneId:        sb.SceneID,
				SequenceNo:     &sequenceNo,
				Title:          &title,
				ShotType:       &shotType,
				CameraMovement: &cameraMove,
				Angle:          &angle,
				Time:           &timeDesc,
				Location:       &locDesc,
				Action:         &action,
				VisualDesc:     &visualDesc,
				Result:         &resultText,
				Atmosphere:     &atmosphere,
				Dialogue:       &dialogue,
				ImagePrompt:    &imgPromptStr,
				VideoPrompt:    &vidPromptStr,
				AudioPrompt:    &audioPrompt,
				DurationMs:     &durationMs,
				Status:         &status,
			}

			// 保存分镜基础信息
			if err := tx.Create(&newShot).Error; err != nil {
				return err
			}

			// 绑定登场角色 (利用 GORM Association 进行 Many2Many 关联)
			if len(sb.CharacterIDs) > 0 {
				var chars []characters.Characters
				if err := tx.Where("id IN ?", sb.CharacterIDs).Find(&chars).Error; err == nil && len(chars) > 0 {
					tx.Model(&newShot).Association("Characters").Append(chars)
				}
			}

			// 绑定相关道具 (利用 GORM Association 进行 Many2Many 关联)
			if len(sb.PropIDs) > 0 {
				var prps []props.Props
				if err := tx.Where("id IN ?", sb.PropIDs).Find(&prps).Error; err == nil && len(prps) > 0 {
					tx.Model(&newShot).Association("Props").Append(prps)
				}
			}
		}
		return nil
	})

	if err != nil {
		taskModel.MarkAsFailed(fmt.Errorf("db transaction failed: %v", err))
		return err
	}

	// 7. 完成
	resultData := map[string]interface{}{
		"total_shots": len(resultWrapper.Storyboards),
	}
	resBytes, _ := json.Marshal(resultData)
	taskModel.MarkAsSuccess(string(resBytes))

	console.Success(fmt.Sprintf("任务[%d] - 分镜拆分完成，共 %d 个镜头", p.AsyncTaskID, len(resultWrapper.Storyboards)))
	return nil
}

// ==========================================
// 辅助方法区
// ==========================================

func getString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// generateImagePrompt 生成专用于图片生成的提示词（提取首帧静态画面）
func generateImagePrompt(sb AIStoryboard) string {
	var parts []string

	if sb.Location != "" {
		locationDesc := sb.Location
		if sb.Time != "" {
			locationDesc += ", " + sb.Time
		}
		parts = append(parts, locationDesc)
	}

	// 角色初始静态姿态（去除动作过程，只保留起始状态）
	if sb.Action != "" {
		initialPose := extractInitialPose(sb.Action)
		if initialPose != "" {
			parts = append(parts, initialPose)
		}
	}

	if sb.Atmosphere != "" {
		parts = append(parts, sb.Atmosphere)
	}

	// 兜底补丁
	parts = append(parts, "cinematic lighting, highly detailed")

	if len(parts) > 0 {
		return strings.Join(parts, ", ")
	}
	return "highly detailed scene"
}

// generateVideoPrompt 生成专用于视频生成的结构化提示词
func generateVideoPrompt(sb AIStoryboard) string {
	var parts []string

	if sb.Action != "" {
		parts = append(parts, fmt.Sprintf("Action: %s", sb.Action))
	}
	if sb.Dialogue != "" {
		parts = append(parts, fmt.Sprintf("Dialogue: %s", sb.Dialogue))
	}
	if sb.CameraMovement != "" {
		parts = append(parts, fmt.Sprintf("Camera movement: %s", sb.CameraMovement))
	}
	if sb.ShotType != "" {
		parts = append(parts, fmt.Sprintf("Shot type: %s", sb.ShotType))
	}
	if sb.Angle != "" {
		parts = append(parts, fmt.Sprintf("Camera angle: %s", sb.Angle))
	}
	if sb.Location != "" {
		locationDesc := sb.Location
		if sb.Time != "" {
			locationDesc += ", " + sb.Time
		}
		parts = append(parts, fmt.Sprintf("Scene: %s", locationDesc))
	}
	if sb.Atmosphere != "" {
		parts = append(parts, fmt.Sprintf("Atmosphere: %s", sb.Atmosphere))
	}
	if sb.VisualDesc != "" {
		parts = append(parts, fmt.Sprintf("Result: %s", sb.VisualDesc))
	}

	if len(parts) > 0 {
		return strings.Join(parts, ". ")
	}
	return "cinematic video scene"
}

// extractInitialPose 提取初始静态姿态（去除过程动词）
func extractInitialPose(action string) string {
	processWords := []string{
		"然后", "接着", "接下来", "随后", "紧接着",
		"向下", "向上", "向前", "向后", "向左", "向右",
		"开始", "继续", "逐渐", "慢慢", "快速", "突然", "猛然",
	}

	result := action
	for _, word := range processWords {
		if idx := strings.Index(result, word); idx > 0 {
			result = result[:idx]
			break
		}
	}
	result = strings.TrimRight(result, "，。,. ")
	return strings.TrimSpace(result)
}
