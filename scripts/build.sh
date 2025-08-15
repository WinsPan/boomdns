#!/usr/bin/env bash

# BoomDNS 构建脚本
# 支持多平台构建、性能优化、版本管理

set -euo pipefail

# 颜色定义
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly PURPLE='\033[0;35m'
readonly CYAN='\033[0;36m'
readonly NC='\033[0m' # No Color

# 项目信息
readonly PROJECT_NAME="boomdns"
readonly PROJECT_VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
readonly BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
readonly GO_VERSION=$(go version | awk '{print $3}')

# 构建配置
readonly BUILD_DIR="build"
readonly DIST_DIR="dist"
readonly CMD_DIR="cmd/boomdns"

# 支持的平台
readonly PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "linux/arm"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
)

# 日志函数
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

info() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] INFO: $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
}

success() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] SUCCESS: $1${NC}"
}

# 检查依赖
check_dependencies() {
    log "检查构建依赖..."
    
    # 检查 Go 版本
    if ! command -v go > /dev/null; then
        error "Go 未安装或不在 PATH 中"
        exit 1
    fi
    
    local go_version
    go_version=$(go version | awk '{print $3}' | sed 's/go//')
    local major_version
    major_version=$(echo "$go_version" | cut -d. -f1)
    local minor_version
    minor_version=$(echo "$go_version" | cut -d. -f2)
    
    if [[ "$major_version" -lt 1 ]] || [[ "$major_version" -eq 1 && "$minor_version" -lt 21 ]]; then
        error "需要 Go 1.21 或更高版本，当前版本: $go_version"
        exit 1
    fi
    
    success "Go 版本检查通过: $go_version"
    
    # 检查其他工具
    local tools=("git" "make")
    for tool in "${tools[@]}"; do
        if ! command -v "$tool" > /dev/null; then
            warn "工具未找到: $tool"
        else
            info "工具已找到: $tool"
        fi
    done
}

# 清理构建目录
clean_build() {
    log "清理构建目录..."
    
    if [[ -d "$BUILD_DIR" ]]; then
        rm -rf "$BUILD_DIR"
        info "已清理构建目录: $BUILD_DIR"
    fi
    
    if [[ -d "$DIST_DIR" ]]; then
        rm -rf "$DIST_DIR"
        info "已清理发布目录: $DIST_DIR"
    fi
    
    # 清理 Go 缓存
    go clean -cache -modcache -testcache
    info "已清理 Go 缓存"
}

# 创建构建目录
create_directories() {
    log "创建构建目录..."
    
    mkdir -p "$BUILD_DIR"
    mkdir -p "$DIST_DIR"
    
    success "构建目录创建完成"
}

# 下载依赖
download_dependencies() {
    log "下载 Go 依赖..."
    
    go mod download
    go mod tidy
    
    success "依赖下载完成"
}

# 运行测试
run_tests() {
    log "运行测试..."
    
    # 运行单元测试
    info "运行单元测试..."
    go test -v ./...
    
    # 运行竞态检测
    info "运行竞态检测..."
    go test -race -v ./...
    
    # 生成测试覆盖率报告
    info "生成测试覆盖率报告..."
    mkdir -p "$BUILD_DIR"
    go test -coverprofile="$BUILD_DIR/coverage.out" ./...
    go tool cover -html="$BUILD_DIR/coverage.out" -o "$BUILD_DIR/coverage.html"
    
    success "测试完成，覆盖率报告: $BUILD_DIR/coverage.html"
}

# 代码质量检查
code_quality_check() {
    log "代码质量检查..."
    
    # 格式化代码
    info "格式化代码..."
    go fmt ./...
    
    # 运行 golangci-lint（如果可用）
    if command -v golangci-lint > /dev/null; then
        info "运行 golangci-lint..."
        golangci-lint run
    else
        warn "golangci-lint 未安装，跳过代码质量检查"
    fi
    
    # 检查 Go 模块
    info "检查 Go 模块..."
    go mod verify
    
    success "代码质量检查完成"
}

# 构建单个平台
build_platform() {
    local platform=$1
    local os_arch
    IFS='/' read -r os arch <<< "$platform"
    
    log "构建平台: $os/$arch"
    
    local output_name="$PROJECT_NAME-$os-$arch"
    if [[ "$os" == "windows" ]]; then
        output_name="$output_name.exe"
    fi
    
    local build_flags=(
        "-ldflags"
        "-X main.Version=$PROJECT_VERSION -X main.BuildTime=$BUILD_TIME -s -w"
        "-o"
        "$DIST_DIR/$output_name"
    )
    
    # 设置环境变量
    export GOOS="$os"
    export GOARCH="$arch"
    export CGO_ENABLED=0  # 静态链接
    
    # 执行构建
    if go build "${build_flags[@]}" "./$CMD_DIR"; then
        success "平台 $os/$arch 构建成功: $output_name"
        
        # 计算文件大小
        local file_size
        file_size=$(du -h "$DIST_DIR/$output_name" | cut -f1)
        info "文件大小: $file_size"
        
        # 创建压缩包
        local archive_name="$output_name.tar.gz"
        tar -czf "$DIST_DIR/$archive_name" -C "$DIST_DIR" "$output_name"
        local archive_size
        archive_size=$(du -h "$DIST_DIR/$archive_name" | cut -f1)
        info "压缩包大小: $archive_size"
        
        return 0
    else
        error "平台 $os/$arch 构建失败"
        return 1
    fi
}

# 构建所有平台
build_all_platforms() {
    log "开始构建所有平台..."
    
    local success_count=0
    local total_count=${#PLATFORMS[@]}
    
    for platform in "${PLATFORMS[@]}"; do
        if build_platform "$platform"; then
            ((success_count++))
        fi
    done
    
    log "构建完成: $success_count/$total_count 个平台成功"
    
    if [[ $success_count -eq $total_count ]]; then
        success "所有平台构建成功！"
    else
        warn "部分平台构建失败"
    fi
}

# 构建当前平台
build_current() {
    log "构建当前平台..."
    
    local current_os
    current_os=$(go env GOOS)
    local current_arch
    current_arch=$(go env GOARCH)
    
    info "当前平台: $current_os/$current_arch"
    
    local output_name="$PROJECT_NAME"
    if [[ "$current_os" == "windows" ]]; then
        output_name="$output_name.exe"
    fi
    
    local build_flags=(
        "-ldflags"
        "-X main.Version=$PROJECT_VERSION -X main.BuildTime=$BUILD_TIME"
        "-o"
        "$BUILD_DIR/$output_name"
    )
    
    if go build "${build_flags[@]}" "./$CMD_DIR"; then
        success "当前平台构建成功: $BUILD_DIR/$output_name"
        
        # 显示文件信息
        local file_size
        file_size=$(du -h "$BUILD_DIR/$output_name" | cut -f1)
        info "文件大小: $file_size"
        
        # 复制到根目录
        cp "$BUILD_DIR/$output_name" .
        info "已复制到根目录: $output_name"
        
        return 0
    else
        error "当前平台构建失败"
        return 1
    fi
}

# 性能基准测试
run_benchmarks() {
    log "运行性能基准测试..."
    
    mkdir -p "$BUILD_DIR"
    
    # 运行基准测试
    go test -bench=. -benchmem ./... > "$BUILD_DIR/benchmark.out" 2>&1 || true
    
    # 显示结果
    if [[ -f "$BUILD_DIR/benchmark.out" ]]; then
        info "基准测试结果:"
        cat "$BUILD_DIR/benchmark.out"
        success "基准测试完成，结果保存到: $BUILD_DIR/benchmark.out"
    else
        warn "基准测试未生成结果"
    fi
}

# 生成构建报告
generate_build_report() {
    log "生成构建报告..."
    
    local report_file="$BUILD_DIR/build-report.txt"
    
    cat > "$report_file" << EOF
BoomDNS 构建报告
================

构建时间: $BUILD_TIME
项目版本: $PROJECT_VERSION
Go 版本: $GO_VERSION

构建配置:
- 构建目录: $BUILD_DIR
- 发布目录: $DIST_DIR
- 支持平台: ${#PLATFORMS[@]} 个

构建结果:
EOF
    
    # 添加构建文件信息
    if [[ -d "$DIST_DIR" ]]; then
        echo "" >> "$report_file"
        echo "发布文件:" >> "$report_file"
        ls -la "$DIST_DIR" >> "$report_file" 2>/dev/null || true
    fi
    
    # 添加测试覆盖率信息
    if [[ -f "$BUILD_DIR/coverage.out" ]]; then
        echo "" >> "$report_file"
        echo "测试覆盖率:" >> "$report_file"
        go tool cover -func="$BUILD_DIR/coverage.out" >> "$report_file" 2>/dev/null || true
    fi
    
    success "构建报告已生成: $report_file"
}

# 显示帮助信息
show_help() {
    cat << EOF
BoomDNS 构建脚本

用法: $0 [选项]

选项:
  -h, --help          显示此帮助信息
  -c, --clean         清理构建目录
  -t, --test          运行测试
  -q, --quality       代码质量检查
  -b, --bench         运行基准测试
  -a, --all           构建所有平台
  -p, --platform      构建指定平台 (格式: os/arch)
  -r, --report        生成构建报告

示例:
  $0                    # 构建当前平台
  $0 --all             # 构建所有平台
  $0 --test            # 运行测试
  $0 --clean           # 清理构建目录
  $0 --platform linux/amd64  # 构建指定平台

支持的平台:
  ${PLATFORMS[*]}

环境变量:
  GOOS     目标操作系统
  GOARCH   目标架构
  CGO_ENABLED  是否启用 CGO (默认: 0)

EOF
}

# 主函数
main() {
    local clean_only=false
    local test_only=false
    local quality_only=false
    local benchmark_only=false
    local build_all=false
    local specific_platform=""
    local generate_report=false
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -c|--clean)
                clean_only=true
                shift
                ;;
            -t|--test)
                test_only=true
                shift
                ;;
            -q|--quality)
                quality_only=true
                shift
                ;;
            -b|--bench)
                benchmark_only=true
                shift
                ;;
            -a|--all)
                build_all=true
                shift
                ;;
            -p|--platform)
                specific_platform="$2"
                shift 2
                ;;
            -r|--report)
                generate_report=true
                shift
                ;;
            *)
                error "未知选项: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    # 显示构建信息
    log "开始 BoomDNS 构建..."
    info "项目名称: $PROJECT_NAME"
    info "项目版本: $PROJECT_VERSION"
    info "构建时间: $BUILD_TIME"
    info "Go 版本: $GO_VERSION"
    echo ""
    
    # 检查依赖
    check_dependencies
    echo ""
    
    # 清理构建目录
    clean_build
    echo ""
    
    # 如果只是清理，则退出
    if [[ "$clean_only" == true ]]; then
        success "清理完成"
        exit 0
    fi
    
    # 创建构建目录
    create_directories
    echo ""
    
    # 下载依赖
    download_dependencies
    echo ""
    
    # 如果只是测试，则运行测试后退出
    if [[ "$test_only" == true ]]; then
        run_tests
        exit 0
    fi
    
    # 如果只是代码质量检查，则运行检查后退出
    if [[ "$quality_only" == true ]]; then
        code_quality_check
        exit 0
    fi
    
    # 如果只是基准测试，则运行测试后退出
    if [[ "$benchmark_only" == true ]]; then
        run_benchmarks
        exit 0
    fi
    
    # 运行测试
    run_tests
    echo ""
    
    # 代码质量检查
    code_quality_check
    echo ""
    
    # 构建
    if [[ "$build_all" == true ]]; then
        build_all_platforms
    elif [[ -n "$specific_platform" ]]; then
        build_platform "$specific_platform"
    else
        build_current
    fi
    echo ""
    
    # 运行基准测试
    run_benchmarks
    echo ""
    
    # 生成构建报告
    if [[ "$generate_report" == true ]]; then
        generate_build_report
        echo ""
    fi
    
    # 显示构建结果
    log "构建完成！"
    if [[ -d "$BUILD_DIR" ]]; then
        info "构建文件位置: $BUILD_DIR"
    fi
    if [[ -d "$DIST_DIR" ]]; then
        info "发布文件位置: $DIST_DIR"
    fi
    
    success "BoomDNS 构建成功完成！"
}

# 脚本入口
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
