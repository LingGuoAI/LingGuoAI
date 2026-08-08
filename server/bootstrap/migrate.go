package bootstrap

import (
	"spiritFruit/app/models"
	"spiritFruit/app/models/admins"
	"spiritFruit/app/models/ai_config"
	"spiritFruit/app/models/characters"
	"spiritFruit/app/models/projects"
	"spiritFruit/app/models/props"
	"spiritFruit/app/models/scenes"
	"spiritFruit/app/models/script_workflows"
	"spiritFruit/app/models/scripts"
	"spiritFruit/app/models/shot_characters"
	"spiritFruit/app/models/shot_frame_image"
	"spiritFruit/app/models/shot_frame_prompts"
	"spiritFruit/app/models/shot_generate_video"
	"spiritFruit/app/models/shot_props"
	"spiritFruit/app/models/source"
	"spiritFruit/app/models/sys_base_menus"
	"spiritFruit/pkg/console"
	"spiritFruit/pkg/database"

	"spiritFruit/app/models/async_tasks"
	"spiritFruit/app/models/shot_video_merge"
	"spiritFruit/app/models/shots"
)

// SetupAutoMigrate 自动同步数据库表结构
func SetupAutoMigrate() {
	console.Success("开始自动同步数据库表结构 (AutoMigrate)...")

	err := database.DB.SetupJoinTable(&shots.Shots{}, "Characters", &shot_characters.ShotCharacters{})
	if err != nil {
		console.Error("注册 Characters 连接表失败: " + err.Error())
	}

	err = database.DB.SetupJoinTable(&shots.Shots{}, "Props", &shot_props.ShotProps{})
	if err != nil {
		console.Error("注册 Props 连接表失败: " + err.Error())
	}

	// 将所有模型结构体实例传入 AutoMigrate
	err = database.DB.AutoMigrate(
		&admins.Admins{},
		&sys_base_menus.SysBaseMenus{},
		&ai_config.AiConfig{},
		&async_tasks.AsyncTask{},
		&async_tasks.AsyncTaskEvent{},
		&characters.Characters{},
		&props.Props{},
		&scenes.Scenes{},
		&shot_characters.ShotCharacters{},
		&shot_frame_image.ShotFrameImages{},
		&shot_frame_prompts.ShotFramePrompts{},
		&shot_generate_video.ShotGenerateVideo{},
		&shot_props.ShotProps{},
		&source.Source{},
		&shot_video_merge.ShotVideoMerge{},
		&shots.Shots{},
		&scripts.Scripts{},
		&script_workflows.ScriptWorkflow{},
		&script_workflows.ScriptWorkflowStep{},
		&script_workflows.ScriptEpisodeDraft{},
		&script_workflows.ScriptEpisodeVersion{},
		&projects.Projects{},
	)

	if err != nil {
		console.Exit("数据库自动迁移失败: " + err.Error())
	}

	console.Success("数据库表结构自动同步完成！")

	// 插入初始默认数据
	seedDefaultData()
}

func ptrStr(s string) *string  { return &s }
func ptrUint(u uint64) *uint64 { return &u }
func ptrInt8(i int8) *int8     { return &i }

// seedDefaultData 播种默认数据
func seedDefaultData() {
	// ==========================================
	// 1. 初始化超级管理员
	// ==========================================
	var adminCount int64
	database.DB.Model(&admins.Admins{}).Count(&adminCount)
	if adminCount == 0 {
		console.Success("检测到管理员表为空，正在插入默认管理员...")

		// 取消注释并替换为你 Admins 模型中的真实字段
		defaultAdmin := admins.Admins{
			Username:    ptrStr("admin"),
			Mobile:      ptrStr("1888888888"),
			Password:    ptrStr("123456"),
			Email:       ptrStr("admin@gmail.com"),
			AuthorityId: ptrUint(666),
		}
		if err := database.DB.Create(&defaultAdmin).Error; err != nil {
			console.Error("默认管理员插入失败: " + err.Error())
		} else {
			console.Success("默认超级管理员创建成功！")
		}
	}

	// ==========================================
	// 2. 初始化系统基础菜单
	// ==========================================
	var menuCount int64
	database.DB.Model(&sys_base_menus.SysBaseMenus{}).Count(&menuCount)
	if menuCount == 0 {
		console.Success("检测到菜单表为空，正在插入默认系统菜单...")

		// 完全按照你提供的截图数据映射
		defaultMenus := []sys_base_menus.SysBaseMenus{
			{BaseModel: models.BaseModel{ID: 1}, ParentId: nil, Path: ptrStr("/dashboard"), Name: ptrStr("Dashboard"), Hidden: ptrInt8(1), Component: ptrStr("Layout"), Sort: ptrUint(1), Title: ptrStr("仪表盘"), Icon: ptrStr("dashboard")},
			{BaseModel: models.BaseModel{ID: 2}, ParentId: ptrUint(1), Path: ptrStr("/dashboard/base"), Name: ptrStr("DashboardStatistics"), Hidden: ptrInt8(0), Component: ptrStr("/dashboard/base/index.vue"), Sort: ptrUint(2), Title: ptrStr("统计报表"), Icon: nil},

			{BaseModel: models.BaseModel{ID: 3}, ParentId: nil, Path: ptrStr("/admin/characters"), Name: ptrStr("AdminCharactersModule"), Hidden: ptrInt8(1), Component: ptrStr("Layout"), Sort: ptrUint(10), Title: ptrStr("角色管理"), Icon: ptrStr("app")},
			{BaseModel: models.BaseModel{ID: 4}, ParentId: ptrUint(3), Path: ptrStr("/admin/characters/list"), Name: ptrStr("AdminCharactersList"), Hidden: ptrInt8(0), Component: ptrStr("/characters/index.vue"), Sort: ptrUint(1), Title: ptrStr("角色列表"), Icon: nil},

			{BaseModel: models.BaseModel{ID: 5}, ParentId: nil, Path: ptrStr("/admin/projects"), Name: ptrStr("AdminProjectsModule"), Hidden: ptrInt8(0), Component: ptrStr("Layout"), Sort: ptrUint(11), Title: ptrStr("短剧项目"), Icon: ptrStr("folder-add")},
			{BaseModel: models.BaseModel{ID: 6}, ParentId: ptrUint(5), Path: ptrStr("/admin/projects/list"), Name: ptrStr("AdminProjectsList"), Hidden: ptrInt8(1), Component: ptrStr("/projects/index.vue"), Sort: ptrUint(0), Title: ptrStr("短剧项目"), Icon: nil},

			{BaseModel: models.BaseModel{ID: 7}, ParentId: nil, Path: ptrStr("/admin/scripts"), Name: ptrStr("AdminScriptsModule"), Hidden: ptrInt8(1), Component: ptrStr("Layout"), Sort: ptrUint(12), Title: ptrStr("剧本管理"), Icon: ptrStr("app")},
			{BaseModel: models.BaseModel{ID: 8}, ParentId: ptrUint(7), Path: ptrStr("/admin/scripts/list"), Name: ptrStr("AdminScriptsList"), Hidden: ptrInt8(0), Component: ptrStr("/scripts/index.vue"), Sort: ptrUint(1), Title: ptrStr("剧本列表"), Icon: nil},

			{BaseModel: models.BaseModel{ID: 9}, ParentId: nil, Path: ptrStr("/admin/shots"), Name: ptrStr("AdminShotsModule"), Hidden: ptrInt8(1), Component: ptrStr("Layout"), Sort: ptrUint(13), Title: ptrStr("镜头管理"), Icon: ptrStr("app")},
			{BaseModel: models.BaseModel{ID: 10}, ParentId: ptrUint(9), Path: ptrStr("/admin/shots/list"), Name: ptrStr("AdminShotsList"), Hidden: ptrInt8(0), Component: ptrStr("/shots/index.vue"), Sort: ptrUint(1), Title: ptrStr("镜头列表"), Icon: nil},

			{BaseModel: models.BaseModel{ID: 11}, ParentId: nil, Path: ptrStr("/admin/admins"), Name: ptrStr("AdminAdminsModule"), Hidden: ptrInt8(1), Component: ptrStr("Layout"), Sort: ptrUint(14), Title: ptrStr("系统管理员"), Icon: ptrStr("user-setting")},
			{BaseModel: models.BaseModel{ID: 12}, ParentId: ptrUint(11), Path: ptrStr("/admin/admins/list"), Name: ptrStr("AdminAdminsList"), Hidden: ptrInt8(0), Component: ptrStr("/admins/index.vue"), Sort: ptrUint(1), Title: ptrStr("系统管理员"), Icon: nil},

			{BaseModel: models.BaseModel{ID: 13}, ParentId: nil, Path: ptrStr("/admin/system"), Name: ptrStr("AdminSystemModule"), Hidden: ptrInt8(1), Component: ptrStr("Layout"), Sort: ptrUint(900), Title: ptrStr("系统管理"), Icon: ptrStr("setting")},
			{BaseModel: models.BaseModel{ID: 14}, ParentId: ptrUint(13), Path: ptrStr("/admin/system/menus"), Name: ptrStr("AdminSysBaseMenuList"), Hidden: ptrInt8(0), Component: ptrStr("/sys_base_menus/index.vue"), Sort: ptrUint(1), Title: ptrStr("菜单管理"), Icon: nil},

			{BaseModel: models.BaseModel{ID: 15}, ParentId: nil, Path: ptrStr("/user"), Name: ptrStr("UserCenter"), Hidden: ptrInt8(1), Component: ptrStr("Layout"), Sort: ptrUint(999), Title: ptrStr("个人中心"), Icon: ptrStr("user-circle")},
			{BaseModel: models.BaseModel{ID: 16}, ParentId: ptrUint(15), Path: ptrStr("/user/index"), Name: ptrStr("UserProfile"), Hidden: ptrInt8(1), Component: ptrStr("/user/index"), Sort: ptrUint(1), Title: ptrStr("个人信息"), Icon: nil},

			{BaseModel: models.BaseModel{ID: 17}, ParentId: ptrUint(5), Path: ptrStr("detail/:id"), Name: ptrStr("ProjectDetail"), Hidden: ptrInt8(1), Component: ptrStr("/projects/detail.vue"), Sort: ptrUint(20), Title: ptrStr("项目工作台"), Icon: nil},
			{BaseModel: models.BaseModel{ID: 18}, ParentId: ptrUint(5), Path: ptrStr("chapter/:id/:episodeNumber"), Name: ptrStr("ProjectChapterCreate"), Hidden: ptrInt8(1), Component: ptrStr("/projects/createChapter.vue"), Sort: ptrUint(0), Title: ptrStr("章节创作"), Icon: nil},
			{BaseModel: models.BaseModel{ID: 19}, ParentId: ptrUint(5), Path: ptrStr("editor/:dramaId/:episodeNumber"), Name: ptrStr("ScriptEditor"), Hidden: ptrInt8(1), Component: ptrStr("/projects/scriptEditor.vue"), Sort: ptrUint(0), Title: ptrStr("视频创作"), Icon: nil},
		}

		if err := database.DB.Create(&defaultMenus).Error; err != nil {
			console.Error("默认系统菜单插入失败: " + err.Error())
		} else {
			console.Success("默认系统菜单创建成功！")
		}
	}
	// 隐藏的剧本工作流路由需要在老数据库中补种。
	workflowMenu := sys_base_menus.SysBaseMenus{BaseModel: models.BaseModel{ID: 20}, ParentId: ptrUint(5), Path: ptrStr(":projectId/script-workflow/:workflowId?"), Name: ptrStr("ProjectScriptWorkflow"), Hidden: ptrInt8(1), Component: ptrStr("/projects/scriptWorkflow.vue"), Sort: ptrUint(0), Title: ptrStr("AI剧本创作"), Icon: nil}
	database.DB.Where("id = ?", 20).FirstOrCreate(&workflowMenu)
}
