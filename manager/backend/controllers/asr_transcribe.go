package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type asrTranscribeResponse struct {
	Text  string `json:"text"`
	Error string `json:"error"`
}

func transcribeAudioFile(serviceURL, filePath string) (string, error) {
	serviceURL = strings.TrimRight(strings.TrimSpace(serviceURL), "/")
	if serviceURL == "" {
		return "", fmt.Errorf("ASR 服务未配置")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开音频文件失败")
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("audio", filepath.Base(filePath))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", err
	}
	_ = writer.WriteField("format", strings.TrimPrefix(filepath.Ext(filePath), "."))
	if err := writer.Close(); err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodPost, serviceURL+"/asr/transcribe", body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ASR 服务请求失败")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取 ASR 响应失败")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ASR 转写失败")
	}

	var result asrTranscribeResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("解析 ASR 响应失败")
	}
	if result.Error != "" {
		return "", fmt.Errorf("%s", result.Error)
	}
	text := strings.TrimSpace(result.Text)
	if text == "" {
		return "", fmt.Errorf("ASR 未识别到有效文本")
	}
	return text, nil
}
