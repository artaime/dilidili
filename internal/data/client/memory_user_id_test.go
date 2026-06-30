package client

import "testing"

func TestGetMemoryUserIDPrefersDeviceID(t *testing.T) {
	state := &ClientState{
		DeviceID: "SN-001",
		AgentID:  "1",
	}
	if got := state.GetMemoryUserID(); got != "SN-001" {
		t.Fatalf("GetMemoryUserID() = %q, want SN-001", got)
	}
}

func TestGetMemoryUserIDFallsBackToAgentID(t *testing.T) {
	state := &ClientState{
		AgentID: "1",
	}
	if got := state.GetMemoryUserID(); got != "1" {
		t.Fatalf("GetMemoryUserID() = %q, want 1", got)
	}
}

func TestGetDeviceIDOrAgentIDStillPrefersAgentID(t *testing.T) {
	state := &ClientState{
		DeviceID: "SN-001",
		AgentID:  "1",
	}
	if got := state.GetDeviceIDOrAgentID(); got != "1" {
		t.Fatalf("GetDeviceIDOrAgentID() = %q, want 1", got)
	}
}
