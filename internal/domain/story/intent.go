package story

import (
	"encoding/json"
	"regexp"
	"strings"
)

const StoryIntentMinConfidence = 0.55

// IntentResult LLM 故事意图识别结果。
type IntentResult struct {
	IsStoryRequest bool    `json:"is_story_request"`
	Confidence     float64 `json:"confidence"`
	Action         string  `json:"action"`
	StoryRef       string  `json:"story_ref"`
	Theme          string  `json:"theme"`
	StoryType      string  `json:"story_type"`
	NarrationMode  string  `json:"narration_mode"`
	IsBedtime      *bool   `json:"is_bedtime"`
	UserSaidCasual bool    `json:"user_said_casual"`
}

var storyIntentJSONPattern = regexp.MustCompile(`\{[\s\S]*\}`)

// BuildStoryIntentSystemPrompt 故事意图分类器 system prompt。
func BuildStoryIntentSystemPrompt() string {
	return `你是儿童语音助手的「故事意图分类器」。根据用户一句话，判断是否在与「听故事/讲故事」相关，并提取结构化参数。

只输出一个 JSON 对象，不要 markdown、不要解释。字段说明：
- is_story_request: bool，用户是否在请求讲故事、听故事、编故事、复播/续讲故事等
- confidence: 0~1，判断置信度
- action: generate|replay|resume|list_recent|none（非故事请求填 none）
- story_ref: replay/resume 时用 last|last_night|favorite|空
- theme: 规范化故事名或主题（如「龟兔赛跑」「女娲补天」「恐龙冒险」）；无则空
- story_type: classic|myth|fable|fairy_tale|bedtime|original（经典童话、神话、寓言、民间童话、睡前、用户要新编）
- narration_mode: canonical|creative
  - canonical：用户点名经典/神话/寓言/民间故事，期望讲广为流传的正篇，不要新编
  - creative：用户要编故事、随便讲、或主题需原创（如「讲个小恐龙」）
- is_bedtime: bool，是否睡前/哄睡场景
- user_said_casual: bool，用户说随便/都可以/你定

规则：
1. 「讲龟兔赛跑」「女娲补天」即使无「故事」二字也是故事请求；theme 填标准故事名，story_type 与 narration_mode 要准确
2. 「编个故事」「讲个关于恐龙的新故事」→ original + creative
3. 「再讲一遍」「昨晚的故事」→ replay；用户点名具体故事名（如「再讲一遍女娲补天」）→ replay 且 theme 填该故事名；「接着讲」→ resume
4. 问天气、闲聊、事实问答 → is_story_request=false，action=none
5. theme 用简短规范名称，不要整句用户原话
6. 用户追问「刚才/刚刚讲了什么故事、什么内容、叫什么名字」→ is_story_request=false，action=none（由主对话根据上下文回答，不是复播/续讲/列表）`
}

// ParseStoryIntentJSON 解析 LLM 输出的故事意图 JSON。
func ParseStoryIntentJSON(raw string) (IntentResult, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return IntentResult{}, errEmptyStoryIntent
	}
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var result IntentResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		match := storyIntentJSONPattern.FindString(raw)
		if match == "" {
			return IntentResult{}, err
		}
		if err2 := json.Unmarshal([]byte(match), &result); err2 != nil {
			return IntentResult{}, err
		}
	}
	result.Action = strings.TrimSpace(result.Action)
	result.StoryRef = strings.TrimSpace(result.StoryRef)
	result.Theme = strings.TrimSpace(result.Theme)
	result.StoryType = normalizeStoryType(result.StoryType)
	result.NarrationMode = normalizeNarrationMode(result.NarrationMode)
	if result.Action == "" && result.IsStoryRequest {
		result.Action = ActionGenerate
	}
	return result, nil
}

var errEmptyStoryIntent = &parseError{"empty story intent"}

type parseError struct{ msg string }

func (e *parseError) Error() string { return e.msg }

// IntentToStoryParams 将意图结果转为故事生成参数。
func IntentToStoryParams(intent IntentResult) StoryParams {
	p := StoryParams{
		RequestType:    intent.StoryType,
		Theme:          intent.Theme,
		NarrationMode:  intent.NarrationMode,
		IsBedtime:      intent.IsBedtime,
		UserSaidCasual: intent.UserSaidCasual,
	}
	if intent.IsBedtime != nil && *intent.IsBedtime && p.RequestType == "" {
		p.RequestType = StoryModeBedtime
	}
	NormalizeStoryParams(&p)
	return p
}
