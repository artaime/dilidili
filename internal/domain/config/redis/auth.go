package redis_config

import (
	"context"
	"dili-esp32-server-golang/internal/domain/config/types"

	"github.com/google/uuid"
)

type activationInfo struct {
	challenge string
	msg       string
}

var verfiyDeviceId = map[string]bool{}
var preActivationInfo = map[string]activationInfo{}

// 设备是否激活?
func (r *UserConfig) IsDeviceActivated(ctx context.Context, deviceId string, clientId string) (bool, error) {
	if _, ok := verfiyDeviceId[deviceId]; ok {
		return true, nil
	}
	return false, nil
}

// 获取激活需要的信息,  code, challenge, msg, timeoutMs（code 已废弃，恒为空）
func (r *UserConfig) GetActivationInfo(ctx context.Context, deviceId string, clientId string) (string, string, string, int) {
	if info, ok := preActivationInfo[deviceId]; ok {
		return "", info.challenge, info.msg, 300
	}
	challenge := uuid.New().String()
	msg := "请在小程序上绑定我，双击电源键进入配网模式"
	preActivationInfo[deviceId] = activationInfo{
		challenge: challenge,
		msg:       msg,
	}
	return "", challenge, msg, 300
}

// 验证 challenge和HMAC是否匹配, 设备是否已激活，此处可以省略hmac的校验, 只查询deviceId是否绑定
func (r *UserConfig) VerifyChallenge(ctx context.Context, deviceId string, clientId string, activationPayload types.ActivationPayload) (bool, error) {
	if _, ok := verfiyDeviceId[deviceId]; ok {
		return true, nil
	}
	if info, ok := preActivationInfo[deviceId]; ok {
		if info.challenge == activationPayload.Challenge {
			verfiyDeviceId[deviceId] = true
			delete(preActivationInfo, deviceId)
			return true, nil
		}
	}
	return false, nil
}
