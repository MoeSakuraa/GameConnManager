# GameConnManager
使用 Golang 编写的基于 mihomo API 的游戏连接管理器，可以保证使用机场线路打游戏时不掉线
## 特性
- 自动在 Hysteria2 的 QUIC 流上测试 HTTP 延迟，当出现连接超时时自动打断端口转发，迅速连接迁移
- 自动故障转移，避免单点故障
- 仅使用香港和日本节点，UDP 会话出问题时，迁移时间小于 100ms；单点故障时，迁移时间小于 1s（且在配置文件中配置更激进的超时可进一步提升迁移速度）
- 仅使用 mihomo API，不需要额外权限
## 使用场景
有的机场（即使是超一线机场）节点提供的 UDP 会话有 BUG，会出现某一段时间代理协议显示 UDP 会话状态正常，但是数据完全没有被发送，导致游戏掉线，此插件可以避免此情况的发生，原理是通过自动 HTTP 检测以及 QUIC 流的连接迁移
## 前提条件
1. 自己拥有一台落地服务器，部署 sing-box，配置文件参考 `example\sing-box\config.json`
## 安装
```bash
git clone https://github.com/MoeSakuraa/GameConnManager.git
go build
```
## 使用方法
1. mihomo 配置文件参考 `example\mihomo\config.yaml`
2. 运行即可，程序会读取当前目录 `config.yaml`
## 授权
本项目采用 BSD 3-Clause 许可证进行授权。