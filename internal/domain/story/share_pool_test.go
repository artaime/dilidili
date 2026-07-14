package story

import (
	"strings"
	"testing"
)

func TestClassifyShareIntent(t *testing.T) {
	bedtime := true
	cases := []struct {
		name string
		p    StoryParams
		want string
	}{
		{"named myth", StoryParams{RequestType: StoryModeMyth, NarrationMode: NarrationCanonical, Theme: "哪吒闹海"}, SharePoolNamed},
		{"open casual", StoryParams{UserSaidCasual: true, Theme: ""}, SharePoolOpen},
		{"open empty theme", StoryParams{RequestType: StoryModeOriginal, Theme: ""}, SharePoolOpen},
		{"bedtime", StoryParams{IsBedtime: &bedtime, Theme: "睡前故事"}, SharePoolBedtime},
		{"creative plot", StoryParams{RequestType: StoryModeOriginal, NarrationMode: NarrationCreative, Theme: "小恐龙找妈妈"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyShareIntent(tc.p); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestNormalizeCanonicalKeyNezha(t *testing.T) {
	a := NormalizeCanonicalKey("哪吒三太子闹海")
	b := NormalizeCanonicalKey("哪吒闹海")
	c := NormalizeCanonicalKey("哪吒脑海")
	if a == "" || b == "" || c == "" {
		t.Fatal("empty keys")
	}
	if a != b {
		t.Fatalf("oral vs standard: %q vs %q", a, b)
	}
	if c != b {
		t.Fatalf("typo 脑海: %q vs %q", c, b)
	}
}

func TestNormalizeCanonicalKeyHouYi(t *testing.T) {
	a := NormalizeCanonicalKey("后羿射日")
	b := NormalizeCanonicalKey("后裔射太阳")
	if a != b {
		t.Fatalf("houyi oral: %q vs %q", a, b)
	}
}

func TestBuildShareLookupKeys(t *testing.T) {
	keys := BuildShareLookupKeys("后羿射日", "后裔射太阳")
	joined := strings.Join(keys, "|")
	if !strings.Contains(joined, "后羿射日") {
		t.Fatalf("missing canonical in %v", keys)
	}
}

func TestBuildAliasKeys(t *testing.T) {
	keys := BuildAliasKeys("哪吒闹海", "哪吒闹海的故事", "哪吒三太子闹海", []string{"哪吒脑海"})
	if len(keys) < 2 {
		t.Fatalf("expected multiple aliases, got %v", keys)
	}
	joined := strings.Join(keys, "|")
	if !strings.Contains(joined, "哪吒") {
		t.Fatalf("missing 哪吒 in %v", keys)
	}
}

func TestApplyShareEnrollmentSkipsIncomplete(t *testing.T) {
	rec := &StoryRecord{
		FullText: "从前有座山。",
		ParamsSnapshot: map[string]any{
			SnapshotKeyPoolKind: SharePoolOpen,
			"draft":             true,
		},
	}
	setGenerationComplete(rec, false)
	ApplyShareEnrollment(rec, StoryParams{UserSaidCasual: true}, StoryMeta{})
	if _, ok := rec.ParamsSnapshot[SnapshotKeyPoolKind]; ok {
		t.Fatal("incomplete story must not keep pool_kind")
	}

	setGenerationComplete(rec, true)
	ApplyShareEnrollment(rec, StoryParams{UserSaidCasual: true}, StoryMeta{})
	if got, _ := rec.ParamsSnapshot[SnapshotKeyPoolKind].(string); got != SharePoolOpen {
		t.Fatalf("complete casual story should enroll open pool, got %q", got)
	}
}
