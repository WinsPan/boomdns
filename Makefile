# BoomDNS Makefile
# 用于管理项目的构建、测试、部署等操作

.PHONY: help build clean test deploy docker-build docker-run lint format

# 项目信息
PROJECT_NAME := boomdns
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION := $(shell go version | awk '{print $$3}')

# 构建参数
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# 目录
CMD_DIR := cmd/boomdns
BUILD_DIR := build
DIST_DIR := dist

# 默认目标
.DEFAULT_GOAL := help

help: ## 显示帮助信息
	@echo "BoomDNS 项目管理工具"
	@echo ""
	@echo "可用的目标:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "环境变量:"
	@echo "  GOOS    目标操作系统 (默认: $(GOOS))"
	@echo "  GOARCH  目标架构 (默认: $(GOARCH))"
	@echo "  VERSION 版本号 (默认: $(VERSION))"

# 构建相关
build: ## 构建项目
	@echo "构建 $(PROJECT_NAME) v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(PROJECT_NAME) ./$(CMD_DIR)
	@echo "构建完成: $(BUILD_DIR)/$(PROJECT_NAME)"

build-all: ## 构建所有平台版本
	@echo "构建所有平台版本..."
	@mkdir -p $(DIST_DIR)
	@for os in linux darwin windows; do \
		for arch in amd64 arm64; do \
			echo "构建 $$os/$$arch..."; \
			GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) \
				-o $(DIST_DIR)/$(PROJECT_NAME)-$$os-$$arch \
				./$(CMD_DIR); \
		done; \
	done
	@echo "所有平台版本构建完成"

clean: ## 清理构建文件
	@echo "清理构建文件..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@go clean -cache
	@echo "清理完成"

# 测试相关
test: ## 运行测试
	@echo "运行测试..."
	go test -v ./...

test-race: ## 运行竞态检测测试
	@echo "运行竞态检测测试..."
	go test -race -v ./...

test-coverage: ## 运行测试并生成覆盖率报告
	@echo "运行测试并生成覆盖率报告..."
	@mkdir -p $(BUILD_DIR)
	go test -coverprofile=$(BUILD_DIR)/coverage.out ./...
	go tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "覆盖率报告已生成: $(BUILD_DIR)/coverage.html"

test-bench: ## 运行基准测试
	@echo "运行基准测试..."
	go test -bench=. -benchmem ./...

# 代码质量
lint: ## 运行代码检查
	@echo "运行代码检查..."
	golangci-lint run

format: ## 格式化代码
	@echo "格式化代码..."
	go fmt ./...
	goimports -w .

# 依赖管理
deps: ## 下载依赖
	@echo "下载依赖..."
	go mod download
	go mod tidy

deps-update: ## 更新依赖
	@echo "更新依赖..."
	go get -u ./...
	go mod tidy

# 部署相关
deploy: build ## 部署项目
	@echo "部署 $(PROJECT_NAME)..."
	@cp $(BUILD_DIR)/$(PROJECT_NAME) .
	@echo "部署完成"

docker-build: ## 构建 Docker 镜像
	@echo "构建 Docker 镜像..."
	docker build -t $(PROJECT_NAME):$(VERSION) -f deploy/docker/Dockerfile .
	@echo "Docker 镜像构建完成: $(PROJECT_NAME):$(VERSION)"

docker-run: ## 运行 Docker 容器
	@echo "运行 Docker 容器..."
	docker run -d --name $(PROJECT_NAME) \
		-p 53:53/udp -p 8080:8080 \
		-v $(PWD)/configs:/app/configs \
		-v $(PWD)/data:/app/data \
		$(PROJECT_NAME):$(VERSION)

docker-stop: ## 停止 Docker 容器
	@echo "停止 Docker 容器..."
	docker stop $(PROJECT_NAME) || true
	docker rm $(PROJECT_NAME) || true

# 开发工具
dev: ## 开发模式运行
	@echo "开发模式运行..."
	go run ./$(CMD_DIR)

dev-watch: ## 开发模式运行（文件变化时自动重启）
	@echo "开发模式运行（文件变化时自动重启）..."
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "请安装 air: go install github.com/cosmtrek/air@latest"; \
		go run ./$(CMD_DIR); \
	fi

# 文档相关
docs: ## 生成文档
	@echo "生成文档..."
	@mkdir -p docs
	@echo "文档生成完成"

# 发布相关
release: clean build-all ## 准备发布版本
	@echo "准备发布版本 v$(VERSION)..."
	@mkdir -p $(DIST_DIR)
	@cd $(DIST_DIR) && for file in *; do \
		tar -czf $$file.tar.gz $$file; \
	done
	@echo "发布版本准备完成"

# 安装 air 开发工具
install-dev-tools: ## 安装开发工具
	@echo "安装开发工具..."
	go install github.com/cosmtrek/air@latest
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "开发工具安装完成"

# 显示项目信息
info: ## 显示项目信息
	@echo "项目信息:"
	@echo "  名称: $(PROJECT_NAME)"
	@echo "  版本: $(VERSION)"
	@echo "  构建时间: $(BUILD_TIME)"
	@echo "  Go 版本: $(GO_VERSION)"
	@echo "  目标平台: $(GOOS)/$(GOARCH)"
	@echo ""
	@echo "目录结构:"
	@echo "  源码: internal/"
	@echo "  公共包: pkg/"
	@echo "  命令行: $(CMD_DIR)"
	@echo "  配置: configs/"
	@echo "  文档: docs/"
	@echo "  构建: $(BUILD_DIR)"
	@echo "  发布: $(DIST_DIR)"

# 快速构建和运行
quick: build deploy ## 快速构建和部署
	@echo "快速构建和部署完成"

# 默认目标
all: clean deps test build ## 完整构建流程
	@echo "完整构建流程完成"
