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
		next = 3
	} else {
		b.WriteString("1. 当前仅可通过下列工具执行真实操作：")
		b.WriteString(strings.Join(names, "、"))
		b.WriteString("。\n")
		b.WriteString("2. 需要执行操作时必须先调用对应工具；禁止仅用文字假装已经完成操作。\n")
		b.WriteString("3. 工具未返回成功结果前，不要说「已经帮你…」「正在…好了」等完成态话术。\n")
		b.WriteString("4. 不在上述工具范围内的能力（如控制未接入的家电、下单支付、查看实时摄像头等），必须明确说做不到。\n")
			if hasFirmwareStatusTools(names) {
				b.WriteString("5. 询问本机音量/电量/亮度等状态时，必须先调用 get_device_status（或同名查询工具）再据返回值回答；禁止猜测数值。调节音量/亮度/睡眠/关机必须调用对应工具；工具已返回成功时，用一句话确认已完成，禁止再说失败或做不到。\n")
				next = 6
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
