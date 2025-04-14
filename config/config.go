package config

import (
	"fmt"
	"log"
	"os"

	C "GameConnManager/constant"

	"gopkg.in/yaml.v3"
)

// Checker 定义检查器配置
type Checker struct {
	Proxy    string `yaml:"proxy"`
	Timeout  string `yaml:"timeout"`
	Interval string `yaml:"interval"`
}

// Switcher 定义切换器配置
type Switcher struct {
	CheckRiseInterval string `yaml:"check-rise-interval"`
}

// ConnClose 定义连接关闭配置
type ConnClose struct {
	ListenIP   string `yaml:"listen-ip"`
	ListenPort int    `yaml:"listen-port"`
}

// Config 定义 YAML 配置结构体
type Config struct {
	MihomoApiUrl           string    `yaml:"mihomo-api-url"`
	MihomoApiToken         string    `yaml:"mihomo-api-token"`
	MihomoApiTimeout       string    `yaml:"mihomo-api-timeout"`
	MihomoLatencyTestGroup string    `yaml:"mihomo-group-latency-test"`
	MihomoPortForwardGroup string    `yaml:"mihomo-group-port-forward"`
	Checker                Checker   `yaml:"checker"`
	Switcher               Switcher  `yaml:"switcher"`
	ConnClose              ConnClose `yaml:"connclose"`
}

// Cfg 全局配置变量
var Cfg *Config

func init() {
	var err error
	Cfg, err = loadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
}

// loadConfig 从指定文件中读取并解析 YAML 配置
func loadConfig() (*Config, error) {
	data, err := os.ReadFile(C.ConfigFilename)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件出错: %v", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件出错: %v", err)
	}
	return &cfg, nil
}
