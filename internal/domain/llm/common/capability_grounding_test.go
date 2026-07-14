package common

import (
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestBuildCapabilityGroundingPolicyEmptyTools(t *testing.T) {
	got := BuildCapabilityGroundingPolicy(nil)
	if !strings.Contains(got, "没有可用工具") {
		t.Fatalf("expected empty-tools rule, got %q", got)
	}
	if !strings.Contains(got, "不要编造") {
		t.Fatalf("expected no-fabricate rule, got %q", got)
	}
}

func TestBuildCapabilityGroundingPolicyWithTools(t *testing.T) {
	tools := []*schema.ToolInfo{
		{Name: "search_knowledge"},
		{Name: "create_child_story"},
		{Name: "search_knowledge"}, // duplicate
		nil,
	}
	got := BuildCapabilityGroundingPolicy(tools)
	if !strings.Contains(got, "search_knowledge") || !strings.Contains(got, "create_child_story") {
		t.Fatalf("expected tool names in policy, got %q", got)
	}
	if !strings.Contains(got, "必须先调用对应工具") {
		t.Fatalf("expected must-call-tool rule, got %q", got)
	}
	// duplicates collapsed
	if strings.Count(got, "search_knowledge") != 1 {
		t.Fatalf("expected search_knowledge once, got %q", got)
	}
	if strings.Contains(got, "get_device_status") {
		t.Fatalf("unexpected firmware rule without firmware tools: %q", got)
	}
}

func TestBuildCapabilityGroundingPolicyWithFirmwareTools(t *testing.T) {
	tools := []*schema.ToolInfo{
		{Name: "get_device_status"},
		{Name: "set_speaker_volume"},
	}
	got := BuildCapabilityGroundingPolicy(tools)
	if !strings.Contains(got, "get_device_status") {
		t.Fatalf("expected firmware get rule, got %q", got)
	}
	if !strings.Contains(got, "禁止猜测数值") {
		t.Fatalf("expected no-guess rule, got %q", got)
	}
	if !strings.Contains(got, "禁止再说失败") {
		t.Fatalf("expected success-confirm rule, got %q", got)
	}
}

func TestMaybeRewriteUngroundedActionClaim(t *testing.T) {
	cases := []struct {
		name         string
		text         string
		hadTools     bool
		wantRewrite  bool
		wantContains string
	}{
		{
			name:         "fake turn off",
			text:         "好的，已经帮你把灯关掉了。",
			wantRewrite:  true,
			wantContains: UngroundedActionFallback,
		},
		{
			name:         "fake volume",
			text:         "我已经帮你调高音量了。",
			wantRewrite:  true,
			wantContains: UngroundedActionFallback,
		},
		{
			name:         "fake power off",
			text:         "好的，已经帮你关机了。",
			wantRewrite:  true,
			wantContains: UngroundedActionFallback,
		},
		{
			name:         "fake sleep",
			text:         "我已经进入睡眠模式啦。",
			wantRewrite:  true,
			wantContains: UngroundedActionFallback,
		},
		{
			name:        "with tools keep",
			text:        "已经帮你把灯关掉了。",
			hadTools:    true,
			wantRewrite: false,
		},
		{
			name:        "normal chat",
			text:        "你今天想听什么故事呀？",
			wantRewrite: false,
		},
		{
			name:        "user self action not claim",
			text:        "如果你已经吃完饭，我们可以一起讲故事。",
			wantRewrite: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, rewritten := MaybeRewriteUngroundedActionClaim(tc.text, tc.hadTools)
			if rewritten != tc.wantRewrite {
				t.Fatalf("rewritten=%v want %v, got %q", rewritten, tc.wantRewrite, got)
			}
			if tc.wantContains != "" && got != tc.wantContains {
				t.Fatalf("got %q want %q", got, tc.wantContains)
			}
			if !tc.wantRewrite && got != tc.text {
				t.Fatalf("unexpected change: %q -> %q", tc.text, got)
			}
		})
	}
}

func TestLooksLikeUngroundedActionClaim(t *testing.T) {
	if !LooksLikeUngroundedActionClaim("操作已经完成啦") {
		t.Fatal("expected match for 操作已经完成")
	}
	if LooksLikeUngroundedActionClaim("明天天气不错") {
		t.Fatal("unexpected match")
	}
}
