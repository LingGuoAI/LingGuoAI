package v1

import (
	"github.com/gin-gonic/gin"
	workflow "spiritFruit/app/models/script_workflows"
	"spiritFruit/app/services"
	"spiritFruit/pkg/database"
	"spiritFruit/pkg/response"
	"strconv"
)

type ScriptWorkflowsController struct{ BaseADMINController }

func workflowID(c *gin.Context) (uint64, error) { return strconv.ParseUint(c.Param("id"), 10, 64) }

func (ctrl *ScriptWorkflowsController) Create(c *gin.Context) {
	var req struct {
		ProjectID uint64                 `json:"projectId" binding:"required"`
		Title     string                 `json:"title"`
		Config    map[string]interface{} `json:"config" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err)
		return
	}
	w, err := new(services.ScriptWorkflowService).Create(ctrl.GetAdminID(c), req.ProjectID, req.Title, req.Config)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, w)
}
func (ctrl *ScriptWorkflowsController) List(c *gin.Context) {
	projectID, _ := strconv.ParseUint(c.Query("projectId"), 10, 64)
	var list []workflow.ScriptWorkflow
	q := database.DB.Where("admin_id=?", ctrl.GetAdminID(c))
	if projectID > 0 {
		q = q.Where("project_id=?", projectID)
	}
	q.Order("id desc").Find(&list)
	response.Data(c, list)
}
func (ctrl *ScriptWorkflowsController) Show(c *gin.Context) {
	id, err := workflowID(c)
	if err != nil {
		response.Abort400(c)
		return
	}
	data, err := new(services.ScriptWorkflowService).Detail(id, ctrl.GetAdminID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Data(c, data)
}
func (ctrl *ScriptWorkflowsController) RunStage(c *gin.Context) {
	id, err := workflowID(c)
	if err != nil {
		response.Abort400(c)
		return
	}
	w, err := workflow.GetWorkflow(id, ctrl.GetAdminID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	var req struct {
		StepKey       string `json:"stepKey" binding:"required"`
		EpisodeNo     int    `json:"episodeNo"`
		ChangeRequest string `json:"changeRequest"`
	}
	if err = c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err)
		return
	}
	task, err := new(services.ScriptWorkflowService).StartStage(&w, req.StepKey, req.EpisodeNo, req.ChangeRequest)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Data(c, gin.H{"taskId": task.ID})
}
func (ctrl *ScriptWorkflowsController) GenerateEpisodes(c *gin.Context) {
	id, err := workflowID(c)
	if err != nil {
		response.Abort400(c)
		return
	}
	w, err := workflow.GetWorkflow(id, ctrl.GetAdminID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	var req struct {
		StartEpisode int `json:"startEpisode" binding:"required"`
		EndEpisode   int `json:"endEpisode" binding:"required"`
	}
	if err = c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err)
		return
	}
	if req.EndEpisode < req.StartEpisode || req.EndEpisode-req.StartEpisode+1 > 5 {
		response.Abort400(c, "每批最多生成5集")
		return
	}
	ids := []uint64{}
	for ep := req.StartEpisode; ep <= req.EndEpisode; ep++ {
		task, e := new(services.ScriptWorkflowService).StartStage(&w, workflow.StepEpisodes, ep, "")
		if e != nil {
			response.Error(c, e)
			return
		}
		ids = append(ids, task.ID)
	}
	response.Data(c, gin.H{"taskIds": ids})
}
func (ctrl *ScriptWorkflowsController) ConfirmStep(c *gin.Context) {
	id, _ := workflowID(c)
	w, err := workflow.GetWorkflow(id, ctrl.GetAdminID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	var req struct {
		StepKey string `json:"stepKey" binding:"required"`
	}
	if err = c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err)
		return
	}
	if err = new(services.ScriptWorkflowService).ConfirmStep(&w, req.StepKey); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c)
}
func (ctrl *ScriptWorkflowsController) ConfirmEpisode(c *gin.Context) {
	id, _ := workflowID(c)
	ep, _ := strconv.Atoi(c.Param("episodeNo"))
	w, err := workflow.GetWorkflow(id, ctrl.GetAdminID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	if err = new(services.ScriptWorkflowService).ConfirmEpisode(&w, ep); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c)
}
func (ctrl *ScriptWorkflowsController) UpdateEpisode(c *gin.Context) {
	id, _ := workflowID(c)
	epNo, _ := strconv.Atoi(c.Param("episodeNo"))
	w, err := workflow.GetWorkflow(id, ctrl.GetAdminID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	var req struct {
		ContentText string `json:"contentText" binding:"required"`
	}
	if err = c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err)
		return
	}
	var ep workflow.ScriptEpisodeDraft
	if err = database.DB.Where("workflow_id=? AND episode_no=?", w.ID, epNo).First(&ep).Error; err != nil {
		response.Error(c, err)
		return
	}
	database.DB.Model(&ep).Updates(map[string]interface{}{"content_text": req.ContentText, "status": "waiting_confirmation"})
	var count int64
	database.DB.Model(&workflow.ScriptEpisodeVersion{}).Where("episode_draft_id=?", ep.ID).Count(&count)
	database.DB.Create(&workflow.ScriptEpisodeVersion{EpisodeDraftID: ep.ID, Version: int(count) + 1, Source: "manual", ContentJSON: ep.ContentJSON, ContentText: req.ContentText, CreatedBy: w.AdminID})
	response.Success(c)
}
func (ctrl *ScriptWorkflowsController) Versions(c *gin.Context) {
	id, _ := workflowID(c)
	epNo, _ := strconv.Atoi(c.Param("episodeNo"))
	w, err := workflow.GetWorkflow(id, ctrl.GetAdminID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	var ep workflow.ScriptEpisodeDraft
	if err = database.DB.Where("workflow_id=? AND episode_no=?", w.ID, epNo).First(&ep).Error; err != nil {
		response.Error(c, err)
		return
	}
	var list []workflow.ScriptEpisodeVersion
	database.DB.Where("episode_draft_id=?", ep.ID).Order("version desc").Find(&list)
	response.Data(c, list)
}
