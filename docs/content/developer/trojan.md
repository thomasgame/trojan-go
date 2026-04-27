---
title: "Trojan协议"
draft: false
weight: 20
---

Trojan-Go 遵循原始 Trojan 协议，协议格式本身可参考 [Trojan 文档](https://trojan-gfw.github.io/trojan/protocol)。

当前实现位于：

- `src/internal/transport/outbound/trojan`

默认情况下，Trojan 协议由 TLS 承载，典型协议栈如下：

| 协议     |
| -------- |
| 真实流量 |
| Trojan   |
| TLS      |
| TCP      |

在 Trojan-Go 中，Trojan 协议既是默认的核心出站协议，也是多种扩展能力的承载基础，例如：

- 在其上叠加 `mux`
- 在其外层叠加 `websocket`
- 在其下层替换默认传输层

因此，理解 Trojan 协议本身之后，再结合 `mux`、`websocket`、`transport plugin` 等专题阅读，会更容易建立完整心智模型。
