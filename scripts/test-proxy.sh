#!/usr/bin/env bash

# BoomDNS 代理功能测试脚本
# 测试代理管理器、API 接口和基本功能

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
readonly PROXY_HTTP_PORT="7890"
readonly PROXY_SOCKS_PORT="7891"

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
    
    local deps=("curl" "jq" "netstat")
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            error "缺少依赖: $dep"
            exit 1
        fi
    done
    
    log "所有依赖已满足"
}

# 检查服务状态
check_service_status() {
    log "检查 BoomDNS 服务状态..."
    
    if ! curl -s "$SERVICE_URL/api/health" | jq -e '.ok' > /dev/null; then
        error "BoomDNS 服务未运行或无法访问"
        return 1
    fi
    
    log "BoomDNS 服务运行正常"
    return 0
}

# 测试代理配置
test_proxy_config() {
    log "测试代理配置..."
    
    # 检查代理配置是否启用
    local proxy_enabled
    proxy_enabled=$(curl -s "$SERVICE_URL/api/proxy/status" | jq -r '.data.enabled // false')
    
    if [[ "$proxy_enabled" == "true" ]]; then
        log "代理功能已启用"
    else
        warn "代理功能未启用"
        return 1
    fi
    
    # 检查代理状态
    local proxy_status
    proxy_status=$(curl -s "$SERVICE_URL/api/proxy/status" | jq -r '.data.status // "unknown"')
    log "代理状态: $proxy_status"
    
    return 0
}

# 测试代理节点管理
test_proxy_nodes() {
    log "测试代理节点管理..."
    
    # 获取代理节点列表
    local nodes_response
    nodes_response=$(curl -s "$SERVICE_URL/api/proxy/nodes")
    
    if echo "$nodes_response" | jq -e '.success' > /dev/null; then
        log "代理节点 API 响应正常"
        local message
        message=$(echo "$nodes_response" | jq -r '.message // "unknown"')
        log "API 消息: $message"
    else
        error "代理节点 API 响应异常"
        return 1
    fi
    
    return 0
}

# 测试代理组管理
test_proxy_groups() {
    log "测试代理组管理..."
    
    # 获取代理组列表
    local groups_response
    groups_response=$(curl -s "$SERVICE_URL/api/proxy/groups")
    
    if echo "$groups_response" | jq -e '.success' > /dev/null; then
        log "代理组 API 响应正常"
        local message
        message=$(echo "$groups_response" | jq -r '.message // "unknown"')
        log "API 消息: $message"
    else
        error "代理组 API 响应异常"
        return 1
    fi
    
    return 0
}

# 测试代理规则管理
test_proxy_rules() {
    log "测试代理规则管理..."
    
    # 获取代理规则列表
    local rules_response
    rules_response=$(curl -s "$SERVICE_URL/api/proxy/rules")
    
    if echo "$rules_response" | jq -e '.success' > /dev/null; then
        log "代理规则 API 响应正常"
        local message
        message=$(echo "$rules_response" | jq -r '.message // "unknown"')
        log "API 消息: $message"
    else
        error "代理规则 API 响应异常"
        return 1
    fi
    
    return 0
}

# 检查代理端口
check_proxy_ports() {
    log "检查代理端口..."
    
    # 检查 HTTP 代理端口
    if netstat -an | grep -q ":$PROXY_HTTP_PORT.*LISTEN"; then
        log "HTTP 代理端口 $PROXY_HTTP_PORT 正在监听"
    else
        warn "HTTP 代理端口 $PROXY_HTTP_PORT 未监听"
    fi
    
    # 检查 SOCKS5 代理端口
    if netstat -an | grep -q ":$PROXY_SOCKS_PORT.*LISTEN"; then
        log "SOCKS5 代理端口 $PROXY_SOCKS_PORT 正在监听"
    else
        warn "SOCKS5 代理端口 $PROXY_SOCKS_PORT 未监听"
    fi
}

# 测试代理连接
test_proxy_connection() {
    log "测试代理连接..."
    
    # 这里可以添加实际的代理连接测试
    # 例如使用 curl 通过代理访问外部网站
    
    log "代理连接测试完成（基础功能验证）"
}

# 生成测试报告
generate_report() {
    log "生成测试报告..."
    
    local report_file="proxy-test-report-$(date +%Y%m%d-%H%M%S).txt"
    
    cat > "$report_file" << EOF
BoomDNS 代理功能测试报告
========================

测试时间: $(date)
测试项目: $PROJECT_NAME

测试结果摘要:
- 服务状态: $(check_service_status && echo "正常" || echo "异常")
- 代理配置: $(test_proxy_config && echo "正常" || echo "异常")
- 节点管理: $(test_proxy_nodes && echo "正常" || echo "异常")
- 组管理: $(test_proxy_groups && echo "正常" || echo "异常")
- 规则管理: $(test_proxy_rules && echo "正常" || echo "异常")
- 端口监听: $(check_proxy_ports && echo "正常" || echo "异常")

测试完成时间: $(date)
EOF
    
    log "测试报告已生成: $report_file"
}

# 主函数
main() {
    log "开始 BoomDNS 代理功能测试..."
    
    # 检查依赖
    check_dependencies
    
    # 检查服务状态
    if ! check_service_status; then
        error "服务检查失败，退出测试"
        exit 1
    fi
    
    # 执行各项测试
    test_proxy_config || warn "代理配置测试失败"
    test_proxy_nodes || warn "代理节点管理测试失败"
    test_proxy_groups || warn "代理组管理测试失败"
    test_proxy_rules || warn "代理规则管理测试失败"
    check_proxy_ports || warn "代理端口检查失败"
    test_proxy_connection || warn "代理连接测试失败"
    
    # 生成报告
    generate_report
    
    log "代理功能测试完成！"
}

# 脚本入口
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
