package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"dili/manager/backend/config"
	"dili/manager/backend/database"
	"dili/manager/backend/models"
	"dili/manager/backend/privacy"
)

func main() {
	var configFile string
	var dryRun bool
	flag.StringVar(&configFile, "config", "config/config.json", "配置文件路径")
	flag.StringVar(&configFile, "c", "config/config.json", "配置文件路径 (简写)")
	flag.BoolVar(&dryRun, "dry-run", false, "仅统计，不写回")
	flag.Parse()

	cfg := config.LoadWithPath(configFile)
	db := database.Init(cfg.Database)
	if db == nil {
		log.Fatal("数据库初始化失败")
	}
	defer database.Close(db)

	kek, err := privacy.LoadKEKFromEnv()
	if err != nil {
		log.Fatalf("加载 KEK 失败: %v", err)
	}
	svc, err := privacy.NewService(db, privacy.Config{
		Enabled: true,
		KeyID:   cfg.Encryption.KeyID,
		KEK:     kek,
	})
	if err != nil {
		log.Fatal(err)
	}

	chatAudioBase := cfg.History.AudioBasePath
	if chatAudioBase == "" {
		chatAudioBase = "./data/chat_history/audio"
	}
	parentAudioBase := cfg.ParentMessage.AudioBasePath
	if parentAudioBase == "" {
		parentAudioBase = "./data/parent_messages/audio"
	}

	var chatMsgs []models.ChatMessage
	if err := db.Where("is_deleted = ?", false).Find(&chatMsgs).Error; err != nil {
		log.Fatalf("查询 chat_messages 失败: %v", err)
	}
	chatTextN, chatAudioN := 0, 0
	deviceCache := map[string]uint{}
	for _, msg := range chatMsgs {
		deviceDBID, ok := deviceCache[msg.DeviceID]
		if !ok {
			var device models.Device
			if err := db.Select("id").Where("device_name = ?", msg.DeviceID).First(&device).Error; err != nil {
				log.Printf("跳过消息 %s：设备 %s 不存在", msg.MessageID, msg.DeviceID)
				continue
			}
			deviceDBID = device.ID
			deviceCache[msg.DeviceID] = deviceDBID
		}
		if msg.Content != "" && !privacy.IsCiphertext(msg.Content) {
			enc, err := svc.EncryptExistingText(deviceDBID, msg.Content)
			if err != nil {
				log.Printf("加密 chat content 失败 id=%d: %v", msg.ID, err)
			} else if !dryRun {
				if err := db.Model(&models.ChatMessage{}).Where("id = ?", msg.ID).Update("content", enc).Error; err != nil {
					log.Printf("写回 chat content 失败 id=%d: %v", msg.ID, err)
				} else {
					chatTextN++
				}
			} else {
				chatTextN++
			}
		}
		if strings.TrimSpace(msg.AudioPath) == "" {
			continue
		}
		full := filepath.Join(chatAudioBase, msg.AudioPath)
		n, err := migrateAudioFile(svc, deviceDBID, full, dryRun)
		if err != nil {
			log.Printf("加密 chat 音频失败 id=%d path=%s: %v", msg.ID, full, err)
			continue
		}
		chatAudioN += n
	}

	var parentMsgs []models.ParentMessage
	if err := db.Find(&parentMsgs).Error; err != nil {
		log.Fatalf("查询 parent_messages 失败: %v", err)
	}
	parentTextN, parentAudioN := 0, 0
	for _, msg := range parentMsgs {
		if msg.TextContent != "" && !privacy.IsCiphertext(msg.TextContent) {
			enc, err := svc.EncryptExistingText(msg.DeviceID, msg.TextContent)
			if err != nil {
				log.Printf("加密 parent text 失败 id=%d: %v", msg.ID, err)
			} else if !dryRun {
				if err := db.Model(&models.ParentMessage{}).Where("id = ?", msg.ID).Update("text_content", enc).Error; err != nil {
					log.Printf("写回 parent text 失败 id=%d: %v", msg.ID, err)
				} else {
					parentTextN++
				}
			} else {
				parentTextN++
			}
		}
		if strings.TrimSpace(msg.AudioPath) == "" {
			continue
		}
		path := msg.AudioPath
		if !filepath.IsAbs(path) {
			path = filepath.Join(parentAudioBase, path)
		}
		n, err := migrateAudioFile(svc, msg.DeviceID, path, dryRun)
		if err != nil {
			log.Printf("加密 parent 音频失败 id=%d path=%s: %v", msg.ID, path, err)
			continue
		}
		parentAudioN += n
	}

	mode := "已写入"
	if dryRun {
		mode = "dry-run 统计"
	}
	fmt.Printf("%s: chat_text=%d chat_audio=%d parent_text=%d parent_audio=%d\n",
		mode, chatTextN, chatAudioN, parentTextN, parentAudioN)
}

func migrateAudioFile(svc *privacy.Service, deviceDBID uint, fullPath string, dryRun bool) (int, error) {
	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	if privacy.IsCiphertext(string(data)) {
		return 0, nil
	}
	enc, err := svc.EncryptExistingFileBytes(deviceDBID, data)
	if err != nil {
		return 0, err
	}
	if dryRun {
		return 1, nil
	}
	if err := os.WriteFile(fullPath, enc, 0644); err != nil {
		return 0, err
	}
	return 1, nil
}
