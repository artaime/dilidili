package story

import (
	"fmt"
	"strings"
)

// BuildSystemPrompt 返回故事生成专用 system prompt（不含 VoiceReplyStylePrompt）。
func BuildSystemPrompt(params StoryParams) string {
	spec := GetAgeBandSpec(params.AgeBand)
	minW, maxW := WordRangeForBand(params.AgeBand)

	p := params
	NormalizeStoryParams(&p)
	canonical := ShouldTellCanonical(p)

	var roleIntro string
	switch {
	case canonical && p.RequestType == StoryModeMyth:
		roleIntro = "你是一位熟悉神话传说的讲述者。请讲述该主题的经典神话叙事。"
	case canonical:
		roleIntro = "你是一位熟悉经典儿童文学、寓言与民间故事的讲述者。请讲述该主题广为流传的经典版本。"
	default:
		roleIntro = "你是一位专业的儿童故事创作师。请严格遵循以下规则创作原创故事："
	}

	prompt := fmt.Sprintf(`%s

## 年龄适配
- 听众年龄档：%s（%s）
- 目标字数：约 %d~%d 字（中文）
- 词汇与句式须符合该年龄认知水平

## 内容安全
禁止：政治、宗教宣传、歧视、色情、暴力细节、恐怖血腥、霸凌美化、犯罪/自残/毒品/赌博/迷信诱导等。

## 价值观
积极健康，通过情节自然传递诚实、勇敢、善良等品质，禁止生硬说教。`, roleIntro, spec.ID, spec.Description, minW, maxW)

	if canonical {
		prompt += `

## 经典正篇讲述要求
- 保留该故事/传说公认的核心人物、主线情节与结局，不要魔改、换名或替换为全新故事
- 允许适龄口语化表达，但不得改变广为流传的情节脉络
- 不要声明「改编版」「新编」；直接像讲经典一样叙述`
	} else {
		prompt += `

## 故事质量（原创）
- 须含：开端、冲突、转折、解决、结尾；逻辑自洽
- 开头吸引听众，中间有节奏，结尾完整收束
- 避免网络热梗、成人流行语、Markdown 与列表格式

## 多样性（原创必须）
- 题材与情节须轮换丰富，禁止总写「小动物在森林里交朋友」一类套路
- 主角姓名每次须新鲜；禁止复用近期人物名，也禁止改回「小明」「小红」「小兔子」等高频默认名
- 若用户提示给出了指定题材、切入点或主角名，必须遵守`
	}

	prompt += `

## 语言
- 适合家长朗读、便于 TTS 播报
- 对话与描写比例符合年龄档`

	if params.IsBedtime != nil && *params.IsBedtime {
		prompt += "\n\n## 睡前模式\n- 节奏舒缓、情绪安稳\n- 避免惊吓与持续紧张\n- 结尾柔和，便于入睡"
	}
	if params.RequestType == "interactive" {
		prompt += "\n\n## 互动故事\n- 可在关键处向孩子提出简单选择题或猜想（1~2 处即可）"
	}

	prompt += `

## 输出格式（必须严格遵守）
- 第一行且仅第一行输出元信息（不会被朗读），格式：
  [[meta:title=故事名称|genre=题材|theme=故事主题|characters=主角,配角|canonical=规范名|aliases=别名1,别名2]]
- title：为本篇故事的名称（2~12 字，如「普罗米修斯盗火」「森林里的新朋友」）
- genre：题材类型，必须从以下选一：童话、历史、神话、寓言、冒险、侦探、科幻、生活
- theme：具体故事主题（可与 title 相同；用户未指定主题时由你拟定）
- characters：主要人物名，逗号分隔 1~4 个（如「阿澄,灯笼精灵」）；原创须填写
- canonical：经典/神话等正篇时填写社会通行规范名（如「哪吒闹海」）；原创可省略
- aliases：常见口语异名，逗号分隔 1~5 个（如「哪吒三太子闹海,哪吒脑海」）；无则省略
- 第二行起直接写故事正文第一句，禁止过渡语、前言、Markdown 与标题行
- 系统会先向听众播报一句简短过渡语，你须直接从故事正文写起`

	prompt += "\n\n只输出上述 meta 行与故事正文，不要后记或任何解释。"
	return prompt
}

// BuildUserPrompt 构建用户侧生成提示。seed 可为 nil。
func BuildUserPrompt(params StoryParams, weakThemes []string, seed *DiversitySeed) string {
	p := params
	NormalizeStoryParams(&p)
	canonical := ShouldTellCanonical(p)

	var parts []string
	if p.Theme != "" {
		if canonical {
			parts = append(parts, fmt.Sprintf("请讲述：%s（经典/神话/寓言正篇，勿新编）", p.Theme))
		} else {
			parts = append(parts, "主题："+p.Theme+"（围绕主题原创，情节新颖）")
		}
	}
	if p.Style != "" {
		parts = append(parts, "风格："+p.Style)
	}
	if len(p.Interests) > 0 {
		parts = append(parts, fmt.Sprintf("兴趣参考：%v", p.Interests))
	}
	if p.DurationHint == "short" {
		parts = append(parts, "篇幅偏短")
	} else if p.DurationHint == "long" {
		parts = append(parts, "篇幅偏长")
	}
	parts = append(parts, FormatDiversityPromptLines(seed)...)
	// 多样性种子未覆盖主题回避时，保留弱主题提示；有种子则已写「禁止复用」
	if !canonical && seed == nil && len(weakThemes) > 0 {
		parts = append(parts, fmt.Sprintf("禁止复用近期主题：%v；本次须创作全新情节与人物", weakThemes))
	}
	if len(parts) == 0 {
		if canonical {
			return "请讲述适合当前年龄的经典儿童故事正文。"
		}
		return "请创作一个适合当前年龄、情节新颖的儿童故事。"
	}
	verb := "请创作儿童故事。"
	if canonical {
		verb = "请开始讲述。"
	}
	result := verb
	for _, part := range parts {
		result += "\n" + part
	}
	return result
}

// BuildContinueUserPrompt 构建续写用户提示：基于已有全文，紧接文末自然续写。
func BuildContinueUserPrompt(params StoryParams, writtenText string) string {
	writtenText = strings.TrimSpace(writtenText)
	p := params
	NormalizeStoryParams(&p)

	var parts []string
	if p.Theme != "" {
		parts = append(parts, "故事主题："+p.Theme)
	}
	if writtenText != "" {
		parts = append(parts, "以下是故事目前已写出的全文（听众刚听完其中未播部分，请勿重复任何已有字句）：\n"+writtenText)
	}
	parts = append(parts, "请紧接最后一个字自然续写，保持情节、人物与语气一致，写至完整结局。")
	parts = append(parts, "只输出新增正文，不要重复上文，不要开场白、过渡语或解释。")

	result := "请续写儿童故事。"
	for _, part := range parts {
		result += "\n" + part
	}
	return result
}

// BuildContinueSystemPrompt 续写专用 system prompt。
func BuildContinueSystemPrompt(params StoryParams) string {
	base := BuildSystemPrompt(params)
	return base + `

## 续写模式
- 上文为已写定稿，听众刚听完；你必须从文末最后一个字无缝衔接
- 首句应像自然接龙，勿用「接着」「然后」等生硬开头，勿复述上文结尾
- 保持人物、情节、语气与上文一致
- 禁止重复、改写或总结已有内容
- 写至故事自然收束的完整结尾`
}
func BuildFillerText(params StoryParams, cfg Config) string {
	if !cfg.FillerEnabled {
		return ""
	}
	p := params
	NormalizeStoryParams(&p)
	theme := strings.TrimSpace(p.Theme)

	if theme != "" {
		label := ThemeSpeakLabel(theme, p.RequestType)
		if p.IsBedtime != nil && *p.IsBedtime {
			return fmt.Sprintf("好呀，那我们讲%s，乖乖听哦。", label)
		}
		return fmt.Sprintf("好呀，我给你讲%s。", label)
	}

	if p.IsBedtime != nil && *p.IsBedtime && strings.TrimSpace(cfg.FillerBedtime) != "" {
		return strings.TrimSpace(cfg.FillerBedtime)
	}
	return strings.TrimSpace(cfg.FillerDefault)
}

// SpeakableStoryTitle 口语化故事标题，避免「…的故事的故事」。
func SpeakableStoryTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" || title == "儿童故事" || looksLikeStoryOpening(title) {
		return ""
	}
	if strings.Contains(title, "故事") || strings.Contains(title, "童话") ||
		strings.Contains(title, "神话") || strings.Contains(title, "寓言") {
		return title
	}
	return title + "的故事"
}

// BuildNarrationIntro 复播/直读正文前的礼貌过渡语（告知标题）。
func BuildNarrationIntro(title string, cfg Config) string {
	if !cfg.FillerEnabled {
		return ""
	}
	if label := SpeakableStoryTitle(title); label != "" {
		return fmt.Sprintf("好呀，接下来给你讲%s。", label)
	}
	return strings.TrimSpace(cfg.FillerDefault)
}

// BuildResumeTitleLead 续讲前点明标题的引导语（接在断点衔接语之前）。
func BuildResumeTitleLead(title string, cfg Config) string {
	if !cfg.FillerEnabled {
		return ""
	}
	if label := SpeakableStoryTitle(title); label != "" {
		return fmt.Sprintf("好的，我们继续讲%s。", label)
	}
	return ""
}

// BuildMetaTitleAnnounce 流式开放生成在收到 meta 标题后、正文前的短告知。
func BuildMetaTitleAnnounce(title string, cfg Config) string {
	if !cfg.FillerEnabled {
		return ""
	}
	title = strings.TrimSpace(title)
	if title == "" || title == "儿童故事" || looksLikeStoryOpening(title) {
		return ""
	}
	return fmt.Sprintf("这篇故事叫%s。", title)
}

// HasNarrationIntroPrefix 判断文本是否已含讲前过渡语，避免重复拼接。
func HasNarrationIntroPrefix(s string) bool {
	s = strings.TrimSpace(s)
	for _, p := range []string{
		"好呀，接下来给你讲",
		"好呀，我给你讲",
		"好呀，那我们讲",
		"好的，我们继续讲",
		"这篇故事叫",
	} {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// HasResumeTransitionPrefix 正文是否已含续讲衔接过渡语。
func HasResumeTransitionPrefix(s string) bool {
	s = strings.TrimSpace(s)
	for _, p := range []string{
		"上次讲到",
		"好的，我们继续讲",
		"上次讲到这儿",
	} {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// StripRedundantStoryOpening 去掉与系统过渡语重复的模型开场白。
// 返回 (处理后文本, 是否应整句跳过)。
func StripRedundantStoryOpening(sentence string) (string, bool) {
	s := strings.TrimSpace(sentence)
	if s == "" {
		return "", true
	}
	if HasNarrationIntroPrefix(s) {
		return "", true
	}
	for _, p := range []string{
		"好的，我来给你讲",
		"好的，我来为你讲",
		"好的，我来讲述",
		"让我来给你讲",
		"让我给你讲",
		"我来给你讲一个故事",
		"我给你讲一个故事",
		"今天给大家讲",
		"今天我给你们讲",
	} {
		if strings.HasPrefix(s, p) {
			return "", true
		}
	}
	return s, false
}

// BuildQuestionForMissing 生成语音友好的追问文案。
func BuildQuestionForMissing(missing []string) string {
	if len(missing) == 0 {
		return ""
	}
	switch missing[0] {
	case "age_band":
		return "小朋友几岁啦？这样我才能讲合适的故事哦。"
	default:
		return "想听什么风格的故事呢？睡前还是冒险？"
	}
}
