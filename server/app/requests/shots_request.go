package requests

import (
	"github.com/gin-gonic/gin"
	"github.com/thedevsaddam/govalidator"
)

// ShotsRequest 镜头表请求结构体
type ShotsRequest struct {
	ProjectId      uint64 `valid:"projectId" json:"projectId" `   // 所属项目ID
	ScriptId       uint64 `valid:"scriptId" json:"scriptId" `     // 所属剧本/分集ID
	SceneId        uint64 `valid:"sceneId" json:"sceneId" `       // 场景ID
	SequenceNo     uint64 `valid:"sequenceNo" json:"sequenceNo" ` // 镜头序号
	Title          string `valid:"title" json:"title"`
	ShotType       string `valid:"shotType" json:"shotType"`             // 景别: 全景/特写/中景
	CameraMovement string `valid:"cameraMovement" json:"cameraMovement"` // 运镜: 推/拉/摇/移
	Angle          string `valid:"angle" json:"angle"`                   // 视角: 俯视/平视
	Dialogue       string `valid:"dialogue" json:"dialogue"`             // 台词/旁白
	Time           string `valid:"time" json:"time"`
	Location       string `valid:"location" json:"location"`
	Action         string `valid:"action" json:"action"`
	VisualDesc     string `valid:"visualDesc" json:"visualDesc"`   // 画面描述
	Result         string `valid:"result" json:"result"`           // 动作结果
	Atmosphere     string `valid:"atmosphere" json:"atmosphere"`   // 氛围/环境描述
	ImagePrompt    string `valid:"imagePrompt" json:"imagePrompt"` // 绘画Prompt
	VideoPrompt    string `valid:"videoPrompt" json:"videoPrompt"` // 视频生成Prompt
	AudioPrompt    string `valid:"audioPrompt" json:"audioPrompt"` // 音效/BGM提示词
	ImageUrl       string `valid:"imageUrl" json:"imageUrl"`       // 分镜图
	VideoUrl       string `valid:"videoUrl" json:"videoUrl"`       // 最终视频片段
	AudioUrl       string `valid:"audioUrl" json:"audioUrl"`       // 配音/音效
	DurationMs     uint64 `valid:"durationMs" json:"durationMs"`   // 时长(毫秒, 原duration*1000)
	Status         int8   `valid:"status" json:"status"`           // 状态
	// 关联关系字段
	CharacterIds []uint64 `valid:"characterIds" json:"characterIds"` // 角色ID
	PropIds      []uint64 `valid:"propIds" json:"propIds"`           // 道具ID
}

// ShotsSave 镜头表保存时的验证规则
func ShotsSave(data interface{}, c *gin.Context) map[string][]string {
	rules := govalidator.MapData{
		"projectId":      []string{"required", "numeric"},
		"scriptId":       []string{"required", "numeric"},
		"sequenceNo":     []string{"required", "numeric"},
		"title":          []string{"max:255"},
		"time":           []string{"max:255"},
		"location":       []string{"max:255"},
		"shotType":       []string{"max:50"},
		"cameraMovement": []string{"max:50"},
		"angle":          []string{"max:50"},
		"imageUrl":       []string{"max:1024"},
		"videoUrl":       []string{"max:1024"},
		"audioUrl":       []string{"max:1024"},
		"durationMs":     []string{"numeric"},
		"status":         []string{"numeric"},
	}

	messages := govalidator.MapData{
		"projectId": []string{
			"required:所属项目ID为必填项",
			"numeric:所属项目ID必须为数字",
		},
		"scriptId": []string{
			"required:所属剧本/分集ID为必填项",
			"numeric:所属剧本/分集ID必须为数字",
		},
		"sequenceNo": []string{
			"required:镜头序号为必填项",
			"numeric:镜头序号必须为数字",
		},
		"shotType": []string{
			"max:景别: 全景/特写/中景长度不能超过 50 个字符",
		},
		"cameraMovement": []string{
			"max:运镜: 推/拉/摇/移长度不能超过 50 个字符",
		},
		"angle": []string{
			"max:视角: 俯视/平视长度不能超过 50 个字符",
		},
		"dialogue": []string{
			"max:台词/旁白长度不能超过 255 个字符",
		},
		"imageUrl": []string{
			"max:分镜图长度不能超过 1024 个字符",
		},
		"videoUrl": []string{
			"max:最终视频片段长度不能超过 1024 个字符",
		},
		"audioUrl": []string{
			"max:配音/音效长度不能超过 1024 个字符",
		},
		"durationMs": []string{
			"numeric:时长(毫秒, 原duration*1000)必须为数字",
		},
		"status": []string{
			"numeric:状态必须为数字",
		},
	}

	return validate(data, rules, messages)
}
