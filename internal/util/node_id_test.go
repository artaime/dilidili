package util

import (
	"os"
	"testing"

	"github.com/spf13/viper"
)

func TestGetNodeIDFromEnv(t *testing.T) {
	viper.Reset()
	t.Setenv("DILI_NODE_ID", "env-node-01")
	if got := GetNodeID(); got != "env-node-01" {
		t.Fatalf("expected env node id, got %s", got)
	}
}

func TestGetNodeIDFromConfig(t *testing.T) {
	viper.Reset()
	os.Unsetenv("DILI_NODE_ID")
	viper.Set("server.node_id", "cfg-node-01")
	if got := GetNodeID(); got != "cfg-node-01" {
		t.Fatalf("expected config node id, got %s", got)
	}
}

func TestGetNodeNameFallbackToNodeID(t *testing.T) {
	viper.Reset()
	os.Unsetenv("DILI_NODE_ID")
	viper.Set("server.node_id", "cfg-node-01")
	if got := GetNodeName(); got != "cfg-node-01" {
		t.Fatalf("expected node name fallback, got %s", got)
	}
}
