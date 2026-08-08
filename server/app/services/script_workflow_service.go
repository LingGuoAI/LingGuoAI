package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"spiritFruit/app/models/async_tasks"
	"spiritFruit/app/models/projects"
	workflow "spiritFruit/app/models/script_workflows"
	myasynq "spiritFruit/pkg/asynq"
	"spiritFruit/pkg/database"
)

type ScriptWorkflowService struct{}

func (s *ScriptWorkflowService) Create(adminID, projectID uint64, title string, config map[string]interface{}) (*workflow.ScriptWorkflow, error) {
	var project projects.Projects
	if err := database.DB.Where("id = ? AND admin_id = ?", projectID, adminID).First(&project).Error; err != nil {
		return nil, err
	}
	b, _ := json.Marshal(config)
	total := 10
	if v, ok := config["totalEpisodes"].(float64); ok && v > 0 {
		total = int(v)
	}
	w := workflow.ScriptWorkflow{ProjectID: projectID, AdminID: adminID, Title: title, Status: workflow.StatusDraft, CurrentStep: 1, StateVersion: "v0.0", TotalEpisodes: total, ConfigJSON: string(b)}
	if err := database.DB.Create(&w).Error; err != nil {
		return nil, err
	}
	return &w, nil
}

func (s *ScriptWorkflowService) Detail(id, adminID uint64) (map[string]interface{}, error) {
	w, err := workflow.GetWorkflow(id, adminID)
	if err != nil {
		return nil, err
	}
	var steps []workflow.ScriptWorkflowStep
	var episodes []workflow.ScriptEpisodeDraft
	database.DB.Where("workflow_id = ?", id).Order("step_key, version desc").Find(&steps)
	database.DB.Where("workflow_id = ?", id).Order("episode_no").Find(&episodes)
	return map[string]interface{}{"workflow": w, "steps": steps, "episodes": episodes}, nil
}

func (s *ScriptWorkflowService) StartStage(w *workflow.ScriptWorkflow, stepKey string, episodeNo int, change string) (*async_tasks.AsyncTask, error) {
	if err := validateTransition(w, stepKey, episodeNo); err != nil {
		return nil, err
	}
	p := myasynq.ScriptWorkflowPayload{WorkflowID: w.ID, StepKey: stepKey, EpisodeNo: episodeNo, ChangeRequest: change}
	raw, _ := json.Marshal(p)
	relID := w.ID
	if stepKey == workflow.StepEpisodes {
		relID = w.ID*1000 + uint64(episodeNo)
	}
	// Worker 按统一任务类型注册，阶段信息保存在 Payload 中；不能把 stepKey 拼到 Asynq type，
	// 否则任务会进入队列但无法被 ServeMux 消费。
	t := async_tasks.AsyncTask{AdminID: &w.AdminID, ProjectID: w.ProjectID, RelID: relID, Type: myasynq.TypeScriptWorkflowStage, Status: async_tasks.StatusPending, Payload: string(raw)}
	t.Create()
	if t.Reused {
		return &t, nil
	}
	p.AsyncTaskID = t.ID
	if stepKey == workflow.StepEpisodes {
		var ep workflow.ScriptEpisodeDraft
		database.DB.Where("workflow_id=? AND episode_no=?", w.ID, episodeNo).FirstOrCreate(&ep, workflow.ScriptEpisodeDraft{WorkflowID: w.ID, EpisodeNo: episodeNo, Status: "queued"})
		database.DB.Model(&ep).Updates(map[string]interface{}{"status": "queued", "async_task_id": t.ID})
	} else {
		var count int64
		database.DB.Model(&workflow.ScriptWorkflowStep{}).Where("workflow_id=? AND step_key=?", w.ID, stepKey).Count(&count)
		st := workflow.ScriptWorkflowStep{WorkflowID: w.ID, StepKey: stepKey, Version: int(count) + 1, Status: "queued", InputJSON: w.ConfigJSON, PromptVersion: "1.1_optimized", AsyncTaskID: t.ID}
		database.DB.Create(&st)
	}
	if _, err := myasynq.EnqueueScriptWorkflow(p); err != nil {
		t.MarkAsFailed(err)
		return &t, err
	}
	database.DB.Model(w).Updates(map[string]interface{}{"status": workflow.StatusRunning})
	return &t, nil
}

func validateTransition(w *workflow.ScriptWorkflow, key string, ep int) error {
	switch key {
	case workflow.StepCreative:
		return nil
	case workflow.StepDesign:
		if w.StateVersion != "v1.0" {
			return errors.New("请先确认创意定型 v1.0")
		}
	case workflow.StepEpisodes:
		if w.StateVersion != "v2.0" && w.StateVersion != "v3.0" {
			return errors.New("请先确认全剧设计 v2.0")
		}
		if ep < 1 || ep > w.TotalEpisodes {
			return fmt.Errorf("集数必须在1-%d之间", w.TotalEpisodes)
		}
	case workflow.StepDelivery:
		if w.ConfirmedEpisodes < w.TotalEpisodes {
			return errors.New("所有单集确认后才能生成AI交付")
		}
	default:
		return errors.New("未知工作流阶段")
	}
	return nil
}

func (s *ScriptWorkflowService) ConfirmStep(w *workflow.ScriptWorkflow, key string) error {
	var st workflow.ScriptWorkflowStep
	if err := database.DB.Where("workflow_id=? AND step_key=? AND (output_json <> '' OR output_text <> '')", w.ID, key).Order("version desc").First(&st).Error; err != nil {
		return err
	}
	if st.Status == "confirmed" {
		return nil
	}
	now := database.DB.NowFunc()
	updates := map[string]interface{}{"status": "confirmed", "confirmed_at": now}
	if err := database.DB.Model(&st).Updates(updates).Error; err != nil {
		return err
	}
	wu := map[string]interface{}{"status": workflow.StatusDraft}
	if key == workflow.StepCreative {
		wu["current_step"] = 2
		wu["state_version"] = "v1.0"
	} else if key == workflow.StepDesign {
		wu["current_step"] = 3
		wu["state_version"] = "v2.0"
	} else if key == workflow.StepDelivery {
		wu["current_step"] = 4
		wu["state_version"] = "v4.0"
		wu["status"] = workflow.StatusCompleted
	}
	return database.DB.Model(w).Updates(wu).Error
}

func (s *ScriptWorkflowService) ConfirmEpisode(w *workflow.ScriptWorkflow, episodeNo int) error {
	var ep workflow.ScriptEpisodeDraft
	if err := database.DB.Where("workflow_id=? AND episode_no=?", w.ID, episodeNo).First(&ep).Error; err != nil {
		return err
	}
	if ep.Status != "waiting_confirmation" && ep.Status != "confirmed" {
		return errors.New("本集尚未生成完成")
	}
	return PublishEpisode(w, &ep)
}
