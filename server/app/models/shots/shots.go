package shots

import (
	"spiritFruit/app/models"
	"spiritFruit/app/models/characters"
	"spiritFruit/app/models/projects"
	"spiritFruit/app/models/props"
	"spiritFruit/app/models/scenes"
	"spiritFruit/app/models/scripts"
	"spiritFruit/app/models/shot_frame_image"
	"spiritFruit/app/models/shot_frame_prompts"
	"spiritFruit/app/models/shot_generate_video"
	"spiritFruit/pkg/database"
)

// Shots 结构体 镜头表
type Shots struct {
	models.BaseModel

	// --- 层级关联 ---
	ProjectId *uint64 `json:"projectId" form:"projectId" gorm:"index;column:project_id;comment:所属项目ID;"`
	ScriptId  *uint64 `json:"scriptId" form:"scriptId" gorm:"index;column:script_id;comment:所属剧本ID;"`
	SceneId   *uint64 `json:"sceneId" form:"sceneId" gorm:"column:scene_id;comment:场景ID;"`

	// --- 镜头属性 ---
	SequenceNo     *uint64 `json:"sequenceNo" form:"sequenceNo" gorm:"column:sequence_no;not null;default:1;comment:镜头序号;"`
	Title          *string `json:"title" form:"title" gorm:"column:title;size:255;comment:镜头标题;"`
	ShotType       *string `json:"shotType" form:"shotType" gorm:"column:shot_type;size:100;comment:景别(远/中/特);"`
	Angle          *string `json:"angle" form:"angle" gorm:"column:angle;size:100;comment:视角(仰/俯/平);"`
	CameraMovement *string `json:"cameraMovement" form:"cameraMovement" gorm:"column:camera_movement;size:100;comment:运镜(推/拉/摇/移);"`

	// --- 内容描述 ---
	Time       *string `json:"time" form:"time" gorm:"column:time;size:255;comment:具体时间描述;"`
	Location   *string `json:"location" form:"location" gorm:"column:location;size:255;comment:具体地点描述;"`
	Action     *string `json:"action" form:"action" gorm:"column:action;type:text;comment:人物动作描述;"`
	Dialogue   *string `json:"dialogue" form:"dialogue" gorm:"column:dialogue;type:text;comment:台词;"`
	VisualDesc *string `json:"visualDesc" form:"visualDesc" gorm:"column:visual_desc;type:text;comment:画面/视觉结果;"`
	Result     *string `json:"result" form:"result" gorm:"column:result;type:text;comment:动作结果;"`
	Atmosphere *string `json:"atmosphere" form:"atmosphere" gorm:"column:atmosphere;type:text;comment:氛围描述;"`

	// --- AIGC 提示词 (核心生产力) ---
	ImagePrompt *string `json:"imagePrompt" form:"imagePrompt" gorm:"column:image_prompt;type:text;comment:AI绘图提示词;"`
	VideoPrompt *string `json:"videoPrompt" form:"videoPrompt" gorm:"column:video_prompt;type:text;comment:视频生成提示词;"`
	AudioPrompt *string `json:"audioPrompt" form:"audioPrompt" gorm:"column:audio_prompt;type:text;comment:音效/BGM提示词;"`

	// --- 资源路径 ---
	ImageUrl *string `json:"imageUrl" form:"imageUrl" gorm:"column:image_url;size:1024;comment:分镜底图/生成图;"`
	VideoUrl *string `json:"videoUrl" form:"videoUrl" gorm:"column:video_url;size:1024;comment:最终合成视频路径;"`
	AudioUrl *string `json:"audioUrl" form:"audioUrl" gorm:"column:audio_url;size:1024;comment:配音/音频路径;"`

	// --- 状态与参数 ---
	DurationMs *uint64 `json:"durationMs" form:"durationMs" gorm:"column:duration_ms;default:3000;comment:持续时间(毫秒);"`
	Status     *int8   `json:"status" form:"status" gorm:"column:status;default:0;comment:状态;"`
	// --- 复杂关联 (学习 houbao-drama 的精华) ---
	Projects *projects.Projects `json:"projects,omitempty" gorm:"foreignKey:ProjectId;constraint:OnDelete:CASCADE;"`
	Scripts  *scripts.Scripts   `json:"scripts,omitempty" gorm:"foreignKey:ScriptId;constraint:OnDelete:CASCADE;"`
	Scenes   *scenes.Scenes     `json:"scenes,omitempty" gorm:"foreignKey:SceneId;references:ID;comment:关联的背景场景"`

	Characters     []characters.Characters                 `json:"characters" gorm:"many2many:shot_characters;joinForeignKey:shot_id;joinReferences:character_id;"`
	Props          []props.Props                           `json:"props" gorm:"many2many:shot_props;joinForeignKey:shot_id;joinReferences:props_id;"`
	FramePrompts   []shot_frame_prompts.ShotFramePrompts   `json:"framePrompts" gorm:"foreignKey:ShotId;references:ID"`
	FrameImages    []shot_frame_image.ShotFrameImages      `json:"frameImages" gorm:"foreignKey:ShotId;references:ID"`
	GenerateVideos []shot_generate_video.ShotGenerateVideo `json:"generateVideos" gorm:"foreignKey:ShotId;references:ID"`

	models.CommonTimestampsField
}

func (s *Shots) TableName() string {
	return "shots"
}

// Create 创建镜头表
func (shots *Shots) Create() {
	database.DB.Create(&shots)
}

// Save 保存镜头表
func (shots *Shots) Save() (rowsAffected int64) {
	result := database.DB.Save(&shots)
	return result.RowsAffected
}

// Delete 删除镜头表
func (shots *Shots) Delete() (rowsAffected int64) {
	result := database.DB.Delete(&shots)
	return result.RowsAffected
}
