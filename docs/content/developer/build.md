---
title: "编译和自定义Trojan-Go"
draft: false
weight: 10
---

当前仓库的 Go 模块根位于 `src/`，并要求 Go 版本不低于 `1.26`。

编译方式非常简单，可以使用Makefile预设步骤进行编译：

```shell
make
make install #安装systemd服务等，可选
```

或者直接使用Go进行编译：

```shell
cd src && go build -tags "full" #编译完整版本
```

可以通过指定GOOS和GOARCH环境变量，指定交叉编译的目标操作系统和架构，例如

```shell
cd src && GOOS=windows GOARCH=386 go build -tags "full" #windows x86
cd src && GOOS=linux GOARCH=arm64 go build -tags "full" #linux arm64
```

发布构建默认通过仓库根的 `Makefile` 和 CI 流程完成。

Trojan-Go 的大多数模块是可插拔的。当前 build tag 的装配入口位于 `src/build/`。如果你不需要其中某些功能，或者需要缩小可执行文件体积，可以使用构建标签（tags）进行模块裁剪，例如

```shell
cd src && go build -tags "full" #编译所有模块
cd src && go build -tags "client" -trimpath -ldflags="-s -w -buildid=" #只有客户端功能，且去除符号表缩小体积
cd src && go build -tags "server mysql" #只有服务端和mysql支持
```

使用full标签等价于

```shell
cd src && go build -tags "api client server forward nat other"
```
