```markdown
### AI玩具协议26031617.pdf

#### AI玩具交互协议

**一、HTTP请求（全交互内容）**

**OTA 版本检查 ( 没有可升级版本时 )**
*   **请求地址：** http://iot.realscene.cn:8002/xiaozhi/ota/
*   **接口状态：** OTA 接口运行正常， websocket 集群数量： 1
*   **触发时机：**设备 4G 网络连接成功，进入激活检查阶段

**请求**
*   **请求方式：** POST
*   **请求头（ HEADER ）：**
    *   Activation-Version: 1.6.0
    *   Device-Id: 00:50:22:ef:07:f9
    *   Client-Id: c147fac7-0012-22fc-021f-37e7b98ca55a
    *   User-Agent: AIToy/2.31.0
    *   Accept-Language: en-US
    *   Content-Type: application/json
*   **请求体（ BODY ）：**
```json
{
"application" : {
"version" :  "{\"pd-v1-gx8006\":\"2.12.0\",\"pd-v1-ln882\":\"2.31.0\",\"pd-v1-re3220\":\"\"}" ,
"elf_sha256" : "c8a8ecb6d6fbcda682494d9675cd1ead240ecf38bdde75282a42365a0e396033"
  },
"board" : {
"type" :  "pangdou-toy" ,
"name" :  "pangdou-board" ,
"ssid" :  "" ,
"rssi" :  - 50 ,
"channel" :  1 ,
"ip" :  "0.0.0.0" ,
"mac" :  "00:50:22:ef:07:f9" ,
"ble_mac" :  "C0:50:22:EF:07:F9" ,
"imei" :  "865229085717303"
  }
}
```

**响应**
```json
{
"server_time" : {
"timestamp" :  1772593434733 ,
"timeZone" :  "Asia/Shanghai" ,
"timezone_offset" :  480
  },
"firmware" : {
"subs" : {
"pd-v1-re3220" : {
"version" :  "2.10.0" ,
"size" :  1407024 ,
"url" :  "http://iot.realscene.cn:8002/xiaozhi/otaMag/download/5e85e47b-5983-4d9c-bbad-a5646d7b5c02?subtype=2"
      }
    }
  },
"websocket" : {
"url" :  "ws://iot.realscene.cn:8000/xiaozhi/v1/" ,
"token" :  ""
  },
"mqtt" : {
"endpoint" :  "121.199.54.249" ,
"client_id" :  "pangdou-toy@@@00_50_22_ef_07_f9@@@00_50_22_ef_07_f9" ,
"username" :  "eyJpcCI6IjM2LjE1MC4xNTguMTc5In0=" ,
"password" :  "OXIrBZ6Xq0hBbSz/jl5K7Tbp5oAhkNFetXJF1BtAjW4=" ,
"publish_topic" :  "device-server" ,
"subscribe_topic" :  "devices/p2p/00_50_22_ef_07_f9"
  },
"pd" : {
"voice" :  1
  }
}
```

**OTA版本检查(有可升级版本时)**
*   **请求地址：** http://iot.realscene.cn:8002/xiaozhi/ota/
*   **接口状态：** OTA 接口运行正常， websocket 集群数量： 1
*   **触发时机：**设备 4G 网络连接成功，进入激活检查阶段（首次请求域名解析失败，重试后成功）

**请求**
*   **请求方式：** POST
*   **请求头（ HEADER ）：**
    *   Activation-Version: 1.6.0
    *   Device-Id: 00:50:1d:29:f4:f8
    *   Client-Id: a0178609-0012-24cc-0221-ea01a1cca55a
    *   User-Agent: AIToy/2.33.0
    *   Accept-Language: en-US
    *   Content-Type: application/json
*   **请求体（ BODY ）：**
```json
{
"application" : {
"version" :  "{\"pd-v1-gx8006\":\"2.12.0\",\"pd-v1-ln882\":\"2.33.0\",\"pd-v1-re3220\":\"2.12.0\"}" , 
"elf_sha256" : "c8a8ecb6d6fbcda682494d9675cd1ead240ecf38bdde75282a42365a0e396033"
  },
"board" : {
"type" :  "pangdou-toy" ,
"name" :  "pangdou-board" ,
"ssid" :  "" ,
"rssi" :  - 50 ,
"channel" :  1 ,
"ip" :  "0.0.0.0" ,
"mac" :  "00:50:1d:29:f4:f8" ,
"ble_mac" :  "C0:50:1D:29:F4:F8" ,
"imei" :  "865229085720976"
  }
}
```

**响应**
```json
{
"server_time" : {
"timestamp" :  1773651541396 ,
"timeZone" :  "Asia/Shanghai" ,
"timezone_offset" :  480
  },
"firmware" : {
"subs" : {
"pd-v1-ln882" : {
"version" :  "2.34.0" ,
"size" :  669154 ,
"url" :  "http://iot.realscene.cn:8002/xiaozhi/otaMag/download/54116515-6f61-48b7-808e-a1995a6a2224?subtype=0"
      },
"pd-v1-gx8006" : {
"version" :  "2.13.0" ,
"size" :  1577024 ,
"url" :  "http://iot.realscene.cn:8002/xiaozhi/otaMag/download/335814dd-b4d8-4f89-ab24-8234b69c751b?subtype=1"
      },
"pd-v1-re3220" : {
"version" :  "2.13.0" ,
"size" :  1408184 ,
"url" :  "http://iot.realscene.cn:8002/xiaozhi/otaMag/download/b22ac639-e13c-4840-bdcc-13c032ddee44?subtype=2"
      }
    }
  },
"websocket" : {
"url" :  "ws://iot.realscene.cn:8000/xiaozhi/v1/" ,
"token" :  ""
  },
"mqtt" : {
"endpoint" :  "121.199.54.249" ,
"client_id" :  "pangdou-toy@@@00_50_1d_29_f4_f8@@@00_50_1d_29_f4_f8" ,
"username" :  "eyJpcCI6IjExMS41NS4xNzYuNjkifQ==" ,
"password" :  "1SnVVvy3j+Fwa5r3fWpzPRoK+1pQcTYpuyYK1j4SJk8=" ,
"publish_topic" :  "device-server" ,
"subscribe_topic" :  "devices/p2p/00_50_1d_29_f4_f8"
  },
"pd" : {
"voice" :  2
}
}
```

**版本对比结果**

| 固件类型 | 当前版本 | 可升级版本 | 版本对比结果 |
| :--- | :--- | :--- | :--- |
| pd-v1-gx8006 | 2.12.0 | 2.13.0 | 需要升级 |
| pd-v1-ln882 | 2.33.0 | 2.34.0 | 需要升级 |
| pd-v1-re3220 | 2.12.0 | 2.13.0 | 需要升级 |

1.  有可升级版本时，设备请求后，响应返回所有可升级固件的版本、大小和下载地址。
2.  响应中 firmware. subs包含 pd-v1-ln882、pd-v1-gx8006、pd-v1-re3220三类固件的升级信息, 且版本号均高于设备当前版本。

**二、MQTT协议交互 (全交互内容，含CONNECT/SEND/RECV全指令)**

**1. MQTT基础配置与连接**

**连接参数(从HTTP响应中提取)：**
*   服务器地址: 121.199.54.249:8883
*   Client ID: pangdou-toy@@@00_50_22_ ef _07_f9@@@00_50_22_ ef _07_f9
*   用户名: eyJpcCI6IjM2LjE1MC4xNTguMTc5In0=
*   密码: 0XIrBZ6xq0hBbSz/j15K7Tbp5oAhkNFetXJF1BtAjw4=
*   发布主题: device-server
*   订阅主题: devices/p2p/00_50_22_ ef _07_f9
*   传输方式: 基于4G (CAT1) 网络
*   连接结果: [CAT1] MQTT connected successfully (MQTT连接成功)
*   状态回调: [M_MQTT] MQTT connection state: Connected successfully
*   设备状态切换: SMARTBOT_SYSTEM_ACTIVATED → SMARTBOT_SYSTEM_SRV_CONNECTED

**2. MCP协议初始化交互 (无会话ID阶段)**

**【RECV】MCP初始化请求**
```json
{
" type": " mcp",
" payload": {
" jsonrpc": "2.0",
" method": " initialize",
" id": 10000,
" params": {
" protocolversion": "2024-11-05",
" capabilities": {},
"clientInfo": {
" name": " xiaozhi-mqtt-client",
" version": "1.0.0"
}
}
}
}
```

**【 SEND 】 MCP 初始化响应**
```json
{
"session_id" :  "" ,
"type" :  "mcp" ,
"payload" : {
"jsonrpc" :  "2.0" ,
"id" :  10000 ,
"result" : {
"protocolVersion" :  "2024-11-05" ,
"capabilities" : {
"tools" : {}
      },
"serverInfo" : {
"name" :  "pangdou-toy" ,
"version" :  "2.31.0"
      }
    }
  }
}
```

**【 RECV 】初始化完成通知**
```json
{
"type" :  "mcp" ,
"payload" : {
"jsonrpc" :  "2.0" ,
"method" :  "notifications/initialized"
  }
}
```

**【 RECV 】 MCP 工具列表查询请求**
```json
{
"type" :  "mcp" ,
"payload" : {
"jsonrpc" :  "2.0" ,
"method" :  "tools/list" ,
"id" :  10001 ,
"params" : {}
  }
}
```

**【 SEND 】 MCP 工具列表响应**
```json
{
"session_id" :  "" ,
"type" :  "mcp" ,
"payload" : {
"jsonrpc" :  "2.0" ,
"id" :  10001 ,
"result" : {
"tools" : [
        {
"name" :  "get_device_status" ,
"description" :  " 用于获得设备的当前状态 ,  如音量，屏幕亮度，" ,
"inputSchema" : {
"type" :  "object" ,
"properties" : {}
          }
        },
        {
"name" :  "set_speaker_volume" ,
"description" :  " 用于设定设备的音量 " ,
"inputSchema" : {
"type" :  "object" ,
"properties" : {
"volume" : {
"type" :  "integer" ,
"minimum" :  0 ,
"maximum" :  100
              }
            },
"required" : [ "volume" ]
          }
        },
        {
"name" :  "set_screen_brightness" ,
"description" :  " 用于设定设备的屏幕亮度 " ,
"inputSchema" : {
"type" :  "object" ,
"properties" : {
"brightness" : {
"type" :  "integer" ,
"minimum" :  0 ,
"maximum" :  100
              }
            },
"required" : [ "brightness" ]
          }
        },
        {
"name" :  "enter_sleep_mode" ,
"description" :  " 用于让设备进入睡眠模式 " ,
"inputSchema" : {
"type" :  "object" ,
"properties" : {}
          }
        },
        {
"name" :  "power_off_device" ,
"description" :  " 用于让设备关机 " ,
"inputSchema" : {
"type" :  "object" ,
"properties" : {}
          }
        }
      ]
    }
  }
}
```

**3. 会话创建与UDP音频通道初始化**

**【 SEND 】会话初始化请求（ hello ）**
```json
{
"type" :  "hello" ,
"version" :  3 ,
"transport" :  "udp" ,
"audio_params" : {
"format" :  "opus" ,
"sample_rate" :  16000 ,
"channels" :  1 ,
"frame_duration" :  40
  },
"features" : {
"consistent_sample_rate" :  true ,
"mcp" :  true
  }
}
```

**【 SEND 】临时终止指令（无会话 ID ）**
```json
{
"session_id" :  "" ,
"type" :  "abort"
}
```

**【 RECV 】会话确认响应（分配唯一 session_id ）**
```json
{
"type" :  "hello" ,
"version" :  3 ,
"session_id" :  "a82e2d88-cd98-43a2-9a11-e4a97205b170" ,
"transport" :  "udp" ,
"udp" : {
"server" :  "121.199.54.249" ,
"port" :  8884 ,
"encryption" :  "aes-128-ctr" ,
"key" :  "022e5d9bd0ed3da564c54a782c69d886" ,
"nonce" :  "0100000023c4e6220000000000000000"
  },
"audio_params" : {
"format" :  "opus" ,
"sample_rate" :  16000 ,
"channels" :  1 ,
"frame_duration" :  40
  }
}
```

**4. 带会话ID的MCP协议二次初始化**

**【 RECV 】 MCP 二次初始化请求（绑定 session_id ）**
```json
{
"type" :  "mcp" ,
"payload" : {
"jsonrpc" :  "2.0" ,
"method" :  "initialize" ,
"id" :  1 ,
"params" : {
"protocolVersion" :  "202ascar24-11-05" ,
"capabilities" : {
"roots" : {
"listChanged" :  true
        },
"sampling" : {},
"vision" : {
"url" :  "http://172.27.5.233:8003/mcp/vision/explain" ,
"token" : "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjoiVEtGZXJuSElXZTNPOXV1MmZoYWgyeGlvWFRPa2FaN094bnhnX25YakhYQXN6eGxpbG1NbWVENGNqM1hpTnBJMFgxTWxvbVBkZGVUb1NyVTFoMXU0RWpUQXZkVE00ZDdiRHpYcEYtNm5CTm9uUG9qcW1yQmhiUT09In0.JWILRLw0ax7yRbdm5BfE1FPiQC9sGKee6BJ8GszBX3c"
        }
      },
"clientInfo" : {
"name" :  "XiaozhiClient" ,
"version" :  "1.0.0"
      }
    }
  }
}
```

**【 SEND 】 MCP 二次初始化响应（绑定 session_id ）**
```json
{
"session_id" :  "a82e2d88-cd98-43a2-9a11-e4a97205b170" ,
"type" :  "mcp" ,
"payload" : {
"jsonrpc" :  "2.0" ,
"id" :  1 ,
"result" : {
"protocolVersion" :  "2024-11-05" ,
"capabilities" : {
"tools" : {}
      },
"serverInfo" : {
"name" :  "pangdou-toy" ,
"version" :  "2.31.0"
      }
    }
  }
}
```

**5. 核心业务指令全交互（绑定session_id）**

**指令1：语音指令「讲个故事」全交互**

**【 RECV 】 TTS 停止指令**
```json
{
"type" :  "tts" ,
"state" :  "stop" ,
"session_id" :  "a82e2d88-cd98-43a2-9a11-e4a97205b170"
}
```

**【 SEND 】开始监听语音指令**
```json
{
"session_id" :  "a82e2d88-cd98-43a2-9a11-e4a97205b170" ,
"type" :  "listen" ,
"state" :  "start" ,
"mode" :  "auto"
}
```

**【 SEND 】停止监听语音指令**
```json
{
"session_id" :  "a82e2d88-cd98-43a2-9a11-e4a97205b170" ,
"type" :  "listen" ,
"state" :  "stop"
}
```

**【 RECV 】 STT 语音识别结果**
```json
{
"type" :  "stt" ,
"text" :  " 讲个故事 " ,
"session_id" :  "a82e2d88-cd98-43a2-9a11-e4a97205b170"
}
```

**【 RECV 】 TTS 开始播报指令**
```json
{
"type" :  "tts" ,
"state" :  "start" ,
"session_id" :  "a82e2d88-cd98-43a2-9a11-e4a97205b170"
}
```

**【 RECV 】 LLM 情感与表情响应**
```json
{
"type" :  "llm" ,
"text" :  " 😱 " ,
"emotion" :  "shocked" ,
"session_id" :  "a82e2d88-cd98-43a2-9a11-e4a97205b170"
}
```

**【 RECV 】 TTS 再次开始播报指令**
```json