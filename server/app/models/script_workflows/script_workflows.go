package script_workflows

import (
	"spiritFruit/app/models"
	"spiritFruit/pkg/database"
	"time"
)

const (
	StatusDraft     = "draft"
	StatusRunning   = "running"
	StatusWaiting   = "waiting_confirmation"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StepCreative    = "creative_lock"
	StepDesign      = "complete_design"
	StepEpisodes    = "episode_generation"
	StepDelivery    = "ai_delivery"
)

type ScriptWorkflow struct {
	models.BaseModel
	ProjectID         uint64 `json:"projectId" gorm:"index;not null"`
	AdminID           uint64 `json:"adminId" gorm:"index;not null"`
	Title             string `json:"title" gorm:"size:255"`
	Status            string `json:"status" gorm:"size:32;index;default:'draft'"`
	CurrentStep       int    `json:"currentStep" gorm:"default:1"`
	StateVersion      string `json:"stateVersion" gorm:"size:16;default:'v0.0'"`
	TotalEpisodes     int    `json:"totalEpisodes" gorm:"default:10"`
	ConfirmedEpisodes int    `json:"confirmedEpisodes" gorm:"default:0"`
	GenerationMode    string `json:"generationMode" gorm:"size:16"`
	ConfigJSON        string `json:"configJson" gorm:"type:longtext"`
	models.CommonTimestampsField
}

func (ScriptWorkflow) TableName() string { return "script_workflows" }

type ScriptWorkflowStep struct {
	models.BaseModel
	WorkflowID    uint64     `json:"workflowId" gorm:"uniqueIndex:idx_workflow_step_version;index"`
	StepKey       string     `json:"stepKey" gorm:"size:64;uniqueIndex:idx_workflow_step_version"`
	Version       int        `json:"version" gorm:"default:1;uniqueIndex:idx_workflow_step_version"`
	Status        string     `json:"status" gorm:"size:32;index"`
	InputJSON     string     `json:"inputJson" gorm:"type:longtext"`
	OutputJSON    string     `json:"outputJson" gorm:"type:longtext"`
	OutputText    string     `json:"outputText" gorm:"type:longtext"`
	PromptVersion string     `json:"promptVersion" gorm:"size:32"`
	ModelSnapshot string     `json:"modelSnapshot" gorm:"type:text"`
	AsyncTaskID   uint64     `json:"asyncTaskId" gorm:"index"`
	ConfirmedAt   *time.Time `json:"confirmedAt"`
	models.CommonTimestampsField
}

func (ScriptWorkflowStep) TableName() string { return "script_workflow_steps" }

type ScriptEpisodeDraft struct {
	models.BaseModel
	WorkflowID        uint64     `json:"workflowId" gorm:"uniqueIndex:idx_workflow_episode;index"`
	EpisodeNo         int        `json:"episodeNo" gorm:"uniqueIndex:idx_workflow_episode"`
	Title             string     `json:"title" gorm:"size:255"`
	Status            string     `json:"status" gorm:"size:32;index"`
	BatchNo           int        `json:"batchNo"`
	OutlineJSON       string     `json:"outlineJson" gorm:"type:longtext"`
	ContentJSON       string     `json:"contentJson" gorm:"type:longtext"`
	ContentText       string     `json:"contentText" gorm:"type:longtext"`
	DurationSeconds   int        `json:"durationSeconds"`
	WordCount         int        `json:"wordCount"`
	QualityReportJSON string     `json:"qualityReportJson" gorm:"type:longtext"`
	PromptVersion     string     `json:"promptVersion" gorm:"size:32"`
	ModelSnapshot     string     `json:"modelSnapshot" gorm:"type:text"`
	AsyncTaskID       uint64     `json:"asyncTaskId" gorm:"index"`
	ConfirmedAt       *time.Time `json:"confirmedAt"`
	PublishedScriptID uint64     `json:"publishedScriptId" gorm:"index"`
	models.CommonTimestampsField
}

func (ScriptEpisodeDraft) TableName() string { return "script_episode_drafts" }

type ScriptEpisodeVersion struct {
	models.BaseModel
	EpisodeDraftID uint64 `json:"episodeDraftId" gorm:"index"`
	Version        int    `json:"version"`
	Source         string `json:"source" gorm:"size:24"`
	ChangeRequest  string `json:"changeRequest" gorm:"type:text"`
	ContentJSON    string `json:"contentJson" gorm:"type:longtext"`
	ContentText    string `json:"contentText" gorm:"type:longtext"`
	CreatedBy      uint64 `json:"createdBy"`
	models.CommonTimestampsField
}

func (ScriptEpisodeVersion) TableName() string { return "script_episode_versions" }

func GetWorkflow(id, adminID uint64) (ScriptWorkflow, error) {
	var item ScriptWorkflow
	err := database.DB.Where("id = ? AND admin_id = ?", id, adminID).First(&item).Error
	return item, err
}
