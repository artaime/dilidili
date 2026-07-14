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
	Theme          string  `json:"theme"`      // 用户口中的主题（可含 ASR 错字）
	Canonical      string  `json:"canonical"`  // 通行规范名（纠错后，用于查库/入池）
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
- theme: 用户口中提到的故事名/主题原意（可保留口语写法；无则空）
- canonical: 经典/神话/寓言点名时的「通行规范故事名」；原创主题可与 theme 相同或空
- story_type: classic|myth|fable|fairy_tale|bedtime|original（经典童话、神话、寓言、民间童话、睡前、用户要新编）
- narration_mode: canonical|creative
  - canonical：用户点名经典/神话/寓言/民间故事，期望讲广为流传的正篇，不要新编
  - creative：用户要编故事、随便讲、或主题需原创（如「讲个小恐龙」）
- is_bedtime: bool，是否睡前/哄睡场景
- user_said_casual: bool，用户说随便/都可以/你定

规则：
1. 「讲龟兔赛跑」「女娲补天」即使无「故事」二字也是故事请求；story_type 与 narration_mode 要准确
2. 「编个故事」「讲个关于恐龙的新故事」→ original + creative
3. 「再讲一遍」「昨晚的故事」→ replay；用户点名具体故事名（如「再讲一遍女娲补天」）→ replay 且填 theme/canonical；「接着讲」→ resume
4. 问天气、闲聊、事实问答 → is_story_request=false，action=none
5. theme/canonical 用简短名称，不要整句用户原话
6. 用户追问「刚才/刚刚讲了什么故事、什么内容、叫什么名字」→ is_story_request=false，action=none（由主对话根据上下文回答，不是复播/续讲/列表）
7. 【重要·规范名纠错】语音识别常有谐音/错字。点名经典时必须把 canonical 纠成社会通行故事名，theme 可保留用户原说法：
   - 「后裔射太阳」「后羿射太阳」→ theme 可写口语，canonical=「后羿射日」
   - 「哪吒三太子闹海」「哪吒脑海」→ canonical=「哪吒闹海」
   - 「女娲补天的故事」「女哇补天」→ canonical=「女娲补天」
   - 「夸父追日」「夸父逐日」→ canonical=「夸父逐日」
   找不到通行名时 canonical 与 theme 同填最接近的短名；原创勿硬扭成经典名。`
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
	result.Canonical = strings.TrimSpace(result.Canonical)
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

// ResolveIntentTheme 返回查库/生成用的规范主题，以及口语原主题（供别名）。
func ResolveIntentTheme(intent IntentResult) (theme, themeRaw string) {
	themeRaw = strings.TrimSpace(intent.Theme)
	theme = strings.TrimSpace(intent.Canonical)
	if theme == "" {
		theme = themeRaw
	}
	if themeRaw == "" {
		themeRaw = theme
	}
	return theme, themeRaw
}

// IntentToStoryParams 将意图结果转为故事生成参数（Theme 优先用规范名）。
func IntentToStoryParams(intent IntentResult) StoryParams {
	theme, themeRaw := ResolveIntentTheme(intent)
	p := StoryParams{
		RequestType:    intent.StoryType,
		Theme:          theme,
		ThemeRaw:       themeRaw,
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
