package chat

import "testing"

func TestIsStoryRecallQuestion(t *testing.T) {
	positive := []string{
		"你刚刚给我讲了一个什么故事？",
		"刚才讲的是什么故事",
		"刚刚那个故事叫什么名字",
		"之前讲的故事讲了什么内容",
	}
	for _, text := range positive {
		if !isStoryRecallQuestion(text) {
			t.Fatalf("expected recall question: %q", text)
		}
	}
	negative := []string{
		"再讲一遍刚才的故事",
		"接着讲刚才那个",
		"讲个新故事",
		"今天天气怎么样",
	}
	for _, text := range negative {
		if isStoryRecallQuestion(text) {
			t.Fatalf("expected not recall question: %q", text)
		}
	}
}
