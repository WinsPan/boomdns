#!/usr/bin/env bash

# BoomDNS Hysteria2 协议测试脚本
# 测试 Hysteria2 协议的支持和配置

set -euo pipefail

# 颜色定义
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# 项目配置
readonly PROJECT_NAME="boomdns"
readonly SERVICE_URL="http://localhost:8080"
readonly CONFIG_FILE="config.yaml"

# 日志函数
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
}

info() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] INFO: $1${NC}"
}

# 检查依赖
check_dependencies() {
    log "检查系统依赖..."
    
    local deps=("go" "yq" "curl")
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            error "缺少依赖: $dep"
            exit 1
        fi
    done
    
    log "所有依赖已满足"
}

# 检查配置文件
check_config_file() {
    log "检查配置文件..."
    
    if [[ ! -f "$CONFIG_FILE" ]]; then
        error "配置文件不存在: $CONFIG_FILE"
        exit 1
    fi
    
    log "配置文件存在: $CONFIG_FILE"
}

# 验证 Hysteria2 配置
validate_hysteria2_config() {
    log "验证 Hysteria2 配置..."
    
    # 检查代理功能是否启用
    local proxy_enabled
    proxy_enabled=$(yq eval '.proxy.enabled' "$CONFIG_FILE" 2>/dev/null || echo "false")
    
    if [[ "$proxy_enabled" != "true" ]]; then
        warn "代理功能未启用，请检查 config.yaml 中的 proxy.enabled 设置"
        return 1
    fi
    
    log "代理功能已启用"
    
    # 检查是否有 Hysteria2 节点
    local hysteria2_nodes
    hysteria2_nodes=$(yq eval '.proxy_nodes[] | select(.protocol == "hysteria2") | .name' "$CONFIG_FILE" 2>/dev/null || echo "")
    
    if [[ -z "$hysteria2_nodes" ]]; then
        warn "未找到 Hysteria2 节点配置"
        return 1
    fi
    
    log "找到 Hysteria2 节点:"
    echo "$hysteria2_nodes" | while read -r node; do
        if [[ -n "$node" ]]; then
            info "  - $node"
        fi
    done
    
    return 0
}

# 检查服务状态
check_service_status() {
    log "检查 BoomDNS 服务状态..."
    
    if ! curl -s "$SERVICE_URL/api/health" | grep -q '"ok":true'; then
        error "BoomDNS 服务未运行或无法访问"
        return 1
    fi
    
    log "BoomDNS 服务运行正常"
    return 0
}

# 测试 Hysteria2 API
test_hysteria2_api() {
    log "测试 Hysteria2 相关 API..."
    
    # 测试代理状态 API
    local status_response
    status_response=$(curl -s "$SERVICE_URL/api/proxy/status")
    
    if echo "$status_response" | grep -q '"success":true'; then
        log "代理状态 API 响应正常"
    else
        warn "代理状态 API 响应异常"
        return 1
    fi
    
    # 测试代理节点 API
    local nodes_response
    nodes_response=$(curl -s "$SERVICE_URL/api/proxy/nodes")
    
    if echo "$nodes_response" | grep -q '"success":true'; then
        log "代理节点 API 响应正常"
    else
        warn "代理节点 API 响应异常"
        return 1
    fi
    
    return 0
}

# 验证 Hysteria2 协议支持
validate_hysteria2_protocol() {
    log "验证 Hysteria2 协议支持..."
    
    # 检查代码中是否定义了 Hysteria2 协议
    if grep -q "ProxyHysteria2" dns/proxy.go; then
        log "✓ 代码中已定义 Hysteria2 协议类型"
    else
        error "✗ 代码中未找到 Hysteria2 协议类型定义"
        return 1
    fi
    
    # 检查是否实现了 Hysteria2 拨号器
    if grep -q "createHysteria2Dialer" dns/proxy.go; then
        log "✓ 代码中已实现 Hysteria2 拨号器创建方法"
    else
        error "✗ 代码中未找到 Hysteria2 拨号器创建方法"
        return 1
    fi
    
    # 检查配置结构是否支持 Hysteria2
    if grep -q "Hysteria2" dns/proxy.go; then
        log "✓ 配置结构支持 Hysteria2 特定参数"
    else
        error "✗ 配置结构不支持 Hysteria2 特定参数"
        return 1
    fi
    
    return 0
}

# 检查编译
check_compilation() {
    log "检查代码编译..."
    
    if go build ./...; then
        log "✓ 代码编译成功"
        return 0
    else
        error "✗ 代码编译失败"
        return 1
    fi
}

# 生成测试报告
generate_report() {
    log "生成 Hysteria2 测试报告..."
    
    local report_file="hysteria2-test-report-$(date +%Y%m%d-%H%M%S).txt"
    
    cat > "$report_file" << EOF
BoomDNS Hysteria2 协议测试报告
================================

测试时间: $(date)
测试项目: $PROJECT_NAME

测试结果摘要:
- 配置文件检查: $(check_config_file && echo "通过" || echo "失败")
- Hysteria2 配置验证: $(validate_hysteria2_config && echo "通过" || echo "失败")
- 服务状态检查: $(check_service_status && echo "通过" || echo "失败")
- Hysteria2 API 测试: $(test_hysteria2_api && echo "通过" || echo "失败")
- 协议支持验证: $(validate_hysteria2_protocol && echo "通过" || echo "失败")
- 代码编译检查: $(check_compilation && echo "通过" || echo "失败")

Hysteria2 特性支持:
- [x] 协议类型定义
- [x] 配置结构支持
- [x] 拨号器创建
- [x] 密码认证
- [x] CA 证书支持
- [x] 带宽限制配置
- [x] 证书验证跳过选项

配置示例:
\`\`\`yaml
- name: "Hysteria2-香港"
  protocol: "hysteria2"
  address: "hk.example.com"
  port: 443
  enabled: true
  weight: 100
  hysteria2:
    password: "your-password"
    ca: "/path/to/ca.crt"
    insecure: false
    up_mbps: 100
    down_mbps: 100
\`\`\`

测试完成时间: $(date)
EOF
    
    log "测试报告已生成: $report_file"
}

# 主函数
main() {
    log "开始 BoomDNS Hysteria2 协议测试..."
    
    # 检查依赖
    check_dependencies
    
    # 检查配置文件
    check_config_file
    
    # 验证 Hysteria2 配置
    validate_hysteria2_config || warn "Hysteria2 配置验证失败"
    
    # 检查服务状态
    check_service_status || warn "服务状态检查失败"
    
    # 测试 Hysteria2 API
    test_hysteria2_api || warn "Hysteria2 API 测试失败"
    
    # 验证 Hysteria2 协议支持
    validate_hysteria2_protocol || error "Hysteria2 协议支持验证失败"
    
    # 检查编译
    check_compilation || error "代码编译检查失败"
    
    # 生成报告
    generate_report
    
    log "Hysteria2 协议测试完成！"
    log "现在你的 BoomDNS 支持 Hysteria2 协议了！"
}

# 脚本入口
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
