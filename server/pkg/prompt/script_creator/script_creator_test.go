package script_creator

import (
	"strings"
	"testing"
)

func TestCleanJSON(t *testing.T) {
	got := CleanJSON("说明\n```json\n{\"version\":\"v1.0\"}\n```\n结束")
	if got != "{\"version\":\"v1.0\"}" {
		t.Fatalf("unexpected clean result: %s", got)
	}
}

func TestRenderAndValidateEpisode(t *testing.T) {
	ep := Episode{EpisodeNo: 1, Title: "死亡列车", EstimatedDuration: 90, CoreEvent: "列车停运", Cliffhanger: "尸体出现", EndingSubtitle: "未完待续...", Scenes: []Scene{{SceneNo: 1, InteriorExterior: "内景", Location: "车厢", Time: "深夜", Duration: 30, Environment: "冷白灯光照射金属扶手", Shots: []Shot{{ShotNo: 1, ShotSize: "全景", Angle: "俯视", Description: "车厢全貌", Action: "乘客散坐"}}}, {SceneNo: 2, InteriorExterior: "内景", Location: "车厢", Time: "深夜", Duration: 30, Environment: "红色应急灯闪烁", Shots: []Shot{{ShotNo: 2, ShotSize: "中景", Angle: "平视", Description: "人物反应"}}}, {SceneNo: 3, InteriorExterior: "内景", Location: "连接处", Time: "深夜", Duration: 30, Environment: "刺眼白光照亮金属车壁", Shots: []Shot{{ShotNo: 3, ShotSize: "特写", Angle: "正面", Description: "尸体出现"}}}}}
	if warnings := ValidateEpisode(ep); len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	out := RenderEpisode(ep)
	for _, want := range []string{"第1集 - 死亡列车", "场景1: 内景 - 车厢 - 深夜 (30秒)", "[镜头: 全景/俯视 - 车厢全貌]"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q", want)
		}
	}
}
