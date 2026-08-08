package asynq

const (
	TypeGenerateScript     = "drama:generate_script"
	TypeGenerateImage      = "drama:generate_image"
	TypeGenerateCharacters = "ai:generate:characters" // 提取角色
	TypeExtractScenes      = "ai:extract:scenes"      // 提取场景
	TypeGenerateSceneImage = "generate:scene:image"
	TypeGenerateShots      = "generate:shots"
	TypeExtractProps       = "ai:extract_props"
	TypeGeneratePropImage  = "ai:generate_prop_image"

	TypeExtractFramePrompt  = "ai:extract_frame_prompt" // 提取帧提示词任务
	TypeGenerateFrameImage  = "ai:generate_frame_image" // 生成分镜帧图片任务
	TypeGenerateVideo       = "ai:generate_video"
	TypeMergeVideo          = "video:merge" // 视频合并任务
	TypeScriptWorkflowStage = "script:workflow_stage"
)

type ScriptWorkflowPayload struct {
	AsyncTaskID   uint64 `json:"asyncTaskId"`
	WorkflowID    uint64 `json:"workflowId"`
	StepKey       string `json:"stepKey"`
	EpisodeNo     int    `json:"episodeNo,omitempty"`
	ChangeRequest string `json:"changeRequest,omitempty"`
}

// GenerateScriptPayload
type GenerateScriptPayload struct {
	AsyncTaskID uint64 `json:"asyncTaskId"` // 关键：关联的任务表ID
	ProjectID   uint64 `json:"projectId"`
	ScriptID    uint64 `json:"scriptId"` // 具体要更新哪个剧本
	Prompt      string `json:"prompt"`
}

// GenerateImagePayload 生成图片的参数
type GenerateImagePayload struct {
	AsyncTaskID uint64 `json:"asyncTaskId"` // 关联的任务ID
	ProjectID   uint64 `json:"projectId"`   // 关联的项目ID

	// 互斥参数：要么传 CharacterID (角色定妆)，要么传 ShotID (分镜图)
	CharacterID uint64 `json:"characterId,omitempty"`
	ShotID      uint64 `json:"shotId,omitempty"`

	Prompt string `json:"prompt"`
}

type GenerateCharactersPayload struct {
	AsyncTaskID uint64 `json:"asyncTaskId"`
	ProjectID   uint64 `json:"projectId"`
	Count       int    `json:"count"`
	Outline     string `json:"outline"`
}

type ExtractScenesPayload struct {
	AsyncTaskID uint64 `json:"asyncTaskId"`
	ScriptID    uint64 `json:"scriptId"`
}

// GenerateSceneImagePayload 场景图片生成载荷
type GenerateSceneImagePayload struct {
	AsyncTaskID uint64 `json:"asyncTaskId"`
	ProjectID   uint64 `json:"projectId"`
	SceneID     uint64 `json:"sceneId"`
	Prompt      string `json:"prompt"`
}

// GenerateShotsPayload 分镜生成载荷
type GenerateShotsPayload struct {
	AsyncTaskID uint64 `json:"asyncTaskId"`
	ProjectID   uint64 `json:"projectId"`
	ScriptID    uint64 `json:"scriptId"`
	Model       string `json:"model"` // 可选：指定模型
}

// GeneratePropImagePayload 道具生图载荷
type GeneratePropImagePayload struct {
	AsyncTaskID uint64 `json:"async_task_id"`
	ProjectID   uint64 `json:"project_id"`
	PropID      uint64 `json:"prop_id"`
	Prompt      string `json:"prompt"`
}

// ExtractPropsPayload 提取道具载荷
type ExtractPropsPayload struct {
	AsyncTaskID uint64 `json:"async_task_id"`
	ProjectID   uint64 `json:"project_id"`
	EpisodeID   uint64 `json:"episode_id"` // 根据哪一集剧本提取
}

// ExtractFramePromptPayload 提取帧提示词载荷
type ExtractFramePromptPayload struct {
	AsyncTaskID uint64 `json:"async_task_id"`
	ProjectID   uint64 `json:"project_id"`
	ShotID      uint64 `json:"shot_id"`
	FrameType   string `json:"frame_type"` // first/last/key/action/panel
	Model       string `json:"model"`      // 可选：指定的文本大模型
}

// GenerateFrameImagePayload 帧图片生成载荷
type GenerateFrameImagePayload struct {
	AsyncTaskID uint64 `json:"async_task_id"`
	ProjectID   uint64 `json:"project_id"`
	ShotID      uint64 `json:"shot_id"`
	FrameType   string `json:"frame_type"`
	Prompt      string `json:"prompt"`
}

// GenerateVideoPayload 视频生成载荷
type GenerateVideoPayload struct {
	AsyncTaskID        uint64   `json:"async_task_id"`
	ProjectID          uint64   `json:"project_id"`
	ShotID             uint64   `json:"shot_id"`
	Model              string   `json:"model"`
	Duration           int      `json:"duration"`
	Prompt             string   `json:"prompt"`
	ReferenceMode      string   `json:"reference_mode"`
	ImageURL           string   `json:"image_url,omitempty"`
	FirstFrameURL      string   `json:"first_frame_url,omitempty"`
	LastFrameURL       string   `json:"last_frame_url,omitempty"`
	ReferenceImageURLs []string `json:"reference_image_urls,omitempty"`
	Style              string   `json:"style,omitempty"`
	AspectRatio        string   `json:"aspect_ratio,omitempty"`
}

// MergeClip 合成视频的片段信息
type MergeClip struct {
	URL        string                 `json:"url"`
	Duration   float64                `json:"duration"`
	StartTime  float64                `json:"start_time"`
	EndTime    float64                `json:"end_time"`
	Transition map[string]interface{} `json:"transition"`
}

// MergeVideoPayload 视频合并任务载荷
type MergeVideoPayload struct {
	AsyncTaskID uint64      `json:"async_task_id"` // 异步任务ID
	MergeID     uint64      `json:"merge_id"`      // 合成记录ID (保存到 video_merges 表)
	ProjectID   uint64      `json:"project_id"`
	EpisodeID   uint64      `json:"episode_id"` // 剧集/脚本ID
	Title       string      `json:"title"`
	Clips       []MergeClip `json:"clips"`
	AspectRatio string      `json:"aspect_ratio"`
}
