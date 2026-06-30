package memobase

import "dili-esp32-server-golang/pkg/memobaseuserid"

// MemobaseUserID 将设备 SN 映射为 Memobase 用户 UUID（单次 UUID v5）。
func MemobaseUserID(deviceSN string) string {
	return memobaseuserid.MemobaseUserID(deviceSN)
}

// LegacyMemobaseUserID 返回修复 double-UUID 之前的历史用户 ID，用于兼容旧数据。
func LegacyMemobaseUserID(deviceSN string) string {
	return memobaseuserid.LegacyMemobaseUserID(deviceSN)
}
