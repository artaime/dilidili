package chat

import "strings"

// isStoryRecallQuestion 用户追问「刚才讲了什么故事/什么内容」，应走主 LLM + 最近故事上下文，而非 create_child_story。
func isStoryRecallQuestion(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	if strings.Contains(text, "再讲") || strings.Contains(text, "继续讲") || strings.Contains(text, "接着讲") ||
		strings.Contains(text, "来一遍") || strings.Contains(text, "重新讲") || strings.Contains(text, "复播") {
		return false
	}
	recallWords := []string{"刚才", "刚刚", "上一", "之前", "上次"}
	hasRecall := false
	for _, w := range recallWords {
		if strings.Contains(text, w) {
			hasRecall = true
			break
		}
	}
	questionWords := []string{
		"什么故事", "哪个故事", "什么内容", "讲了什么", "讲的是什么", "什么童话",
		"什么寓言", "什么神话", "叫什么名字", "名叫什么", "故事名", "故事叫",
	}
	hasQuestion := strings.Contains(text, "?") || strings.Contains(text, "？")
	for _, w := range questionWords {
		if strings.Contains(text, w) {
			hasQuestion = true
			break
		}
	}
	return hasRecall && hasQuestion
}
