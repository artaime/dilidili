package chat

import (
	"errors"
	"testing"

	data_client "dili-esp32-server-golang/internal/data/client"
)

func TestShouldIgnoreAsrLoopFatalErrorDuringAssistantOutput(t *testing.T) {
	state := &data_client.ClientState{
		Status: data_client.ClientStatusTTSStart,
	}
	state.SetTtsStart(true)

	err := errors.New("ASR短时间内连续返回空结果(3次/3s)，触发保护并断开连接")
	if !shouldIgnoreAsrLoopFatalErrorDuringAssistantOutput(state, err) {
		t.Fatal("expected empty-result storm to be ignored during assistant output")
	}

	state.SetTtsStart(false)
	state.SetStatus(data_client.ClientStatusListening)
	if shouldIgnoreAsrLoopFatalErrorDuringAssistantOutput(state, err) {
		t.Fatal("expected empty-result storm to remain fatal while assistant is idle")
	}
}

func TestShouldDeferAsrRecoveryDuringAssistantOutput(t *testing.T) {
	manager := &ASRManager{}
	state := &data_client.ClientState{Status: data_client.ClientStatusTTSStart}
	state.SetTtsStart(true)

	if !manager.shouldDeferAsrRecoveryDuringAssistantOutput(state) {
		t.Fatal("expected defer during TTS output")
	}

	state.SetTtsStart(false)
	state.SetStatus(data_client.ClientStatusListening)
	if manager.shouldDeferAsrRecoveryDuringAssistantOutput(state) {
		t.Fatal("expected no defer while listening without assistant output")
	}
}
