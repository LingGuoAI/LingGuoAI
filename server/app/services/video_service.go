package services

import (
	"encoding/json"
	"fmt"
	"spiritFruit/app/models/async_tasks"
	"spiritFruit/app/models/scripts"
	"spiritFruit/app/models/shots"
	"spiritFruit/app/models/source"

	"spiritFruit/pkg/asynq"
	"spiritFruit/pkg/console"
	"spiritFruit/pkg/database"
)

type TimelineClipReq struct {
	AssetID    interface{}            `json:"asset_id"`
	ShotID     uint64                 `json:"shotId"`
	URL        string                 `json:"url"`
	Order      int                    `json:"order"`
	StartTime  float64                `json:"start_time"`
	EndTime    float64                `json:"end_time"`
	Duration   float64                `json:"duration"`
	Transition map[string]interface{} `json:"transition"`
}

type FinalizeEpisodeReq struct {
	ProjectID     uint64            `json:"projectId" binding:"required"`
	EpisodeNumber uint64            `json:"episodeNumber" binding:"required"`
	ScriptID      uint64            `json:"scriptId"`
	Clips         []TimelineClipReq `json:"clips"`
}

type VideoService struct{}

// FinalizeEpisode 参数类型改为使用本包的 FinalizeEpisodeReq
func (s *VideoService) FinalizeEpisode(req FinalizeEpisodeReq) (map[string]interface{}, error) {
	// episodeNumber 是展示序号，shots.script_id 保存的是 scripts.id，二者不能混用。
	scriptID := req.ScriptID
	if scriptID == 0 {
		var script scripts.Scripts
		if err := database.DB.Where("project_id = ? AND episode_no = ?", req.ProjectID, req.EpisodeNumber).First(&script).Error; err == nil {
			scriptID = script.ID
		}
	}

	var shotList []shots.Shots
	if scriptID > 0 {
		database.DB.Where("project_id = ? AND script_id = ?", req.ProjectID, scriptID).Order("sequence_no asc").Find(&shotList)
	}

	if len(req.Clips) == 0 && len(shotList) == 0 {
		return nil, fmt.Errorf("该集数下没有找到任何分镜")
	}

	sceneMap := make(map[uint64]shots.Shots)
	for _, shot := range shotList {
		sceneMap[shot.ID] = shot
	}

	var mergeClips []asynq.MergeClip
	var skippedScenes []uint64

	if len(req.Clips) > 0 {
		console.Success(fmt.Sprintf("使用前端时间线数据合成，片段数: %d", len(req.Clips)))

		for _, clip := range req.Clips {
			videoURL := clip.URL

			// URL 未提交时，从素材表解析 assetId。
			if videoURL == "" && clip.AssetID != nil {
				var asset source.Source
				if id, ok := clip.AssetID.(float64); ok && id > 0 {
					database.DB.First(&asset, uint64(id))
				} else if id, ok := clip.AssetID.(string); ok && id != "" {
					database.DB.First(&asset, id)
				}
				if asset.VideoUrl != nil {
					videoURL = *asset.VideoUrl
				}
			}

			if videoURL == "" && clip.ShotID != 0 {
				if scene, exists := sceneMap[clip.ShotID]; exists {
					if scene.VideoUrl != nil && *scene.VideoUrl != "" {
						videoURL = *scene.VideoUrl
					}
				}
			}

			if videoURL == "" {
				skippedScenes = append(skippedScenes, clip.ShotID)
				continue
			}

			mergeClips = append(mergeClips, asynq.MergeClip{
				URL:        videoURL,
				Duration:   clip.Duration,
				StartTime:  clip.StartTime,
				EndTime:    clip.EndTime,
				Transition: clip.Transition,
			})
		}
	} else {
		fmt.Println("[INFO] 无时间线数据，按分镜默认顺序拼接")

		for _, scene := range shotList {
			var videoURL string

			if scene.VideoUrl != nil && *scene.VideoUrl != "" {
				videoURL = *scene.VideoUrl
			}

			if videoURL == "" {
				// 🔴 将 seqNo 声明为 uint64，并做对应转换
				var seqNo uint64 = 0
				if scene.SequenceNo != nil {
					seqNo = uint64(*scene.SequenceNo)
				}
				skippedScenes = append(skippedScenes, seqNo)
				continue
			}

			duration := 3.0
			if scene.DurationMs != nil {
				duration = float64(*scene.DurationMs) / 1000.0
			}

			mergeClips = append(mergeClips, asynq.MergeClip{
				URL:      videoURL,
				Duration: duration,
			})
		}
	}

	if len(mergeClips) == 0 {
		return nil, fmt.Errorf("没有找到任何可用的视频片段用于合成")
	}

	mergeRecordID := uint64(1) // 您的实际 Merge ID

	payload := asynq.MergeVideoPayload{
		MergeID:   mergeRecordID,
		ProjectID: req.ProjectID,
		EpisodeID: scriptID,
		Title:     fmt.Sprintf("项目%d-第%d集合成", req.ProjectID, req.EpisodeNumber),
		Clips:     mergeClips,
	}

	payloadBytes, _ := json.Marshal(payload)
	task := async_tasks.AsyncTask{
		ProjectID: req.ProjectID,
		RelID:     mergeRecordID,
		Type:      asynq.TypeMergeVideo,
		Status:    async_tasks.StatusPending,
		Payload:   string(payloadBytes),
	}
	database.DB.Create(&task)

	payload.AsyncTaskID = task.ID
	_, err := asynq.EnqueueMergeVideo(payload)
	if err != nil {
		task.MarkAsFailed(err)
		return nil, err
	}

	result := map[string]interface{}{
		"message":      "视频合成任务已创建，正在后台处理",
		"merge_id":     mergeRecordID,
		"task_id":      task.ID,
		"scenes_count": len(mergeClips),
	}

	if len(skippedScenes) > 0 {
		result["skipped_scenes"] = skippedScenes
		result["warning"] = fmt.Sprintf("已跳过 %d 个未生成视频的场景", len(skippedScenes))
	}

	return result, nil
}
