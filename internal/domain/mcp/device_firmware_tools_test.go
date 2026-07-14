package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnrichFirmwareToolDescription(t *testing.T) {
	got := enrichFirmwareToolDescription("get_device_status", "用于获得设备的当前状态")
	assert.Contains(t, got, "用于获得设备的当前状态")
	assert.Contains(t, got, "必须调用本工具")
	assert.Contains(t, got, "不要猜测")

	vol := enrichFirmwareToolDescription("set_speaker_volume", "设定音量")
	assert.Contains(t, vol, "±10")
	assert.Contains(t, vol, "get_device_status")
	assert.Contains(t, vol, "0-100")

	sleep := enrichFirmwareToolDescription("enter_sleep_mode", "")
	assert.Contains(t, sleep, "睡眠模式")

	power := enrichFirmwareToolDescription("power_off_device", "关机")
	assert.Contains(t, power, "关机")

	unknown := enrichFirmwareToolDescription("play_music", "播放音乐")
	assert.Equal(t, "播放音乐", unknown)

	// 幂等：已含引导不重复追加
	once := enrichFirmwareToolDescription("get_device_status", got)
	assert.Equal(t, got, once)
}

func TestAnnotateFirmwareToolSuccess(t *testing.T) {
	got := AnnotateFirmwareToolSuccess("set_speaker_volume", `{"volume":60}`, `{"ok":true}`)
	assert.Contains(t, got, "成功")
	assert.Contains(t, got, "60")
	assert.Contains(t, got, "已设置为")

	got2 := AnnotateFirmwareToolSuccess("enter_sleep_mode", `{}`, "")
	assert.Contains(t, got2, "睡眠模式")

	passthrough := AnnotateFirmwareToolSuccess("get_device_status", `{}`, "vol=1")
	assert.Equal(t, "vol=1", passthrough)
}

func TestConvertMcpToolListEnrichesFirmwareDescriptions(t *testing.T) {
	tools := []mcp.Tool{
		mcp.NewTool("get_device_status", mcp.WithDescription("获得设备状态")),
		mcp.NewTool("set_speaker_volume", mcp.WithDescription("设定音量"),
			mcp.WithNumber("volume", mcp.Required(), mcp.Min(0), mcp.Max(100))),
	}
	converted := ConvertMcpToolListToInvokableToolList(tools, "iot_test", new(client.Client))
	require.Contains(t, converted, "get_device_status")
	require.Contains(t, converted, "set_speaker_volume")

	info, err := converted["get_device_status"].(tool.InvokableTool).Info(context.Background())
	require.NoError(t, err)
	assert.True(t, strings.Contains(info.Desc, "必须调用") || strings.Contains(info.Desc, "实时结果"))

	volInfo, err := converted["set_speaker_volume"].(*McpTool).Info(context.Background())
	require.NoError(t, err)
	assert.Contains(t, volInfo.Desc, "±10")
}
