# 项目配置
BINARY=ait
BIN_DIR=bin

# Go 相关变量
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod

# 构建标志
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-w -s -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildTime=$(BUILD_TIME)"
BUILD_FLAGS=-trimpath $(LDFLAGS)
WEB_DIR=internal/web

## help: 显示此帮助信息
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## build: 构建当前平台二进制
.PHONY: build
build:
	@echo "正在构建 $(BINARY)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY) ./cmd/$(BINARY)/

## web-build: 构建 Web UI 静态产物
.PHONY: web-build
web-build:
	@echo "正在构建 Web UI..."
	cd $(WEB_DIR) && npm ci && npm run build

## build-web: 构建当前平台二进制并嵌入 Web UI
.PHONY: build-web
build-web: web-build
	@echo "正在构建嵌入 Web UI 的 $(BINARY)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -tags webembed $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY) ./cmd/$(BINARY)/

## run-web: 构建并启动 Web UI（监听 127.0.0.1:18180）
.PHONY: run-web
run-web: build-web
	./$(BIN_DIR)/$(BINARY) --web

## test-web: 验证 Web UI、Go 测试与嵌入构建
.PHONY: test-web
test-web:
	cd $(WEB_DIR) && npm ci && npm run lint && npm run build
	$(GOTEST) ./cmd/$(BINARY) ./internal/web
	$(GOBUILD) -tags webembed $(BUILD_FLAGS) -o /tmp/$(BINARY)-webembed ./cmd/$(BINARY)/

## build-all: 交叉编译所有平台
.PHONY: build-all
build-all:
	@echo "正在交叉编译所有平台..."
	@mkdir -p $(BIN_DIR)
	GOOS=linux   GOARCH=amd64  $(GOBUILD) $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY)-linux-amd64   ./cmd/$(BINARY)/
	GOOS=linux   GOARCH=arm64  $(GOBUILD) $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY)-linux-arm64   ./cmd/$(BINARY)/
	GOOS=linux   GOARCH=386    $(GOBUILD) $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY)-linux-386     ./cmd/$(BINARY)/
	GOOS=linux   GOARCH=arm    $(GOBUILD) $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY)-linux-arm     ./cmd/$(BINARY)/
	GOOS=darwin  GOARCH=amd64  $(GOBUILD) $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY)-darwin-amd64  ./cmd/$(BINARY)/
	GOOS=darwin  GOARCH=arm64  $(GOBUILD) $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY)-darwin-arm64  ./cmd/$(BINARY)/
	GOOS=windows GOARCH=amd64  $(GOBUILD) $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY)-windows-amd64.exe ./cmd/$(BINARY)/

## test: 运行所有测试
.PHONY: test
test:
	@echo "正在运行测试..."
	$(GOTEST) -v ./...

## clean: 清理构建文件
.PHONY: clean
clean:
	@echo "正在清理构建文件..."
	$(GOCLEAN)
	rm -rf $(BIN_DIR)/

## tidy: 格式化代码并整理模块依赖
.PHONY: tidy
tidy:
	@echo "正在格式化代码..."
	$(GOCMD) fmt ./...
	@echo "正在整理模块依赖..."
	$(GOMOD) tidy -v
