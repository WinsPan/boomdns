#!/bin/bash

# MosDNS 自定义三层DNS架构部署脚本
# 架构: MosDNS(10.0.0.4) -> AdGuardHome(10.0.0.5) + mihomo(10.0.0.6)

set -euo pipefail

# 配置变量
MOSDNS_IP="10.0.0.4"
ADGUARDHOME_IP="10.0.0.5"
MIHOMO_IP="10.0.0.6"
ADGUARDHOME_PORT="53"
MIHOMO_PORT="1053"

SERVICE_USER="homeserver"
INSTALL_DIR="/opt/homeserver"
CONFIG_DIR="/etc/homeserver"
LOG_DIR="/var/log/homeserver"
BACKUP_DIR="/opt/homeserver/backup"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

log_step() {
    echo -e "${PURPLE}[STEP]${NC} $*"
}

# 显示横幅
show_banner() {
    clear
    echo -e "${CYAN}"
    cat << 'EOF'
╔══════════════════════════════════════════════════════════════╗
║                    MosDNS 三层DNS架构部署                     ║
║                                                              ║
║  MosDNS(10.0.0.4) ──┬── AdGuardHome(10.0.0.5) [国内域名]    ║
║                     └── mihomo(10.0.0.6) [国外域名]         ║
╚══════════════════════════════════════════════════════════════╝
EOF
    echo -e "${NC}"
    echo
}

# 检查运行权限
check_privileges() {
    if [[ $EUID -ne 0 ]]; then
        log_error "请以root权限运行此脚本"
        exit 1
    fi
}

# 检查网络连通性
check_network() {
    log_step "检查网络架构和连通性..."
    
    # 检查当前服务器IP
    local current_ip=$(hostname -I | awk '{print $1}')
    log_info "当前服务器IP: $current_ip"
    
    if [[ "$current_ip" != "$MOSDNS_IP" ]]; then
        log_warn "当前服务器IP($current_ip)与配置的MosDNS IP($MOSDNS_IP)不匹配"
        read -p "是否继续部署？(y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "部署已取消"
            exit 0
        fi
    fi
    
    # 检查AdGuardHome连通性
    log_info "检查AdGuardHome连通性 ($ADGUARDHOME_IP:$ADGUARDHOME_PORT)..."
    if timeout 5 bash -c "</dev/tcp/$ADGUARDHOME_IP/$ADGUARDHOME_PORT" 2>/dev/null; then
        log_success "AdGuardHome 连接正常"
    else
        log_error "无法连接到AdGuardHome ($ADGUARDHOME_IP:$ADGUARDHOME_PORT)"
        log_error "请确保AdGuardHome服务正在运行"
        exit 1
    fi
    
    # 检查mihomo连通性
    log_info "检查mihomo连通性 ($MIHOMO_IP:$MIHOMO_PORT)..."
    if timeout 5 bash -c "</dev/tcp/$MIHOMO_IP/$MIHOMO_PORT" 2>/dev/null; then
        log_success "mihomo 连接正常"
    else
        log_error "无法连接到mihomo ($MIHOMO_IP:$MIHOMO_PORT)"
        log_error "请确保mihomo服务正在运行且DNS端口已开启"
        exit 1
    fi
    
    # 测试DNS解析
    log_info "测试上游DNS解析..."
    
    if nslookup baidu.com $ADGUARDHOME_IP >/dev/null 2>&1; then
        log_success "AdGuardHome DNS解析正常"
    else
        log_warn "AdGuardHome DNS解析测试失败"
    fi
    
    if nslookup google.com $MIHOMO_IP -port=$MIHOMO_PORT >/dev/null 2>&1; then
        log_success "mihomo DNS解析正常"
    else
        log_warn "mihomo DNS解析测试失败"
    fi
}

# 创建服务用户
create_service_user() {
    log_step "创建服务用户..."
    
    if ! id "$SERVICE_USER" &>/dev/null; then
        useradd -r -s /bin/false -d /opt/homeserver -c "HomeServer Service User" $SERVICE_USER
        log_success "服务用户 $SERVICE_USER 创建成功"
    else
        log_info "服务用户 $SERVICE_USER 已存在"
    fi
}

# 创建目录结构
create_directories() {
    log_step "创建目录结构..."
    
    local dirs=(
        "$INSTALL_DIR"
        "$INSTALL_DIR/mosdns"
        "$INSTALL_DIR/data"
        "$INSTALL_DIR/rules"
        "$CONFIG_DIR"
        "$CONFIG_DIR/mosdns"
        "$LOG_DIR"
        "$LOG_DIR/mosdns"
        "$BACKUP_DIR"
        "$BACKUP_DIR/config"
    )
    
    for dir in "${dirs[@]}"; do
        mkdir -p "$dir"
        log_info "创建目录: $dir"
    done
    
    # 设置目录权限
    chown -R $SERVICE_USER:$SERVICE_USER $INSTALL_DIR $CONFIG_DIR $LOG_DIR
    chmod -R 755 $INSTALL_DIR $CONFIG_DIR
    chmod -R 750 $LOG_DIR
    
    log_success "目录结构创建完成"
}

# 安装MosDNS
install_mosdns() {
    log_step "安装MosDNS..."
    
    # 检查是否已安装
    if command -v mosdns >/dev/null 2>&1; then
        local version=$(mosdns version 2>/dev/null | head -n1 || echo "unknown")
        log_info "MosDNS已安装: $version"
        read -p "是否要重新安装？(y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            return 0
        fi
    fi
    
    # 检测系统架构
    local arch
    case $(uname -m) in
        x86_64)  arch="amd64" ;;
        aarch64) arch="arm64" ;;
        armv7l)  arch="arm-7" ;;
        *)       
            log_error "不支持的系统架构: $(uname -m)"
            exit 1
            ;;
    esac
    
    log_info "系统架构: $arch"
    
    # 获取最新版本
    log_info "获取MosDNS最新版本..."
    local latest_version
    latest_version=$(curl -s https://api.github.com/repos/IrineSistiana/mosdns/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [[ -z "$latest_version" ]]; then
        log_error "无法获取MosDNS最新版本"
        exit 1
    fi
    
    log_info "最新版本: $latest_version"
    
    # 下载MosDNS
    local download_url="https://github.com/IrineSistiana/mosdns/releases/download/$latest_version/mosdns-linux-$arch.zip"
    local temp_dir=$(mktemp -d)
    
    log_info "下载MosDNS..."
    if ! curl -L -o "$temp_dir/mosdns.zip" "$download_url"; then
        log_error "下载MosDNS失败"
        rm -rf "$temp_dir"
        exit 1
    fi
    
    # 解压并安装
    cd "$temp_dir"
    unzip -q mosdns.zip
    
    # 复制到安装目录
    cp mosdns "$INSTALL_DIR/mosdns/"
    chmod +x "$INSTALL_DIR/mosdns/mosdns"
    
    # 创建符号链接
    ln -sf "$INSTALL_DIR/mosdns/mosdns" /usr/local/bin/mosdns
    
    # 清理临时文件
    rm -rf "$temp_dir"
    
    log_success "MosDNS安装完成"
    log_info "版本: $(mosdns version 2>/dev/null | head -n1)"
}

# 下载数据文件
download_data_files() {
    log_step "下载地理位置数据文件..."
    
    local files=(
        "https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat:geosite.dat"
        "https://github.com/v2fly/geoip/releases/latest/download/geoip.dat:geoip.dat"
    )
    
    for file_info in "${files[@]}"; do
        local url="${file_info%%:*}"
        local filename="${file_info##*:}"
        local filepath="$INSTALL_DIR/data/$filename"
        
        log_info "下载 $filename..."
        if curl -L -o "$filepath" "$url"; then
            log_success "$filename 下载完成"
        else
            log_error "$filename 下载失败"
            exit 1
        fi
    done
    
    # 设置文件权限
    chown -R $SERVICE_USER:$SERVICE_USER "$INSTALL_DIR/data"
}

# 创建配置文件
create_config_files() {
    log_step "创建配置文件..."
    
    # 备份现有配置
    if [[ -f "$CONFIG_DIR/mosdns/config.yaml" ]]; then
        local backup_file="$BACKUP_DIR/config/config.yaml.$(date +%Y%m%d_%H%M%S)"
        cp "$CONFIG_DIR/mosdns/config.yaml" "$backup_file"
        log_info "现有配置已备份到: $backup_file"
    fi
    
    # 检查自定义配置文件是否存在
    if [[ -f "./mosdns-custom-config.yaml" ]]; then
        cp "./mosdns-custom-config.yaml" "$CONFIG_DIR/mosdns/config.yaml"
        log_success "使用自定义配置文件"
    else
        log_error "找不到mosdns-custom-config.yaml配置文件"
        log_error "请确保配置文件存在于当前目录"
        exit 1
    fi
    
    # 复制本地域名规则文件
    if [[ -f "./local-domains.txt" ]]; then
        cp "./local-domains.txt" "$INSTALL_DIR/rules/local.txt"
        log_success "本地域名规则文件已复制"
    else
        log_warn "本地域名规则文件不存在，创建空文件"
        touch "$INSTALL_DIR/rules/local.txt"
    fi
    
    # 创建其他规则文件
    touch "$INSTALL_DIR/rules/direct.txt"
    touch "$INSTALL_DIR/rules/proxy.txt"
    
    # 设置权限
    chown -R $SERVICE_USER:$SERVICE_USER "$CONFIG_DIR/mosdns" "$INSTALL_DIR/rules"
    
    log_success "配置文件创建完成"
}

# 创建systemd服务
create_systemd_service() {
    log_step "创建systemd服务..."
    
    cat > /etc/systemd/system/mosdns.service << EOF
[Unit]
Description=MosDNS DNS Server (Custom 3-Tier Architecture)
Documentation=https://github.com/IrineSistiana/mosdns
After=network-online.target
Wants=network-online.target
RequiresMountsFor=$INSTALL_DIR $CONFIG_DIR $LOG_DIR

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
ExecStart=$INSTALL_DIR/mosdns/mosdns start -c $CONFIG_DIR/mosdns/config.yaml
ExecReload=/bin/kill -HUP \$MAINPID
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=mosdns
KillMode=mixed
KillSignal=SIGTERM
TimeoutStopSec=30

# 安全设置
NoNewPrivileges=true
PrivateTmp=true
ProtectHome=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictSUIDSGID=true
RestrictRealtime=true
LockPersonality=true

# 网络权限
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    log_success "systemd服务创建完成"
}

# 配置防火墙
configure_firewall() {
    log_step "配置防火墙规则..."
    
    # 检查防火墙类型
    if command -v ufw >/dev/null 2>&1 && ufw status | grep -q "Status: active"; then
        log_info "配置UFW防火墙..."
        ufw allow 53/udp comment "MosDNS DNS"
        ufw allow 53/tcp comment "MosDNS DNS"
        ufw allow 1053/udp comment "MosDNS DNS (backup)"
        log_success "UFW防火墙规则已添加"
        
    elif command -v firewall-cmd >/dev/null 2>&1 && systemctl is-active firewalld >/dev/null 2>&1; then
        log_info "配置firewalld防火墙..."
        firewall-cmd --permanent --add-port=53/udp --add-port=53/tcp
        firewall-cmd --permanent --add-port=1053/udp
        firewall-cmd --reload
        log_success "firewalld防火墙规则已添加"
        
    elif command -v iptables >/dev/null 2>&1; then
        log_info "配置iptables防火墙..."
        iptables -A INPUT -p udp --dport 53 -j ACCEPT
        iptables -A INPUT -p tcp --dport 53 -j ACCEPT
        iptables -A INPUT -p udp --dport 1053 -j ACCEPT
        
        # 尝试保存规则
        if command -v iptables-save >/dev/null 2>&1; then
            iptables-save > /etc/iptables/rules.v4 2>/dev/null || true
        fi
        log_success "iptables防火墙规则已添加"
        
    else
        log_warn "未检测到活动的防火墙，请手动开放53和1053端口"
    fi
}

# 测试配置
test_configuration() {
    log_step "测试配置文件..."
    
    # 验证配置文件语法
    if mosdns verify -c "$CONFIG_DIR/mosdns/config.yaml"; then
        log_success "配置文件语法正确"
    else
        log_error "配置文件语法错误"
        return 1
    fi
    
    # 测试上游连接
    log_info "测试上游服务连接..."
    
    # 测试AdGuardHome
    if timeout 5 bash -c "</dev/tcp/$ADGUARDHOME_IP/$ADGUARDHOME_PORT" 2>/dev/null; then
        log_success "AdGuardHome ($ADGUARDHOME_IP:$ADGUARDHOME_PORT) 连接正常"
    else
        log_error "AdGuardHome连接失败"
        return 1
    fi
    
    # 测试mihomo
    if timeout 5 bash -c "</dev/tcp/$MIHOMO_IP/$MIHOMO_PORT" 2>/dev/null; then
        log_success "mihomo ($MIHOMO_IP:$MIHOMO_PORT) 连接正常"
    else
        log_error "mihomo连接失败"
        return 1
    fi
    
    return 0
}

# 启动服务
start_services() {
    log_step "启动MosDNS服务..."
    
    # 启动服务
    systemctl enable mosdns
    systemctl start mosdns
    
    # 等待服务启动
    sleep 3
    
    # 检查服务状态
    if systemctl is-active --quiet mosdns; then
        log_success "MosDNS服务启动成功"
    else
        log_error "MosDNS服务启动失败"
        log_error "查看日志: journalctl -u mosdns -f"
        return 1
    fi
    
    # 检查端口监听
    if ss -tlnp | grep -q ":53 "; then
        log_success "DNS端口(53)监听正常"
    else
        log_error "DNS端口(53)未监听"
        return 1
    fi
}

# 功能测试
test_functionality() {
    log_step "测试DNS分流功能..."
    
    # 等待服务完全启动
    sleep 5
    
    local test_results=()
    
    # 测试国内域名解析
    log_info "测试国内域名解析 (baidu.com)..."
    if timeout 10 nslookup baidu.com 127.0.0.1 >/dev/null 2>&1; then
        log_success "国内域名解析正常"
        test_results+=("国内域名: ✓")
    else
        log_error "国内域名解析失败"
        test_results+=("国内域名: ✗")
    fi
    
    # 测试国外域名解析
    log_info "测试国外域名解析 (google.com)..."
    if timeout 10 nslookup google.com 127.0.0.1 >/dev/null 2>&1; then
        log_success "国外域名解析正常"
        test_results+=("国外域名: ✓")
    else
        log_warn "国外域名解析失败或较慢"
        test_results+=("国外域名: ⚠")
    fi
    
    # 测试本地域名解析
    log_info "测试本地域名解析 (pve.local)..."
    if timeout 5 nslookup pve.local 127.0.0.1 >/dev/null 2>&1; then
        log_success "本地域名解析正常"
        test_results+=("本地域名: ✓")
    else
        log_warn "本地域名解析失败(可能域名不存在)"
        test_results+=("本地域名: ⚠")
    fi
    
    # 显示测试结果
    echo
    log_info "DNS分流测试结果:"
    for result in "${test_results[@]}"; do
        echo "  $result"
    done
}

# 创建管理脚本
create_management_script() {
    log_step "创建管理脚本..."
    
    cat > /usr/local/bin/mosdns-ctl << 'EOF'
#!/bin/bash

# MosDNS 管理脚本

CMD="${1:-help}"

case "$CMD" in
    status|st)
        echo "=== MosDNS Service Status ==="
        systemctl status mosdns --no-pager
        echo
        echo "=== Port Listening ==="
        ss -tlnp | grep ":53"
        ;;
        
    start)
        echo "启动MosDNS服务..."
        systemctl start mosdns
        ;;
        
    stop)
        echo "停止MosDNS服务..."
        systemctl stop mosdns
        ;;
        
    restart)
        echo "重启MosDNS服务..."
        systemctl restart mosdns
        ;;
        
    reload)
        echo "重载MosDNS配置..."
        systemctl reload mosdns
        ;;
        
    logs)
        journalctl -u mosdns -f
        ;;
        
    test)
        echo "测试DNS解析..."
        echo "国内域名(baidu.com):"
        nslookup baidu.com 127.0.0.1 | grep -A2 "Non-authoritative"
        echo
        echo "国外域名(google.com):"
        nslookup google.com 127.0.0.1 | grep -A2 "Non-authoritative"
        ;;
        
    config)
        nano /etc/homeserver/mosdns/config.yaml
        ;;
        
    verify)
        echo "验证配置文件..."
        mosdns verify -c /etc/homeserver/mosdns/config.yaml
        ;;
        
    help|*)
        cat << HELP
MosDNS 管理脚本

用法: $(basename "$0") <command>

命令:
  status, st    查看服务状态
  start         启动服务
  stop          停止服务  
  restart       重启服务
  reload        重载配置
  logs          查看实时日志
  test          测试DNS解析
  config        编辑配置文件
  verify        验证配置文件
  help          显示此帮助

示例:
  $(basename "$0") status
  $(basename "$0") test
  $(basename "$0") logs
HELP
        ;;
esac
EOF

    chmod +x /usr/local/bin/mosdns-ctl
    log_success "管理脚本创建完成: /usr/local/bin/mosdns-ctl"
}

# 显示部署总结
show_summary() {
    echo
    echo -e "${CYAN}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║                        部署完成总结                           ║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo
    
    log_success "MosDNS 三层DNS架构部署完成！"
    echo
    
    echo -e "${BLUE}架构信息:${NC}"
    echo "  MosDNS DNS服务器: $MOSDNS_IP:53"
    echo "  AdGuardHome (国内): $ADGUARDHOME_IP:$ADGUARDHOME_PORT"
    echo "  mihomo (国外): $MIHOMO_IP:$MIHOMO_PORT"
    echo
    
    echo -e "${BLUE}服务状态:${NC}"
    if systemctl is-active --quiet mosdns; then
        echo "  MosDNS服务: ✓ 运行中"
    else
        echo "  MosDNS服务: ✗ 未运行"
    fi
    
    if ss -tlnp | grep -q ":53 "; then
        echo "  DNS端口: ✓ 监听中"
    else
        echo "  DNS端口: ✗ 未监听"
    fi
    echo
    
    echo -e "${BLUE}管理命令:${NC}"
    echo "  查看状态: mosdns-ctl status"
    echo "  测试解析: mosdns-ctl test"
    echo "  查看日志: mosdns-ctl logs"
    echo "  重启服务: mosdns-ctl restart"
    echo
    
    echo -e "${BLUE}配置文件:${NC}"
    echo "  主配置: $CONFIG_DIR/mosdns/config.yaml"
    echo "  本地域名: $INSTALL_DIR/rules/local.txt"
    echo "  日志文件: $LOG_DIR/mosdns/mosdns.log"
    echo
    
    echo -e "${BLUE}下一步:${NC}"
    echo "  1. 在ROS路由器中设置DNS为: $MOSDNS_IP"
    echo "  2. 测试客户端DNS解析功能"
    echo "  3. 监控服务运行状态"
    echo "  4. 根据需要调整域名分流规则"
    echo
    
    echo -e "${GREEN}部署成功！您的三层DNS分流架构已就绪。${NC}"
}

# 主函数
main() {
    show_banner
    
    log_info "开始部署MosDNS三层DNS架构..."
    log_info "目标架构: MosDNS($MOSDNS_IP) -> AdGuardHome($ADGUARDHOME_IP) + mihomo($MIHOMO_IP)"
    echo
    
    # 执行部署步骤
    check_privileges
    check_network
    create_service_user
    create_directories
    install_mosdns
    download_data_files
    create_config_files
    create_systemd_service
    configure_firewall
    
    # 测试和启动
    if test_configuration; then
        start_services
        test_functionality
        create_management_script
        show_summary
    else
        log_error "配置测试失败，请检查配置文件和上游服务"
        exit 1
    fi
}

# 帮助信息
show_help() {
    cat << EOF
MosDNS 自定义三层DNS架构部署脚本

架构: MosDNS(10.0.0.4) -> AdGuardHome(10.0.0.5) + mihomo(10.0.0.6)

用法: $0 [选项]

选项:
  -h, --help     显示此帮助信息
  -t, --test     仅测试环境，不执行部署
  --check        检查环境和依赖

前置要求:
  1. AdGuardHome在10.0.0.5:53运行
  2. mihomo在10.0.0.6:1053运行DNS服务
  3. 当前服务器IP为10.0.0.4
  4. 配置文件mosdns-custom-config.yaml存在

示例:
  $0              # 执行完整部署
  $0 --test       # 测试环境
  $0 --check      # 检查依赖
EOF
}

# 解析命令行参数
case "${1:-}" in
    -h|--help)
        show_help
        exit 0
        ;;
    -t|--test)
        show_banner
        check_privileges
        check_network
        log_success "环境测试通过"
        exit 0
        ;;
    --check)
        show_banner
        check_privileges
        check_network
        exit 0
        ;;
    "")
        main
        ;;
    *)
        echo "未知选项: $1"
        show_help
        exit 1
        ;;
esac
