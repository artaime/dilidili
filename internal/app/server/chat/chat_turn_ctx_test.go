package chat

import (
	"context"
	"errors"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	client "dili-esp32-server-golang/internal/data/client"
	"dili-esp32-server-golang/internal/util"
)

func TestAddAsrResultToQueueTurnSurvivesAfterAsrCancel(t *testing.T) {
	session, clientState := newAbortTestSession("realtime")

	require.NoError(t, session.AddAsrResultToQueue("你从哪里来？", nil))
	item, err := session.chatTextQueue.Pop(context.Background(), 0)
	require.NoError(t, err)
	require.NoError(t, item.ctx.Err())
	require.True(t, session.hasActiveChatTurn())

	clientState.AfterAsrSessionCtx.CancelWithReason("test: simulate idle ASR first-text AfterAsr cancel")
	require.NoError(t, item.ctx.Err(), "本轮对话 ctx 不应随 AfterAsrSessionCtx 被误杀")

	session.StopAssistantOutputAfterAsrWithReason(false, "test: explicit barge-in")
	require.ErrorIs(t, item.ctx.Err(), context.Canceled)
	require.False(t, session.hasActiveChatTurn())
}

func TestTryRecoverStuckVoiceCaptureSkipsActiveChatTurn(t *testing.T) {
	session, clientState := newAbortTestSession("auto")
	clientState.SetClientVoiceStop(true)
	clientState.SetStatus(client.ClientStatusListenStop)
	session.asrManager = NewASRManager(clientState, nil)

	require.NoError(t, session.AddAsrResultToQueue("将音量设置到50。", nil))
	require.True(t, session.hasActiveChatTurn())
	require.False(t, session.TryRecoverStuckVoiceCapture(), "对话处理中不应自动恢复拾音")
}

func TestTryRecoverStuckVoiceCaptureClearsSoftVoiceStop(t *testing.T) {
	session, clientState := newAbortTestSession("auto")
	clientState.SetClientVoiceStop(true)
	clientState.SetStatus(client.ClientStatusListening)
	clientState.SetListenPhase(client.ListenPhaseListening)
	clientState.Asr.AsrAudioChannel = make(chan []float32, 1)
	session.asrManager = NewASRManager(clientState, nil)
	session.asrManager.recognitionLoopActive.Store(true)

	require.True(t, session.TryRecoverStuckVoiceCapture())
	require.False(t, clientState.GetClientVoiceStop())
	require.NotNil(t, clientState.Asr.AsrAudioChannel, "soft recover must not close ASR channel")
}

func TestIsBenignAsrDisconnectError(t *testing.T) {
	require.True(t, isBenignAsrDisconnectError(errors.New("aliyun funasr task failed: EmptyAudio")))
	require.False(t, isBenignAsrDisconnectError(errors.New("unrelated failure")))
}

func TestRealtimeMode4IdleFirstTextDoesNotCancelChatTurn(t *testing.T) {
	viper.Set("chat.realtime_mode", 4)
	t.Cleanup(func() { viper.Set("chat.realtime_mode", 0) })

	clientState := &client.ClientState{
		Ctx:         context.Background(),
		ListenMode:  "realtime",
		ListenPhase: client.ListenPhaseListening,
		DeviceID:    "device-mode4",
		Status:      client.ClientStatusListening,
	}
	// NewChatSession 会挂上生产路径的 OnAsrFirstTextCallback
	session := NewChatSession(clientState, NewServerTransport(nil, clientState), nil, nil)
	session.chatTextQueue = util.NewQueue[AsrResponseChannelItem](2)
	if session.ttsManager == nil {
		session.ttsManager = NewTTSManager(clientState, nil, nil)
	}
	if session.llmManager == nil {
		session.llmManager = NewLLMManager(clientState, nil, session.ttsManager, nil, nil)
	}

	require.NotNil(t, clientState.OnAsrFirstTextCallback)
	require.NoError(t, session.AddAsrResultToQueue("你从哪里来？", nil))
	item, err := session.chatTextQueue.Pop(context.Background(), 0)
	require.NoError(t, err)
	require.NoError(t, item.ctx.Err())

	// 空闲态触发 mode=4 首字回调：不得取消本轮 chat turn
	clientState.OnAsrFirstTextCallback("回声", false)
	require.NoError(t, item.ctx.Err())
}
