package chat

import (
	"context"
	"sort"
	"strings"
	"time"

	parentmsg "xiaozhi-esp32-server-golang/internal/data/parentmessage"
	"xiaozhi-esp32-server-golang/internal/domain/chat/intent"
	log "xiaozhi-esp32-server-golang/logger"
)

const parentMessageSelectSearchLimit = 100

type parentMessageSelectFilter struct {
	FamilyRole string
	Start      time.Time
	End        time.Time
	HasStart   bool
	HasEnd     bool
}

func (c *ChatManager) listAccessibleParentMessages(ctx context.Context) ([]parentMessageItem, error) {
	return c.searchParentMessages(ctx, parentMessageSelectFilter{})
}

func (c *ChatManager) searchParentMessages(ctx context.Context, filter parentMessageSelectFilter) ([]parentMessageItem, error) {
	if c.parentMessageClient == nil {
		return nil, nil
	}
	params := parentmsg.SearchParams{Limit: parentMessageSelectSearchLimit}
	if filter.HasStart {
		params.Start = formatParentMessageSearchTime(filter.Start)
	}
	if filter.HasEnd {
		params.End = formatParentMessageSearchTime(filter.End)
	}

	messages, err := c.parentMessageClient.SearchMessages(ctx, c.DeviceID, params)
	if err != nil {
		return nil, err
	}
	out := make([]parentMessageItem, 0, len(messages))
	for _, msg := range messages {
		if !c.hasPlayableParentMessage(msg) {
			continue
		}
		if filter.FamilyRole != "" && !familyRoleMatches(msg.FamilyRole, filter.FamilyRole) {
			continue
		}
		out = append(out, msg)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func formatParentMessageSearchTime(t time.Time) string {
	return t.Format("2006-01-02T15:04:05")
}

func selectParentMessageByFilter(messages []parentMessageItem, filter parentMessageSelectFilter) (parentMessageItem, bool) {
	if len(messages) == 0 {
		return parentMessageItem{}, false
	}
	matches := make([]parentMessageItem, 0)
	for _, msg := range messages {
		if parentMessageMatchesFilter(msg, filter) {
			matches = append(matches, msg)
		}
	}
	if len(matches) == 0 {
		return parentMessageItem{}, false
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].CreatedAt.After(matches[j].CreatedAt)
	})
	return matches[0], true
}

func parentMessageMatchesFilter(msg parentMessageItem, filter parentMessageSelectFilter) bool {
	if filter.FamilyRole != "" && !familyRoleMatches(msg.FamilyRole, filter.FamilyRole) {
		return false
	}
	createdAt := msg.CreatedAt
	if filter.HasStart && createdAt.Before(filter.Start) {
		return false
	}
	if filter.HasEnd && createdAt.After(filter.End) {
		return false
	}
	return true
}

func familyRoleMatches(stored, query string) bool {
	stored = normalizeFamilyRoleLabel(stored)
	query = normalizeFamilyRoleLabel(query)
	if stored == query {
		return true
	}
	aliases := map[string][]string{
		"爸爸": {"爸", "父亲"},
		"妈妈": {"妈", "母亲"},
		"外公": {"姥爷"},
		"外婆": {"姥姥"},
	}
	for canonical, aliasList := range aliases {
		if stored == canonical || query == canonical {
			for _, alias := range aliasList {
				if stored == alias || query == alias {
					return true
				}
			}
		}
		if stored == canonical && containsAlias(aliasList, query) {
			return true
		}
		if query == canonical && containsAlias(aliasList, stored) {
			return true
		}
	}
	return strings.Contains(stored, query) || strings.Contains(query, stored)
}

func containsAlias(aliases []string, label string) bool {
	for _, alias := range aliases {
		if label == alias || strings.Contains(label, alias) || strings.Contains(alias, label) {
			return true
		}
	}
	return false
}

func (c *ChatManager) playLatestParentMessage(ctx context.Context) error {
	messages, err := c.listAccessibleParentMessages(ctx)
	if err != nil {
		log.Warnf("设备 %s 拉取留言列表失败: %v", c.DeviceID, err)
		return c.InjectMessage("播放留言失败了，稍后再试试吧。", true, true)
	}
	if len(messages) == 0 {
		return c.InjectMessage("没有找到可以播放的留言哦。", true, true)
	}
	return c.playSingleParentMessage(ctx, messages[0])
}

func (c *ChatManager) playSelectedParentMessage(ctx context.Context, data intent.MsgPlayData) error {
	filter, ok := buildParentMessageSelectFilter(data)
	if !ok {
		return c.InjectMessage("没有找到符合条件的留言哦。", true, true)
	}
	messages, err := c.searchParentMessages(ctx, filter)
	if err != nil {
		log.Warnf("设备 %s 搜索留言失败: %v", c.DeviceID, err)
		return c.InjectMessage("播放留言失败了，稍后再试试吧。", true, true)
	}
	msg, matched := selectParentMessageByFilter(messages, filter)
	if !matched {
		desc := describeParentMessageFilter(filter, data)
		return c.InjectMessage("没有找到"+desc+"的留言哦。", true, true)
	}
	log.Infof("设备 %s msg_play select matched id=%d role=%s status=%s filter=%+v",
		c.DeviceID, msg.ID, msg.FamilyRole, msg.Status, filter)
	return c.playSingleParentMessage(ctx, msg)
}

func buildParentMessageSelectFilter(data intent.MsgPlayData) (parentMessageSelectFilter, bool) {
	filter := parentMessageSelectFilter{
		FamilyRole: strings.TrimSpace(data.FamilyRole),
	}
	loc := time.Local
	if start, ok := parseIntentTime(data.Start, loc); ok {
		filter.Start = start
		filter.HasStart = true
	}
	if end, ok := parseIntentTimeEnd(data.End, loc); ok {
		filter.End = end
		filter.HasEnd = true
	}
	if filter.FamilyRole == "" && !filter.HasStart && !filter.HasEnd {
		return parentMessageSelectFilter{}, false
	}
	return filter, true
}

func parseIntentTime(raw string, loc *time.Location) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, raw, loc)
		if err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func parseIntentTimeEnd(raw string, loc *time.Location) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	if t, ok := parseIntentTime(raw, loc); ok {
		if len(raw) == len("2006-01-02") {
			return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), loc), true
		}
		return t, true
	}
	return time.Time{}, false
}

func describeParentMessageFilter(filter parentMessageSelectFilter, data intent.MsgPlayData) string {
	var parts []string
	if filter.FamilyRole != "" {
		parts = append(parts, filter.FamilyRole)
	}
	if filter.HasStart || filter.HasEnd {
		if strings.TrimSpace(data.Start) != "" || strings.TrimSpace(data.End) != "" {
			parts = append(parts, "指定时间")
		}
	}
	if len(parts) == 0 {
		return "符合条件"
	}
	return strings.Join(parts, "")
}

func (c *ChatManager) playSingleParentMessage(ctx context.Context, msg parentMessageItem) error {
	if !c.hasPlayableParentMessage(msg) {
		return c.InjectMessage("这条留言暂时无法播放。", true, true)
	}
	transition := buildTransitionPrompt(msg.FamilyRole, msg.CreatedAt, time.Now())
	if err := c.injectSpeechSegment(transition, true, ttsTurnEndPolicyNone); err != nil {
		return err
	}
	c.waitInjectedSpeechSettled(ctx, transition)
	if err := c.playParentMessage(ctx, msg); err != nil {
		log.Warnf("设备 %s 播放留言失败 id=%d: %v", c.DeviceID, msg.ID, err)
		return c.InjectMessage("播放留言失败了，稍后再试试吧。", true, true)
	}
	status := strings.TrimSpace(msg.Status)
	if status == "" || status == "pending" || status == "notified" {
		c.updateParentMessageStatus(ctx, msg.ID, "played")
	}
	c.recordPlayedMessageProfile(msg)
	return nil
}
