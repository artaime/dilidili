package story_persist

import "testing"

func TestJoinChatCompletionsURL(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://api.openai.com/v1", "https://api.openai.com/v1/chat/completions"},
		{"https://api.openai.com/v1/", "https://api.openai.com/v1/chat/completions"},
		{"https://x.com/v1/chat/completions", "https://x.com/v1/chat/completions"},
		{"https://proxy.example", "https://proxy.example/v1/chat/completions"},
	}
	for _, tc := range cases {
		if got := joinChatCompletionsURL(tc.in); got != tc.want {
			t.Fatalf("%q => %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestSegmentText(t *testing.T) {
	segs := SegmentText("从前有座山。山里有座庙。庙里有个老和尚。")
	if len(segs) == 0 {
		t.Fatal("empty segments")
	}
}

func TestStripLeadingMetaTitle(t *testing.T) {
	title, body := stripLeadingMetaTitle("[[meta:title=后羿射日|theme=后羿射日]]\n很久很久以前。")
	if title != "后羿射日" || body == "" {
		t.Fatalf("title=%q body=%q", title, body)
	}
}
