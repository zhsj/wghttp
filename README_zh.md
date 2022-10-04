# wghttp

[English](./README.md) \| [中文简体](./README_zh.md)

本软件的作用是转换 WireGuard 为 HTTP & SOCKS5 代理

本软件的 HTTP 和 SOCKS5 服务将会监听相同的端口, 运行在用户空间中
本软件工作无需依赖WireGuard内核模块或TUN网卡设备

在 `exit-mode` 选项被设置为 `remote` 时, 本服务代理本地网络的流量从代理服务器连接到WireGuard网络, 反之亦然

详细的使用, 请查阅文档:  
<https://github.com/zhsj/wghttp/tree/master/docs>