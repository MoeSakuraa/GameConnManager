package connclose

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"GameConnManager/config"
	C "GameConnManager/constant"
)

// GroupInfo 表示策略组的信息
type GroupInfo struct {
	All  []string `json:"all"`
	Now  string   `json:"now"`
	Type string   `json:"type"`
}

// connection 表示从Mihomo API返回的连接信息的简化结构
type connection struct {
	ID       string `json:"id"`
	Metadata struct {
		Type        string `json:"type"`
		InboundIP   string `json:"inboundIP"`
		InboundPort string `json:"inboundPort"`
	} `json:"metadata"`
}

// connectionsResponse 表示API响应的结构
type connectionsResponse struct {
	Connections []connection `json:"connections"`
}

// getGroupStatus 获取策略组状态
func getGroupStatus(groupName string) (*GroupInfo, error) {
	// 构建请求路径
	path := fmt.Sprintf("%s%s", C.GroupPath, url.PathEscape(groupName))

	// 发送请求
	resp, err := doRequest("GET", path, http.StatusOK)
	if err != nil {
		return nil, fmt.Errorf("获取策略组 %s 状态失败: %v", groupName, err)
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应内容失败: %v", err)
	}

	// 解析JSON响应
	var info GroupInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("解析JSON响应失败: %v", err)
	}

	return &info, nil
}

// findIndex 找出元素在数组中的索引位置，不存在则返回-1
func findIndex(arr []string, element string) int {
	for i, v := range arr {
		if v == element {
			return i
		}
	}
	return -1
}

// updateGroupSelection 更新策略组选择
func updateGroupSelection(groupName string, proxyName string) error {
	// 构建请求路径
	path := fmt.Sprintf("%s%s", C.ProxiesPath, url.PathEscape(groupName))

	// 构建请求体数据
	data := map[string]string{"name": proxyName}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化JSON数据失败: %v", err)
	}

	// 创建HTTP客户端
	client, err := createHTTPClient()
	if err != nil {
		return err
	}

	// 构建完整URL
	fullUrl := config.Cfg.MihomoApiUrl + path

	// 创建请求
	req, err := http.NewRequest("PUT", fullUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add(C.AuthHeader, C.BearerPrefix+config.Cfg.MihomoApiToken)

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("执行HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("API返回非预期状态码: %d", resp.StatusCode)
	}

	return nil
}

// SyncGroups 同步"自动回退"和"端口转发用"策略组的选择
func SyncGroups(closeCurrentConnection bool) error {
	// 获取"自动回退"策略组状态
	fallbackGroup := config.Cfg.MihomoLatencyTestGroup
	fallbackInfo, err := getGroupStatus(fallbackGroup)
	if err != nil {
		return fmt.Errorf("获取 %s 策略组状态失败: %v", fallbackGroup, err)
	}

	// 查找"now"在"all"数组中的位置
	fallbackIndex := findIndex(fallbackInfo.All, fallbackInfo.Now)
	if fallbackIndex == -1 {
		return fmt.Errorf("%s 策略组的当前选择 %s 不在可用列表中", fallbackGroup, fallbackInfo.Now)
	}

	// 检查是否为默认节点（第一个元素）
	// isNotDefault := fallbackIndex == 0
	// fmt.Printf("%s 当前选择: %s (索引: %d, 是否默认节点: %v)\n",
	// 	fallbackGroup, fallbackInfo.Now, fallbackIndex, isNotDefault)

	// 获取"端口转发用"策略组状态
	portForwardGroup := config.Cfg.MihomoPortForwardGroup
	portForwardInfo, err := getGroupStatus(portForwardGroup)
	if err != nil {
		return fmt.Errorf("获取 %s 策略组状态失败: %v", portForwardGroup, err)
	}

	// 查找"now"在"all"数组中的位置
	portForwardIndex := findIndex(portForwardInfo.All, portForwardInfo.Now)
	if portForwardIndex == -1 {
		return fmt.Errorf("%s 策略组的当前选择 %s 不在可用列表中", portForwardGroup, portForwardInfo.Now)
	}
	// fmt.Printf("%s 当前选择: %s (索引: %d)\n",
	// 	portForwardGroup, portForwardInfo.Now, portForwardIndex)

	// 比较两个策略组的选择位置
	if fallbackIndex != portForwardIndex && fallbackIndex < len(portForwardInfo.All) {
		// 如果位置不同且fallback的索引有效，则更新"端口转发用"策略组
		proxyToSelect := portForwardInfo.All[fallbackIndex]
		fmt.Printf("同步选择: 将 %s 的选择更新为 %s (索引 %d)\n",
			portForwardGroup, proxyToSelect, fallbackIndex)

		err := updateGroupSelection(portForwardGroup, proxyToSelect)
		if err != nil {
			return fmt.Errorf("更新 %s 策略组选择失败: %v", portForwardGroup, err)
		}
		fmt.Printf("已成功更新 %s 策略组选择, 将关闭现有连接以切换到新节点\n", portForwardGroup)
		// 关闭连接以应用新的策略组设置，传入false防止递归调用
		if closeCurrentConnection {
			err := CloseCurrentConnection(true)
			if err != nil {
				return fmt.Errorf("关闭连接失败: %s", err)
			}
		}
	} else {
		// fmt.Printf("两个策略组已同步，无需更新\n")
	}

	return nil
}

// createHTTPClient 创建一个带有适当超时设置的HTTP客户端
func createHTTPClient() (*http.Client, error) {
	// 解析API超时设置
	timeout, err := time.ParseDuration(config.Cfg.MihomoApiTimeout)
	if err != nil {
		return nil, fmt.Errorf("解析API超时设置失败: %v", err)
	}

	// 创建HTTP客户端
	return &http.Client{
		Timeout: timeout,
	}, nil
}

// doRequest 执行HTTP请求并返回响应
func doRequest(method, path string, expectStatusCodes ...int) (*http.Response, error) {
	client, err := createHTTPClient()
	if err != nil {
		return nil, err
	}

	// 构建完整URL
	url := config.Cfg.MihomoApiUrl + path

	// 创建请求
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %v", err)
	}

	// 使用常量添加认证头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add(C.AuthHeader, C.BearerPrefix+config.Cfg.MihomoApiToken)

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("执行HTTP请求失败: %v", err)
	}

	// 检查响应状态码
	var statusOK bool
	if len(expectStatusCodes) == 0 {
		statusOK = resp.StatusCode == http.StatusOK
	} else {
		statusOK = slices.Contains(expectStatusCodes, resp.StatusCode)
	}

	if !statusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("API返回非预期状态码: %d", resp.StatusCode)
	}

	return resp, nil
}

// testGroupDelay 测试指定策略组的延迟
func testGroupDelay() error {
	// 构建查询参数
	// 解析timeout字符串为Duration对象
	timeoutDuration, err := time.ParseDuration(config.Cfg.Checker.Timeout)
	// 获取毫秒数并转换为字符串
	timeoutMs := strconv.FormatInt(timeoutDuration.Milliseconds(), 10)
	if err != nil {
		return fmt.Errorf("解析超时设置失败: %v", err)
	}
	queryParams := url.Values{}
	queryParams.Add("url", C.SpeedtestUrl)
	queryParams.Add("timeout", timeoutMs)

	// 构建请求路径
	path := fmt.Sprintf("%s%s/delay?%s",
		C.GroupPath,
		url.PathEscape(config.Cfg.MihomoLatencyTestGroup),
		queryParams.Encode())

	// 发送请求
	resp, err := doRequest("GET", path, http.StatusOK)
	if err != nil {
		return fmt.Errorf("测试策略组延迟失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应内容失败: %v", err)
	}

	// 打印响应内容
	fmt.Printf("策略组 '%s' 的延迟测试响应: %s\n", config.Cfg.MihomoLatencyTestGroup, string(body))

	return nil
}

// getConnections 获取当前所有连接信息
func getConnections() ([]connection, error) {
	resp, err := doRequest("GET", C.ConnectionsPath, http.StatusOK)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应内容失败: %v", err)
	}

	// 解析JSON响应
	var response connectionsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析JSON响应失败: %v", err)
	}

	return response.Connections, nil
}

// deleteConnection 通过ID删除指定连接
func deleteConnection(id string) error {
	_, err := doRequest("DELETE", C.ConnectionsPath+"/"+id, http.StatusOK, http.StatusNoContent)
	return err
}

// CloseCurrentConnection 根据配置查找并关闭符合条件的连接
// 条件: 类型为TypeTunnel且入站IP和端口与配置匹配
func CloseCurrentConnection(syncGroups bool) error {
	// 获取所有连接
	connections, err := getConnections()
	if err != nil {
		return fmt.Errorf("获取连接信息失败: %v", err)
	}

	// 从配置获取目标IP和端口
	targetIP := config.Cfg.ConnClose.ListenIP
	targetPort := strconv.Itoa(config.Cfg.ConnClose.ListenPort)

	// 用于标记是否找到了符合条件的连接
	found := false

	// 遍历所有连接，寻找符合条件的连接
	for _, conn := range connections {
		// 检查是否符合所有条件:
		// 1. 类型为TypeTunnel
		// 2. 入站IP匹配
		// 3. 入站端口匹配
		if conn.Metadata.Type == C.TypeTunnel &&
			conn.Metadata.InboundIP == targetIP &&
			conn.Metadata.InboundPort == targetPort {

			// 找到匹配的连接，尝试关闭它
			if err := deleteConnection(conn.ID); err != nil {
				return fmt.Errorf("关闭连接 %s 失败: %v", conn.ID, err)
			}
			found = true
			fmt.Printf("成功关闭连接 ID: %s\n", conn.ID)
			// 在关闭连接后，测试策略组的延迟
			if err := testGroupDelay(); err != nil {
				fmt.Printf("警告: 测试策略组延迟失败: %v\n", err)
			}
			// 根据传入参数决定是否同步策略组
			if syncGroups {
				if err := SyncGroups(true); err != nil {
					fmt.Printf("警告: 同步策略组失败: %v\n", err)
				}
			}
		}
	}

	if !found {
		// 即使无匹配的连接可以关闭，也测延迟并同步策略组
		if err := testGroupDelay(); err != nil {
			fmt.Printf("警告: 测试策略组延迟失败: %v\n", err)
		}
		if syncGroups {
			if err := SyncGroups(false); err != nil {
				fmt.Printf("警告: 同步策略组失败: %v\n", err)
			}
		}
		return fmt.Errorf("未找到符合条件的连接(类型: %s, 入站IP: %s, 入站端口: %s)",
			C.TypeTunnel, targetIP, targetPort)
	}

	return nil
}
