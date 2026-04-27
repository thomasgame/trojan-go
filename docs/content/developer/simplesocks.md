---
title: "SimpleSocks协议"
draft: false
weight: 50
---

SimpleSocks 是一个无鉴权的轻量代理协议，可以把它理解为“去掉认证部分后的简化版 Trojan 头部”。

当前实现位于：

- `src/internal/transport/outbound/simplesocks`

它的主要用途是减少多路复用场景下每条逻辑连接的额外开销，因此：

- 只有启用多路复用之后，被复用出来的逻辑连接才会使用 SimpleSocks
- 在当前实现里，SimpleSocks 总是承载在 smux 之上

头部结构如下：

```text
+-----+------+----------+----------+-----------+
| CMD | ATYP | DST.ADDR | DST.PORT |  Payload  |
+-----+------+----------+----------+-----------+
|  1  |  1   | Variable |    2     |  Variable |
+-----+------+----------+----------+-----------+
```

各字段含义与 Trojan 协议中的目标地址描述基本一致，因此这里只保留最小头部，不再重复引入完整鉴权信息。
