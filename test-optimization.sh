#!/bin/bash
# MosDNS 优化效果测试脚本
# 用途: 测试 MosDNS 优化配置的性能和功能
# 作者: hnet 项目

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 配置变量
MOSDNS_HOST="127.0.0.1"
MOSDNS_PORT="53"
MOSDNS_API="http://127.0.0.1:9091"
TEST_DOMAINS=(
    "baidu.com"           # 国内域名
    "google.com"          # 国外域名 
    "github.com"          # 代码托管
    "youtube.com"         # 视频平台
    "10.0.0.5"           # 内网IP
    "pve.local"          # 本地域名
    "doubleclick.net"    # 广告域名（应该被拦截）
)

# 日志函数
log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_debug() { echo -e "${BLUE}[DEBUG]${NC} $1"; }

# 检查依赖
check_dependencies() {
    log_info "检查测试依赖..."
    
    local deps=("dig" "nslookup" "curl" "bc")
    for dep in "${deps[@]}"; do
        if ! command -v $dep &> /dev/null; then
            log_error "缺少依赖: $dep"
            echo "请安装: apt-get install dnsutils curl bc"
            exit 1
        fi
    done
}

# 测试DNS服务可用性
test_dns_availability() {
    log_info "测试DNS服务可用性..."
    
    if ! nc -z -w3 $MOSDNS_HOST $MOSDNS_PORT 2>/dev/null; then
        log_error "MosDNS 服务不可用 ($MOSDNS_HOST:$MOSDNS_PORT)"
        exit 1
    fi
    
    log_info "✓ MosDNS 服务可用"
}

# 测试API接口
test_api_interface() {
    log_info "测试API接口..."
    
    # 测试指标接口
    if curl -s --connect-timeout 5 "$MOSDNS_API/metrics" >/dev/null; then
        log_info "✓ API接口正常"
        
        # 获取一些基础指标
        metrics=$(curl -s "$MOSDNS_API/metrics" 2>/dev/null || echo "")
        if [[ -n "$metrics" ]]; then
            log_debug "API指标可用"
        fi
    else
        log_warn "✗ API接口不可用"
    fi
}

# 基础DNS解析测试
test_basic_resolution() {
    log_info "执行基础DNS解析测试..."
    
    local success_count=0
    local total_tests=${#TEST_DOMAINS[@]}
    
    for domain in "${TEST_DOMAINS[@]}"; do
        log_debug "测试域名: $domain"
        
        if dig @$MOSDNS_HOST +short "$domain" | grep -q "."; then
            log_info "✓ $domain 解析成功"
            ((success_count++))
        else
            log_warn "✗ $domain 解析失败"
        fi
    done
    
    local success_rate=$(echo "scale=1; $success_count * 100 / $total_tests" | bc)
    log_info "基础解析成功率: $success_rate% ($success_count/$total_tests)"
}

# 缓存效果测试
test_cache_performance() {
    log_info "测试缓存效果..."
    
    local test_domain="github.com"
    local iterations=3
    
    log_debug "测试域名: $test_domain (进行 $iterations 次查询)"
    
    # 清除本地DNS缓存（如果可能）
    # sudo systemctl flush-dns 2>/dev/null || true
    
    local total_time=0
    local times=()
    
    for i in $(seq 1 $iterations); do
        local start_time=$(date +%s%3N)
        dig @$MOSDNS_HOST +short "$test_domain" >/dev/null 2>&1
        local end_time=$(date +%s%3N)
        local query_time=$((end_time - start_time))
        
        times+=($query_time)
        total_time=$((total_time + query_time))
        
        log_debug "第${i}次查询: ${query_time}ms"
        sleep 0.1  # 短暂间隔
    done
    
    local avg_time=$(echo "scale=1; $total_time / $iterations" | bc)
    log_info "平均查询时间: ${avg_time}ms"
    
    # 分析缓存效果
    local first_time=${times[0]}
    local last_time=${times[-1]}
    
    if [[ $last_time -lt $first_time ]]; then
        local improvement=$(echo "scale=1; ($first_time - $last_time) * 100 / $first_time" | bc)
        log_info "✓ 缓存工作正常，性能提升: ${improvement}%"
    else
        log_warn "✗ 缓存效果不明显"
    fi
}

# 并发性能测试
test_concurrent_performance() {
    log_info "测试并发性能..."
    
    local concurrent_count=10
    local test_domain="baidu.com"
    
    log_debug "并发查询数: $concurrent_count"
    log_debug "测试域名: $test_domain"
    
    # 创建临时目录存储结果
    local temp_dir=$(mktemp -d)
    local start_time=$(date +%s%3N)
    
    # 启动并发查询
    for i in $(seq 1 $concurrent_count); do
        {
            local query_start=$(date +%s%3N)
            dig @$MOSDNS_HOST +short "$test_domain" >/dev/null 2>&1
            local query_end=$(date +%s%3N)
            echo $((query_end - query_start)) > "$temp_dir/result_$i"
        } &
    done
    
    # 等待所有查询完成
    wait
    local total_time=$(($(date +%s%3N) - start_time))
    
    # 分析结果
    local success_count=0
    local total_query_time=0
    
    for i in $(seq 1 $concurrent_count); do
        if [[ -f "$temp_dir/result_$i" ]]; then
            local query_time=$(cat "$temp_dir/result_$i")
            total_query_time=$((total_query_time + query_time))
            ((success_count++))
        fi
    done
    
    # 清理临时文件
    rm -rf "$temp_dir"
    
    local avg_query_time=$(echo "scale=1; $total_query_time / $success_count" | bc 2>/dev/null || echo "0")
    local qps=$(echo "scale=1; $success_count * 1000 / $total_time" | bc 2>/dev/null || echo "0")
    
    log_info "并发测试结果:"
    log_info "  - 成功查询: $success_count/$concurrent_count"
    log_info "  - 总耗时: ${total_time}ms"
    log_info "  - 平均查询时间: ${avg_query_time}ms"
    log_info "  - QPS: $qps"
}

# 分流规则测试
test_routing_rules() {
    log_info "测试分流规则..."
    
    # 测试国内域名分流
    log_debug "测试国内域名分流..."
    local cn_domain="baidu.com"
    local cn_result=$(dig @$MOSDNS_HOST +short "$cn_domain" A)
    if [[ -n "$cn_result" ]]; then
        log_info "✓ 国内域名解析正常: $cn_domain"
    else
        log_warn "✗ 国内域名解析失败: $cn_domain"
    fi
    
    # 测试国外域名分流
    log_debug "测试国外域名分流..."
    local foreign_domain="google.com"
    local foreign_result=$(dig @$MOSDNS_HOST +short "$foreign_domain" A)
    if [[ -n "$foreign_result" ]]; then
        log_info "✓ 国外域名解析正常: $foreign_domain"
    else
        log_warn "✗ 国外域名解析失败: $foreign_domain"
    fi
    
    # 测试本地域名分流
    log_debug "测试本地域名分流..."
    local local_domain="pve.local"
    local local_result=$(dig @$MOSDNS_HOST +short "$local_domain" A)
    # 本地域名可能没有配置，不一定有结果
    log_info "ℹ 本地域名测试: $local_domain"
}

# 广告拦截测试
test_ad_blocking() {
    log_info "测试广告拦截功能..."
    
    local ad_domains=("doubleclick.net" "googleadservices.com" "googlesyndication.com")
    local blocked_count=0
    
    for domain in "${ad_domains[@]}"; do
        log_debug "测试广告域名: $domain"
        
        # 使用dig获取响应码
        local response=$(dig @$MOSDNS_HOST "$domain" A +short)
        local exit_code=$?
        
        if [[ $exit_code -ne 0 ]] || [[ -z "$response" ]] || echo "$response" | grep -q "NXDOMAIN"; then
            log_info "✓ $domain 已被拦截"
            ((blocked_count++))
        else
            log_warn "✗ $domain 未被拦截"
        fi
    done
    
    local block_rate=$(echo "scale=1; $blocked_count * 100 / ${#ad_domains[@]}" | bc)
    log_info "广告拦截率: $block_rate% ($blocked_count/${#ad_domains[@]})"
}

# 延迟测试
test_latency() {
    log_info "测试查询延迟..."
    
    local test_domains=("baidu.com" "google.com" "github.com")
    local total_time=0
    local test_count=0
    
    for domain in "${test_domains[@]}"; do
        log_debug "延迟测试: $domain"
        
        # 进行多次测试取平均值
        local domain_total=0
        local domain_count=3
        
        for i in $(seq 1 $domain_count); do
            local start_time=$(date +%s%3N)
            dig @$MOSDNS_HOST +short "$domain" >/dev/null 2>&1
            local end_time=$(date +%s%3N)
            local query_time=$((end_time - start_time))
            
            domain_total=$((domain_total + query_time))
        done
        
        local domain_avg=$(echo "scale=1; $domain_total / $domain_count" | bc)
        log_info "  $domain: ${domain_avg}ms"
        
        total_time=$((total_time + domain_total))
        test_count=$((test_count + domain_count))
    done
    
    local overall_avg=$(echo "scale=1; $total_time / $test_count" | bc)
    log_info "整体平均延迟: ${overall_avg}ms"
    
    # 延迟评估
    if (( $(echo "$overall_avg < 50" | bc -l) )); then
        log_info "✓ 延迟表现优秀 (<50ms)"
    elif (( $(echo "$overall_avg < 100" | bc -l) )); then
        log_info "✓ 延迟表现良好 (<100ms)"
    elif (( $(echo "$overall_avg < 200" | bc -l) )); then
        log_warn "△ 延迟表现一般 (<200ms)"
    else
        log_warn "✗ 延迟表现较差 (>200ms)"
    fi
}

# 上游DNS健康检查
test_upstream_health() {
    log_info "测试上游DNS健康状态..."
    
    # 测试AdGuardHome
    log_debug "测试 AdGuardHome (10.0.0.5:53)..."
    if nc -z -w3 10.0.0.5 53 2>/dev/null; then
        log_info "✓ AdGuardHome 连通性正常"
        
        # 测试解析功能
        if dig @10.0.0.5 +short baidu.com >/dev/null 2>&1; then
            log_info "✓ AdGuardHome 解析功能正常"
        else
            log_warn "✗ AdGuardHome 解析功能异常"
        fi
    else
        log_warn "✗ AdGuardHome 无法连接"
    fi
    
    # 测试mihomo
    log_debug "测试 mihomo (10.0.0.6:1053)..."
    if nc -z -w3 10.0.0.6 1053 2>/dev/null; then
        log_info "✓ mihomo 连通性正常"
        
        # 测试解析功能
        if dig @10.0.0.6 -p 1053 +short google.com >/dev/null 2>&1; then
            log_info "✓ mihomo 解析功能正常"
        else
            log_warn "✗ mihomo 解析功能异常"
        fi
    else
        log_warn "✗ mihomo 无法连接"
    fi
}

# 生成测试报告
generate_test_report() {
    log_info "生成测试报告..."
    
    local report_file="/tmp/mosdns-test-report.txt"
    cat > "$report_file" << EOF
MosDNS 优化效果测试报告
测试时间: $(date)
测试主机: $(uname -n)

测试配置:
- MosDNS地址: $MOSDNS_HOST:$MOSDNS_PORT
- API地址: $MOSDNS_API

服务状态:
$(systemctl is-active mosdns 2>/dev/null && echo "✓ 服务运行中" || echo "✗ 服务未运行")

端口监听:
$(ss -tulnp | grep :53 || echo "未检测到DNS端口监听")

内存使用:
$(ps aux | grep mosdns | grep -v grep || echo "未找到MosDNS进程")

配置文件状态:
$(ls -la /etc/homeserver/mosdns/config.yaml 2>/dev/null || echo "配置文件不存在")

缓存文件状态:
$(ls -la /opt/homeserver/data/cache.dump 2>/dev/null || echo "缓存文件不存在")

API响应测试:
$(curl -s --connect-timeout 3 "$MOSDNS_API/metrics" >/dev/null && echo "✓ API响应正常" || echo "✗ API无响应")

建议的优化操作:
1. 如果延迟较高，考虑调整并发数
2. 如果缓存效果不佳，检查缓存配置
3. 如果分流不准确，更新规则文件
4. 定期检查上游DNS服务状态

EOF
    
    log_info "测试报告已生成: $report_file"
    echo
    cat "$report_file"
}

# 主测试函数
run_all_tests() {
    log_info "开始 MosDNS 优化效果测试..."
    echo "=========================================="
    
    check_dependencies
    test_dns_availability
    test_api_interface
    test_basic_resolution
    echo
    test_cache_performance
    echo
    test_concurrent_performance
    echo
    test_routing_rules
    echo
    test_ad_blocking
    echo
    test_latency
    echo
    test_upstream_health
    echo
    generate_test_report
    
    echo "=========================================="
    log_info "🎉 测试完成！请查看上方报告了解详细结果。"
}

# 参数处理
case "${1:-}" in
    "--basic")
        log_info "执行基础测试..."
        check_dependencies
        test_dns_availability
        test_basic_resolution
        ;;
    "--performance")
        log_info "执行性能测试..."
        check_dependencies
        test_cache_performance
        test_concurrent_performance
        test_latency
        ;;
    "--routing")
        log_info "执行分流测试..."
        check_dependencies
        test_routing_rules
        test_ad_blocking
        ;;
    "--upstream")
        log_info "执行上游测试..."
        test_upstream_health
        ;;
    "--help"|"-h")
        echo "MosDNS 优化效果测试脚本"
        echo
        echo "用法: $0 [选项]"
        echo
        echo "选项:"
        echo "  --basic       执行基础功能测试"
        echo "  --performance 执行性能测试"
        echo "  --routing     执行分流规则测试"
        echo "  --upstream    执行上游DNS测试"
        echo "  --help, -h    显示此帮助信息"
        echo
        echo "无参数时执行完整测试套件"
        exit 0
        ;;
    "")
        run_all_tests
        ;;
    *)
        log_error "未知参数: $1"
        echo "使用 $0 --help 查看帮助"
        exit 1
        ;;
esac
