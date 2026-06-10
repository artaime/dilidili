package chat

import "testing"

func TestClassifyParentMessageIntent(t *testing.T) {
	cases := []struct {
		text   string
		expect parentMessageIntent
	}{
		{"要", parentMessageIntentAffirmative},
		{"好的听一下", parentMessageIntentAffirmative},
		{"想听", parentMessageIntentAffirmative},
		{"不要", parentMessageIntentNegative},
		{"不用了", parentMessageIntentNegative},
		{"算了", parentMessageIntentNegative},
		{"今天天气怎么样", parentMessageIntentUnknown},
		{"", parentMessageIntentUnknown},
	}

	for _, tc := range cases {
		got := classifyParentMessageIntent(tc.text)
		if got != tc.expect {
			t.Fatalf("classifyParentMessageIntent(%q) = %d, want %d", tc.text, got, tc.expect)
		}
	}
}
