#!/bin/bash
# MosDNS 优化配置部署脚本
# 用途: 自动部署优化后的 MosDNS 配置
# 作者: hnet 项目
# 版本: 1.0

set -e  # 遇到错误立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
}

# 检查是否为 root 用户
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "此脚本需要 root 权限运行"
        echo "请使用: sudo $0"
        exit 1
    fi
}

# 检查系统依赖
check_dependencies() {
    log_info "检查系统依赖..."
    
    local deps=("wget" "curl" "systemctl" "nc")
    for dep in "${deps[@]}"; do
        if ! command -v $dep &> /dev/null; then
            log_error "缺少依赖: $dep"
            exit 1
        fi
    done
    
    log_info "系统依赖检查完成"
}

# 备份现有配置
backup_config() {
    log_info "备份现有配置..."
    
    local backup_dir="/backup/mosdns-$(date +%Y%m%d-%H%M%S)"
    mkdir -p "$backup_dir"
    
    # 备份配置文件
    if [[ -f "/etc/mosdns/config.yaml" ]]; then
        cp "/etc/mosdns/config.yaml" "$backup_dir/"
        log_info "已备份配置文件到: $backup_dir/config.yaml"
    fi
    
    # 备份规则文件
    if [[ -d "/opt/homeserver/rules" ]]; then
        cp -r "/opt/homeserver/rules" "$backup_dir/"
        log_info "已备份规则文件到: $backup_dir/rules/"
    fi
    
    echo "$backup_dir" > /tmp/mosdns_backup_path
    log_info "备份完成: $backup_dir"
}

# 创建目录结构
create_directories() {
    log_info "创建目录结构..."
    
    local dirs=(
        "/opt/homeserver/data"
        "/opt/homeserver/rules"
        "/var/log/homeserver/mosdns"
        "/etc/homeserver/mosdns"
        "/backup"
    )
    
    for dir in "${dirs[@]}"; do
        mkdir -p "$dir"
        log_debug "创建目录: $dir"
    done
    
    # 设置权限
    chown -R mosdns:mosdns /opt/homeserver 2>/dev/null || true
    chown -R mosdns:mosdns /var/log/homeserver 2>/dev/null || true
    
    log_info "目录结构创建完成"
}

# 下载数据文件
download_data_files() {
    log_info "下载 geosite 和 geoip 数据文件..."
    
    local data_dir="/opt/homeserver/data"
    cd "$data_dir"
    
    # 下载 geosite CN 列表
    log_debug "下载 geosite CN 数据..."
    wget -q --timeout=30 --tries=3 \
        -O geosite_cn.txt.tmp \
        "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/direct-list.txt" || {
        log_warn "geosite CN 下载失败，使用备用源..."
        wget -q --timeout=30 --tries=3 \
            -O geosite_cn.txt.tmp \
            "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/direct-list.txt"
    }
    mv geosite_cn.txt.tmp geosite_cn.txt
    
    # 下载 geoip CN 列表
    log_debug "下载 geoip CN 数据..."
    wget -q --timeout=30 --tries=3 \
        -O geoip_cn.txt.tmp \
        "https://raw.githubusercontent.com/Loyalsoldier/geoip/release/text/cn.txt" || {
        log_warn "geoip CN 下载失败，使用备用源..."
        wget -q --timeout=30 --tries=3 \
            -O geoip_cn.txt.tmp \
            "https://github.com/Loyalsoldier/geoip/releases/latest/download/text/cn.txt"
    }
    mv geoip_cn.txt.tmp geoip_cn.txt
    
    # 设置权限
    chown mosdns:mosdns *.txt 2>/dev/null || true
    chmod 644 *.txt
    
    log_info "数据文件下载完成"
}

# 创建规则文件
create_rule_files() {
    log_info "创建规则文件..."
    
    local rules_dir="/opt/homeserver/rules"
    
    # 创建本地域名文件
    if [[ -f "local-domains-optimized.txt" ]]; then
        cp "local-domains-optimized.txt" "$rules_dir/local-domains.txt"
        log_debug "已复制本地域名文件"
    else
        log_warn "本地域名文件不存在，创建空文件"
        touch "$rules_dir/local-domains.txt"
    fi
    
    # 创建 DDNS 域名文件
    cat > "$rules_dir/ddns-domains.txt" << 'EOF'
# DDNS 域名列表 - 需要低 TTL
# 这些域名的解析结果会频繁变化
myhost.ddns.net
*.no-ip.org
*.duckdns.org
*.dynv6.net
homeserver.ddns.net
EOF
    
    # 创建代理域名文件（从远程获取）
    log_debug "下载代理域名列表..."
    wget -q --timeout=30 --tries=3 \
        -O "$rules_dir/proxy-domains.txt" \
        "https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/proxy-list.txt" || {
        log_warn "代理域名列表下载失败，创建基础列表"
        cat > "$rules_dir/proxy-domains.txt" << 'EOF'
# 基础代理域名列表
google.com
youtube.com
twitter.com
facebook.com
instagram.com
telegram.org
EOF
    }
    
    # 创建广告拦截文件
    touch "$rules_dir/ad-block.txt"
    
    # 设置权限
    chown -R mosdns:mosdns "$rules_dir" 2>/dev/null || true
    chmod -R 644 "$rules_dir"/*.txt
    
    log_info "规则文件创建完成"
}

# 部署配置文件
deploy_config() {
    log_info "部署 MosDNS 配置文件..."
    
    # 检查优化配置文件是否存在
    if [[ ! -f "mosdns-optimized-config.yaml" ]]; then
        log_error "优化配置文件 mosdns-optimized-config.yaml 不存在"
        exit 1
    fi
    
    # 验证配置文件语法
    log_debug "验证配置文件语法..."
    if command -v mosdns &> /dev/null; then
        if ! mosdns start -c mosdns-optimized-config.yaml --dry-run &> /dev/null; then
            log_error "配置文件语法验证失败"
            exit 1
        fi
        log_debug "配置文件语法验证通过"
    else
        log_warn "MosDNS 未安装，跳过语法验证"
    fi
    
    # 部署配置文件
    cp mosdns-optimized-config.yaml /etc/homeserver/mosdns/config.yaml
    
    # 创建软链接（如果原配置目录存在）
    if [[ -d "/etc/mosdns" ]]; then
        ln -sf /etc/homeserver/mosdns/config.yaml /etc/mosdns/config.yaml
    fi
    
    # 设置权限
    chown mosdns:mosdns /etc/homeserver/mosdns/config.yaml 2>/dev/null || true
    chmod 644 /etc/homeserver/mosdns/config.yaml
    
    log_info "配置文件部署完成"
}

# 测试配置
test_configuration() {
    log_info "测试配置..."
    
    # 检查上游 DNS 连通性
    log_debug "测试 AdGuardHome (10.0.0.5:53)..."
    if nc -z -w3 10.0.0.5 53 2>/dev/null; then
        log_info "✓ AdGuardHome 连通性正常"
    else
        log_warn "✗ AdGuardHome 无法连接"
    fi
    
    log_debug "测试 mihomo (10.0.0.6:1053)..."
    if nc -z -w3 10.0.0.6 1053 2>/dev/null; then
        log_info "✓ mihomo 连通性正常"
    else
        log_warn "✗ mihomo 无法连接"
    fi
    
    # 测试备用 DNS
    log_debug "测试备用 DNS..."
    if nc -z -w3 223.5.5.5 53 2>/dev/null; then
        log_info "✓ 阿里 DNS 连通性正常"
    else
        log_warn "✗ 阿里 DNS 无法连接"
    fi
    
    log_info "配置测试完成"
}

# 重启服务
restart_service() {
    log_info "重启 MosDNS 服务..."
    
    if systemctl is-active --quiet mosdns; then
        systemctl restart mosdns
        log_info "MosDNS 服务已重启"
    else
        systemctl start mosdns
        log_info "MosDNS 服务已启动"
    fi
    
    # 等待服务启动
    sleep 3
    
    # 检查服务状态
    if systemctl is-active --quiet mosdns; then
        log_info "✓ MosDNS 服务运行正常"
    else
        log_error "✗ MosDNS 服务启动失败"
        echo "请检查日志: journalctl -u mosdns -f"
        exit 1
    fi
    
    # 检查端口监听
    if ss -tulnp | grep -q ":53 "; then
        log_info "✓ DNS 端口监听正常"
    else
        log_warn "✗ DNS 端口未监听"
    fi
}

# 性能测试
performance_test() {
    log_info "执行性能测试..."
    
    # 等待服务完全启动
    sleep 5
    
    # 测试基础解析
    log_debug "测试基础 DNS 解析..."
    if nslookup baidu.com 127.0.0.1 >/dev/null 2>&1; then
        log_info "✓ 基础 DNS 解析正常"
    else
        log_warn "✗ 基础 DNS 解析失败"
    fi
    
    # 测试缓存效果
    log_debug "测试缓存效果..."
    start_time=$(date +%s%3N)
    nslookup github.com 127.0.0.1 >/dev/null 2>&1
    first_time=$(($(date +%s%3N) - start_time))
    
    start_time=$(date +%s%3N)
    nslookup github.com 127.0.0.1 >/dev/null 2>&1
    second_time=$(($(date +%s%3N) - start_time))
    
    log_info "首次查询用时: ${first_time}ms"
    log_info "缓存查询用时: ${second_time}ms"
    
    if [[ $second_time -lt $first_time ]]; then
        log_info "✓ 缓存工作正常"
    else
        log_warn "✗ 缓存可能未工作"
    fi
}

# 生成使用报告
generate_report() {
    log_info "生成部署报告..."
    
    local report_file="/tmp/mosdns-deployment-report.txt"
    cat > "$report_file" << EOF
MosDNS 优化配置部署报告
部署时间: $(date)
操作系统: $(uname -a)

配置文件位置:
- 主配置: /etc/homeserver/mosdns/config.yaml
- 数据目录: /opt/homeserver/data/
- 规则目录: /opt/homeserver/rules/
- 日志目录: /var/log/homeserver/mosdns/

备份位置:
$(cat /tmp/mosdns_backup_path 2>/dev/null || echo "无备份")

服务状态:
$(systemctl status mosdns --no-pager -l)

端口监听:
$(ss -tulnp | grep :53)

配置验证:
$(mosdns start -c /etc/homeserver/mosdns/config.yaml --dry-run 2>&1 || echo "验证失败")

建议的下一步操作:
1. 检查 API 接口: curl http://127.0.0.1:9091/metrics
2. 查看实时日志: journalctl -u mosdns -f
3. 测试分流效果: ./test_deployment.sh
4. 配置定时更新: crontab -e

EOF
    
    log_info "部署报告已生成: $report_file"
    echo
    cat "$report_file"
}

# 清理函数
cleanup() {
    log_info "清理临时文件..."
    rm -f /tmp/mosdns_backup_path
}

# 主函数
main() {
    log_info "开始部署 MosDNS 优化配置..."
    echo
    
    # 检查权限
    check_root
    
    # 检查依赖
    check_dependencies
    
    # 备份现有配置
    backup_config
    
    # 创建目录结构
    create_directories
    
    # 下载数据文件
    download_data_files
    
    # 创建规则文件
    create_rule_files
    
    # 部署配置文件
    deploy_config
    
    # 测试配置
    test_configuration
    
    # 重启服务
    restart_service
    
    # 性能测试
    performance_test
    
    # 生成报告
    generate_report
    
    # 清理
    cleanup
    
    echo
    log_info "🎉 MosDNS 优化配置部署完成！"
    log_info "请查看上方的部署报告了解详细信息"
}

# 信号处理
trap cleanup EXIT
trap 'log_error "部署过程被中断"; exit 1' INT TERM

# 参数处理
case "${1:-}" in
    "--test")
        log_info "测试模式：只验证配置，不实际部署"
        check_dependencies
        test_configuration
        exit 0
        ;;
    "--backup-only")
        log_info "仅备份模式"
        check_root
        backup_config
        exit 0
        ;;
    "--help"|"-h")
        echo "MosDNS 优化配置部署脚本"
        echo
        echo "用法: $0 [选项]"
        echo
        echo "选项:"
        echo "  --test        测试模式，只验证不部署"
        echo "  --backup-only 仅备份现有配置"
        echo "  --help, -h    显示此帮助信息"
        echo
        exit 0
        ;;
    "")
        # 无参数，执行主部署
        main
        ;;
    *)
        log_error "未知参数: $1"
        echo "使用 $0 --help 查看帮助"
        exit 1
        ;;
esac
