---
title: "Websocket"
draft: false
weight: 40
---

Trojan-Go 支持使用 WebSocket 承载 Trojan 流量，以便在 CDN 或常见 Web 基础设施后面转发代理连接。

当前实现主要位于：

- `src/internal/transport/layer/websocket`
- `src/internal/transport/outbound/shadowsocks`

由于 CDN 可以看到 WebSocket 的传输内容，而 Trojan 协议本身并不提供额外的内容加密，因此如果 WebSocket 流量经过不可信中间层，通常建议叠加 Shadowsocks AEAD。

**如果你使用中国境内运营商提供的 CDN，建议开启 AEAD 加密。**

启用 AEAD 后，WebSocket 承载的数据会先经过 Shadowsocks AEAD 再进入 Trojan 协议层。

典型协议栈如下：

| 协议        | 备注             |
| ----------- | ---------------- |
| 真实流量    |                  |
| SimpleSocks | 如果开启多路复用 |
| smux        | 如果开启多路复用 |
| Trojan      |                  |
| Shadowsocks | 如果开启 AEAD    |
| Websocket   |                  |
| 传输层协议  | 例如 TLS         |

因此，WebSocket 在 Trojan-Go 中更适合作为一个“可叠加传输层”，而不是独立的代理协议本身。
