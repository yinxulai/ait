# 项目配置
BINARY_NAME=ait
BIN_DIR=bin

# Go 相关变量
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod

# 构建标志
LDFLAGS=-ldflags "-w -s"
BUILD_FLAGS=-trimpath $(LDFLAGS)

## help: 显示此帮助信息
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## build: 构建二进制文件
.PHONY: build
build:
	@echo "正在构建 $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/

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
