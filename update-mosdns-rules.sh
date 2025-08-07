#!/bin/bash

# MosDNS 规则自动更新脚本
# 用于定期更新远程规则列表

set -euo pipefail

# 配置变量
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RULES_DIR="/opt/homeserver/rules"
BACKUP_DIR="/opt/homeserver/backup/rules"
LOG_FILE="/var/log/homeserver/mosdns/update-rules.log"
LOCK_FILE="/var/run/mosdns-update.lock"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log() {
    local level="$1"
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${timestamp} [${level}] ${message}" | tee -a "${LOG_FILE}"
}

log_info() {
    log "INFO" "$@"
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_warn() {
    log "WARN" "$@"
    echo -e "${YELLOW}[WARN]${NC} $*"
}

log_error() {
    log "ERROR" "$@"
    echo -e "${RED}[ERROR]${NC} $*"
}

log_success() {
    log "SUCCESS" "$@"
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

# 清理函数
cleanup() {
    if [[ -f "$LOCK_FILE" ]]; then
        rm -f "$LOCK_FILE"
    fi
}

# 捕获退出信号
trap cleanup EXIT

# 检查是否已有实例在运行
check_lock() {
    if [[ -f "$LOCK_FILE" ]]; then
        local pid=$(cat "$LOCK_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            log_error "更新脚本已在运行 (PID: $pid)"
            exit 1
        else
            log_warn "发现孤立的锁文件，正在清理..."
            rm -f "$LOCK_FILE"
        fi
    fi
    echo $$ > "$LOCK_FILE"
}

# 创建必要目录
create_directories() {
    local dirs=("$RULES_DIR" "$BACKUP_DIR" "$(dirname "$LOG_FILE")")
    for dir in "${dirs[@]}"; do
        if [[ ! -d "$dir" ]]; then
            mkdir -p "$dir"
            log_info "创建目录: $dir"
        fi
    done
}

# 备份现有规则
backup_rules() {
    local backup_timestamp=$(date '+%Y%m%d_%H%M%S')
    local backup_path="${BACKUP_DIR}/rules_${backup_timestamp}"
    
    if [[ -d "$RULES_DIR" ]] && [[ -n "$(ls -A "$RULES_DIR" 2>/dev/null)" ]]; then
        cp -r "$RULES_DIR" "$backup_path"
        log_info "规则已备份到: $backup_path"
        
        # 保留最近7天的备份
        find "$BACKUP_DIR" -name "rules_*" -type d -mtime +7 -exec rm -rf {} \; 2>/dev/null || true
    fi
}

# 下载文件函数
download_file() {
    local url="$1"
    local output_file="$2"
    local max_retries=3
    local retry_delay=5
    
    for ((i=1; i<=max_retries; i++)); do
        log_info "下载 $url (尝试 $i/$max_retries)..."
        
        if curl -L --connect-timeout 10 --max-time 60 -o "$output_file.tmp" "$url" 2>/dev/null; then
            # 检查文件是否为空或无效
            if [[ -s "$output_file.tmp" ]]; then
                mv "$output_file.tmp" "$output_file"
                log_success "下载成功: $(basename "$output_file")"
                return 0
            else
                log_warn "下载的文件为空: $url"
                rm -f "$output_file.tmp"
            fi
        else
            log_warn "下载失败: $url"
            rm -f "$output_file.tmp"
        fi
        
        if [[ $i -lt $max_retries ]]; then
            log_info "等待 ${retry_delay} 秒后重试..."
            sleep $retry_delay
        fi
    done
    
    log_error "下载失败，已达最大重试次数: $url"
    return 1
}

# 处理域名格式
process_domain_list() {
    local input_file="$1"
    local output_file="$2"
    
    # 提取域名，过滤注释和空行，去重排序
    grep -v '^#' "$input_file" 2>/dev/null | \
    grep -v '^$' | \
    sed 's/^[[:space:]]*//' | \
    sed 's/[[:space:]]*$//' | \
    grep -E '^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$' | \
    sort -u > "$output_file.processed"
    
    mv "$output_file.processed" "$output_file"
}

# 更新广告拦截列表
update_reject_list() {
    log_info "更新广告拦截列表..."
    
    local temp_dir=$(mktemp -d)
    local reject_urls=(
        "https://raw.githubusercontent.com/privacy-protection-tools/anti-AD/master/anti-ad-domains.txt"
        "https://raw.githubusercontent.com/jdlingyu/ad-wars/master/hosts"
        "https://raw.githubusercontent.com/AdguardTeam/AdguardFilters/master/BaseFilter/sections/adservers.txt"
    )
    
    local combined_file="${temp_dir}/reject_combined.txt"
    : > "$combined_file"  # 创建空文件
    
    local success_count=0
    for url in "${reject_urls[@]}"; do
        local filename=$(basename "$url" .txt)_$(date +%s).txt
        local temp_file="${temp_dir}/${filename}"
        
        if download_file "$url" "$temp_file"; then
            # 处理不同格式的文件
            if [[ "$url" == *"hosts"* ]]; then
                # 处理hosts格式
                grep '^0.0.0.0\|^127.0.0.1' "$temp_file" 2>/dev/null | \
                awk '{print $2}' | \
                grep -v '^localhost$\|^local$' >> "$combined_file" || true
            else
                # 处理域名列表格式
                cat "$temp_file" >> "$combined_file"
            fi
            ((success_count++))
        fi
    done
    
    if [[ $success_count -gt 0 ]]; then
        process_domain_list "$combined_file" "${RULES_DIR}/reject.txt"
        local count=$(wc -l < "${RULES_DIR}/reject.txt")
        log_success "广告拦截列表更新完成，共 $count 个域名"
    else
        log_error "所有广告拦截列表下载失败"
    fi
    
    rm -rf "$temp_dir"
}

# 更新直连域名列表
update_direct_list() {
    log_info "更新直连域名列表..."
    
    local temp_dir=$(mktemp -d)
    local direct_urls=(
        "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/direct-list.txt"
        "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/apple-cn.txt"
    )
    
    local combined_file="${temp_dir}/direct_combined.txt"
    : > "$combined_file"
    
    local success_count=0
    for url in "${direct_urls[@]}"; do
        local filename=$(basename "$url" .txt)_$(date +%s).txt
        local temp_file="${temp_dir}/${filename}"
        
        if download_file "$url" "$temp_file"; then
            cat "$temp_file" >> "$combined_file"
            ((success_count++))
        fi
    done
    
    if [[ $success_count -gt 0 ]]; then
        process_domain_list "$combined_file" "${RULES_DIR}/direct.txt"
        local count=$(wc -l < "${RULES_DIR}/direct.txt")
        log_success "直连域名列表更新完成，共 $count 个域名"
    else
        log_error "所有直连域名列表下载失败"
    fi
    
    rm -rf "$temp_dir"
}

# 更新代理域名列表
update_proxy_list() {
    log_info "更新代理域名列表..."
    
    local temp_dir=$(mktemp -d)
    local proxy_urls=(
        "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/gfw.txt"
        "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/greatfire.txt"
        "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/telegram.txt"
    )
    
    local combined_file="${temp_dir}/proxy_combined.txt"
    : > "$combined_file"
    
    local success_count=0
    for url in "${proxy_urls[@]}"; do
        local filename=$(basename "$url" .txt)_$(date +%s).txt
        local temp_file="${temp_dir}/${filename}"
        
        if download_file "$url" "$temp_file"; then
            cat "$temp_file" >> "$combined_file"
            ((success_count++))
        fi
    done
    
    if [[ $success_count -gt 0 ]]; then
        process_domain_list "$combined_file" "${RULES_DIR}/proxy.txt"
        local count=$(wc -l < "${RULES_DIR}/proxy.txt")
        log_success "代理域名列表更新完成，共 $count 个域名"
    else
        log_error "所有代理域名列表下载失败"
    fi
    
    rm -rf "$temp_dir"
}

# 更新geosite和geoip数据
update_geo_data() {
    log_info "更新地理位置数据..."
    
    local data_dir="/opt/homeserver/data"
    mkdir -p "$data_dir"
    
    # 更新geosite数据
    if download_file "https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat" "${data_dir}/geosite.dat.new"; then
        mv "${data_dir}/geosite.dat.new" "${data_dir}/geosite.dat"
        log_success "geosite.dat 更新完成"
    else
        log_error "geosite.dat 更新失败"
    fi
    
    # 更新geoip数据
    if download_file "https://github.com/v2fly/geoip/releases/latest/download/geoip.dat" "${data_dir}/geoip.dat.new"; then
        mv "${data_dir}/geoip.dat.new" "${data_dir}/geoip.dat"
        log_success "geoip.dat 更新完成"
    else
        log_error "geoip.dat 更新失败"
    fi
}

# 检查MosDNS配置有效性
check_config() {
    log_info "检查MosDNS配置有效性..."
    
    local config_file="/etc/homeserver/mosdns/config.yaml"
    if [[ -f "$config_file" ]]; then
        if command -v mosdns >/dev/null 2>&1; then
            if mosdns verify -c "$config_file" >/dev/null 2>&1; then
                log_success "配置文件验证通过"
                return 0
            else
                log_error "配置文件验证失败"
                return 1
            fi
        else
            log_warn "MosDNS未安装，跳过配置验证"
            return 0
        fi
    else
        log_warn "配置文件不存在: $config_file"
        return 0
    fi
}

# 重启MosDNS服务
restart_service() {
    log_info "重启MosDNS服务..."
    
    if systemctl is-active --quiet mosdns; then
        if systemctl restart mosdns; then
            log_success "MosDNS服务重启成功"
        else
            log_error "MosDNS服务重启失败"
            return 1
        fi
    else
        log_warn "MosDNS服务未运行，跳过重启"
    fi
}

# 显示统计信息
show_stats() {
    log_info "规则统计信息:"
    
    local files=("reject.txt" "direct.txt" "proxy.txt")
    for file in "${files[@]}"; do
        local path="${RULES_DIR}/${file}"
        if [[ -f "$path" ]]; then
            local count=$(wc -l < "$path")
            local size=$(du -h "$path" | cut -f1)
            log_info "  ${file}: ${count} 个域名 (${size})"
        else
            log_warn "  ${file}: 文件不存在"
        fi
    done
}

# 主函数
main() {
    local start_time=$(date +%s)
    
    log_info "开始更新MosDNS规则..."
    log_info "脚本版本: 1.0"
    log_info "执行时间: $(date)"
    
    # 检查运行权限
    if [[ $EUID -ne 0 ]]; then
        log_error "请以root权限运行此脚本"
        exit 1
    fi
    
    # 检查锁文件
    check_lock
    
    # 创建必要目录
    create_directories
    
    # 备份现有规则
    backup_rules
    
    # 更新各类规则
    local update_success=true
    
    update_reject_list || update_success=false
    update_direct_list || update_success=false
    update_proxy_list || update_success=false
    update_geo_data || update_success=false
    
    # 检查配置
    if ! check_config; then
        log_error "配置验证失败，不重启服务"
        update_success=false
    fi
    
    # 重启服务
    if [[ "$update_success" == "true" ]]; then
        restart_service
    else
        log_warn "由于更新过程中出现错误，跳过服务重启"
    fi
    
    # 显示统计信息
    show_stats
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [[ "$update_success" == "true" ]]; then
        log_success "规则更新完成！耗时: ${duration} 秒"
        exit 0
    else
        log_error "规则更新过程中出现错误，请检查日志"
        exit 1
    fi
}

# 帮助信息
show_help() {
    cat << EOF
MosDNS 规则自动更新脚本

用法: $0 [选项]

选项:
  -h, --help     显示此帮助信息
  -v, --version  显示版本信息
  -t, --test     测试模式（不重启服务）

示例:
  $0              # 执行完整更新
  $0 --test       # 测试更新（不重启服务）

日志文件: $LOG_FILE
EOF
}

# 解析命令行参数
case "${1:-}" in
    -h|--help)
        show_help
        exit 0
        ;;
    -v|--version)
        echo "MosDNS规则更新脚本 v1.0"
        exit 0
        ;;
    -t|--test)
        TEST_MODE=true
        ;;
    "")
        # 无参数，正常执行
        ;;
    *)
        echo "未知选项: $1"
        show_help
        exit 1
        ;;
esac

# 执行主函数
main "$@"
