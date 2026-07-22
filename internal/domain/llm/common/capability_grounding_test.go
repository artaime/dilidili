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
	if !strings.Contains(got, "禁止声称或邀请尝试查天气") {
		t.Fatalf("expected no-advertise rule, got %q", got)
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
	if !strings.Contains(got, "禁止主动列举、推销") {
		t.Fatalf("expected no-advertise rule, got %q", got)
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

func TestLooksLikeUngroundedCapabilityOffer(t *testing.T) {
	offer := "嘿嘿，我可以陪你聊天、讲故事、唱歌，还可以帮你查天气、定闹钟，甚至当你的小闹钟叫你起床哦！你想先试试哪一个呀？"
	if !LooksLikeUngroundedCapabilityOffer(offer, nil) {
		t.Fatal("expected ungrounded offer without tools")
	}
	if !LooksLikeUngroundedCapabilityOffer(offer, []*schema.ToolInfo{{Name: "search_knowledge"}}) {
		t.Fatal("expected ungrounded offer when tools lack weather/alarm")
	}
	if LooksLikeUngroundedCapabilityOffer(offer, []*schema.ToolInfo{
		{Name: "maps_weather", Desc: "天气查询"},
		{Name: "set_alarm", Desc: "设置闹钟"},
		{Name: "play_music"},
	}) {
		t.Fatal("should allow offer when tools cover weather/alarm/music")
	}
	if LooksLikeUngroundedCapabilityOffer("明天天气不错，我们出去玩吧。", nil) {
		t.Fatal("casual weather chat should not count as offer")
	}
	if LooksLikeUngroundedCapabilityOffer("嘿嘿，虽然我很想帮你查天气，可是我还不会联网查天气预报呢~", nil) {
		t.Fatal("refusal should not count as offer")
	}
	if LooksLikeUngroundedCapabilityOffer("我可以陪你聊天、讲故事呀。", nil) {
		t.Fatal("companion-only offer should pass")
	}
}

func TestMaybeRewriteUngroundedCapabilityOffer(t *testing.T) {
	text := "我还可以帮你查天气、定闹钟哦！你想先试试哪一个呀？"
	got, ok := MaybeRewriteUngroundedCapabilityOffer(text, nil)
	if !ok || got != UngroundedCapabilityOfferFallback {
		t.Fatalf("got %q ok=%v", got, ok)
	}
	keep := "你今天想听什么故事呀？"
	got, ok = MaybeRewriteUngroundedCapabilityOffer(keep, nil)
	if ok || got != keep {
		t.Fatalf("unexpected rewrite: %q ok=%v", got, ok)
	}
}
