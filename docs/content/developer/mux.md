---
title: "多路复用"
draft: false
weight: 30
---

Trojan-Go 使用 [smux](https://github.com/xtaci/smux) 实现多路复用，并配合 SimpleSocks 协议承载被复用后的代理流量。

当前实现主要位于：

- `src/internal/transport/layer/mux`
- `src/internal/transport/outbound/simplesocks`

当启用多路复用时，客户端会先建立一条正常的 Trojan 连接，但将协议中的 `Command` 字段设置为 `0x7f`（`protocol.Mux`），表示该连接将被升级为复用连接。之后，这条底层连接交由 smux 客户端管理。

服务端识别到该命令后，也会把这条底层连接交给 smux 服务端处理。每一条从 smux 中拆分出来的逻辑连接，再使用 SimpleSocks 协议标明目标地址与目标端口。

因此，多路复用启用后的典型协议栈如下：

| 协议        | 备注     |
| ----------- | -------- |
| 真实流量    |          |
| SimpleSocks |          |
| smux        |          |
| Trojan      | 用于鉴权 |
| 底层协议    | 例如 TLS |
