package scripts

import (
	"spiritFruit/app/models"
	"spiritFruit/app/models/projects"
	"spiritFruit/app/models/shot_video_merge"
	"spiritFruit/app/models/source"
	"spiritFruit/pkg/database"
	"time"
)

// Scripts 结构体 剧本
type Scripts struct {
	models.BaseModel
	ProjectId        *uint64    `json:"projectId" form:"projectId" gorm:"column:project_id;comment:所属项目ID, 外键约束(project_id) -> projects(id);"` //所属项目ID
	Title            *string    `json:"title" form:"title" gorm:"column:title;comment:分集标题;size:255;"`                                         //分集标题
	Content          *string    `json:"content" form:"content" gorm:"column:content;comment:剧本正文;"`                                            //剧本正文
	Outline          *string    `json:"outline" form:"outline" gorm:"column:outline;comment:大纲/简介;"`                                           //大纲/简介
	EpisodeNo        *uint64    `json:"episodeNo" form:"episodeNo" gorm:"default:1;column:episode_no;comment:第几集;"`                            //第几集
	IsLocked         *int8      `json:"isLocked" form:"isLocked" gorm:"default:0;column:is_locked;comment:是否定稿 0-否 1-是;size:1;"`               //是否定稿
	WorkflowId       *uint64    `json:"workflowId" gorm:"column:workflow_id;index"`
	EpisodeDraftId   *uint64    `json:"episodeDraftId" gorm:"column:episode_draft_id;index"`
	ContentJSON      *string    `json:"contentJson" gorm:"column:content_json;type:longtext"`
	GenerationStatus *string    `json:"generationStatus" gorm:"column:generation_status;size:32"`
	Version          *uint64    `json:"version" gorm:"default:1"`
	GeneratedAt      *time.Time `json:"generatedAt"`

	// 关联关系
	Projectss  *projects.Projects `json:"projects,omitempty" gorm:"foreignKey:ProjectId;references:ID"` // 所属短剧项目
	Source     []source.Source    `json:"source" gorm:"foreignKey:ShotId;references:ID"`
	ShotsCount int64              `json:"shotsCount" gorm:"->"`

	ShotVideoMerges []shot_video_merge.ShotVideoMerge `json:"shotVideoMerges" gorm:"foreignKey:ScriptId;references:ID"`

	models.CommonTimestampsField
}

// TableName 剧本 Scripts自定义表名 scripts
func (Scripts) TableName() string {
	return "scripts"
}

// Create 创建剧本
func (scripts *Scripts) Create() {
	database.DB.Create(&scripts)
}

// Save 保存剧本
func (scripts *Scripts) Save() (rowsAffected int64) {
	result := database.DB.Save(&scripts)
	return result.RowsAffected
}

// Delete 删除剧本
func (scripts *Scripts) Delete() (rowsAffected int64) {
	result := database.DB.Delete(&scripts)
	return result.RowsAffected
}
