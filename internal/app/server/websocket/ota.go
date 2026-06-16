package websocket

import (
	"dili-esp32-server-golang/internal/data/client"
	user_config "dili-esp32-server-golang/internal/domain/config"
	ctypes "dili-esp32-server-golang/internal/domain/config/types"
	"dili-esp32-server-golang/internal/ota/aitoy"
	"dili-esp32-server-golang/internal/util"
	log "dili-esp32-server-golang/logger"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type ActivationRequest struct {
	Payload ctypes.ActivationPayload `json:"Payload"`
}

func (s *WebSocketServer) handleOta(w http.ResponseWriter, r *http.Request) {
	//获取客户端ip
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}

	//从header头部获取Device-Id和Client-Id
	headerDeviceId := r.Header.Get("Device-Id")
	clientId := r.Header.Get("Client-Id")

	if headerDeviceId == "" || clientId == "" {
		log.Errorf("缺少Device-Id或Client-Id")
		http.Error(w, "缺少Device-Id或Client-Id", http.StatusBadRequest)
		return
	}

	var otaReq OtaRequest
	if body, err := io.ReadAll(r.Body); err != nil {
		log.Errorf("读取OTA请求体失败: %v", err)
		http.Error(w, "读取请求体失败", http.StatusBadRequest)
		return
	} else if len(body) > 0 {
		if err := json.Unmarshal(body, &otaReq); err != nil {
			log.Warnf("OTA请求体解析失败(将按默认设备处理): %v", err)
		}
	}

	aitoyDevice := isAIToyOTARequest(&otaReq, r.Header.Get("User-Agent"))
	if aitoyDevice && strings.TrimSpace(otaReq.Board.Sn) == "" {
		log.Errorf("AIToy OTA 缺少 board.sn")
		http.Error(w, "缺少 board.sn", http.StatusBadRequest)
		return
	}

	deviceId := resolveOTADeviceID(headerDeviceId, &otaReq)
	if deviceId != headerDeviceId {
		log.Infof("OTA 设备标识: header=%s resolved=%s", headerDeviceId, deviceId)
	}
	if deviceId == "" {
		log.Errorf("OTA 缺少设备 SN")
		http.Error(w, "缺少设备 SN", http.StatusBadRequest)
		return
	}

	if aitoyDevice {
		log.Infof("OTA AIToy 设备: board=%s device=%s", otaReq.Board.Type, deviceId)
		log.DeviceInfof(log.DeviceTagOTA, deviceId,
			"AIToy OTA 请求 board=%s ua=%s ip=%s header=%s", otaReq.Board.Type, r.Header.Get("User-Agent"), ip, headerDeviceId)
	}

	//根据ip选择不同的配置
	clientIp := r.Header.Get("X-Real-IP")
	if clientIp == "" {
		clientIp = r.Header.Get("X-Forwarded-For")
	}
	if clientIp == "" {
		clientIp = r.RemoteAddr
	}

	var activationInfo *ActivationInfo
	authEnable := viper.GetBool("auth.enable")
	log.Debugf("authEnable: %v", authEnable)
	if authEnable {
		configProvider, err := user_config.GetProvider(viper.GetString("config_provider.type"))
		//检查此deviceId是否已认证
		isActivited, err := configProvider.IsDeviceActivated(r.Context(), deviceId, clientId)
		if err != nil {
			log.Errorf("检查设备是否认证失败: %v", err)
			http.Error(w, "内部服务器错误", http.StatusInternalServerError)
			return
		}
		if !isActivited {
			code, challenge, msg, timeoutMs := configProvider.GetActivationInfo(r.Context(), deviceId, clientId)
			activationInfo = &ActivationInfo{
				Code:      code,
				Message:   msg,
				Challenge: challenge,
				TimeoutMs: timeoutMs,
			}
			log.Infof("激活信息: &{Code:%s Message:%s Challenge:%s TimeoutMs:%d}", code, msg, challenge, timeoutMs)
		}
	}

	otaConfigPrefix := "ota.external."
	//如果ip是192.168开头的，则选择test配置
	if strings.HasPrefix(clientIp, "192.168") || strings.HasPrefix(clientIp, "10.") || strings.HasPrefix(clientIp, "127.0.0.1") {
		otaConfigPrefix = "ota.test."
	} else {
		otaConfigPrefix = "ota.external."
	}

	mqttInfo := getMqttInfo(deviceId, clientId, otaConfigPrefix, ip)
	serverTime := ServerTimeInfo{
		Timestamp:      time.Now().UnixMilli(),
		TimezoneOffset: 480,
	}
	var firmware FirmwareInfo
	if aitoyDevice {
		serverTime.TimeZone = "Asia/Shanghai"
		firmware = buildAIToyFirmware(&otaReq)
	} else {
		firmware = FirmwareInfo{
			Version: "0.9.9",
			Url:     "",
		}
	}

	respData := &OtaResponse{
		Websocket: WebsocketInfo{
			Url:   viper.GetString(otaConfigPrefix + "websocket.url"),
			Token: viper.GetString(otaConfigPrefix + "websocket.token"),
		},
		Mqtt:       mqttInfo,
		ServerTime: serverTime,
		Activation: activationInfo,
		Firmware:   firmware,
	}
	if aitoyDevice {
		respData.Pd = &PdInfo{Voice: aitoy.PdVoiceDefault}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(respData); err != nil {
		log.Errorf("OTA响应序列化失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}
	if respData.Mqtt != nil {
		log.DeviceInfof(log.DeviceTagOTA, deviceId,
			"OTA 响应 mqtt=%s pub=%s sub=%s ws=%s firmware=%s pd_voice=%v",
			respData.Mqtt.Endpoint, respData.Mqtt.PublishTopic, respData.Mqtt.SubscribeTopic,
			respData.Websocket.Url, respData.Firmware.Version, respData.Pd != nil)
	} else {
		log.DeviceInfof(log.DeviceTagOTA, deviceId, "OTA 响应 ws=%s (无MQTT)", respData.Websocket.Url)
	}
	return
}

func getMqttInfo(deviceId, clientId, otaConfigPrefix, ip string) *MqttInfo {
	if !viper.GetBool(otaConfigPrefix + "mqtt.enable") {
		return nil
	}

	// 生成MQTT凭据
	signatureKey := viper.GetString("ota.signature_key")
	credentials, err := util.GenerateMqttCredentials(deviceId, clientId, ip, signatureKey)
	if err != nil {
		log.Errorf("生成MQTT凭据失败: %v", err)
		return nil
	}

	deviceTopic := strings.ReplaceAll(deviceId, ":", "_")
	// 协议文档为 devices/p2p/{deviceId}；本服务下行主题为 /p2p/device_sub/{deviceId}（连接时亦会自动订阅）
	subscribeTopic := fmt.Sprintf("%s%s", client.DeviceSubTopicPrefix, deviceTopic)

	return &MqttInfo{
		Endpoint:       viper.GetString(otaConfigPrefix + "mqtt.endpoint"),
		ClientId:       credentials.ClientId,
		Username:       credentials.Username,
		Password:       credentials.Password,
		PublishTopic:   client.DeviceMockPubTopicPrefix,
		SubscribeTopic: subscribeTopic,
	}
}

// handleOtaActivate 设备激活接口
func (s *WebSocketServer) handleOtaActivate(w http.ResponseWriter, r *http.Request) {
	headerDeviceId := r.Header.Get("Device-Id")
	clientId := r.Header.Get("Client-Id")
	if headerDeviceId == "" || clientId == "" {
		log.Errorf("缺少Device-Id或Client-Id")
		http.Error(w, "缺少Device-Id或Client-Id", http.StatusBadRequest)
		return
	}
	var req ActivationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Errorf("激活请求解析失败: %v", err)
		http.Error(w, "请求体解析失败", http.StatusBadRequest)
		return
	}
	deviceId := resolveActivateDeviceID(headerDeviceId, req.Payload.SerialNumber)
	if deviceId != headerDeviceId {
		log.Infof("OTA 激活设备标识: header=%s resolved=%s", headerDeviceId, deviceId)
	}
	// 校验算法
	if req.Payload.Algorithm != "hmac-sha256" {
		http.Error(w, "不支持的算法", http.StatusBadRequest)
		return
	}

	// 调用配置Provider进行绑定校验
	configProvider, err := user_config.GetProvider(viper.GetString("config_provider.type"))
	if err != nil {
		log.Errorf("获取配置Provider失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}
	ok, err := configProvider.VerifyChallenge(r.Context(), deviceId, clientId, req.Payload)
	if err != nil {
		log.Errorf("设备激活校验失败: %v", err)
		http.Error(w, "设备激活校验失败", http.StatusInternalServerError)
		return
	}
	if !ok {
		log.Warnf("设备激活校验未通过: deviceId=%s, clientId=%s", deviceId, clientId)
		http.Error(w, "设备激活校验未通过", http.StatusAccepted)
		return
	}
	// 激活成功，返回200
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("激活成功"))
}
