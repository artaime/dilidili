package memobaseuserid

import "github.com/google/uuid"

var deviceNamespace = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

func deviceIDToUUID(deviceID string) string {
	return uuid.NewSHA1(deviceNamespace, []byte(deviceID)).String()
}

// MemobaseUserID 将设备 SN 映射为 Memobase 用户 UUID（单次 UUID v5）。
func MemobaseUserID(deviceSN string) string {
	return deviceIDToUUID(deviceSN)
}

// LegacyMemobaseUserID 返回修复 double-UUID 之前的历史用户 ID，用于兼容旧数据。
func LegacyMemobaseUserID(deviceSN string) string {
	return deviceIDToUUID(MemobaseUserID(deviceSN))
}
