package v1

import (
	"spiritFruit/app/models"
	"spiritFruit/app/models/ai_config"
	"spiritFruit/app/requests"
	"spiritFruit/pkg/database"
	"spiritFruit/pkg/response"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type AiConfigController struct {
	BaseADMINController
}

// Index AI配置列表
// @Summary AI配置列表
// @Description 获取AI服务配置列表，支持多种搜索条件
// @Tags AiConfig
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param per_page query int false "每页数量"
// @Param service_type query string false "服务类型"
// @Param provider query string false "厂商提供商"
// @Param is_active query int false "状态(1-启用 0-禁用)"
// @Success 200 {object} response.Response{data=[]ai_config.AiConfig} "AI配置列表"
// @Router /admin/v1/ai-config [get]
func (ctrl *AiConfigController) Index(c *gin.Context) {
	// 构建搜索条件
	where := ctrl.buildSearchConditions(c)

	// 获取分页参数
	perPage := 10
	if perPageStr := c.Query("pageSize"); perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= 100 {
			perPage = pp
		}
	}

	data, pager := ai_config.Paginate(c, perPage, where)
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
func (ctrl *AiConfigController) buildSearchConditions(c *gin.Context) map[string]interface{} {
	where := map[string]interface{}{}

	// 配置只获取当前登录用户的配置
	if adminID := c.GetString("current_admin_id"); adminID != "" {
		where["admin_id"] = adminID
	}

	if serviceType := strings.TrimSpace(c.Query("service_type")); serviceType != "" {
		where["service_type"] = serviceType
	}

	if provider := strings.TrimSpace(c.Query("provider")); provider != "" {
		where["provider"] = provider
	}

	if isActive := strings.TrimSpace(c.Query("is_active")); isActive != "" {
		where["is_active"] = isActive
	}

	return where
}

// Show AI配置详情
// @Summary AI配置详情
// @Router /admin/v1/ai-config/{id} [get]
func (ctrl *AiConfigController) Show(c *gin.Context) {
	configModel := ai_config.Get(c.Param("id"))
	if configModel.ID == 0 || (configModel.AdminID != nil && strconv.FormatUint(*configModel.AdminID, 10) != c.GetString("current_admin_id")) {
		response.JSON(c, gin.H{
			"code":    404,
			"message": "数据不存在",
			"data":    nil,
		})
		return
	}
	response.JSON(c, gin.H{
		"code":    0,
		"data":    configModel,
		"message": "success",
	})
}

// Store 创建AI配置
// @Summary 创建AI配置
// @Router /admin/v1/ai-config [post]
func (ctrl *AiConfigController) Store(c *gin.Context) {
	request := requests.AiConfigRequest{}
	if ok := requests.Validate(c, &request, requests.AiConfigSave); !ok {
		return
	}

	currentAdminIDStr := c.GetString("current_admin_id")
	currentAdminID, _ := strconv.ParseUint(currentAdminIDStr, 10, 64)

	configModel := ai_config.AiConfig{
		Name:        &request.Name,
		ServiceType: &request.ServiceType,
		Provider:    &request.Provider,
		BaseUrl:     &request.BaseUrl,
		ApiKey:      &request.ApiKey,
		Model:       request.Model,
		Priority:    &request.Priority,
		IsActive:    &request.IsActive,
		AdminID:     &currentAdminID,
	}

	configModel.Create()
	if configModel.ID > 0 {
		response.JSON(c, gin.H{
			"code":    0,
			"data":    configModel,
			"message": "success",
		})
	} else {
		response.Abort500(c, "创建失败，请稍后尝试~")
	}
}

// Update 更新AI配置
// @Summary 更新AI配置
// @Router /admin/v1/ai-config/{id} [put]
func (ctrl *AiConfigController) Update(c *gin.Context) {
	id := c.Param("id")
	existingConfig := ai_config.Get(id)
	if existingConfig.ID == 0 || (existingConfig.AdminID != nil && strconv.FormatUint(*existingConfig.AdminID, 10) != c.GetString("current_admin_id")) {
		response.JSON(c, gin.H{
			"code":    404,
			"message": "数据不存在",
			"data":    nil,
		})
		return
	}

	request := requests.AiConfigRequest{}
	if bindOk := requests.Validate(c, &request, requests.AiConfigSave); !bindOk {
		return
	}

	// 增量/全量更新
	updateConfig := &ai_config.AiConfig{
		BaseModel: models.BaseModel{ID: existingConfig.ID},
	}
	updateConfig.AdminID = existingConfig.AdminID
	updateConfig.Name = &request.Name
	updateConfig.ServiceType = &request.ServiceType
	updateConfig.Provider = &request.Provider
	updateConfig.BaseUrl = &request.BaseUrl
	updateConfig.ApiKey = &request.ApiKey
	updateConfig.Model = request.Model
	updateConfig.Priority = &request.Priority
	updateConfig.IsActive = &request.IsActive

	updateConfig.UpdatedAt = time.Now()
	updateConfig.CreatedAt = existingConfig.CreatedAt

	result := database.DB.Save(updateConfig)

	if result.Error != nil {
		response.Abort500(c, "更新失败："+result.Error.Error())
		return
	}

	if result.RowsAffected > 0 {
		updatedConfig := ai_config.Get(id)
		response.JSON(c, gin.H{
			"code":    0,
			"data":    updatedConfig,
			"message": "success",
		})
	} else {
		response.Abort500(c, "更新失败，请稍后尝试~")
	}
}

// Delete 删除AI配置
// @Summary 删除AI配置
// @Router /admin/v1/ai-config/{id} [delete]
func (ctrl *AiConfigController) Delete(c *gin.Context) {
	configModel := ai_config.Get(c.Param("id"))
	if configModel.ID == 0 || (configModel.AdminID != nil && strconv.FormatUint(*configModel.AdminID, 10) != c.GetString("current_admin_id")) {
		response.JSON(c, gin.H{
			"code":    404,
			"message": "数据不存在",
			"data":    nil,
		})
		return
	}

	rowsAffected := configModel.Delete()
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
