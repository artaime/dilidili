package story

import "strings"

const (
	MessageKindStoryCard = "story_card"

	ExtraKeyKind               = "kind"
	ExtraKeyStoryID            = "story_id"
	ExtraKeyTitle              = "title"
	ExtraKeyGenerationComplete = "generation_complete"
)

// StoryCardContent 小程序/历史列表展示的短文案。
func StoryCardContent(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "故事"
	}
	return "播放故事：" + title
}

// StoryCardExtra 写入 schema.Message.Extra，并由 message worker 落入 chat_messages.metadata。
func StoryCardExtra(storyID, title string, generationComplete bool) map[string]any {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "故事"
	}
	return map[string]any{
		ExtraKeyKind:               MessageKindStoryCard,
		ExtraKeyStoryID:            strings.TrimSpace(storyID),
		ExtraKeyTitle:              title,
		ExtraKeyGenerationComplete: generationComplete,
	}
}

// MergeStoryCardMetadata 将 Extra 中的故事卡片字段并入 metadata。
func MergeStoryCardMetadata(dst map[string]any, extra map[string]any) {
	if dst == nil || extra == nil {
		return
	}
	kind, _ := extra[ExtraKeyKind].(string)
	if kind != MessageKindStoryCard {
		return
	}
	dst[ExtraKeyKind] = MessageKindStoryCard
	if v, ok := extra[ExtraKeyStoryID]; ok {
		dst[ExtraKeyStoryID] = v
	}
	if v, ok := extra[ExtraKeyTitle]; ok {
		dst[ExtraKeyTitle] = v
	}
	if v, ok := extra[ExtraKeyGenerationComplete]; ok {
		dst[ExtraKeyGenerationComplete] = v
	}
}
