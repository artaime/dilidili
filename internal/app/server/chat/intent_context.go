package chat

import (
	"fmt"
	"strings"
	"unicode/utf8"

	. "dili-esp32-server-golang/internal/data/client"

	"github.com/cloudwego/eino/schema"
)

const (
	classifierContextMaxMessages = 6
	classifierContextMaxRunes    = 200
)

// buildClassifierUserPrompt 将近期 Dialogue 与当前句拼成分类器 user 内容（无关键词规则）。
func buildClassifierUserPrompt(state *ClientState, currentText string) string {
	currentText = strings.TrimSpace(currentText)
	var b strings.Builder
	recent := classifierRecentMessages(state)
	if len(recent) > 0 {
		b.WriteString("近期对话：\n")
		for _, msg := range recent {
			if msg == nil {
				continue
			}
			role := classifierRoleLabel(msg.Role)
			content := truncateRunes(strings.TrimSpace(msg.Content), classifierContextMaxRunes)
			if content == "" && len(msg.ToolCalls) == 0 {
				continue
			}
			if content == "" {
				content = "(工具调用)"
			}
			b.WriteString(role)
			b.WriteString("：")
			b.WriteString(content)
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}
	b.WriteString("当前用户说：")
	b.WriteString(currentText)
	return b.String()
}

func classifierRecentMessages(state *ClientState) []*schema.Message {
	if state == nil {
		return nil
	}
	limit := shortContextRecentMessageLimit()
	if limit > classifierContextMaxMessages {
		limit = classifierContextMaxMessages
	}
	if limit <= 0 {
		limit = classifierContextMaxMessages
	}
	return state.GetMessages(limit)
}

func classifierRoleLabel(role schema.RoleType) string {
	switch role {
	case schema.User:
		return "用户"
	case schema.Assistant:
		return "助手"
	case schema.System:
		return "系统"
	case schema.Tool:
		return "工具"
	default:
		return fmt.Sprintf("%s", role)
	}
}

func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max]) + "…"
}
