package util

import (
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

// GetNodeID 返回主服务节点唯一标识，用于 WS 连接与管理端运行时监控关联。
// 优先级：环境变量 DILI_NODE_ID > server.node_id > 主机名 > 随机 UUID（进程内稳定）。
func GetNodeID() string {
	if id := strings.TrimSpace(os.Getenv("DILI_NODE_ID")); id != "" {
		return id
	}
	if id := strings.TrimSpace(viper.GetString("server.node_id")); id != "" {
		return id
	}
	if hostname, err := os.Hostname(); err == nil {
		if h := strings.TrimSpace(hostname); h != "" {
			return h
		}
	}
	return fallbackNodeID()
}

// GetNodeName 返回节点展示名称。
func GetNodeName() string {
	if name := strings.TrimSpace(viper.GetString("server.node_name")); name != "" {
		return name
	}
	return GetNodeID()
}

var generatedNodeID = uuid.New().String()

func fallbackNodeID() string {
	return generatedNodeID
}
