package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
	"spiritFruit/app/models/async_tasks"
	workflow "spiritFruit/app/models/script_workflows"
	"spiritFruit/app/services"
	myasynq "spiritFruit/pkg/asynq"
	"spiritFruit/pkg/database"
	"spiritFruit/pkg/openai"
	creator "spiritFruit/pkg/prompt/script_creator"
	"unicode/utf8"
)

func HandleScriptWorkflow(ctx context.Context, t *asynq.Task) error {
	var p myasynq.ScriptWorkflowPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	var task async_tasks.AsyncTask
	if err := database.DB.First(&task, p.AsyncTaskID).Error; err != nil {
		return nil
	}
	task.MarkAsProcessing()
	w, err := workflow.GetWorkflow(p.WorkflowID, *task.AdminID)
	if err != nil {
		task.MarkAsFailed(err)
		return err
	}
	provider, model := services.NewTextProvider(w.AdminID)
	task.UpdateProgress(20)
	prompt, err := workflowPrompt(&w, p)
	if err != nil {
		task.MarkAsFailed(err)
		return err
	}
	task.UpdateProgress(30)
	raw, err := provider.GenerateScript(openai.ScriptRequest{Messages: []openai.ChatMessage{{Role: "user", Content: prompt}}, Temperature: .7, MaxTokens: 16000})
	if err != nil {
		task.MarkAsFailed(err)
		markWorkflowFailure(p)
		return err
	}
	task.UpdateProgress(75)
	clean, err := creator.ValidateJSON(raw)
	if err != nil {
		err = fmt.Errorf("模型输出不是有效JSON: %w", err)
		task.MarkAsFailed(err)
		markWorkflowFailure(p)
		return err
	}
	if err = saveWorkflowResult(&w, p, clean, model, task.ID); err != nil {
		task.MarkAsFailed(err)
		return err
	}
	task.MarkAsSuccess(fmt.Sprintf(`{"workflowId":%d,"stepKey":%q,"episodeNo":%d}`, w.ID, p.StepKey, p.EpisodeNo))
	return nil
}

func latestStep(workflowID uint64, key string) (workflow.ScriptWorkflowStep, error) {
	var s workflow.ScriptWorkflowStep
	err := database.DB.Where("workflow_id=? AND step_key=?", workflowID, key).Order("version desc").First(&s).Error
	return s, err
}

func workflowPrompt(w *workflow.ScriptWorkflow, p myasynq.ScriptWorkflowPayload) (string, error) {
	switch p.StepKey {
	case workflow.StepCreative:
		return creator.CreativePrompt(w.ConfigJSON), nil
	case workflow.StepDesign:
		c, e := latestStep(w.ID, workflow.StepCreative)
		if e != nil {
			return "", e
		}
		return creator.DesignPrompt(c.OutputJSON), nil
	case workflow.StepEpisodes:
		c, e := latestStep(w.ID, workflow.StepCreative)
		if e != nil {
			return "", e
		}
		d, e := latestStep(w.ID, workflow.StepDesign)
		if e != nil {
			return "", e
		}
		prev := ""
		if p.EpisodeNo > 1 {
			var ep workflow.ScriptEpisodeDraft
			database.DB.Where("workflow_id=? AND episode_no=?", w.ID, p.EpisodeNo-1).First(&ep)
			prev = ep.ContentText
		}
		return creator.EpisodePrompt(c.OutputJSON, d.OutputJSON, p.EpisodeNo, prev, p.ChangeRequest), nil
	case workflow.StepDelivery:
		d, e := latestStep(w.ID, workflow.StepDesign)
		if e != nil {
			return "", e
		}
		var eps []workflow.ScriptEpisodeDraft
		database.DB.Where("workflow_id=? AND status=?", w.ID, "confirmed").Order("episode_no").Find(&eps)
		b, _ := json.Marshal(eps)
		return creator.DeliveryPrompt(d.OutputJSON, string(b)), nil
	}
	return "", errors.New("未知工作流阶段")
}

func saveWorkflowResult(w *workflow.ScriptWorkflow, p myasynq.ScriptWorkflowPayload, clean, model string, taskID uint64) error {
	if p.StepKey == workflow.StepEpisodes {
		ep, err := creator.ParseEpisode(clean)
		if err != nil {
			return err
		}
		if ep.EpisodeNo == 0 {
			ep.EpisodeNo = p.EpisodeNo
		}
		warnings := creator.ValidateEpisode(ep)
		report, _ := json.Marshal(map[string]interface{}{"warnings": warnings, "passed": len(warnings) == 0})
		text := creator.RenderEpisode(ep)
		var draft workflow.ScriptEpisodeDraft
		if err = database.DB.Where("workflow_id=? AND episode_no=?", w.ID, p.EpisodeNo).First(&draft).Error; err != nil {
			return err
		}
		var count int64
		database.DB.Model(&workflow.ScriptEpisodeVersion{}).Where("episode_draft_id=?", draft.ID).Count(&count)
		ver := workflow.ScriptEpisodeVersion{EpisodeDraftID: draft.ID, Version: int(count) + 1, Source: "generated", ChangeRequest: p.ChangeRequest, ContentJSON: clean, ContentText: text, CreatedBy: w.AdminID}
		return database.DB.Transaction(func(tx *gorm.DB) error {
			if e := tx.Create(&ver).Error; e != nil {
				return e
			}
			return tx.Model(&draft).Updates(map[string]interface{}{"title": ep.Title, "status": "waiting_confirmation", "content_json": clean, "content_text": text, "duration_seconds": ep.EstimatedDuration, "word_count": utf8.RuneCountInString(text), "quality_report_json": string(report), "prompt_version": creator.Version, "model_snapshot": model, "async_task_id": taskID}).Error
		})
	}
	var step workflow.ScriptWorkflowStep
	if err := database.DB.Where("workflow_id=? AND step_key=? AND async_task_id=?", w.ID, p.StepKey, taskID).First(&step).Error; err != nil {
		return err
	}
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if e := tx.Model(&step).Updates(map[string]interface{}{"status": "waiting_confirmation", "output_json": clean, "output_text": clean, "model_snapshot": model}).Error; e != nil {
			return e
		}
		return tx.Model(w).Updates(map[string]interface{}{"status": workflow.StatusWaiting}).Error
	})
}

func markWorkflowFailure(p myasynq.ScriptWorkflowPayload) {
	if p.StepKey == workflow.StepEpisodes {
		database.DB.Model(&workflow.ScriptEpisodeDraft{}).Where("workflow_id=? AND episode_no=?", p.WorkflowID, p.EpisodeNo).Update("status", "failed")
	} else {
		database.DB.Model(&workflow.ScriptWorkflowStep{}).Where("workflow_id=? AND step_key=? AND async_task_id=?", p.WorkflowID, p.StepKey, p.AsyncTaskID).Update("status", "failed")
	}
}
