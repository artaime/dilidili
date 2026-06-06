// 修补控制台 SQLite：OTA/UDP/MQTT（方案 A，AIToy TLS 8883）
// 用法: cd manager/backend && go run ./cmd/patch_ota -db <path/to/xiaozhi.db>
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type configRow struct {
	ID       uint   `gorm:"column:id"`
	ConfigID string `gorm:"column:config_id"`
	Type     string `gorm:"column:type"`
	JsonData string `gorm:"column:json_data"`
}

func (configRow) TableName() string { return "configs" }

func main() {
	dbPath := flag.String("db", "", "path to xiaozhi.db")
	lanIP := flag.String("ip", "192.168.0.55", "LAN IP")
	flag.Parse()
	if *dbPath == "" {
		fmt.Fprintln(os.Stderr, "missing -db")
		os.Exit(1)
	}

	db, err := gorm.Open(sqlite.Open(*dbPath), &gorm.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "open db: %v\n", err)
		os.Exit(1)
	}

	for _, typ := range []string{"ota", "udp", "mqtt_server", "mqtt"} {
		if err := patchType(db, typ, *lanIP); err != nil {
			fmt.Fprintf(os.Stderr, "patch %s: %v\n", typ, err)
			os.Exit(1)
		}
	}
	fmt.Println("done")
}

func patchType(db *gorm.DB, typ, lanIP string) error {
	var rows []configRow
	if err := db.Where("type = ?", typ).Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		fmt.Printf("skip %s (no rows)\n", typ)
		return nil
	}
	for _, row := range rows {
		data := map[string]interface{}{}
		if row.JsonData != "" {
			_ = json.Unmarshal([]byte(row.JsonData), &data)
		}
		switch typ {
		case "ota":
			applyOTA(data, lanIP)
		case "udp":
			applyUDP(data, lanIP)
		case "mqtt_server":
			applyMQTTServer(data)
		case "mqtt":
			applyMQTTClient(data)
		}
		raw, err := json.Marshal(data)
		if err != nil {
			return err
		}
		if err := db.Model(&configRow{}).Where("id = ?", row.ID).Update("json_data", string(raw)).Error; err != nil {
			return err
		}
		fmt.Printf("updated %s %s\n", typ, row.ConfigID)
	}
	return nil
}

func applyOTA(data map[string]interface{}, lanIP string) {
	endpoint := lanIP + ":8883"
	wsURL := "ws://" + lanIP + ":8989/xiaozhi/v1/"
	for _, key := range []string{"test", "external"} {
		section, _ := data[key].(map[string]interface{})
		if section == nil {
			section = map[string]interface{}{}
		}
		mqtt, _ := section["mqtt"].(map[string]interface{})
		if mqtt == nil {
			mqtt = map[string]interface{}{}
		}
		mqtt["enable"] = true
		mqtt["endpoint"] = endpoint
		section["mqtt"] = mqtt
		section["websocket"] = map[string]interface{}{"url": wsURL}
		data[key] = section
	}
}

func applyUDP(data map[string]interface{}, lanIP string) {
	data["external_host"] = lanIP
	if _, ok := data["external_port"]; !ok {
		data["external_port"] = 8990
	}
}

func applyMQTTServer(data map[string]interface{}) {
	data["enable"] = true
	data["listen_host"] = "0.0.0.0"
	data["listen_port"] = 2883
	data["enable_auth"] = false
	if _, ok := data["username"]; !ok {
		data["username"] = "admin"
	}
	if _, ok := data["password"]; !ok {
		data["password"] = "test!@#"
	}
	tls, _ := data["tls"].(map[string]interface{})
	if tls == nil {
		tls = map[string]interface{}{}
	}
	tls["enable"] = true
	tls["port"] = 8883
	tls["pem"] = "config/server.pem"
	tls["key"] = "config/server.key"
	data["tls"] = tls
}

func applyMQTTClient(data map[string]interface{}) {
	data["enable"] = true
	data["broker"] = "127.0.0.1"
	data["type"] = "tcp"
	data["port"] = 2883
	data["client_id"] = "xiaozhi_server"
	if _, ok := data["username"]; !ok {
		data["username"] = "admin"
	}
	if _, ok := data["password"]; !ok {
		data["password"] = "test!@#"
	}
}
