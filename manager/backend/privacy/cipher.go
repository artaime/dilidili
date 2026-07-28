package privacy

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

const (
	CipherPrefix = "v1|"
	nonceSize    = 12
)

// IsCiphertext 判断是否为 v1| 格式密文。
func IsCiphertext(s string) bool {
	return strings.HasPrefix(s, CipherPrefix)
}

// Encrypt 使用 DEK 加密明文，返回 v1|<keyID>|<b64(nonce||ct||tag)>。
func Encrypt(dek []byte, keyID, plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	raw, err := encryptBytes(dek, []byte(plaintext))
	if err != nil {
		return "", err
	}
	return formatCiphertext(keyID, raw), nil
}

// Decrypt 解密密文；非密文格式原样返回（双读兼容）。
func Decrypt(dek []byte, ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	if !IsCiphertext(ciphertext) {
		return ciphertext, nil
	}
	raw, err := parseCiphertext(ciphertext)
	if err != nil {
		return "", err
	}
	plain, err := decryptBytes(dek, raw)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// EncryptBytes 加密任意字节，返回密文字符串（可直接写文件）。
func EncryptBytes(dek []byte, keyID string, plain []byte) (string, error) {
	if len(plain) == 0 {
		return "", nil
	}
	raw, err := encryptBytes(dek, plain)
	if err != nil {
		return "", err
	}
	return formatCiphertext(keyID, raw), nil
}

// DecryptBytes 解密文件/二进制密文；非密文格式原样返回。
func DecryptBytes(dek []byte, data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	s := string(data)
	if !IsCiphertext(s) {
		return data, nil
	}
	raw, err := parseCiphertext(s)
	if err != nil {
		return nil, err
	}
	return decryptBytes(dek, raw)
}

// WrapDEK 用 KEK 包装 DEK，格式同字段密文（无业务 key_id，用 "wrap"）。
func WrapDEK(kek, dek []byte) (string, error) {
	if len(kek) != 32 {
		return "", fmt.Errorf("KEK 必须为 32 字节")
	}
	if len(dek) != 32 {
		return "", fmt.Errorf("DEK 必须为 32 字节")
	}
	raw, err := encryptBytes(kek, dek)
	if err != nil {
		return "", err
	}
	return formatCiphertext("wrap", raw), nil
}

// UnwrapDEK 解包 DEK。
func UnwrapDEK(kek []byte, wrapped string) ([]byte, error) {
	if len(kek) != 32 {
		return nil, fmt.Errorf("KEK 必须为 32 字节")
	}
	raw, err := parseCiphertext(wrapped)
	if err != nil {
		return nil, err
	}
	dek, err := decryptBytes(kek, raw)
	if err != nil {
		return nil, err
	}
	if len(dek) != 32 {
		return nil, fmt.Errorf("解包后 DEK 长度无效")
	}
	return dek, nil
}

// GenerateDEK 生成 32 字节随机 DEK。
func GenerateDEK() ([]byte, error) {
	dek := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return nil, fmt.Errorf("生成 DEK 失败: %w", err)
	}
	return dek, nil
}

func formatCiphertext(keyID string, raw []byte) string {
	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		keyID = "k1"
	}
	return CipherPrefix + keyID + "|" + base64.StdEncoding.EncodeToString(raw)
}

func parseCiphertext(s string) ([]byte, error) {
	if !IsCiphertext(s) {
		return nil, fmt.Errorf("非密文格式")
	}
	// v1|<key_id>|<b64>
	rest := s[len(CipherPrefix):]
	idx := strings.IndexByte(rest, '|')
	if idx < 0 || idx == len(rest)-1 {
		return nil, fmt.Errorf("密文格式无效")
	}
	b64 := rest[idx+1:]
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("密文 base64 无效: %w", err)
	}
	return raw, nil
}

func encryptBytes(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建 AES 失败: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建 GCM 失败: %w", err)
	}
	if gcm.NonceSize() != nonceSize {
		return nil, fmt.Errorf("意外的 nonce 大小: %d", gcm.NonceSize())
	}
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("生成 nonce 失败: %w", err)
	}
	ct := gcm.Seal(nil, nonce, plaintext, nil)
	out := make([]byte, 0, nonceSize+len(ct))
	out = append(out, nonce...)
	out = append(out, ct...)
	return out, nil
}

func decryptBytes(key, raw []byte) ([]byte, error) {
	if len(raw) < nonceSize+16 {
		return nil, fmt.Errorf("密文过短")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建 AES 失败: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建 GCM 失败: %w", err)
	}
	nonce := raw[:nonceSize]
	ct := raw[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("解密失败: %w", err)
	}
	return plain, nil
}
