package conf

import (
	"fmt"
	"net"
	"os"
	"strings"

	"cncamp/pkg/third_party/nightingale/alert/aconf"
	"cncamp/pkg/third_party/nightingale/center/cconf"
	"cncamp/pkg/third_party/nightingale/pkg/cfg"
	"cncamp/pkg/third_party/nightingale/pkg/httpx"
	"cncamp/pkg/third_party/nightingale/pkg/logx"
	"cncamp/pkg/third_party/nightingale/pkg/ormx"
	"cncamp/pkg/third_party/nightingale/pushgw/pconf"
	"cncamp/pkg/third_party/nightingale/storage"
)

type ConfigType struct {
	Global GlobalConfig
	Log    logx.Config
	HTTP   httpx.Config
	DB     ormx.DBConfig
	Redis  storage.RedisConfig

	Pushgw pconf.Pushgw
	Alert  aconf.Alert
	Center cconf.Center
}

type GlobalConfig struct {
	RunMode string
}

func InitConfig(configDir, cryptoKey string) (*ConfigType, error) {
	var config = new(ConfigType)

	if err := cfg.LoadConfigByDir(configDir, config); err != nil {
		return nil, fmt.Errorf("failed to load configs of directory: %s error: %s", configDir, err)
	}

	config.Pushgw.PreCheck()
	config.Alert.PreCheck()
	config.Center.PreCheck()

	err := decryptConfig(config, cryptoKey)
	if err != nil {
		return nil, err
	}

	if config.Alert.Heartbeat.IP == "" {
		// auto detect
		config.Alert.Heartbeat.IP = fmt.Sprint(GetOutboundIP())
		if config.Alert.Heartbeat.IP == "" {
			hostname, err := os.Hostname()
			if err != nil {
				fmt.Println("failed to get hostname:", err)
				os.Exit(1)
			}

			if strings.Contains(hostname, "localhost") {
				fmt.Println("Warning! hostname contains substring localhost, setting a more unique hostname is recommended")
			}

			config.Alert.Heartbeat.IP = hostname
		}
	}

	config.Alert.Heartbeat.Endpoint = fmt.Sprintf("%s:%d", config.Alert.Heartbeat.IP, config.HTTP.Port)

	return config, nil
}

func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "223.5.5.5:80")
	if err != nil {
		fmt.Println("auto get outbound ip fail:", err)
		return []byte{}
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}
