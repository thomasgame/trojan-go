# Trojan-Go [![Go Report Card](https://goreportcard.com/badge/github.com/thomasgame/trojan-go)](https://goreportcard.com/report/github.com/thomasgame/trojan-go) [![Downloads](https://img.shields.io/github/downloads/thomasgame/trojan-go/total?label=downloads&logo=github&style=flat-square)](https://img.shields.io/github/downloads/thomasgame/trojan-go/total?label=downloads&logo=github&style=flat-square)

使用 Go 实现的 Trojan 代理，兼容原版 Trojan 协议与配置格式，并在此基础上扩展了多路复用、WebSocket、路由、API、AEAD 和传输层插件等能力。

预编译二进制可执行文件可在 [Release 页面](https://github.com/thomasgame/trojan-go/releases)下载。  
完整配置和使用文档请见 [Trojan-Go 文档](https://p4gefau1t.github.io/trojan-go)。

## 功能概览

Trojan-Go 当前支持：

- 原版 Trojan 协议兼容
- TLS 隧道传输
- UDP 代理
- HTTP / SOCKS 自动识别
- 多路复用（Mux）
- WebSocket over TLS
- 路由分流
- Shadowsocks AEAD 二次加密
- 可插拔传输层插件
- gRPC API
- MySQL 用户认证与流量持久化
- YAML / JSON 配置
- TProxy / NAT / forward / custom 等运行模式

## 快速开始

### 简易模式

服务端：

```shell
sudo ./trojan-go -server -remote 127.0.0.1:80 -local 0.0.0.0:443 -key ./your_key.key -cert ./your_cert.crt -password your_password
```

客户端：

```shell
./trojan-go -client -remote example.com:443 -local 127.0.0.1:1080 -password your_password
```

### 配置文件模式

```shell
./trojan-go -config config.json
```

### URL 模式

```shell
./trojan-go -url 'trojan-go://password@cloudflare.com/?type=ws&path=%2Fpath&host=your-site.com'
```

### Docker

```shell
docker run \
  --name trojan-go \
  -d \
  -v /etc/trojan-go/:/etc/trojan-go \
  --network host \
  thomasgame/trojan-go
```

或显式传入配置路径：

```shell
docker run \
  --name trojan-go \
  -d \
  -v /path/to/host/config:/path/in/container \
  --network host \
  thomasgame/trojan-go \
  /path/in/container/config.json
```

## 构建

> 请确保 Go 版本 >= 1.26

仓库根是工作区根，真正的 Go 模块位于 `src/`。

使用 `make`：

```shell
git clone https://github.com/thomasgame/trojan-go.git
cd trojan-go
make
make test
```

或直接在模块根构建：

```shell
cd src
go build -tags "full"
```

交叉编译示例：

```shell
cd src
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags "full"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags "full"
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -tags "full"
```

仅启用部分功能时，也可以使用 build tags：

```shell
cd src
go build -tags "client"
go build -tags "server mysql"
go build -tags "api client server forward nat other"
```

## 仓库结构

当前仓库已经切换为 `src/` 作为 Go 模块根。可以粗略理解为：

```text
repo/
  README.md
  Makefile
  Dockerfile
  go.work
  docs/
  example/
  src/
    go.mod
    cmd/
    build/
    internal/
    pkg/
    common/
    constant/
    config/
    mobilebind/
```

其中：

- `src/cmd`
  进程入口
- `src/build`
  编译期装配层
- `src/internal/app`
  启动流程、运行模式、CLI 特性、注册入口
- `src/internal/core`
  协议抽象、代理编排、配置、认证、转发原语
- `src/internal/control`
  gRPC 与 CLI 控制面
- `src/internal/infra`
  日志、统计、geodata 等基础设施实现
- `src/internal/transport`
  入站、出站和传输层协议实现
- `src/pkg`
  对外稳定导出的公共能力

## 开发者入口

如果你是第一次阅读这个仓库，建议按下面顺序进入：

1. `src/cmd/trojan-go/main.go`
2. `src/internal/app/bootstrap`
3. `src/build`
4. `src/internal/app/wiring`
5. `src/internal/app/mode`
6. `src/internal/core/proxy`
7. `src/internal/core/tunnel`
8. `src/internal/transport`

更详细的结构说明见：

- `docs/project-structure.md`
- `docs/content/developer/overview.md`
- `docs/content/developer/build.md`

## 配置示例

服务端配置示例：

```json
{
  "run_type": "server",
  "local_addr": "0.0.0.0",
  "local_port": 443,
  "remote_addr": "127.0.0.1",
  "remote_port": 80,
  "password": ["your_awesome_password"],
  "ssl": {
    "cert": "your_cert.crt",
    "key": "your_key.key",
    "sni": "www.your-awesome-domain-name.com"
  }
}
```

客户端配置示例：

```json
{
  "run_type": "client",
  "local_addr": "127.0.0.1",
  "local_port": 1080,
  "remote_addr": "www.your-awesome-domain-name.com",
  "remote_port": 443,
  "password": ["your_awesome_password"]
}
```

等价的 YAML 示例：

```yaml
run-type: client
local-addr: 127.0.0.1
local-port: 1080
remote-addr: www.your-awesome-domain-name.com
remote-port: 443
password:
  - your_awesome_password
```

## 生态与客户端

Trojan-Go 服务端兼容所有原 Trojan 客户端，如 Igniter、ShadowRocket 等。  
以下客户端对 Trojan-Go 扩展特性支持更完整：

- [Qv2ray](https://github.com/Qv2ray/Qv2ray)
- [Igniter-Go](https://github.com/p4gefau1t/trojan-go-android)

## 致谢

- [Trojan](https://github.com/trojan-gfw/trojan)
- [V2Fly](https://github.com/v2fly)
- [utls](https://github.com/refraction-networking/utls)
- [smux](https://github.com/xtaci/smux)
- [go-tproxy](https://github.com/LiamHaworth/go-tproxy)

## Stargazers over time

[![Stargazers over time](https://starchart.cc/thomasgame/trojan-go.svg)](https://starchart.cc/thomasgame/trojan-go)
