// Package routes 注册路由
package routes

import (
	controllers "spiritFruit/app/http/controllers/admin/v1"
	"spiritFruit/app/http/controllers/admin/v1/auth"
	"spiritFruit/app/http/middlewares"
	"spiritFruit/pkg/config"

	"github.com/gin-gonic/gin"
)

// RegisterAdminAPIRoutes 注册管理端路由
func RegisterAdminAPIRoutes(r *gin.Engine) {
	var v1 *gin.RouterGroup
	if len(config.Get("app.api_domain")) == 0 {
		v1 = r.Group("/admin/v1")
	} else {
		v1 = r.Group("/v1")
	}

	// 静态文件服务 - 直接注册到根路径
	r.Static("/uploads", "./uploads")

	// 认证路由组 - 无需认证
	authGroup := v1.Group("/auth")
	{
		lgc := new(auth.LoginController)
		authGroup.POST("/login/using-phone", lgc.LoginByPassword)
		authGroup.POST("/register", lgc.Register)
	}

	// 需要认证的路由组
	{
		// 上传图片相关路由
		uploadGroup := v1.Group("/upload")
		{
			uploadController := new(controllers.UploadsController)
			uploadGroup.POST("/singleUpload", uploadController.Upload)
		}

		// 统计数据路由
		statisticsController := new(controllers.StatisticsController)
		protectedGroup := v1.Group("/statistics").Use(middlewares.AuthAdminJWT())
		protectedGroup.GET("/statistics", statisticsController.StatisticsData)
		protectedGroup.GET("/trend", statisticsController.GetTableTrend)
		protectedGroup.GET("/detail", statisticsController.GetDetailStatistics)

		// 角色相关路由
		charactersGroup := v1.Group("/characters").Use(middlewares.AuthAdminJWT())
		{
			charactersController := new(controllers.CharactersController)

			// 基础CRUD路由
			charactersGroup.GET("", charactersController.Index)    // 获取角色列表
			charactersGroup.GET("/:id", charactersController.Show) // 获取角色详情
			charactersGroup.POST("", charactersController.Store)   // 创建角色

			charactersGroup.PUT("/:id", charactersController.Update)    // 更新角色
			charactersGroup.DELETE("/:id", charactersController.Delete) // 删除角色
			// 短剧项目选择列表路由
			charactersGroup.GET("/getProjectsSelectList", charactersController.GetProjectsSelectList) // 获取短剧项目选择列表
		}

		// 短剧项目相关路由
		projectsGroup := v1.Group("/projects").Use(middlewares.AuthAdminJWT())
		{
			projectsController := new(controllers.ProjectsController)

			// 基础CRUD路由
			projectsGroup.GET("", projectsController.Index)    // 获取短剧项目列表
			projectsGroup.GET("/:id", projectsController.Show) // 获取短剧项目详情
			projectsGroup.POST("", projectsController.Store)   // 创建短剧项目

			projectsGroup.PUT("/:id", projectsController.Update)    // 更新短剧项目
			projectsGroup.DELETE("/:id", projectsController.Delete) // 删除短剧项目
		}

		// ============== 任务状态查询路由 ==============
		tasksGroup := v1.Group("/tasks").Use(middlewares.AuthAdminJWT())
		{
			// 假设你创建了 TasksController
			tasksController := new(controllers.TasksController)
			videoController := new(controllers.VideoController)
			aiController := new(controllers.AiController)
			tasksGroup.GET("/:id", tasksController.Show)                                                // 查询任务详情(包含进度和结果)
			tasksGroup.POST("/:id/cancel", tasksController.Cancel)                                      // 取消任务
			tasksGroup.POST("/generateCharacters", aiController.GenerateCharacters)                     // 提取角色
			tasksGroup.POST("/extractScenes", aiController.ExtractScenes)                               // 提取场景
			tasksGroup.POST("/generateCharacterImage", aiController.GenerateCharacterImage)             // 单个角色生图
			tasksGroup.POST("/batchGenerateCharacterImages", aiController.BatchGenerateCharacterImages) // 多个角色生图
			tasksGroup.POST("/generateSceneImage", aiController.GenerateSceneImage)                     // 单个场景生图
			tasksGroup.POST("/batchGenerateSceneImages", aiController.BatchGenerateSceneImages)         // 批量场景生图
			tasksGroup.POST("/generateShots", aiController.GenerateShots)                               // 拆分分镜
			// 道具相关
			tasksGroup.POST("/extractProps", aiController.ExtractProps)                       // 提取道具
			tasksGroup.POST("/generatePropImage", aiController.GeneratePropImage)             // 单个道具生图
			tasksGroup.POST("/batchGeneratePropImages", aiController.BatchGeneratePropImages) // 批量道具生图

			tasksGroup.POST("/extractPrompt", aiController.ExtractPrompt)                 // 提取提示词
			tasksGroup.POST("/generateImageByPrompt", aiController.GenerateImageByPrompt) // 根据帧提示词生成图片

			tasksGroup.POST("/generateVideo", aiController.GenerateVideo)
			tasksGroup.POST("/mergeVideo", videoController.FinalizeEpisode) // 合并视频

			// AI 配置测试接口
			tasksGroup.POST("/testTextConfig", aiController.TestTextConfig)
			tasksGroup.POST("/testImageConfig", aiController.TestImageConfig)
			tasksGroup.POST("/testVideoConfig", aiController.TestVideoConfig)
		}

		// 剧本相关路由
		workflowGroup := v1.Group("/script-workflows").Use(middlewares.AuthAdminJWT())
		{
			ctrl := new(controllers.ScriptWorkflowsController)
			workflowGroup.POST("", ctrl.Create)
			workflowGroup.GET("", ctrl.List)
			workflowGroup.GET("/:id", ctrl.Show)
			workflowGroup.POST("/:id/run", ctrl.RunStage)
			workflowGroup.POST("/:id/episodes/generate", ctrl.GenerateEpisodes)
			workflowGroup.POST("/:id/confirm", ctrl.ConfirmStep)
			workflowGroup.POST("/:id/episodes/:episodeNo/confirm", ctrl.ConfirmEpisode)
			workflowGroup.PUT("/:id/episodes/:episodeNo", ctrl.UpdateEpisode)
			workflowGroup.GET("/:id/episodes/:episodeNo/versions", ctrl.Versions)
		}

		scriptsGroup := v1.Group("/scripts").Use(middlewares.AuthAdminJWT())
		{
			scriptsController := new(controllers.ScriptsController)

			// 基础CRUD路由
			scriptsGroup.GET("", scriptsController.Index)              // 获取剧本列表
			scriptsGroup.GET("/:id", scriptsController.Show)           // 获取剧本详情
			scriptsGroup.POST("", scriptsController.Store)             // 创建剧本
			scriptsGroup.POST("/generate", scriptsController.Generate) // AI生成剧本 (异步)

			scriptsGroup.PUT("/:id", scriptsController.Update)    // 更新剧本
			scriptsGroup.DELETE("/:id", scriptsController.Delete) // 删除剧本
			// 短剧项目选择列表路由
			scriptsGroup.GET("/getProjectsSelectList", scriptsController.GetProjectsSelectList) // 获取短剧项目选择列表
		}

		// 分镜图片相关路由
		shotFrameImagesGroup := v1.Group("/shot_frame_images").Use(middlewares.AuthAdminJWT())
		{
			shotFrameImagesController := new(controllers.ShotFrameImagesController)

			// 基础CRUD路由
			shotFrameImagesGroup.POST("", shotFrameImagesController.Store) // 创建分镜图片

			shotFrameImagesGroup.DELETE("/:id", shotFrameImagesController.Delete) // 删除分镜图片
		}

		// 分镜视频合并记录
		shotVideoMergeGroup := v1.Group("/shot_video_merges").Use(middlewares.AuthAdminJWT())
		{
			shotVideoMergeController := new(controllers.ShotVideoMergesController)

			// 基础CRUD路由
			shotVideoMergeGroup.GET("", shotVideoMergeController.Index)         // 获取分镜视频列表
			shotVideoMergeGroup.DELETE("/:id", shotVideoMergeController.Delete) // 删除分镜视频
		}

		// 素材相关路由
		sourceGroup := v1.Group("/source").Use(middlewares.AuthAdminJWT())
		{
			sourceController := new(controllers.SourceController)

			// 基础CRUD路由
			sourceGroup.GET("", sourceController.Index)  // 获取素材列表
			sourceGroup.POST("", sourceController.Store) // 创建素材

			sourceGroup.DELETE("/:id", sourceController.Delete) // 删除素材
		}

		// 分镜生成视频相关路由
		shotGenerateVideosGroup := v1.Group("/shot_generate_videos").Use(middlewares.AuthAdminJWT())
		{
			shotGenerateVideosCtrl := new(controllers.ShotGenerateVideosController)

			shotGenerateVideosGroup.POST("", shotGenerateVideosCtrl.Store)
			shotGenerateVideosGroup.DELETE("/:id", shotGenerateVideosCtrl.Delete)
		}

		// 场景相关路由
		scenesGroup := v1.Group("/scenes").Use(middlewares.AuthAdminJWT())
		{
			scenesController := new(controllers.ScenesController)

			// 基础CRUD路由
			scenesGroup.GET("", scenesController.Index)    // 获取场景列表
			scenesGroup.GET("/:id", scenesController.Show) // 获取场景详情
			scenesGroup.POST("", scenesController.Store)   // 创建场景

			scenesGroup.PUT("/:id", scenesController.Update)    // 更新场景
			scenesGroup.DELETE("/:id", scenesController.Delete) // 删除场景
		}

		aiConfigGroup := v1.Group("/ai-config").Use(middlewares.AuthAdminJWT())
		{
			aiConfigController := new(controllers.AiConfigController)

			// 基础CRUD路由
			aiConfigGroup.GET("", aiConfigController.Index)         // 获取AI配置列表
			aiConfigGroup.GET("/:id", aiConfigController.Show)      // 获取AI配置详情
			aiConfigGroup.POST("", aiConfigController.Store)        // 创建AI配置
			aiConfigGroup.PUT("/:id", aiConfigController.Update)    // 更新AI配置
			aiConfigGroup.DELETE("/:id", aiConfigController.Delete) // 删除AI配置
		}

		// 道具相关路由
		propsGroup := v1.Group("/props").Use(middlewares.AuthAdminJWT())
		{
			propsController := new(controllers.PropsController)

			// 基础CRUD路由
			propsGroup.GET("", propsController.Index)    // 获取道具列表
			propsGroup.GET("/:id", propsController.Show) // 获取道具详情
			propsGroup.POST("", propsController.Store)   // 创建道具

			propsGroup.PUT("/:id", propsController.Update)    // 更新道具
			propsGroup.DELETE("/:id", propsController.Delete) // 删除道具
		}

		// 镜头表相关路由
		shotsGroup := v1.Group("/shots").Use(middlewares.AuthAdminJWT())
		{
			shotsController := new(controllers.ShotsController)

			// 基础CRUD路由
			shotsGroup.GET("", shotsController.Index)    // 获取镜头表列表
			shotsGroup.GET("/:id", shotsController.Show) // 获取镜头表详情
			shotsGroup.POST("", shotsController.Store)   // 创建镜头表

			shotsGroup.PUT("/:id", shotsController.Update)    // 更新镜头表
			shotsGroup.DELETE("/:id", shotsController.Delete) // 删除镜头表
			// 短剧项目选择列表路由
			shotsGroup.GET("/getProjectsSelectList", shotsController.GetProjectsSelectList) // 获取短剧项目选择列表
			// 剧本选择列表路由
			shotsGroup.GET("/getScriptsSelectList", shotsController.GetScriptsSelectList) // 获取剧本选择列表
		}

		// 系统管理员相关路由
		adminsGroup := v1.Group("/admins").Use(middlewares.AuthAdminJWT())
		{
			adminsController := new(controllers.AdminsController)

			// 基础CRUD路由
			adminsGroup.GET("", adminsController.Index)    // 获取系统管理员列表
			adminsGroup.GET("/:id", adminsController.Show) // 获取系统管理员详情
			adminsGroup.POST("", adminsController.Store)   // 创建系统管理员

			adminsGroup.PUT("/:id", adminsController.Update)    // 更新系统管理员
			adminsGroup.DELETE("/:id", adminsController.Delete) // 删除系统管理员
		}

		// 系统菜单相关路由
		sysBaseMenuController := new(controllers.SysBaseMenusesController)
		sysBaseMenuGroup := v1.Group("/sys_base_menuses").Use(middlewares.AuthAdminJWT())
		{
			sysBaseMenuGroup.GET("/getMenuList", sysBaseMenuController.GetMenuList) // 获取系统菜单列表
			// 基础CRUD路由
			sysBaseMenuGroup.GET("", sysBaseMenuController.Index)                                           // 获取系统菜单列表
			sysBaseMenuGroup.POST("", sysBaseMenuController.Store)                                          // 创建系统菜单
			sysBaseMenuGroup.GET("/:id", sysBaseMenuController.Show)                                        // 获取系统菜单详情
			sysBaseMenuGroup.PUT("/:id", sysBaseMenuController.Update)                                      // 更新系统菜单
			sysBaseMenuGroup.DELETE("/:id", sysBaseMenuController.Delete)                                   // 删除系统菜单
			sysBaseMenuGroup.GET("/getSysBaseMenusTreeList", sysBaseMenuController.GetSysBaseMenusTreeList) // 获取树形菜单下拉列表
		}

	}
}
