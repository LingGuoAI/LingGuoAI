package script_creator

import (
	"encoding/json"
	"fmt"
	"strings"
)

const Version = "1.1_optimized"

type Dialogue struct {
	Character string `json:"character"`
	Direction string `json:"direction"`
	Line      string `json:"line"`
	Type      string `json:"type"`
}
type Shot struct {
	ShotNo      int        `json:"shotNo"`
	ShotSize    string     `json:"shotSize"`
	Angle       string     `json:"angle"`
	Description string     `json:"description"`
	Action      string     `json:"action"`
	Dialogues   []Dialogue `json:"dialogues"`
	SoundEffect string     `json:"soundEffect"`
}
type Scene struct {
	SceneNo          int    `json:"sceneNo"`
	InteriorExterior string `json:"interiorExterior"`
	Location         string `json:"location"`
	Time             string `json:"time"`
	Duration         int    `json:"duration"`
	Environment      string `json:"environment"`
	Shots            []Shot `json:"shots"`
}
type Episode struct {
	EpisodeNo         int     `json:"episodeNo"`
	Title             string  `json:"title"`
	EstimatedDuration int     `json:"estimatedDuration"`
	CoreEvent         string  `json:"coreEvent"`
	Cliffhanger       string  `json:"cliffhanger"`
	Scenes            []Scene `json:"scenes"`
	EndingSubtitle    string  `json:"endingSubtitle"`
}

func BaseSystem() string {
	return `你是“剧本创作师 v1.1 优化版”，专门创作5-30集、单集1-2分钟、适合AI视频制作的中文短剧。严格遵守当前阶段，不跳步。输出必须是纯JSON，不要Markdown代码块或解释。场景环境必须视觉化、具体、可生成；对白简短有力；动作清晰可执行。`
}

func CreativePrompt(input string) string {
	return BaseSystem() + `\n当前步骤1：创意定型。根据用户资料整理或补全创意。返回JSON字段：version固定为v1.0、startingPoint、genre、totalEpisodes、coreStory、targetAudience、episodeDuration、visualStyle、currentStep固定为1。\n用户资料：` + input
}

func DesignPrompt(creative string) string {
	return BaseSystem() + `\n当前步骤2：全剧设计。仅接受v1.0创意。一次性完成三幕结构、逐集大纲、节奏、伏笔、角色、世界观和冲突。返回JSON：version=v2.0, threeActStructure数组, episodes数组（episodeNo,title,act,coreEvent,hookType,hookDesign,estimatedDuration）, rhythmCurve数组, foreshadowing数组（type,setupEpisode,payoffEpisode,content）, characters数组（name,role,age,gender,occupation,appearance,personality,goal,weakness,arc,signatureAction）, relationships数组, worldview对象, conflicts对象。逐集数量必须等于totalEpisodes。\nv1.0：` + creative
}

func EpisodePrompt(creative, design string, episodeNo int, previousEnding, changeRequest string) string {
	extra := ""
	if previousEnding != "" {
		extra += "\n上一集结尾：" + previousEnding
	}
	if changeRequest != "" {
		extra += "\n重写要求：" + changeRequest
	}
	return fmt.Sprintf(`%s
当前步骤3：生成第%d集标准短视频AI剧本。只能使用全剧设计中第%d集的大纲。
硬性要求：3-5个场景，总时长90-120秒，开头15秒有钩子，结尾必须有悬念或反转。环境包含可见的色彩、材质、光线；同一地点描述保持一致。
只返回JSON：episodeNo,title,estimatedDuration,coreEvent,cliffhanger,scenes,endingSubtitle。scenes元素为sceneNo,interiorExterior(内景/外景),location,time,duration,environment,shots。shots元素为shotNo,shotSize(远景/全景/中景/近景/特写),angle(平视/俯视/仰视/侧面/正面),description,action,dialogues,soundEffect。dialogues元素为character,direction,line,type(dialogue/voiceover/inner)。
创意：%s
全剧设计：%s%s`, BaseSystem(), episodeNo, episodeNo, creative, design, extra)
}

func DeliveryPrompt(design string, episodes string) string {
	return BaseSystem() + `\n当前步骤4：AI交付。所有剧本已完成。返回JSON：version=v4.0, scenes数组（sceneId,name,interiorExterior,time,location,visualDescription,episodeNos）, characters数组（characterId,name,gender,age,appearanceKeywords,costume,episodeNos）, shotStatistics对象（total,shotSizeDistribution,angleDistribution,recommendedOrder）, visualStyle对象, documentTitle。场景和角色必须去重。\n全剧设计：` + design + `\n所有剧本：` + episodes
}

func CleanJSON(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	startObj, startArr := strings.Index(s, "{"), strings.Index(s, "[")
	start := startObj
	if start < 0 || (startArr >= 0 && startArr < start) {
		start = startArr
	}
	endObj, endArr := strings.LastIndex(s, "}"), strings.LastIndex(s, "]")
	end := endObj
	if endArr > end {
		end = endArr
	}
	if start >= 0 && end >= start {
		return strings.TrimSpace(s[start : end+1])
	}
	return strings.TrimSpace(s)
}

func ValidateJSON(raw string) (string, error) {
	clean := CleanJSON(raw)
	var v interface{}
	if err := json.Unmarshal([]byte(clean), &v); err != nil {
		return "", err
	}
	return clean, nil
}

func ParseEpisode(raw string) (Episode, error) {
	var ep Episode
	clean, err := ValidateJSON(raw)
	if err != nil {
		return ep, err
	}
	err = json.Unmarshal([]byte(clean), &ep)
	return ep, err
}

func RenderEpisode(ep Episode) string {
	var b strings.Builder
	fmt.Fprintf(&b, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n第%d集 - %s\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n【时长: 约%d秒】\n【核心事件: %s】\n【卡点: %s】\n\n", ep.EpisodeNo, ep.Title, ep.EstimatedDuration, ep.CoreEvent, ep.Cliffhanger)
	for _, sc := range ep.Scenes {
		fmt.Fprintf(&b, "场景%d: %s - %s - %s (%d秒)\n%s\n", sc.SceneNo, sc.InteriorExterior, sc.Location, sc.Time, sc.Duration, sc.Environment)
		for _, sh := range sc.Shots {
			fmt.Fprintf(&b, "[镜头: %s/%s - %s]\n", sh.ShotSize, sh.Angle, sh.Description)
			if sh.Action != "" {
				fmt.Fprintf(&b, "(%s)\n", sh.Action)
			}
			if sh.SoundEffect != "" {
				fmt.Fprintf(&b, "[音效: %s]\n", sh.SoundEffect)
			}
			for _, d := range sh.Dialogues {
				name := d.Character
				if d.Type == "inner" {
					name += "心声"
				}
				if d.Type == "voiceover" {
					name += "(画外音)"
				}
				fmt.Fprintf(&b, "%s: (%s) %s\n", name, d.Direction, d.Line)
			}
		}
		b.WriteString("\n")
	}
	b.WriteString("[镜头: 黑屏]\n【字幕】")
	if ep.EndingSubtitle != "" {
		b.WriteString(ep.EndingSubtitle)
	} else {
		b.WriteString("未完待续...")
	}
	b.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	return b.String()
}

func ValidateEpisode(ep Episode) []string {
	w := []string{}
	if len(ep.Scenes) < 3 || len(ep.Scenes) > 5 {
		w = append(w, "场景数量应为3-5个")
	}
	total := 0
	for _, s := range ep.Scenes {
		total += s.Duration
		if len(s.Shots) == 0 {
			w = append(w, fmt.Sprintf("场景%d没有镜头", s.SceneNo))
		}
	}
	if total < 90 || total > 120 {
		w = append(w, fmt.Sprintf("场景总时长%d秒，不在90-120秒", total))
	}
	if ep.Cliffhanger == "" {
		w = append(w, "缺少结尾卡点")
	}
	return w
}
