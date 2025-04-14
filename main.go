package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"GameConnManager/common/checker"
	"GameConnManager/common/switcher"
)

func main() {
	// 创建连接检查器
	chk, err := checker.New()
	if err != nil {
		fmt.Printf("创建连接检查器失败: %v\n", err)
		os.Exit(1)
	}

	// 创建节点切换器
	swt, err := switcher.New()
	if err != nil {
		fmt.Printf("创建节点切换器失败: %v\n", err)
		os.Exit(1)
	}

	// 使用goroutine运行连接检查器
	go chk.Run()

	// 使用goroutine运行节点切换器
	go swt.Run()

	fmt.Println("GameConnManager 已启动")

	// 等待中断信号以优雅地关闭程序
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("GameConnManager 正在关闭...")
}
