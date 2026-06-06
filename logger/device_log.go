package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	log "github.com/sirupsen/logrus"
)

const (
	DeviceTagMQTT   = "MQTT"
	DeviceTagOTA    = "OTA"
	DeviceTagMCP    = "MCP"
	DeviceTagProto  = "PROTO"
	DeviceTagUDP    = "UDP"
	DeviceTagSystem = "SYS"
)

var (
	deviceLogger   *log.Logger
	deviceLogOnce  sync.Once
	deviceLogReady bool
)

// InitDeviceLog 初始化设备交互专用日志（项目 logs/device.log）。
// path 为空时由 ResolveDeviceLogPath 解析。
func InitDeviceLog(path string) error {
	var initErr error
	deviceLogOnce.Do(func() {
		if path == "" {
			path = ResolveDeviceLogPath()
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			initErr = fmt.Errorf("创建设备日志目录失败: %w", err)
			return
		}
		writer, err := rotatelogs.New(
			path+".%Y%m%d",
			rotatelogs.WithLinkName(path),
			rotatelogs.WithRotationCount(7),
			rotatelogs.WithRotationTime(24*time.Hour),
		)
		if err != nil {
			initErr = fmt.Errorf("初始化设备日志轮转失败: %w", err)
			return
		}
		l := log.New()
		l.SetOutput(writer)
		l.SetLevel(log.DebugLevel)
		l.SetFormatter(&log.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05.000",
			FullTimestamp:   true,
			DisableColors:   true,
		})
		deviceLogger = l
		deviceLogReady = true
		DeviceInfof(DeviceTagSystem, "", "设备交互日志已启用: %s", path)
	})
	return initErr
}

// ResolveDeviceLogPath 解析 device.log 路径：配置 > 环境变量 > 仓库根目录 logs/device.log。
func ResolveDeviceLogPath() string {
	if p := os.Getenv("XIAOZHI_DEVICE_LOG"); p != "" {
		return p
	}
	if root := findRepoRoot(); root != "" {
		return filepath.Join(root, "logs", "device.log")
	}
	binPath, _ := os.Executable()
	return filepath.Join(filepath.Dir(binPath), "logs", "device.log")
}

// FindRepoRootForLog 供主程序解析相对路径的设备日志位置。
func FindRepoRootForLog() string {
	return findRepoRoot()
}

func findRepoRoot() string {
	if env := os.Getenv("XIAOZHI_REPO_ROOT"); env != "" {
		return env
	}
	for _, start := range []func() string {
		func() string { d, _ := os.Getwd(); return d },
		func() string { p, _ := os.Executable(); return filepath.Dir(p) },
	} {
		dir := start()
		for i := 0; i < 12; i++ {
			if dir == "" {
				break
			}
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return ""
}

func deviceLog(tag, deviceID, format string, args ...interface{}) {
	if !deviceLogReady || deviceLogger == nil {
		return
	}
	prefix := fmt.Sprintf("[%s]", tag)
	if deviceID != "" {
		prefix = fmt.Sprintf("[%s][%s]", tag, deviceID)
	}
	deviceLogger.Infof(prefix+" "+format, args...)
}

func DeviceInfof(tag, deviceID, format string, args ...interface{}) {
	deviceLog(tag, deviceID, format, args...)
}

func DeviceWarnf(tag, deviceID, format string, args ...interface{}) {
	if !deviceLogReady || deviceLogger == nil {
		return
	}
	prefix := fmt.Sprintf("[%s]", tag)
	if deviceID != "" {
		prefix = fmt.Sprintf("[%s][%s]", tag, deviceID)
	}
	deviceLogger.Warnf(prefix+" "+format, args...)
}

func DeviceErrorf(tag, deviceID, format string, args ...interface{}) {
	if !deviceLogReady || deviceLogger == nil {
		return
	}
	prefix := fmt.Sprintf("[%s]", tag)
	if deviceID != "" {
		prefix = fmt.Sprintf("[%s][%s]", tag, deviceID)
	}
	deviceLogger.Errorf(prefix+" "+format, args...)
}

func DeviceDebugf(tag, deviceID, format string, args ...interface{}) {
	if !deviceLogReady || deviceLogger == nil {
		return
	}
	prefix := fmt.Sprintf("[%s]", tag)
	if deviceID != "" {
		prefix = fmt.Sprintf("[%s][%s]", tag, deviceID)
	}
	deviceLogger.Debugf(prefix+" "+format, args...)
}
