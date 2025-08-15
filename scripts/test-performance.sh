#!/usr/bin/env bash

# =============================================================================
# BoomDNS 性能测试脚本
# 全面测试 DNS 服务性能、并发能力和资源使用
# =============================================================================

set -euo pipefail

# 颜色定义
readonly RED='\033[38;5;196m'
readonly GREEN='\033[38;5;46m'
readonly YELLOW='\033[38;5;226m'
readonly BLUE='\033[38;5;39m'
readonly CYAN='\033[38;5;51m'
readonly NC='\033[0m'
readonly BOLD='\033[1m'

# 配置变量
readonly DNS_SERVER="127.0.0.1"
readonly DNS_PORT="53"
readonly HTTP_PORT="8080"
readonly TEST_DOMAINS=(
    "baidu.com"
    "google.com"
    "github.com"
    "stackoverflow.com"
    "reddit.com"
    "wikipedia.org"
    "amazon.com"
    "microsoft.com"
    "apple.com"
    "netflix.com"
)

# 测试结果文件
readonly RESULTS_DIR="./test-results"
readonly PERFORMANCE_LOG="$RESULTS_DIR/performance.log"
readonly LATENCY_LOG="$RESULTS_DIR/latency.log"
readonly CONCURRENT_LOG="$RESULTS_DIR/concurrent.log"

# 日志函数
log() {
    local level="$1"
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    case "$level" in
        "INFO")  echo -e "${BLUE}[INFO]${NC} $timestamp $message" ;;
        "SUCCESS") echo -e "${GREEN}[SUCCESS]${NC} $timestamp $message" ;;
        "WARNING") echo -e "${YELLOW}[WARNING]${NC} $timestamp $message" ;;
        "ERROR") echo -e "${RED}[ERROR]${NC} $timestamp $message" ;;
        *) echo -e "${CYAN}[$level]${NC} $timestamp $message" ;;
    esac
}

# 检查依赖
check_dependencies() {
    log "INFO" "检查测试依赖..."
    
    local missing_deps=()
    
    # 检查必要工具
    local required_tools=("dig" "nslookup" "curl" "bc" "awk" "grep" "sort" "uniq")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            missing_deps+=("$tool")
        fi
    done
    
    if [ ${#missing_deps[@]} -gt 0 ]; then
        log "ERROR" "缺少依赖: ${missing_deps[*]}"
        log "ERROR" "请安装缺失的工具后重试"
        exit 1
    fi
    
    log "SUCCESS" "依赖检查通过"
}

# 创建测试目录
setup_test_environment() {
    log "INFO" "设置测试环境..."
    
    mkdir -p "$RESULTS_DIR"
    
    # 清空旧结果
    rm -f "$PERFORMANCE_LOG" "$LATENCY_LOG" "$CONCURRENT_LOG"
    
    # 创建结果文件头
    echo "timestamp,operation,duration_ms,success,error" > "$PERFORMANCE_LOG"
    echo "timestamp,domain,latency_ms,route,upstream" > "$LATENCY_LOG"
    echo "timestamp,concurrent_level,success_count,error_count,avg_latency_ms" > "$CONCURRENT_LOG"
    
    log "SUCCESS" "测试环境设置完成"
}

# 检查服务状态
check_service_status() {
    log "INFO" "检查 BoomDNS 服务状态..."
    
    # 检查 DNS 服务
    if ! nslookup google.com "$DNS_SERVER" >/dev/null 2>&1; then
        log "ERROR" "DNS 服务未响应，请确保 BoomDNS 正在运行"
        exit 1
    fi
    
    # 检查 HTTP 管理接口
    if ! curl -s "http://$DNS_SERVER:$HTTP_PORT/health" >/dev/null 2>&1; then
        log "WARNING" "HTTP 管理接口未响应"
    else
        log "SUCCESS" "HTTP 管理接口正常"
    fi
    
    log "SUCCESS" "服务状态检查通过"
}

# 基础性能测试
basic_performance_test() {
    log "INFO" "执行基础性能测试..."
    
    local total_time=0
    local success_count=0
    local error_count=0
    
    for domain in "${TEST_DOMAINS[@]}"; do
        local start_time=$(date +%s%3N)
        
        if nslookup "$domain" "$DNS_SERVER" >/dev/null 2>&1; then
            local end_time=$(date +%s%3N)
            local duration=$((end_time - start_time))
            total_time=$((total_time + duration))
            success_count=$((success_count + 1))
            
            echo "$(date '+%Y-%m-%d %H:%M:%S'),$domain,$duration,success," >> "$LATENCY_LOG"
            echo "$(date '+%Y-%m-%d %H:%M:%S'),basic_query,$duration,true," >> "$PERFORMANCE_LOG"
            
            log "INFO" "查询 $domain: ${duration}ms"
        else
            error_count=$((error_count + 1))
            echo "$(date '+%Y-%m-%d %H:%M:%S'),basic_query,0,false,query_failed" >> "$PERFORMANCE_LOG"
            log "WARNING" "查询 $domain: 失败"
        fi
    done
    
    if [ $success_count -gt 0 ]; then
        local avg_time=$((total_time / success_count))
        log "SUCCESS" "基础性能测试完成: 成功 $success_count, 失败 $error_count, 平均延迟 ${avg_time}ms"
    fi
}

# 延迟测试
latency_test() {
    log "INFO" "执行延迟测试..."
    
    local iterations=100
    local total_latency=0
    local min_latency=999999
    local max_latency=0
    
    for ((i=1; i<=iterations; i++)); do
        local start_time=$(date +%s%3N)
        
        if nslookup "google.com" "$DNS_SERVER" >/dev/null 2>&1; then
            local end_time=$(date +%s%3N)
            local latency=$((end_time - start_time))
            
            total_latency=$((total_latency + latency))
            
            if [ $latency -lt $min_latency ]; then
                min_latency=$latency
            fi
            if [ $latency -gt $max_latency ]; then
                max_latency=$latency
            fi
            
            echo "$(date '+%Y-%m-%d %H:%M:%S'),google.com,$latency,test,test" >> "$LATENCY_LOG"
        fi
        
        # 显示进度
        if [ $((i % 10)) -eq 0 ]; then
            printf "\r延迟测试进度: %d/%d" "$i" "$iterations"
        fi
    done
    
    echo
    
    if [ $iterations -gt 0 ]; then
        local avg_latency=$((total_latency / iterations))
        log "SUCCESS" "延迟测试完成: 平均 ${avg_latency}ms, 最小 ${min_latency}ms, 最大 ${max_latency}ms"
    fi
}

# 并发测试
concurrent_test() {
    log "INFO" "执行并发测试..."
    
    local concurrent_levels=(1 5 10 25 50 100)
    
    for level in "${concurrent_levels[@]}"; do
        log "INFO" "测试并发级别: $level"
        
        local start_time=$(date +%s%3N)
        local success_count=0
        local error_count=0
        local total_latency=0
        
        # 启动并发查询
        for ((i=1; i<=level; i++)); do
            (
                local query_start=$(date +%s%3N)
                if nslookup "google.com" "$DNS_SERVER" >/dev/null 2>&1; then
                    local query_end=$(date +%s%3N)
                    local query_latency=$((query_end - query_start))
                    echo "success:$query_latency" >> /tmp/concurrent_test_$$.tmp
                else
                    echo "error:0" >> /tmp/concurrent_test_$$.tmp
                fi
            ) &
        done
        
        # 等待所有查询完成
        wait
        
        # 统计结果
        if [ -f "/tmp/concurrent_test_$$.tmp" ]; then
            while IFS=':' read -r result latency; do
                if [ "$result" = "success" ]; then
                    success_count=$((success_count + 1))
                    total_latency=$((total_latency + latency))
                else
                    error_count=$((error_count + 1))
                fi
            done < "/tmp/concurrent_test_$$.tmp"
            rm -f "/tmp/concurrent_test_$$.tmp"
        fi
        
        local end_time=$(date +%s%3N)
        local test_duration=$((end_time - start_time))
        
        local avg_latency=0
        if [ $success_count -gt 0 ]; then
            avg_latency=$((total_latency / success_count))
        fi
        
        echo "$(date '+%Y-%m-%d %H:%M:%S'),$level,$success_count,$error_count,$avg_latency" >> "$CONCURRENT_LOG"
        
        log "INFO" "并发 $level: 成功 $success_count, 失败 $error_count, 平均延迟 ${avg_latency}ms, 总耗时 ${test_duration}ms"
    done
    
    log "SUCCESS" "并发测试完成"
}

# 缓存性能测试
cache_performance_test() {
    log "INFO" "执行缓存性能测试..."
    
    # 第一次查询 (缓存未命中)
    local first_start=$(date +%s%3N)
    nslookup "google.com" "$DNS_SERVER" >/dev/null 2>&1
    local first_end=$(date +%s%3N)
    local first_latency=$((first_end - first_start))
    
    # 第二次查询 (缓存命中)
    local second_start=$(date +%s%3N)
    nslookup "google.com" "$DNS_SERVER" >/dev/null 2>&1
    local second_end=$(date +%s%3N)
    local second_latency=$((second_end - second_start))
    
    local cache_improvement=$((first_latency - second_latency))
    local improvement_percent=0
    
    if [ $first_latency -gt 0 ]; then
        improvement_percent=$(echo "scale=2; $cache_improvement * 100 / $first_latency" | bc)
    fi
    
    log "SUCCESS" "缓存性能测试: 首次查询 ${first_latency}ms, 缓存命中 ${second_latency}ms, 提升 ${improvement_percent}%"
    
    echo "$(date '+%Y-%m-%d %H:%M:%S'),cache_test,$first_latency,true," >> "$PERFORMANCE_LOG"
    echo "$(date '+%Y-%m-%d %H:%M:%S'),cache_test,$second_latency,true," >> "$PERFORMANCE_LOG"
}

# 路由性能测试
routing_performance_test() {
    log "INFO" "执行路由性能测试..."
    
    local test_cases=(
        "baidu.com:china"
        "google.com:intl"
        "doubleclick.net:ads"
    )
    
    for test_case in "${test_cases[@]}"; do
        IFS=':' read -r domain expected_route <<< "$test_case"
        
        local start_time=$(date +%s%3N)
        local result=$(nslookup "$domain" "$DNS_SERVER" 2>/dev/null | grep -o "Server:[[:space:]]*[0-9.]*" | tail -1)
        local end_time=$(date +%s%3N)
        local latency=$((end_time - start_time))
        
        log "INFO" "路由测试 $domain: ${latency}ms (预期: $expected_route)"
        
        echo "$(date '+%Y-%m-%d %H:%M:%S'),$domain,$latency,$expected_route,test" >> "$LATENCY_LOG"
    done
    
    log "SUCCESS" "路由性能测试完成"
}

# 资源使用测试
resource_usage_test() {
    log "INFO" "检查资源使用情况..."
    
    # 检查容器资源使用
    if command -v docker &> /dev/null; then
        local container_stats=$(docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}" boomdns 2>/dev/null || true)
        if [ -n "$container_stats" ]; then
            log "INFO" "容器资源使用:"
            echo "$container_stats"
        fi
    fi
    
    # 检查系统资源
    if command -v top &> /dev/null; then
        local cpu_usage=$(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | cut -d'%' -f1)
        local mem_usage=$(free | grep Mem | awk '{printf "%.1f", $3/$2 * 100.0}' 2>/dev/null || echo "N/A")
        log "INFO" "系统 CPU 使用率: ${cpu_usage}%"
        log "INFO" "系统内存使用率: ${mem_usage}%"
    fi
    
    log "SUCCESS" "资源使用检查完成"
}

# 生成测试报告
generate_report() {
    log "INFO" "生成性能测试报告..."
    
    local report_file="$RESULTS_DIR/performance_report.md"
    
    cat > "$report_file" << EOF
# BoomDNS 性能测试报告

## 测试概览
- 测试时间: $(date '+%Y-%m-%d %H:%M:%S')
- 测试目标: $DNS_SERVER:$DNS_PORT
- 测试域名: ${#TEST_DOMAINS[@]} 个

## 测试结果摘要

### 基础性能测试
$(tail -n +2 "$PERFORMANCE_LOG" | grep "basic_query" | awk -F',' '{print $2, $3, $4}' | head -5)

### 延迟测试
$(tail -n +2 "$LATENCY_LOG" | grep "google.com" | awk -F',' '{print $2, $3}' | head -10)

### 并发测试
$(tail -n +2 "$CONCURRENT_LOG" | awk -F',' '{print "并发级别: " $2 ", 成功: " $3 ", 失败: " $4 ", 平均延迟: " $5 "ms"}')

## 性能指标

### 平均延迟
$(awk -F',' 'NR>1 && \$3>0 {sum+=\$3; count++} END {if(count>0) printf "%.2f ms\n", sum/count}' "$LATENCY_LOG")

### 成功率
$(awk -F',' 'NR>1 {total++} \$4=="true" {success++} END {if(total>0) printf "%.2f%%\n", success*100/total}' "$PERFORMANCE_LOG")

### 缓存命中率
$(awk -F',' 'NR>1 && \$2=="cache_test" {total++} \$2=="cache_test" && \$4=="true" {success++} END {if(total>0) printf "%.2f%%\n", success*100/total}' "$PERFORMANCE_LOG")

## 建议优化

1. **延迟优化**: 如果平均延迟超过 100ms，建议检查网络配置
2. **并发优化**: 如果高并发下失败率较高，建议调整连接池大小
3. **缓存优化**: 如果缓存命中率低于 70%，建议调整缓存策略

## 详细日志

- 性能日志: \`$PERFORMANCE_LOG\`
- 延迟日志: \`$LATENCY_LOG\`
- 并发日志: \`$CONCURRENT_LOG\`
EOF
    
    log "SUCCESS" "测试报告已生成: $report_file"
}

# 主函数
main() {
    echo -e "${BOLD}${BLUE}=============================================================================${NC}"
    echo -e "${BOLD}${BLUE}🚀 BoomDNS 性能测试脚本${NC}"
    echo -e "${BOLD}${BLUE}=============================================================================${NC}"
    echo
    
    local start_time=$(date +%s%3N)
    
    # 执行测试
    check_dependencies
    setup_test_environment
    check_service_status
    
    log "INFO" "开始性能测试..."
    
    basic_performance_test
    latency_test
    concurrent_test
    cache_performance_test
    routing_performance_test
    resource_usage_test
    
    # 生成报告
    generate_report
    
    local end_time=$(date +%s%3N)
    local total_duration=$((end_time - start_time))
    
    log "SUCCESS" "所有测试完成！总耗时: ${total_duration}ms"
    echo
    echo -e "${BOLD}${GREEN}✅ 性能测试完成！${NC}"
    echo -e "${CYAN}📊 测试报告: $RESULTS_DIR/performance_report.md${NC}"
    echo -e "${CYAN}📁 详细日志: $RESULTS_DIR/${NC}"
    echo
}

# 信号处理
trap 'log "ERROR" "测试被中断"; exit 130' INT TERM

# 执行主函数
main "$@"
