#!/usr/bin/env bash

# BoomDNS Hysteria2 功能演示脚本
# 展示 Hysteria2 协议的各种功能和 API 使用方法

set -euo pipefail

# 颜色定义
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# 配置
readonly SERVICE_URL="http://localhost:8080"
readonly ADMIN_TOKEN="boomdns-secret-token-2024"
readonly AUTH_HEADER="Authorization: Bearer $ADMIN_TOKEN"

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

# 检查服务状态
check_service() {
    log "检查 BoomDNS 服务状态..."
    
    local response
    response=$(curl -s "$SERVICE_URL/api/health")
    
    if echo "$response" | grep -q '"ok":true'; then
        log "✓ 服务运行正常"
        return 0
    else
        error "✗ 服务未运行或无法访问"
        return 1
    fi
}

# 演示 1: 获取代理节点列表
demo_get_nodes() {
    log "演示 1: 获取代理节点列表"
    
    local response
    response=$(curl -s -H "$AUTH_HEADER" "$SERVICE_URL/api/proxy/nodes")
    
    if echo "$response" | grep -q '"success":true'; then
        log "✓ 成功获取代理节点列表"
        
        # 显示 Hysteria2 节点信息
        local hysteria2_nodes
        hysteria2_nodes=$(echo "$response" | jq -r '.data[] | select(.protocol == "hysteria2") | "  - \(.name): \(.address):\(.port)"')
        
        if [[ -n "$hysteria2_nodes" ]]; then
            info "找到的 Hysteria2 节点:"
            echo "$hysteria2_nodes"
        else
            warn "未找到 Hysteria2 节点"
        fi
    else
        error "✗ 获取代理节点失败"
        echo "$response"
    fi
}

# 演示 2: 测试 Hysteria2 节点
demo_test_node() {
    log "演示 2: 测试 Hysteria2 节点"
    
    local response
    response=$(curl -s -H "$AUTH_HEADER" "$SERVICE_URL/api/proxy/nodes/1/test")
    
    if [[ -n "$response" ]]; then
        log "✓ 节点测试完成"
        echo "$response" | jq .
    else
        warn "节点测试无响应"
    fi
}

# 演示 3: 配置验证
demo_config_validation() {
    log "演示 3: Hysteria2 配置验证"
    
    # 测试有效配置
    info "测试有效配置..."
    local valid_config='{"protocol":"hysteria2","config":{"password":"test123","address":"test.com","port":443,"up_mbps":100,"down_mbps":100}}'
    
    local response
    response=$(curl -s -X POST -H "$AUTH_HEADER" -H "Content-Type: application/json" -d "$valid_config" "$SERVICE_URL/api/proxy/validate")
    
    if echo "$response" | grep -q '"success":true'; then
        log "✓ 有效配置验证通过"
    else
        warn "有效配置验证失败"
        echo "$response" | jq .
    fi
    
    # 测试无效配置
    info "测试无效配置..."
    local invalid_config='{"protocol":"hysteria2","config":{"address":"test.com","port":99999}}'
    
    response=$(curl -s -X POST -H "$AUTH_HEADER" -H "Content-Type: application/json" -d "$invalid_config" "$SERVICE_URL/api/proxy/validate")
    
    if echo "$response" | grep -q '"success":false'; then
        log "✓ 无效配置验证失败（符合预期）"
        echo "$response" | jq .
    else
        warn "无效配置验证意外通过"
        echo "$response" | jq .
    fi
}

# 演示 4: 代理状态
demo_proxy_status() {
    log "演示 4: 获取代理状态"
    
    local response
    response=$(curl -s -H "$AUTH_HEADER" "$SERVICE_URL/api/proxy/status")
    
    if [[ -n "$response" ]]; then
        log "✓ 代理状态获取成功"
        echo "$response" | jq .
    else
        warn "代理状态获取失败"
    fi
}

# 演示 5: 协议支持检查
demo_protocol_support() {
    log "演示 5: 检查协议支持"
    
    info "支持的代理协议:"
    echo "  - HTTP/HTTPS"
    echo "  - SOCKS5"
    echo "  - Shadowsocks"
    echo "  - V2Ray"
    echo "  - Trojan"
    echo "  - WireGuard"
    echo "  - Hysteria2 ✨"
    
    info "Hysteria2 特性:"
    echo "  - 基于 QUIC 协议"
    echo "  - 支持密码认证"
    echo "  - 支持 CA 证书"
    echo "  - 支持带宽限制"
    echo "  - 支持证书验证跳过"
    echo "  - 优秀的抗封锁能力"
}

# 演示 6: 配置示例
demo_config_examples() {
    log "演示 6: Hysteria2 配置示例"
    
    cat << 'EOF'

Hysteria2 节点配置示例:
```yaml
proxy_nodes:
  - name: "Hysteria2-香港"
    protocol: "hysteria2"
    address: "hk.example.com"
    port: 443
    enabled: true
    weight: 100
    hysteria2:
      password: "your-hysteria2-password"
      ca: "/path/to/ca.crt"        # 可选
      insecure: false              # 可选
      up_mbps: 100                 # 可选
      down_mbps: 100               # 可选
```

代理组配置示例:
```yaml
proxy_groups:
  - name: "自动选择"
    type: "url-test"
    strategy: "latency"
    test_url: "http://www.google.com"
    interval: 300
    timeout: 10
    nodes: [1, 2, 3]
    enabled: true
```

代理规则配置示例:
```yaml
proxy_rules:
  - type: "domain"
    value: "google.com"
    action: "proxy"
    proxy_group: "自动选择"
    priority: 100
    enabled: true
```
EOF
}

# 演示 7: 性能测试
demo_performance_test() {
    log "演示 7: Hysteria2 性能测试"
    
    info "测试配置验证 API 性能..."
    
    local start_time
    start_time=$(date +%s)
    
    local success_count=0
    local total_count=10
    
    for i in $(seq 1 $total_count); do
        local response
        response=$(curl -s -X POST -H "$AUTH_HEADER" -H "Content-Type: application/json" \
            -d '{"protocol":"hysteria2","config":{"password":"test123","address":"test.com","port":443}}' \
            "$SERVICE_URL/api/proxy/validate")
        
        if echo "$response" | grep -q '"success":true'; then
            ((success_count++))
        fi
        
        echo -n "."
    done
    
    local end_time
    end_time=$(date +%s)
    
    local duration
    duration=$((end_time - start_time))
    
    echo ""
    log "性能测试完成:"
    info "  总请求数: $total_count"
    info "  成功请求数: $success_count"
    info "  成功率: $((success_count * 100 / total_count))%"
    info "  总耗时: ${duration}ms"
    info "  平均响应时间: $((duration / total_count))ms"
}

# 主演示函数
main() {
    log "🚀 开始 BoomDNS Hysteria2 功能演示..."
    echo ""
    
    # 检查服务状态
    if ! check_service; then
        error "服务未运行，请先启动 BoomDNS 服务"
        exit 1
    fi
    
    echo ""
    
    # 执行各个演示
    demo_get_nodes
    echo ""
    
    demo_test_node
    echo ""
    
    demo_config_validation
    echo ""
    
    demo_proxy_status
    echo ""
    
    demo_protocol_support
    echo ""
    
    demo_config_examples
    echo ""
    
    demo_performance_test
    echo ""
    
    log "🎉 Hysteria2 功能演示完成！"
    log "现在你的 BoomDNS 已经完全支持 Hysteria2 协议了！"
    echo ""
    
    info "下一步建议:"
    echo "  1. 配置真实的 Hysteria2 服务器信息"
    echo "  2. 测试实际的代理连接"
    echo "  3. 配置分流规则"
    echo "  4. 监控代理性能"
    echo "  5. 集成到你的网络架构中"
}

# 脚本入口
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
