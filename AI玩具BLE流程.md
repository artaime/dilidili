**AI 玩具 BLE 通信流程**

**一、小程序端：设备扫描与连接**

1.  调用微信小程序蓝牙 API 启动 BLE 设备扫描，枚举所有扫描到的 BLE 设备；

2.  提取每个设备的设备名称和advertiseData，校验 advertiseData 前 5 个字节为 0xFF, 0xFF, 0x02, 0x00, 0x01 时，判定为目标 AI 玩具设备；

3.  将所有目标设备展示在小程序界面，供用户选择；

4.  用户点击目标设备后，调用小程序 API 发起 BLE 连接，同时将 MTU 值设置为 128 字节；

5.  连接成功后枚举设备所有服务，按属性分类记录服务 ID：

    含 READ 属性：记录为 ReadServiceID

    含 WRITE 属性：记录为 WriteServiceID

    含 NOTIFY 属性：记录为 NotifyServiceID

6.  基于WriteServiceID（发送）和NotifyServiceID（接收）建立数据交换通道，发送指令

    {"ForceGetSSID": "true"} 至设备端，请求获取周边 WIFI 列表及设备 SN。

**二、设备端：WIFI 与 SN 信息回传**

设备接收到小程序的ForceGetSSID 请求后，通过 Notify 通道依次回传数据，包含扫描到的 WIFI 信息

（单条独立报文）和设备 SN，数据格式如下：

WIFI 单条信息： {"rssi":-66,"mac":"C6B25B112A5D","ssid":"boya"} 、



设备 SN： {"sn":"SN-XXXXXXXX-XXXXXXXX"} （sn 作为设备唯一标识，也兼容 serial_number 字段）。

**三、小程序端：设备绑定与 WIFI 配置**

1.  接收到设备的 sn 后，小程序向后台发送 `POST /api/mp/devices/bind`（body 含 `sn`）完成绑定（同时将设备标记为已激活）；绑定前通过 `GET /api/mp/devices/check?sn=` 预检登记状态；

2.  绑定成功后，进入 WiFi 配网：接收到设备回传的 WIFI 列表后，将所有 WIFI 名称展示在界面，供用户选择；

3.  用户选择目标 WIFI 并输入密码后，小程序通过 Write 通道发送 WIFI 配置指令

    {"ssid":"xxx","password":"xxx"} 至设备端；用户可选择跳过此步骤，不进行 WIFI 配置。

4.  WiFi 配网完成或跳过后，小程序开放音量/亮度系统设置页面。

**四、设备端：WIFI 配网与服务器连接（含状态码回传）**

设备接收到小程序的 WIFI 配置指令后，按流程执行操作，并通过 Notify 通道实时回传sta_code 状态码，全程流程及对应状态码如下：

1.  接收到 WIFI 配置指令，立即回传： {"sta_code":"1"} （已接收配置）；

2.  根据 SSID 和 PASSWORD 尝试连接 WIFI，连接成功后回传： {"sta_code":"2"} （WIFI 连接成功）；

3.  WIFI 连接成功后，发起后台 HTTP 服务器连接并发送报文，根据服务器返回的激活状态分支处理：

    设备未激活：回传{"sta_code":"4"} （未激活，进入重试等待）；

    设备已激活：回传{"sta_code":"3"} （已激活，进入下一步）；

4.  设备已激活且 HTTP 交互完成后，发起 MQTT 服务器连接，连接成功后回传： {"sta_code":"5"}

    （MQTT 连接成功，WIFI 配网流程全部完成）。
