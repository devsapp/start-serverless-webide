# The current OS type.
OS = $(shell uname)

# The current architecture type.
ARCH = $(shell uname -m)

ifeq (${OS}, Darwin)
	OS = darwin
else ifeq (${OS}, Linux)
	OS = linux
else 
	$(error Unspported OS ${OS})
endif

ifeq (${ARCH}, x86_64)
	ARCH = amd64
endif

# Binary name
BINARY = webide-server

# The config file loaded by the BINARY process.
CONFIG_FILE = "config.yaml"

VSCODE_SERVER_VERSION = 1.67.0

VSCODE_SERVER = openvscode-server-v${VSCODE_SERVER_VERSION}-${OS}-${ARCH}

# Release target, which is usually passed from the command line.
# For example, make release TARGET=fc will build the artifacts for FC runtime.
TARGET = ""

ifeq (${TARGET}, fc)
	CONFIG_FILE = "fc.yaml"
endif

# 如果对应版本的 vsocode server 不存在，则下载
third_party:
	@if [ -d third_party/${VSCODE_SERVER} ]; then echo "vscode server is ready"; else mkdir -p third_party/${VSCODE_SERVER} && curl https://s-public-packages.oss-cn-hangzhou.aliyuncs.com/openvscode-server/${VSCODE_SERVER}.tar.gz -o third_party/${VSCODE_SERVER}.tar.gz && tar zxvf third_party/${VSCODE_SERVER}.tar.gz -C third_party; fi

build: third_party
	GOOS=${OS} GOARCH=${ARCH} CGO_ENABLED=0 go build -o target/${BINARY} ./cmd/...
	cp configs/dev.yaml target/config.yaml

test: third_party
	GOOS=${OS} GOARCH=${ARCH} CGO_ENABLED=0 go test -v ./...

# Run: make release to build artifacts for FC runtime.
release:
	go clean
	rm -rf ./target/*
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o target/${BINARY} ./cmd/...
	cp configs/fc.yaml target/config.yaml

download-layer:
	@if [ -e /tmp/openvscode-server-v${VSCODE_SERVER_VERSION}-linux-amd64.tar.gz ]; then echo "vscode server is ready"; else mkdir -p /tmp && curl https://s-public-packages.oss-cn-hangzhou.aliyuncs.com/openvscode-server/openvscode-server-v${VSCODE_SERVER_VERSION}-linux-amd64.tar.gz -o /tmp/openvscode-server-v${VSCODE_SERVER_VERSION}-linux-amd64.tar.gz; fi
	@if [ -e /tmp/fc-layer/openvscode-server ]; then echo "openvscode-server dir is ready"; else mkdir -p /tmp/fc-layer/openvscode-server && tar zxf /tmp/openvscode-server-v${VSCODE_SERVER_VERSION}-linux-amd64.tar.gz -C /tmp/fc-layer/openvscode-server --strip-components 1; fi

layer: download-layer
	s layer publish --layer-name openvscode-server --code /tmp/fc-layer
	
clean:
	go clean
	rm -rf ./target/*

.PHONY: third_party build test release layer clean

