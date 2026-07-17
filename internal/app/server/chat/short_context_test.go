package chat

import (
	"testing"

	"github.com/spf13/viper"
)

func TestHasValidShortContextIdentity(t *testing.T) {
	if hasValidShortContextIdentity(0, "SN", "1") {
		t.Fatal("user_id=0 should be invalid")
	}
	if hasValidShortContextIdentity(1, "", "1") {
		t.Fatal("empty device should be invalid")
	}
	if hasValidShortContextIdentity(1, "SN", "0") {
		t.Fatal("agent_id=0 should be invalid")
	}
	if hasValidShortContextIdentity(1, "SN", "") {
		t.Fatal("empty agent should be invalid")
	}
	if !hasValidShortContextIdentity(9, "SN", "12") {
		t.Fatal("valid triple should pass")
	}
}

func TestShortContextDefaults(t *testing.T) {
	viper.Reset()
	if !shortContextEnabled() {
		t.Fatal("short_context should default enabled")
	}
	if got := shortContextRecentMessageLimit(); got != 10 {
		t.Fatalf("recent limit=%d want 10", got)
	}
	if got := shortContextLoadLimit(); got != 20 {
		t.Fatalf("load limit=%d want 20", got)
	}
	if !shortContextReuseSessionOnFreshHello() {
		t.Fatal("reuse_session_on_fresh_hello should default true")
	}
}
