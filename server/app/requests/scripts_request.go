package requests

import (
	"github.com/gin-gonic/gin"
	"github.com/thedevsaddam/govalidator"
)

// ScriptsRequest 剧本请求结构体
type ScriptsRequest struct {
	ProjectId uint64 `valid:"projectId" json:"projectId" ` // 所属项目ID
	Title     string `valid:"title" json:"title"`          // 分集标题
	Content   string `valid:"content" json:"content"`      // 剧本正文
	Outline   string `valid:"outline" json:"outline"`      // 大纲/简介
	EpisodeNo uint64 `valid:"episodeNo" json:"episodeNo"`  // 第几集
	IsLocked  int8   `valid:"isLocked" json:"isLocked"`    // 是否定稿
}

// ScriptsSave 剧本保存时的验证规则
func ScriptsSave(data interface{}, c *gin.Context) map[string][]string {
	rules := govalidator.MapData{
		"projectId": []string{"required", "numeric"},
		"title":     []string{"max:255"},
		"content":   []string{"max:200000"},
		"outline":   []string{"max:255"},
		"episodeNo": []string{"numeric"},
		"isLocked":  []string{"numeric"},
	}

	messages := govalidator.MapData{
		"projectId": []string{
			"required:所属项目ID为必填项",
			"numeric:所属项目ID必须为数字",
		},
		"title": []string{
			"max:分集标题长度不能超过 255 个字符",
		},
		"content": []string{
			"max:剧本正文长度不能超过 200000 个字符",
		},
		"outline": []string{
			"max:大纲/简介长度不能超过 255 个字符",
		},
		"episodeNo": []string{
			"numeric:第几集必须为数字",
		},
		"isLocked": []string{
			"numeric:是否定稿必须为数字",
		},
	}

	return validate(data, rules, messages)
}
