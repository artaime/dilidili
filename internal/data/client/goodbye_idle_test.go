package client

import (
	"testing"
	"time"
)

func TestGoodbyeIdleArmedEnablesAudioIdleClockForManualMode(t *testing.T) {
	state := &ClientState{ListenMode: "manual"}
	if state.UsesAudioIdleClock() {
		t.Fatal("expected manual mode without goodbye idle to skip audio idle clock")
	}

	state.ArmGoodbyeIdleWindow(time.Now())
	if !state.UsesAudioIdleClock() {
		t.Fatal("expected armed goodbye idle to enable audio idle clock in manual mode")
	}
	if !state.AudioIdleStarted() {
		t.Fatal("expected goodbye idle arming to start audio idle window")
	}

	state.DisarmGoodbyeIdleWindow()
	if state.UsesAudioIdleClock() {
		t.Fatal("expected disarmed goodbye idle to restore manual mode behavior")
	}
}

func TestResetGoodbyeIdleWindowRestartsTimer(t *testing.T) {
	state := &ClientState{ListenMode: "manual"}
	state.ArmGoodbyeIdleWindow(time.Now().Add(-5 * time.Second))
	state.ResetGoodbyeIdleWindow(time.Now())

	if elapsed := state.GetAudioIdleElapsed(time.Now()); elapsed > time.Second {
		t.Fatalf("expected reset to restart idle window, got elapsed=%s", elapsed)
	}
}

func TestNoteUplinkActivityResetsNormalAudioIdle(t *testing.T) {
	state := &ClientState{ListenMode: "auto"}
	state.StartAudioIdleWindow(time.Now().Add(-10 * time.Second))
	state.NoteUplinkActivity(time.Now())

	if elapsed := state.GetAudioIdleElapsed(time.Now()); elapsed > time.Second {
		t.Fatalf("expected uplink activity to restart idle window, got elapsed=%s", elapsed)
	}
}
