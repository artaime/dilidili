package aliyun_cosyvoice

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"dili-esp32-server-golang/internal/data/audio"
	"dili-esp32-server-golang/internal/util"
	log "dili-esp32-server-golang/logger"

	"github.com/gopxl/beep"
	sse "github.com/tmaxmax/go-sse"
)

const (
	defaultModel       = "cosyvoice-v3-flash"
	defaultVoice       = "longanyang"
	defaultFormat      = "wav"
	defaultSampleRate  = 24000
	speechSynthesizer  = "/api/v1/services/audio/tts/SpeechSynthesizer"
	defaultBeijingHost = "cn-beijing.maas.aliyuncs.com"
)

var (
	httpClient     *http.Client
	httpClientOnce sync.Once
)

func getHTTPClient() *http.Client {
	httpClientOnce.Do(func() {
		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
		httpClient = &http.Client{
			Transport: transport,
			Timeout:   60 * time.Second,
		}
	})
	return httpClient
}

// AliyunCosyVoiceProvider 阿里云百炼 CosyVoice TTS 提供者
type AliyunCosyVoiceProvider struct {
	APIKey        string
	APIURL        string
	Model         string
	Voice         string
	Format        string
	SampleRate    int
	Instruction   string
	Stream        bool
	FrameDuration int
}

type cosyVoiceRequest struct {
	Model string              `json:"model"`
	Input cosyVoiceRequestInput `json:"input"`
}

type cosyVoiceRequestInput struct {
	Text         string `json:"text"`
	Voice        string `json:"voice"`
	Format       string `json:"format,omitempty"`
	SampleRate   int    `json:"sample_rate,omitempty"`
	Instruction  string `json:"instruction,omitempty"`
}

type cosyVoiceResponse struct {
	StatusCode int             `json:"status_code"`
	RequestID  string          `json:"request_id"`
	Code       string          `json:"code"`
	Message    string          `json:"message"`
	Output     cosyVoiceOutput `json:"output"`
}

type cosyVoiceOutput struct {
	FinishReason string          `json:"finish_reason"`
	Audio        cosyVoiceAudio  `json:"audio"`
}

type cosyVoiceAudio struct {
	Data      string `json:"data"`
	URL       string `json:"url"`
	ID        string `json:"id"`
	ExpiresAt int64  `json:"expires_at"`
}

// NewAliyunCosyVoiceProvider 创建百炼 CosyVoice TTS 提供者
func NewAliyunCosyVoiceProvider(config map[string]interface{}) *AliyunCosyVoiceProvider {
	apiKey, _ := config["api_key"].(string)
	apiURL, _ := config["api_url"].(string)
	workspaceID, _ := config["workspace_id"].(string)
	model, _ := config["model"].(string)
	voice, _ := config["voice"].(string)
	format, _ := config["format"].(string)
	instruction, _ := config["instruction"].(string)
	stream, _ := config["stream"].(bool)
	frameDuration, _ := config["frame_duration"].(float64)

	sampleRate := defaultSampleRate
	if v, ok := config["sample_rate"].(float64); ok && v > 0 {
		sampleRate = int(v)
	} else if v, ok := config["sample_rate"].(int); ok && v > 0 {
		sampleRate = v
	}

	if model == "" {
		model = defaultModel
	}
	if voice == "" {
		voice = defaultVoice
	}
	if format == "" {
		format = defaultFormat
	}
	if frameDuration == 0 {
		frameDuration = audio.FrameDuration
	}

	resolvedURL := strings.TrimSpace(apiURL)
	if resolvedURL == "" {
		resolvedURL = buildAPIURL(strings.TrimSpace(workspaceID))
	}

	return &AliyunCosyVoiceProvider{
		APIKey:        strings.TrimSpace(apiKey),
		APIURL:        resolvedURL,
		Model:         model,
		Voice:         voice,
		Format:        format,
		SampleRate:    sampleRate,
		Instruction:   strings.TrimSpace(instruction),
		Stream:        stream,
		FrameDuration: int(frameDuration),
	}
}

func buildAPIURL(workspaceID string) string {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return ""
	}
	return fmt.Sprintf("https://%s.%s%s", workspaceID, defaultBeijingHost, speechSynthesizer)
}

func (p *AliyunCosyVoiceProvider) validate() error {
	if p == nil {
		return fmt.Errorf("百炼 CosyVoice provider 未初始化")
	}
	if p.APIKey == "" {
		return fmt.Errorf("百炼 CosyVoice api_key 未配置")
	}
	if p.APIURL == "" {
		return fmt.Errorf("百炼 CosyVoice 需配置 api_url 或 workspace_id")
	}
	if strings.TrimSpace(p.Model) == "" {
		return fmt.Errorf("百炼 CosyVoice model 未配置")
	}
	if strings.TrimSpace(p.Voice) == "" {
		return fmt.Errorf("百炼 CosyVoice voice 未配置")
	}
	return nil
}

func (p *AliyunCosyVoiceProvider) buildRequestBody(text string) ([]byte, error) {
	input := cosyVoiceRequestInput{
		Text:       text,
		Voice:      p.Voice,
		Format:     p.Format,
		SampleRate: p.SampleRate,
	}
	if p.Instruction != "" {
		input.Instruction = p.Instruction
	}
	reqBody := cosyVoiceRequest{
		Model: p.Model,
		Input: input,
	}
	return json.Marshal(reqBody)
}

func (p *AliyunCosyVoiceProvider) newHTTPRequest(ctx context.Context, text string, stream bool) (*http.Request, error) {
	jsonData, err := p.buildRequestBody(text)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.APIKey))
	if stream {
		req.Header.Set("X-DashScope-SSE", "enable")
	}
	return req, nil
}

// TextToSpeech 非流式文本转语音
func (p *AliyunCosyVoiceProvider) TextToSpeech(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) ([][]byte, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}
	startTs := time.Now().UnixMilli()

	req, err := p.newHTTPRequest(ctx, text, false)
	if err != nil {
		return nil, err
	}

	client := getHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("百炼 CosyVoice API 请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var ttsResp cosyVoiceResponse
	if err := json.Unmarshal(body, &ttsResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, 响应体: %s", err, string(body))
	}
	if ttsResp.StatusCode != 0 && ttsResp.StatusCode != 200 {
		return nil, fmt.Errorf("百炼 CosyVoice API 错误 [%s]: %s", ttsResp.Code, ttsResp.Message)
	}
	if code := strings.TrimSpace(ttsResp.Code); code != "" && code != "Success" {
		return nil, fmt.Errorf("百炼 CosyVoice API 错误 [%s]: %s", ttsResp.Code, ttsResp.Message)
	}
	if ttsResp.Output.Audio.URL == "" {
		return nil, fmt.Errorf("响应中未包含音频 URL")
	}

	wavReq, err := http.NewRequestWithContext(ctx, http.MethodGet, ttsResp.Output.Audio.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建音频下载请求失败: %v", err)
	}
	wavResp, err := client.Do(wavReq)
	if err != nil {
		return nil, fmt.Errorf("下载音频失败: %v", err)
	}
	defer wavResp.Body.Close()
	if wavResp.StatusCode != http.StatusOK {
		dlBody, _ := io.ReadAll(wavResp.Body)
		return nil, fmt.Errorf("下载音频失败，状态码: %d, 响应: %s", wavResp.StatusCode, string(dlBody))
	}

	decodeFormat := p.Format
	if decodeFormat == "" {
		decodeFormat = "wav"
	}
	outputChan := make(chan []byte, 1000)
	decoder, err := util.CreateAudioDecoderWithSampleRate(ctx, wavResp.Body, outputChan, frameDuration, decodeFormat, sampleRate)
	if err != nil {
		return nil, fmt.Errorf("创建音频解码器失败: %v", err)
	}
	go func() {
		if err := decoder.Run(startTs); err != nil {
			log.Errorf("百炼 CosyVoice 非流式音频解码失败: %v", err)
		}
	}()

	var frames [][]byte
	for frame := range outputChan {
		frames = append(frames, frame)
	}
	if len(frames) == 0 {
		return nil, fmt.Errorf("百炼 CosyVoice 未解码出有效音频帧")
	}
	return frames, nil
}

// TextToSpeechStream 流式文本转语音
func (p *AliyunCosyVoiceProvider) TextToSpeechStream(ctx context.Context, text string, sampleRate int, channels int, frameDuration int) (outputChan chan []byte, err error) {
	if err := p.validate(); err != nil {
		return nil, err
	}

	startTs := time.Now().UnixMilli()
	req, err := p.newHTTPRequest(ctx, text, true)
	if err != nil {
		return nil, err
	}

	client := getHTTPClient()
	outputChan = make(chan []byte, 100)

	go func() {
		resp, err := client.Do(req)
		if err != nil {
			log.Errorf("发送百炼 CosyVoice 流式请求失败: %v", err)
			close(outputChan)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			log.Errorf("百炼 CosyVoice 流式 API 请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
			close(outputChan)
			return
		}

		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "text/event-stream") {
			body, _ := io.ReadAll(resp.Body)
			log.Errorf("百炼 CosyVoice 流式 API 返回非 SSE，Content-Type: %s, 响应: %s", contentType, previewString(string(body), 500))
			close(outputChan)
			return
		}

		pipeReader, pipeWriter := io.Pipe()
		go func() {
			defer func() {
				if err := pipeWriter.Close(); err != nil {
					log.Debugf("关闭百炼 CosyVoice 管道写入端失败: %v", err)
				}
			}()
			if err := p.parseEventStream(ctx, resp.Body, pipeWriter, text); err != nil {
				log.Errorf("解析百炼 CosyVoice Event Stream 失败: %v", err)
			}
		}()

		decodeFormat := "pcm"
		if strings.EqualFold(p.Format, "wav") {
			decodeFormat = "pcm"
		} else if p.Format != "" {
			decodeFormat = p.Format
		}

		decoder, err := util.CreateAudioDecoderWithSampleRate(ctx, pipeReader, outputChan, frameDuration, decodeFormat, sampleRate)
		if err != nil {
			log.Errorf("创建百炼 CosyVoice 流式音频解码器失败: %v", err)
			close(outputChan)
			pipeReader.Close()
			return
		}

		decoder.WithFormat(beep.Format{
			SampleRate:  beep.SampleRate(p.SampleRate),
			NumChannels: 1,
		})

		if err := decoder.Run(startTs); err != nil {
			log.Errorf("百炼 CosyVoice 流式音频解码失败: %v", err)
			return
		}

		select {
		case <-ctx.Done():
			log.Debugf("百炼 CosyVoice 流式合成取消, 文本: %s", text)
		default:
			log.Debugf("百炼 CosyVoice 流式耗时: %d ms", time.Now().UnixMilli()-startTs)
		}
	}()

	return outputChan, nil
}

func (p *AliyunCosyVoiceProvider) parseEventStream(ctx context.Context, reader io.Reader, writer *io.PipeWriter, text string) error {
	var leadingAudio bytes.Buffer
	wroteLeadingAudio := false

	for ev, evErr := range sse.Read(reader, nil) {
		if evErr != nil {
			return fmt.Errorf("读取 SSE 事件失败: %w", evErr)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		dataValue := strings.TrimSpace(ev.Data)
		if dataValue == "" {
			continue
		}

		var eventResp cosyVoiceResponse
		if err := json.Unmarshal([]byte(dataValue), &eventResp); err != nil {
			log.Warnf("解析百炼 CosyVoice Event Stream JSON 失败: %v, 数据: %s", err, previewString(dataValue, 200))
			continue
		}

		if eventResp.StatusCode != 0 && eventResp.StatusCode != 200 {
			return fmt.Errorf("百炼 CosyVoice 流式 API 错误 [%s]: %s", eventResp.Code, eventResp.Message)
		}
		if code := strings.TrimSpace(eventResp.Code); code != "" && code != "Success" {
			return fmt.Errorf("百炼 CosyVoice 流式 API 错误 [%s]: %s", eventResp.Code, eventResp.Message)
		}

		if eventResp.Output.Audio.URL != "" {
			continue
		}

		if eventResp.Output.Audio.Data != "" {
			encoded := cleanBase64(eventResp.Output.Audio.Data)
			audioBytes, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				log.Errorf("解码百炼 CosyVoice Base64 音频失败: %v", err)
				continue
			}
			if len(audioBytes) == 0 {
				continue
			}

			if !wroteLeadingAudio && strings.EqualFold(p.Format, "wav") {
				leadingAudio.Write(audioBytes)
				normalized, needMore, detectedWAV, err := normalizeLeadingAudio(leadingAudio.Bytes())
				if err != nil {
					return fmt.Errorf("解析流式音频头失败: %w", err)
				}
				if needMore {
					continue
				}
				wroteLeadingAudio = true
				if detectedWAV {
					log.Infof("百炼 CosyVoice 流式音频检测到 WAV 头，已剥离后按 PCM 处理")
				}
				if len(normalized) == 0 {
					continue
				}
				if _, err := writer.Write(normalized); err != nil {
					return fmt.Errorf("写入 PCM 到管道失败: %v", err)
				}
				continue
			}

			if _, err := writer.Write(audioBytes); err != nil {
				return fmt.Errorf("写入音频到管道失败: %v", err)
			}
		}

		if eventResp.Output.FinishReason == "stop" {
			return nil
		}
	}

	return nil
}

func normalizeLeadingAudio(data []byte) (normalized []byte, needMore bool, detectedWAV bool, err error) {
	if len(data) < 12 {
		return nil, true, false, nil
	}
	if !bytes.HasPrefix(data, []byte("RIFF")) || !bytes.Equal(data[8:12], []byte("WAVE")) {
		return data, false, false, nil
	}
	offset, needMore, err := wavDataOffset(data)
	if err != nil {
		return nil, false, true, err
	}
	if needMore {
		return nil, true, true, nil
	}
	if offset > len(data) {
		return nil, false, true, fmt.Errorf("WAV data offset 越界: %d > %d", offset, len(data))
	}
	return data[offset:], false, true, nil
}

func wavDataOffset(data []byte) (offset int, needMore bool, err error) {
	if len(data) < 12 {
		return 0, true, nil
	}
	if !bytes.HasPrefix(data, []byte("RIFF")) || !bytes.Equal(data[8:12], []byte("WAVE")) {
		return 0, false, fmt.Errorf("不是有效的 WAV 头")
	}

	offset = 12
	for {
		if len(data) < offset+8 {
			return 0, true, nil
		}
		chunkID := string(data[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		if chunkSize < 0 {
			return 0, false, fmt.Errorf("非法 WAV chunk size: %d", chunkSize)
		}
		offset += 8
		if chunkID == "data" {
			return offset, false, nil
		}
		nextOffset := offset + chunkSize
		if chunkSize%2 == 1 {
			nextOffset++
		}
		if len(data) < nextOffset {
			return 0, true, nil
		}
		offset = nextOffset
	}
}

func (p *AliyunCosyVoiceProvider) SetVoice(voiceConfig map[string]interface{}) error {
	if voice, ok := voiceConfig["voice"].(string); ok && voice != "" {
		p.Voice = voice
		return nil
	}
	return fmt.Errorf("无效的音色配置: 缺少 voice")
}

func (p *AliyunCosyVoiceProvider) Close() error {
	return nil
}

func (p *AliyunCosyVoiceProvider) IsValid() bool {
	return p != nil
}

func cleanBase64(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == ' ' || ch == '\n' || ch == '\r' || ch == '\t' {
			continue
		}
		b.WriteByte(ch)
	}
	return b.String()
}

func previewString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
