package checker

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"GameConnManager/common/connclose"
	"GameConnManager/config"
	C "GameConnManager/constant"
)

// Checker 结构体包含连接检查功能所需的配置
type Checker struct {
	client *http.Client
}

// New 创建一个新的连接检查器
func New() (*Checker, error) {
	// 1. 解析 timeout 字符串为 time.Duration 类型
	timeout, err := time.ParseDuration(config.Cfg.Checker.Timeout)
	if err != nil {
		return nil, fmt.Errorf("解析超时 %q 失败: %v", config.Cfg.Checker.Timeout, err)
	}

	// 2. 解析代理地址
	proxyURL, err := url.Parse(config.Cfg.Checker.Proxy)
	if err != nil {
		return nil, fmt.Errorf("解析代理地址 %q 失败: %v", config.Cfg.Checker.Proxy, err)
	}

	// 3. 构造 HTTP Transport 和 Client
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	return &Checker{client: client}, nil
}

// Run 开始运行连接检查循环
func (c *Checker) Run() {
	interval, err := time.ParseDuration(config.Cfg.Checker.Interval)
	if err != nil {
		fmt.Printf("解析间隔时间 %q 失败: %v\n", config.Cfg.Checker.Interval, err)
		return
	}
	fmt.Println("开始连接检查，间隔:", interval)

	// 创建定时器，每隔指定时间执行一次检查
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		resp, err := c.client.Get(C.SpeedtestUrl)
		if err != nil {
			fmt.Printf("请求 %s 失败: %v\n", C.SpeedtestUrl, err)
			// 当请求失败或超时时，关闭连接
			if closeErr := connclose.CloseCurrentConnection(true); closeErr != nil {
				fmt.Printf("关闭连接失败: %v\n", closeErr)
			}
		} else {
			resp.Body.Close()
		}
	}
}
