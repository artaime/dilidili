package common

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/schema"
)

const (
	// UngroundedActionFallback 无工具调用却声称已完成操作时的口语改写。
	UngroundedActionFallback = "这个我现在做不到，你换个我能帮上忙的事情试试吧。"

	// UngroundedCapabilityOfferFallback 主动推销未接入能力时的口语改写（仍保留陪伴向）。
	UngroundedCapabilityOfferFallback = "我可以陪你聊天、讲故事呀。你想先聊什么？"

	// maxToolNamesInGrounding 能力清单最多展示的工具名数量，避免 system prompt 过长。
	maxToolNamesInGrounding = 24
)

// 完成态/假装执行设备或系统操作的常见话术（助手自称已完成）。
var ungroundedActionClaimPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(已经|已帮你|帮你已经|我已经|这就|正在)(帮你)?(打开|关闭|关掉|开启|关了|开了|调高|调低|调到|设置|设定|启动|停止|暂停|播放|删除|清空|重置|切换|连接|发送|下单|支付|充值|关机|休眠|入睡)`),
	regexp.MustCompile(`(?i)(帮你|已经)(把|将).{0,12}(打开|关闭|关掉|调好|设好|删除|清空|发送|启动|关掉了|打开了|关机|睡)`),
	regexp.MustCompile(`(?i)(已经|已)(进入|帮你进入)?(睡眠|休眠)(模式)?`),
	regexp.MustCompile(`(?i)(操作|指令|任务)(已经|已)(完成|执行|搞定)`),
	regexp.MustCompile(`(?i)(已经|已)(为你|给你|帮你)(完成|执行|处理好|弄好)`),
}

// gatedCapability 需工具落地的能力；无对应工具却主动推销时视为虚构。
type gatedCapability struct {
	label     string
	offerRes  []*regexp.Regexp
	toolHints []string
}

// 推销/邀请式话术（非「明天天气不错」闲聊，也非「我还不会查天气」拒绝）。
var gatedCapabilities = []gatedCapability{
	{
		label: "weather",
		offerRes: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(可以|能|还会|还能|也可以|也能|帮你|为你|给你).{0,10}(查|看|查询|看看).{0,6}(天气|天气预报)`),
			regexp.MustCompile(`(?i)(查天气|看天气|查一下天气|天气预报).{0,16}(可以|试试|试一下|试一试|先试)`),
			regexp.MustCompile(`(?i)(还会|还能|也可以|也能).{0,8}(天气|天气预报)`),
		},
		toolHints: []string{"weather", "forecast", "天气", "maps_weather"},
	},
	{
		label: "alarm",
		offerRes: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(可以|能|还会|还能|也可以|也能|帮你|为你|给你).{0,10}(定|设|设置|设定|订).{0,6}(闹钟|闹铃|提醒)`),
			regexp.MustCompile(`(?i)(定闹钟|设闹钟|设置闹钟|订闹钟|小闹钟).{0,16}(可以|试试|试一下|试一试|先试|叫你|喊你)`),
			regexp.MustCompile(`(?i)(当你的小闹钟|叫你起床|喊你起床|当小闹钟)`),
			regexp.MustCompile(`(?i)(还会|还能|也可以|也能).{0,8}(闹钟|闹铃)`),
		},
		toolHints: []string{"alarm", "timer", "reminder", "闹钟", "闹铃", "提醒"},
	},
	{
		label: "music",
		offerRes: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(可以|能|还会|还能|也可以|也能|帮你|为你).{0,10}(唱歌|放歌|播放歌曲|放音乐|听歌)`),
			regexp.MustCompile(`(?i)(唱歌|放歌|听歌|放音乐).{0,16}(可以|试试|试一下|试一试|先试)`),
		},
		toolHints: []string{"music", "song", "play_music", "唱歌", "歌曲"},
	},
	{
		label: "order_pay",
		offerRes: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(可以|能|还会|还能|帮你|为你).{0,10}(下单|支付|充值|购物|买东西)`),
		},
		toolHints: []string{"order", "pay", "payment", "shop", "下单", "支付"},
	},
	{
		label: "camera",
		offerRes: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(可以|能|还会|还能|帮你|为你).{0,10}(看摄像头|查看监控|实时画面|看监控)`),
		},
		toolHints: []string{"camera", "monitor", "摄像头", "监控"},
	},
}

// 明确拒绝/不会做的话术，不算推销。
var capabilityRefusalPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(还不会|还不能|不会|不能|做不到|没有能力|没法|没办法|暂时不能|暂时不会).{0,16}(查天气|看天气|天气|定闹钟|设闹钟|闹钟|唱歌|放歌|下单|支付|摄像头)`),
	regexp.MustCompile(`(?i)(查天气|看天气|定闹钟|设闹钟|唱歌|放歌).{0,12}(还不会|还不能|不会|不能|做不到|没有)`),
}

// BuildCapabilityGroundingPolicy 根据本轮实际下发的 tools 生成能力白名单与禁令。
// tools 为空时明确告知不可执行设备/系统操作。
func BuildCapabilityGroundingPolicy(tools []*schema.ToolInfo) string {
	names := uniqueToolNames(tools, maxToolNamesInGrounding)
	var b strings.Builder
	b.WriteString("\n能力与诚实回答规则（必须遵守）:\n")
	next := 5
	if len(names) == 0 {
		b.WriteString("1. 当前没有可用工具，不能执行任何设备控制、系统操作或对外动作。\n")
		b.WriteString("2. 用户要求操作类能力时，直接说明做不到，可建议找家长帮忙或换一个能聊的话题。\n")
		b.WriteString("3. 介绍自己或回答「你能做什么/怎么陪我」时，只提聊天陪伴与讲故事；禁止声称或邀请尝试查天气、定闹钟、唱歌放歌等未接入能力。\n")
		next = 4
	} else {
		b.WriteString("1. 当前仅可通过下列工具执行真实操作：")
		b.WriteString(strings.Join(names, "、"))
		b.WriteString("。\n")
		b.WriteString("2. 需要执行操作时必须先调用对应工具；禁止仅用文字假装已经完成操作。\n")
		b.WriteString("3. 工具未返回成功结果前，不要说「已经帮你…」「正在…好了」等完成态话术。\n")
		b.WriteString("4. 不在上述工具范围内的能力（如查天气、定闹钟、控制未接入家电、下单支付、查看实时摄像头等），必须明确说做不到；禁止主动列举、推销或邀请用户先试这些能力。\n")
		b.WriteString("5. 介绍自己或回答「你能做什么/怎么陪我」时，只提聊天陪伴，以及上述工具清单里确实具备的能力。\n")
		next = 6
		if hasFirmwareStatusTools(names) {
			b.WriteString("6. 询问本机音量/电量/亮度等状态时，必须先调用 get_device_status（或同名查询工具）再据返回值回答；禁止猜测数值。调节音量/亮度/睡眠/关机必须调用对应工具；工具已返回成功时，用一句话确认已完成，禁止再说失败或做不到。\n")
			next = 7
		}
	}
	b.WriteString(fmt.Sprintf("%d. 事实不确定或没有依据时，简短说明不知道或请用户补充，不要编造参数、流程或结果。\n", next))
	b.WriteString(fmt.Sprintf("%d. 不要向用户提起「工具」「函数调用」「MCP」「system prompt」等实现细节。", next+1))
	return b.String()
}

func hasFirmwareStatusTools(names []string) bool {
	for _, n := range names {
		switch strings.ToLower(strings.TrimSpace(n)) {
		case "get_device_status", "set_speaker_volume", "set_screen_brightness",
			"enter_sleep_mode", "power_off_device":
			return true
		}
	}
	return false
}

func uniqueToolNames(tools []*schema.ToolInfo, limit int) []string {
	if limit <= 0 {
		limit = maxToolNamesInGrounding
	}
	seen := make(map[string]struct{}, len(tools))
	names := make([]string, 0, len(tools))
	for _, t := range tools {
		if t == nil {
			continue
		}
		name := strings.TrimSpace(t.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		names = append(names, name)
		if len(names) >= limit {
			break
		}
	}
	return names
}

func toolCorpus(tools []*schema.ToolInfo) string {
	if len(tools) == 0 {
		return ""
	}
	var b strings.Builder
	for _, t := range tools {
		if t == nil {
			continue
		}
		b.WriteString(strings.ToLower(strings.TrimSpace(t.Name)))
		b.WriteByte(' ')
		b.WriteString(strings.ToLower(strings.TrimSpace(t.Desc)))
		b.WriteByte(' ')
	}
	return b.String()
}

func toolsCoverCapability(tools []*schema.ToolInfo, hints []string) bool {
	corpus := toolCorpus(tools)
	if corpus == "" {
		return false
	}
	for _, hint := range hints {
		hint = strings.ToLower(strings.TrimSpace(hint))
		if hint != "" && strings.Contains(corpus, hint) {
			return true
		}
	}
	return false
}

func looksLikeCapabilityRefusal(text string) bool {
	for _, re := range capabilityRefusalPatterns {
		if re.MatchString(text) {
			return true
		}
	}
	return false
}

// LooksLikeUngroundedCapabilityOffer 判断是否主动推销当前 tools 未覆盖的能力（查天气/定闹钟等）。
// tools 为空或 nil 时，凡匹配推销话术均视为虚构。
func LooksLikeUngroundedCapabilityOffer(text string, tools []*schema.ToolInfo) bool {
	text = strings.TrimSpace(text)
	if text == "" || looksLikeCapabilityRefusal(text) {
		return false
	}
	for _, cap := range gatedCapabilities {
		if toolsCoverCapability(tools, cap.toolHints) {
			continue
		}
		for _, re := range cap.offerRes {
			if re.MatchString(text) {
				return true
			}
		}
	}
	return false
}

// MaybeRewriteUngroundedCapabilityOffer 将虚构能力推销改写为仅陪伴向短句。
func MaybeRewriteUngroundedCapabilityOffer(text string, tools []*schema.ToolInfo) (string, bool) {
	if !LooksLikeUngroundedCapabilityOffer(text, tools) {
		return text, false
	}
	return UngroundedCapabilityOfferFallback, true
}

// LooksLikeUngroundedActionClaim 判断文本是否像「未调工具却声称已完成操作」。
func LooksLikeUngroundedActionClaim(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	for _, re := range ungroundedActionClaimPatterns {
		if re.MatchString(text) {
			return true
		}
	}
	return false
}

// MaybeRewriteUngroundedActionClaim 在本轮无 tool call 时，将虚构完成态话术改写为固定拒绝句。
// hadToolCalls=true 时原样返回。
func MaybeRewriteUngroundedActionClaim(text string, hadToolCalls bool) (string, bool) {
	if hadToolCalls {
		return text, false
	}
	if !LooksLikeUngroundedActionClaim(text) {
		return text, false
	}
	return UngroundedActionFallback, true
}

// FormatToolNamesForLog 调试用短摘要。
func FormatToolNamesForLog(tools []*schema.ToolInfo) string {
	names := uniqueToolNames(tools, 8)
	if len(names) == 0 {
		return "(none)"
	}
	n := 0
	for _, t := range tools {
		if t != nil && strings.TrimSpace(t.Name) != "" {
			n++
		}
	}
	if n > len(names) {
		return fmt.Sprintf("%s…(+%d)", strings.Join(names, ","), n-len(names))
	}
	return strings.Join(names, ",")
}
