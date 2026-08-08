package services

import (
	"errors"
	"gorm.io/gorm"
	workflow "spiritFruit/app/models/script_workflows"
	"spiritFruit/app/models/scripts"
	"spiritFruit/pkg/database"
	"time"
)

func PublishEpisode(w *workflow.ScriptWorkflow, ep *workflow.ScriptEpisodeDraft) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var script scripts.Scripts
		err := tx.Where("project_id=? AND episode_no=?", w.ProjectID, ep.EpisodeNo).First(&script).Error
		now := time.Now()
		projectID := w.ProjectID
		episodeNo := uint64(ep.EpisodeNo)
		title := ep.Title
		content := ep.ContentText
		jsonContent := ep.ContentJSON
		locked := int8(0)
		status := "confirmed"
		workflowID := w.ID
		draftID := ep.ID
		version := uint64(1)
		if err != nil {
			script = scripts.Scripts{ProjectId: &projectID, EpisodeNo: &episodeNo, Title: &title, Content: &content, ContentJSON: &jsonContent, IsLocked: &locked, GenerationStatus: &status, WorkflowId: &workflowID, EpisodeDraftId: &draftID, Version: &version, GeneratedAt: &now}
			if e := tx.Create(&script).Error; e != nil {
				return e
			}
		} else {
			if script.IsLocked != nil && *script.IsLocked == 1 {
				return errors.New("剧本已锁定，不能覆盖")
			}
			if script.Version != nil {
				version = *script.Version + 1
			}
			if e := tx.Model(&script).Updates(map[string]interface{}{"title": title, "content": content, "content_json": jsonContent, "workflow_id": workflowID, "episode_draft_id": draftID, "generation_status": status, "version": version, "generated_at": now}).Error; e != nil {
				return e
			}
		}
		if e := tx.Model(ep).Updates(map[string]interface{}{"status": "confirmed", "confirmed_at": now, "published_script_id": script.ID}).Error; e != nil {
			return e
		}
		var count int64
		tx.Model(&workflow.ScriptEpisodeDraft{}).Where("workflow_id=? AND status=?", w.ID, "confirmed").Count(&count)
		updates := map[string]interface{}{"confirmed_episodes": count, "status": workflow.StatusDraft}
		if int(count) == w.TotalEpisodes {
			updates["state_version"] = "v3.0"
			updates["current_step"] = 4
		}
		return tx.Model(w).Updates(updates).Error
	})
}
