package v1

import (
	"fmt"
	"spiritFruit/app/models"
	"spiritFruit/app/models/projects"
	"spiritFruit/app/models/scripts"
	"spiritFruit/app/models/shot_characters"
	"spiritFruit/app/models/shot_props"
	"spiritFruit/app/models/shots"
	"spiritFruit/app/requests"
	"spiritFruit/pkg/database"
	"spiritFruit/pkg/response"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type ShotsController struct {
	BaseADMINController
}

// Index 镜头表列表
// @Summary 镜头表列表
// @Description 获取镜头表列表，支持多种搜索条件
// @Tags Shots
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param per_page query int false "每页数量"
// @Param projectId query string false "所属项目ID"
// @Param scriptId query string false "所属剧本/分集ID"
// @Param sequenceNo query string false "镜头序号"
// @Success 200 {object} response.Response{data=[]shots.Shots} "镜头表列表"
// @Failure 400 {object} response.ErrorResponse "参数错误"
// @Failure 500 {object} response.ErrorResponse "服务器错误"
// @Router /admin/v1/shots [get]
func (ctrl *ShotsController) Index(c *gin.Context) {
	// 构建搜索条件
	where := ctrl.buildSearchConditions(c)

	// 获取分页参数
	perPage := 10
	if perPageStr := c.Query("pageSize"); perPageStr != "" {
		fmt.Println(perPageStr)
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= 1000 {
			perPage = pp
		}
	}
	fmt.Println(perPage)

	where["ORDER"] = "id asc"
	data, pager := shots.Paginate(c, perPage, where)
	response.JSON(c, gin.H{
		"code": 0,
		"data": map[string]interface{}{
			"total": pager.TotalCount,
			"list":  data,
		},
		"message": "success",
	})
}

// buildSearchConditions 构建搜索条件
func (ctrl *ShotsController) buildSearchConditions(c *gin.Context) map[string]interface{} {
	where := map[string]interface{}{}

	// 所属项目ID搜索

	if projectId := strings.TrimSpace(c.Query("projectId")); projectId != "" {
		where["project_id"] = projectId
	}

	// 所属剧本/分集ID搜索

	if scriptId := strings.TrimSpace(c.Query("scriptId")); scriptId != "" {
		where["script_id"] = scriptId
	}

	// 镜头序号搜索

	if sequenceNo := strings.TrimSpace(c.Query("sequenceNo")); sequenceNo != "" {
		where["sequence_no"] = sequenceNo
	}

	return where
}

// GetProjectsSelectList 获取短剧项目选择列表
// @Summary 获取短剧项目选择列表
// @Description 获取短剧项目的简化列表，用于镜头表中的短剧项目选择
// @Tags Shots
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]projects.Projects} "短剧项目选择列表"
// @Failure 500 {object} response.ErrorResponse "服务器错误"
// @Router /admin/v1/shots/getProjectsSelectList [get]
func (ctrl *ShotsController) GetProjectsSelectList(c *gin.Context) {
	list := projects.All()
	response.JSON(c, gin.H{
		"code":    0,
		"data":    list,
		"message": "success",
	})
}

// GetScriptsSelectList 获取剧本选择列表
// @Summary 获取剧本选择列表
// @Description 获取剧本的简化列表，用于镜头表中的剧本选择
// @Tags Shots
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]scripts.Scripts} "剧本选择列表"
// @Failure 500 {object} response.ErrorResponse "服务器错误"
// @Router /admin/v1/shots/getScriptsSelectList [get]
func (ctrl *ShotsController) GetScriptsSelectList(c *gin.Context) {
	list := scripts.All()
	response.JSON(c, gin.H{
		"code":    0,
		"data":    list,
		"message": "success",
	})
}

// Show 镜头表详情
// @Summary 镜头表详情
// @Description 获取镜头表详情
// @Tags Shots
// @Accept json
// @Produce json
// @Param id path string true "Shots ID"
// @Success 200 {object} response.Response{data=shots.Shots} "镜头表详情"
// @Failure 404 {object} response.ErrorResponse "镜头表不存在"
// @Failure 500 {object} response.ErrorResponse "服务器错误"
// @Router /admin/v1/shots/{id} [get]
func (ctrl *ShotsController) Show(c *gin.Context) {
	shotsModel := shots.Get(c.Param("id"))
	if shotsModel.ID == 0 {
		response.JSON(c, gin.H{
			"code":    404,
			"message": "数据不存在",
			"data":    nil,
		})
		return
	}
	response.JSON(c, gin.H{
		"code":    0,
		"data":    shotsModel,
		"message": "success",
	})
}

// Store 创建镜头表
// @Summary 创建镜头表
// @Description 创建新的镜头表
// @Tags Shots
// @Accept json
// @Produce json
// @Param request body requests.ShotsRequest true "镜头表信息"
// @Success 201 {object} response.Response{data=shots.Shots} "创建成功"
// @Failure 400 {object} response.ErrorResponse "参数错误"
// @Failure 422 {object} response.ErrorResponse "验证失败"
// @Failure 500 {object} response.ErrorResponse "服务器错误"
// @Router /admin/v1/shots [post]
func (ctrl *ShotsController) Store(c *gin.Context) {
	request := requests.ShotsRequest{}
	if ok := requests.Validate(c, &request, requests.ShotsSave); !ok {
		return
	}
	var sceneID *uint64
	if request.SceneId != 0 {
		sceneID = &request.SceneId
	}
	shotsModel := shots.Shots{
		ProjectId:      &request.ProjectId,
		ScriptId:       &request.ScriptId,
		SceneId:        sceneID,
		SequenceNo:     &request.SequenceNo,
		Title:          &request.Title,
		ShotType:       &request.ShotType,
		CameraMovement: &request.CameraMovement,
		Angle:          &request.Angle,
		Dialogue:       &request.Dialogue,
		Time:           &request.Time,
		Location:       &request.Location,
		Action:         &request.Action,
		VisualDesc:     &request.VisualDesc,
		Result:         &request.Result,
		Atmosphere:     &request.Atmosphere,
		ImagePrompt:    &request.ImagePrompt,
		VideoPrompt:    &request.VideoPrompt,
		AudioPrompt:    &request.AudioPrompt,
		ImageUrl:       &request.ImageUrl,
		VideoUrl:       &request.VideoUrl,
		AudioUrl:       &request.AudioUrl,
		DurationMs:     &request.DurationMs,
		Status:         &request.Status,
	}

	shotsModel.Create()
	if shotsModel.ID > 0 {
		response.JSON(c, gin.H{
			"code":    0,
			"data":    shotsModel,
			"message": "success",
		})
	} else {
		response.Abort500(c, "创建失败，请稍后尝试~")
	}
}

// Update 更新镜头表
// @Summary 更新镜头表
// @Description 更新镜头表信息
// @Tags Shots
// @Accept json
// @Produce json
// @Param id path string true "Shots ID"
// @Param request body requests.ShotsRequest true "镜头表信息"
// @Success 200 {object} response.Response{data=shots.Shots} "更新成功"
// @Failure 400 {object} response.ErrorResponse "参数错误"
// @Failure 404 {object} response.ErrorResponse "镜头表不存在"
// @Failure 422 {object} response.ValidationErrorResponse "验证失败"
// @Failure 500 {object} response.ErrorResponse "服务器错误"
// @Router /admin/v1/shots/{id} [put]
func (ctrl *ShotsController) Update(c *gin.Context) {
	// 1. 验证数据是否存在
	id := c.Param("id")
	existingShots := shots.Get(id)
	if existingShots.ID == 0 {
		response.JSON(c, gin.H{
			"code":    404,
			"message": "数据不存在",
			"data":    nil,
		})
		return
	}

	// 2. 验证请求参数
	request := requests.ShotsRequest{}
	if bindOk := requests.Validate(c, &request, requests.ShotsSave); !bindOk {
		return
	}

	// 3. 开启数据库事务
	db := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			db.Rollback()
			response.Abort500(c, "更新失败，请稍后尝试~")
		}
	}()

	// 4. 使用新的模型实例进行更新，避免关联对象的影响
	updateShots := &shots.Shots{
		BaseModel: models.BaseModel{ID: existingShots.ID},
	}

	// 赋值字段
	updateShots.ProjectId = &request.ProjectId
	updateShots.ScriptId = &request.ScriptId
	if request.SceneId != 0 {
		updateShots.SceneId = &request.SceneId
	} else {
		updateShots.SceneId = nil
	}
	updateShots.SequenceNo = &request.SequenceNo
	updateShots.Title = &request.Title
	updateShots.ShotType = &request.ShotType
	updateShots.CameraMovement = &request.CameraMovement
	updateShots.Angle = &request.Angle
	updateShots.Dialogue = &request.Dialogue
	updateShots.Time = &request.Time
	updateShots.Location = &request.Location
	updateShots.Action = &request.Action
	updateShots.VisualDesc = &request.VisualDesc
	updateShots.Result = &request.Result
	updateShots.Atmosphere = &request.Atmosphere
	updateShots.ImagePrompt = &request.ImagePrompt
	updateShots.VideoPrompt = &request.VideoPrompt
	updateShots.AudioPrompt = &request.AudioPrompt
	updateShots.ImageUrl = &request.ImageUrl
	updateShots.VideoUrl = &request.VideoUrl
	updateShots.AudioUrl = &request.AudioUrl
	updateShots.DurationMs = &request.DurationMs
	updateShots.Status = &request.Status
	updateShots.UpdatedAt = time.Now()
	updateShots.CreatedAt = existingShots.CreatedAt

	// 5. 执行更新镜头基本信息
	result := db.Save(updateShots)
	if result.Error != nil {
		db.Rollback()
		response.Abort500(c, "更新镜头信息失败："+result.Error.Error())
		return
	}

	// ===========================
	// 6. 处理【角色】关联表更新
	// ===========================
	if request.CharacterIds != nil {
		// 删除旧的角色关联
		if err := db.Where("shot_id = ?", existingShots.ID).Delete(&shot_characters.ShotCharacters{}).Error; err != nil {
			db.Rollback()
			response.Abort500(c, "清理旧角色关联失败："+err.Error())
			return
		}

		// 批量插入新的角色关联
		if len(request.CharacterIds) > 0 {
			shotChars := make([]shot_characters.ShotCharacters, len(request.CharacterIds))
			for i, charId := range request.CharacterIds {
				shotChars[i] = shot_characters.ShotCharacters{
					ShotId:      existingShots.ID,
					CharacterId: charId,
				}
			}
			if err := db.Create(&shotChars).Error; err != nil {
				db.Rollback()
				response.Abort500(c, "创建新角色关联失败："+err.Error())
				return
			}
		}
	}

	// ===========================
	// 7. 处理【道具】关联表更新
	// ===========================
	if request.PropIds != nil {
		// 删除旧的道具关联
		if err := db.Where("shot_id = ?", existingShots.ID).Delete(&shot_props.ShotProps{}).Error; err != nil {
			db.Rollback()
			response.Abort500(c, "清理旧道具关联失败："+err.Error())
			return
		}

		// 批量插入新的道具关联
		if len(request.PropIds) > 0 {
			shotPropsList := make([]shot_props.ShotProps, len(request.PropIds))
			for i, propId := range request.PropIds {
				shotPropsList[i] = shot_props.ShotProps{
					ShotId:  existingShots.ID,
					PropsId: propId,
				}
			}
			if err := db.Create(&shotPropsList).Error; err != nil {
				db.Rollback()
				response.Abort500(c, "创建新道具关联失败："+err.Error())
				return
			}
		}
	}

	// 8. 提交事务
	if err := db.Commit().Error; err != nil {
		response.Abort500(c, "提交事务失败："+err.Error())
		return
	}

	// 9. 重新获取完整数据返回给前端
	updatedShots := shots.Get(id)
	response.JSON(c, gin.H{
		"code":    0,
		"data":    updatedShots,
		"message": "success",
	})
}

// Delete 删除镜头表
// @Summary 删除镜头表
// @Description 删除镜头表
// @Tags Shots
// @Accept json
// @Produce json
// @Param id path string true "Shots ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 404 {object} response.ErrorResponse "镜头表不存在"
// @Failure 403 {object} response.ErrorResponse "权限不足"
// @Failure 500 {object} response.ErrorResponse "服务器错误"
// @Router /admin/v1/shots/{id} [delete]
func (ctrl *ShotsController) Delete(c *gin.Context) {
	shotsModel := shots.Get(c.Param("id"))
	if shotsModel.ID == 0 {
		response.JSON(c, gin.H{
			"code":    404,
			"message": "数据不存在",
			"data":    nil,
		})
		return
	}

	rowsAffected := shotsModel.Delete()
	if rowsAffected > 0 {
		response.JSON(c, gin.H{
			"code":    0,
			"data":    "",
			"message": "success",
		})
		return
	}

	response.Abort500(c, "删除失败，请稍后尝试~")
}
