package switcher

import (
	"fmt"
	"time"

	"GameConnManager/common/connclose"
	"GameConnManager/config"
)

// Switcher 结构体用于定时同步策略组
type Switcher struct {
	interval time.Duration
}

// New 创建一个新的 Switcher 实例
func New() (*Switcher, error) {
	// 解析 CheckRiseInterval 字符串为 time.Duration 类型
	interval, err := time.ParseDuration(config.Cfg.Switcher.CheckRiseInterval)
	if err != nil {
		return nil, fmt.Errorf("解析检查间隔时间 %q 失败: %v", config.Cfg.Switcher.CheckRiseInterval, err)
	}

	return &Switcher{interval: interval}, nil
}

// Run 启动 Switcher，定期同步策略组
func (s *Switcher) Run() {
	fmt.Println("启动策略组同步器，间隔:", s.interval)

	// 首次执行一次同步，无需等待
	if err := connclose.SyncGroups(true); err != nil {
		fmt.Printf("策略组同步失败: %v\n", err)
	}

	// 创建定时器，每隔指定时间执行一次同步
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for range ticker.C {
		if err := connclose.SyncGroups(true); err != nil {
			fmt.Printf("策略组同步失败: %v\n", err)
		}
	}
}
