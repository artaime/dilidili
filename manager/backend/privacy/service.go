package privacy

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"sync"

	"dili/manager/backend/models"

	"gorm.io/gorm"
)

const KEKEnv = "PRIVACY_KEK_BASE64"

// Config 隐私加密运行时配置。
type Config struct {
	Enabled bool
	KeyID   string
	KEK     []byte
}

// Service 设备级 DEK 管理与加解密。
type Service struct {
	db       *gorm.DB
	cfg      Config
	mu       sync.Mutex
	dekCache map[uint][]byte // deviceDBID -> DEK
}

// LoadKEKFromEnv 从环境变量加载 32 字节 KEK。
func LoadKEKFromEnv() ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv(KEKEnv))
	if raw == "" {
		return nil, fmt.Errorf("环境变量 %s 未设置", KEKEnv)
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("%s 必须是 Base64: %w", KEKEnv, err)
	}
	if len(decoded) != 32 {
		return nil, fmt.Errorf("%s 解码后必须为 32 字节，当前 %d", KEKEnv, len(decoded))
	}
	return decoded, nil
}

// NewService 创建服务。enabled=true 时必须提供有效 KEK。
func NewService(db *gorm.DB, cfg Config) (*Service, error) {
	if cfg.KeyID == "" {
		cfg.KeyID = "k1"
	}
	if len(cfg.KEK) == 0 {
		if kek, err := LoadKEKFromEnv(); err == nil {
			cfg.KEK = kek
		} else if cfg.Enabled {
			return nil, fmt.Errorf("隐私加密已启用但 KEK 无效: %w", err)
		}
	}
	if cfg.Enabled && len(cfg.KEK) != 32 {
		return nil, fmt.Errorf("KEK 必须为 32 字节")
	}
	return &Service{
		db:       db,
		cfg:      cfg,
		dekCache: make(map[uint][]byte),
	}, nil
}

// Enabled 是否对新写入强制加密。
func (s *Service) Enabled() bool {
	return s != nil && s.cfg.Enabled
}

// KeyID 当前写入使用的 key_id。
func (s *Service) KeyID() string {
	if s == nil || s.cfg.KeyID == "" {
		return "k1"
	}
	return s.cfg.KeyID
}

// EncryptText 加密文本；未启用时原样返回。
func (s *Service) EncryptText(deviceDBID uint, plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	if s == nil || !s.cfg.Enabled {
		return plaintext, nil
	}
	dek, err := s.getOrCreateDEK(deviceDBID)
	if err != nil {
		return "", err
	}
	return Encrypt(dek, s.KeyID(), plaintext)
}

// DecryptText 解密文本；未启用或明文时双读兼容。若为密文则需 DEK。
func (s *Service) DecryptText(deviceDBID uint, value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if !IsCiphertext(value) {
		return value, nil
	}
	if s == nil {
		return "", fmt.Errorf("隐私服务未初始化，无法解密")
	}
	dek, err := s.getOrCreateDEK(deviceDBID)
	if err != nil {
		return "", err
	}
	return Decrypt(dek, value)
}

// EncryptFileBytes 加密音频等二进制；未启用原样返回字节。
func (s *Service) EncryptFileBytes(deviceDBID uint, plain []byte) ([]byte, error) {
	if len(plain) == 0 {
		return plain, nil
	}
	if s == nil || !s.cfg.Enabled {
		return plain, nil
	}
	dek, err := s.getOrCreateDEK(deviceDBID)
	if err != nil {
		return nil, err
	}
	ct, err := EncryptBytes(dek, s.KeyID(), plain)
	if err != nil {
		return nil, err
	}
	return []byte(ct), nil
}

// DecryptFileBytes 解密文件内容；明文透传。
func (s *Service) DecryptFileBytes(deviceDBID uint, data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	if !IsCiphertext(string(data)) {
		return data, nil
	}
	if s == nil {
		return nil, fmt.Errorf("隐私服务未初始化，无法解密")
	}
	dek, err := s.getOrCreateDEK(deviceDBID)
	if err != nil {
		return nil, err
	}
	return DecryptBytes(dek, data)
}

// EncryptExistingText 迁移用：无论 Enabled，只要有 KEK 就加密明文。
func (s *Service) EncryptExistingText(deviceDBID uint, plaintext string) (string, error) {
	if plaintext == "" || IsCiphertext(plaintext) {
		return plaintext, nil
	}
	if s == nil || len(s.cfg.KEK) != 32 {
		return "", fmt.Errorf("迁移需要有效 KEK")
	}
	dek, err := s.getOrCreateDEK(deviceDBID)
	if err != nil {
		return "", err
	}
	return Encrypt(dek, s.KeyID(), plaintext)
}

// EncryptExistingFileBytes 迁移用文件加密。
func (s *Service) EncryptExistingFileBytes(deviceDBID uint, plain []byte) ([]byte, error) {
	if len(plain) == 0 || IsCiphertext(string(plain)) {
		return plain, nil
	}
	if s == nil || len(s.cfg.KEK) != 32 {
		return nil, fmt.Errorf("迁移需要有效 KEK")
	}
	dek, err := s.getOrCreateDEK(deviceDBID)
	if err != nil {
		return nil, err
	}
	ct, err := EncryptBytes(dek, s.KeyID(), plain)
	if err != nil {
		return nil, err
	}
	return []byte(ct), nil
}

func (s *Service) getOrCreateDEK(deviceDBID uint) ([]byte, error) {
	if deviceDBID == 0 {
		return nil, fmt.Errorf("device_id 无效")
	}
	if len(s.cfg.KEK) != 32 {
		return nil, fmt.Errorf("KEK 未配置")
	}

	s.mu.Lock()
	if dek, ok := s.dekCache[deviceDBID]; ok {
		s.mu.Unlock()
		return dek, nil
	}
	s.mu.Unlock()

	var row models.DeviceEncryptionKey
	err := s.db.Where("device_id = ?", deviceDBID).First(&row).Error
	if err == nil {
		dek, err := UnwrapDEK(s.cfg.KEK, row.WrappedDEK)
		if err != nil {
			return nil, fmt.Errorf("解包设备 DEK 失败: %w", err)
		}
		s.mu.Lock()
		s.dekCache[deviceDBID] = dek
		s.mu.Unlock()
		return dek, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	dek, err := GenerateDEK()
	if err != nil {
		return nil, err
	}
	wrapped, err := WrapDEK(s.cfg.KEK, dek)
	if err != nil {
		return nil, err
	}
	row = models.DeviceEncryptionKey{
		DeviceID:   deviceDBID,
		KeyID:      s.KeyID(),
		WrappedDEK: wrapped,
	}
	if err := s.db.Create(&row).Error; err != nil {
		// 并发创建时再读一次
		var existing models.DeviceEncryptionKey
		if e2 := s.db.Where("device_id = ?", deviceDBID).First(&existing).Error; e2 == nil {
			dek2, err := UnwrapDEK(s.cfg.KEK, existing.WrappedDEK)
			if err != nil {
				return nil, err
			}
			s.mu.Lock()
			s.dekCache[deviceDBID] = dek2
			s.mu.Unlock()
			return dek2, nil
		}
		return nil, fmt.Errorf("保存设备 DEK 失败: %w", err)
	}
	s.mu.Lock()
	s.dekCache[deviceDBID] = dek
	s.mu.Unlock()
	return dek, nil
}
