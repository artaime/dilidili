package chat

import (
	"errors"
	"testing"
)

func TestIsTencentASRNoAudioTimeout(t *testing.T) {
	err := errors.New("tencent_asr error code=4008 message=客户端超过15秒未发送音频数据 voice_id=abc")
	if !isTencentASRNoAudioTimeout(err) {
		t.Fatal("expected tencent 4008 to match")
	}
	if isTencentASRNoAudioTimeout(errors.New("tencent_asr error code=5000 message=other")) {
		t.Fatal("expected unrelated error not to match")
	}
}
