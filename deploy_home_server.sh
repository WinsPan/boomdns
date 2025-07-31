#!/bin/bash

# PVE家庭服务器自动部署脚本
# 功能: 在Ubuntu VM中部署MosDNS + mihomo分流方案
# 作者: AI Assistant
# 版本: 1.0

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 配置变量
MOSDNS_VERSION="5.3.1"
MIHOMO_VERSION="1.18.1"
INSTALL_DIR="/opt/homeserver"
CONFIG_DIR="/etc/homeserver"
LOG_DIR="/var/log/homeserver"
SERVICE_USER="homeserver"

# 网络配置
DNS_PORT="53"
HTTP_PORT="7890"
SOCKS_PORT="7891"
DASHBOARD_PORT="9090"
MOSDNS_ALT_PORT="1053"

# 打印带颜色的消息
print_message() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# 显示标题
show_header() {
    clear
    echo "=========================================="
    echo "    PVE家庭服务器自动部署脚本"
    echo "    MosDNS + mihomo 分流方案"
    echo "=========================================="
    echo
}

# 检查系统环境
check_environment() {
    print_message $BLUE "检查系统环境..."
    
    # 检查是否为Ubuntu系统
    if [[ ! -f /etc/lsb-release ]] || ! grep -q "Ubuntu" /etc/lsb-release; then
        print_message $RED "错误: 此脚本仅支持Ubuntu系统"
        exit 1
    fi
    
    # 检查是否为root用户
    if [[ $EUID -ne 0 ]]; then
        print_message $RED "错误: 此脚本需要root权限运行"
        exit 1
    fi
    
    # 检查网络连接
    if ! ping -c 1 8.8.8.8 &> /dev/null; then
        print_message $RED "错误: 网络连接失败，请检查网络设置"
        exit 1
    fi
    
    print_message $GREEN "✅ 系统环境检查通过"
}

# 创建用户和目录
setup_directories() {
    print_message $BLUE "创建用户和目录结构..."
    
    # 创建服务用户
    if ! id -u $SERVICE_USER &> /dev/null; then
        useradd -r -s /bin/false -d $INSTALL_DIR $SERVICE_USER
        print_message $GREEN "✅ 创建服务用户: $SERVICE_USER"
    fi
    
    # 创建目录
    for dir in $INSTALL_DIR $CONFIG_DIR $LOG_DIR; do
        mkdir -p $dir
        chown $SERVICE_USER:$SERVICE_USER $dir
        chmod 755 $dir
    done
    
    # 创建子目录
    mkdir -p $INSTALL_DIR/{mosdns,mihomo,data,rules}
    mkdir -p $CONFIG_DIR/{mosdns,mihomo}
    mkdir -p $LOG_DIR/{mosdns,mihomo}
    
    chown -R $SERVICE_USER:$SERVICE_USER $INSTALL_DIR $CONFIG_DIR $LOG_DIR
    
    print_message $GREEN "✅ 目录结构创建完成"
}

# 安装系统依赖
install_dependencies() {
    print_message $BLUE "安装系统依赖..."
    
    # 更新包列表
    apt update -y
    
    # 安装必要软件包
    apt install -y \
        curl \
        wget \
        unzip \
        jq \
        systemd \
        cron \
        iptables \
        ipset \
        dnsutils \
        net-tools \
        htop \
        vim \
        git
    
    print_message $GREEN "✅ 系统依赖安装完成"
}

# 下载和安装MosDNS
install_mosdns() {
    print_message $BLUE "下载和安装MosDNS..."
    
    local arch=""
    case $(uname -m) in
        x86_64) arch="amd64" ;;
        aarch64) arch="arm64" ;;
        armv7l) arch="arm-7" ;;
        *) 
            print_message $RED "不支持的架构: $(uname -m)"
            exit 1
            ;;
    esac
    
    local download_url="https://github.com/IrineSistiana/mosdns/releases/download/v${MOSDNS_VERSION}/mosdns-linux-${arch}.zip"
    local temp_file="/tmp/mosdns.zip"
    
    # 下载MosDNS
    print_message $YELLOW "下载MosDNS v${MOSDNS_VERSION}..."
    if wget -O $temp_file $download_url; then
        print_message $GREEN "✅ MosDNS下载完成"
    else
        print_message $RED "❌ MosDNS下载失败"
        exit 1
    fi
    
    # 解压安装
    cd $INSTALL_DIR/mosdns
    unzip -o $temp_file
    chmod +x mosdns
    chown $SERVICE_USER:$SERVICE_USER mosdns
    rm $temp_file
    
    print_message $GREEN "✅ MosDNS安装完成"
}

# 下载和安装mihomo
install_mihomo() {
    print_message $BLUE "下载和安装mihomo..."
    
    local arch=""
    case $(uname -m) in
        x86_64) arch="amd64" ;;
        aarch64) arch="arm64" ;;
        armv7l) arch="armv7" ;;
        *) 
            print_message $RED "不支持的架构: $(uname -m)"
            exit 1
            ;;
    esac
    
    local download_url="https://github.com/MetaCubeX/mihomo/releases/download/v${MIHOMO_VERSION}/mihomo-linux-${arch}-v${MIHOMO_VERSION}.gz"
    local temp_file="/tmp/mihomo.gz"
    
    # 下载mihomo
    print_message $YELLOW "下载mihomo v${MIHOMO_VERSION}..."
    if wget -O $temp_file $download_url; then
        print_message $GREEN "✅ mihomo下载完成"
    else
        print_message $RED "❌ mihomo下载失败"
        exit 1
    fi
    
    # 解压安装
    cd $INSTALL_DIR/mihomo
    gunzip -c $temp_file > mihomo
    chmod +x mihomo
    chown $SERVICE_USER:$SERVICE_USER mihomo
    rm $temp_file
    
    print_message $GREEN "✅ mihomo安装完成"
}

# 下载规则文件
download_rules() {
    print_message $BLUE "下载规则文件..."
    
    cd $INSTALL_DIR/data
    
    # 下载GeoIP和GeoSite数据
    local files=(
        "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat"
        "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat"
        "https://github.com/Loyalsoldier/clash-rules/releases/latest/download/cncidr.txt"
        "https://github.com/Loyalsoldier/clash-rules/releases/latest/download/direct.txt"
        "https://github.com/Loyalsoldier/clash-rules/releases/latest/download/proxy.txt"
        "https://github.com/Loyalsoldier/clash-rules/releases/latest/download/reject.txt"
    )
    
    for file_url in "${files[@]}"; do
        local filename=$(basename $file_url)
        print_message $YELLOW "下载 $filename..."
        if wget -O $filename $file_url; then
            print_message $GREEN "✅ $filename 下载完成"
        else
            print_message $YELLOW "⚠️ $filename 下载失败，跳过"
        fi
    done
    
    chown -R $SERVICE_USER:$SERVICE_USER $INSTALL_DIR/data
    print_message $GREEN "✅ 规则文件下载完成"
}

# 创建MosDNS配置文件
create_mosdns_config() {
    print_message $BLUE "创建MosDNS配置文件..."
    
    cat > $CONFIG_DIR/mosdns/config.yaml << 'EOF'
# MosDNS配置文件

# 日志配置
log:
  level: info
  file: "/var/log/homeserver/mosdns/mosdns.log"

# 数据源配置
data_sources:
  - tag: geosite
    type: v2ray_geodata
    args:
      file: "/opt/homeserver/data/geosite.dat"
  
  - tag: geoip
    type: v2ray_geodata
    args:
      file: "/opt/homeserver/data/geoip.dat"

# 插件配置
plugins:
  # 缓存插件
  - tag: cache
    type: cache
    args:
      size: 2048
      lazy_cache_ttl: 86400
      dump_file: "/opt/homeserver/data/cache.dump"

  # 国内DNS上游
  - tag: domestic_upstream
    type: forward
    args:
      concurrent: 3
      upstreams:
        - addr: "223.5.5.5:53"
          dial_addr: "223.5.5.5:53"
          trusted: true
        - addr: "114.114.114.114:53"
          dial_addr: "114.114.114.114:53"
          trusted: true
        - addr: "119.29.29.29:53"
          dial_addr: "119.29.29.29:53"
          trusted: true

  # 国外DNS上游
  - tag: foreign_upstream
    type: forward
    args:
      concurrent: 2
      upstreams:
        - addr: "https://1.1.1.1/dns-query"
          dial_addr: "1.1.1.1:443"
          trusted: true
          enable_pipeline: true
        - addr: "https://8.8.8.8/dns-query"
          dial_addr: "8.8.8.8:443"
          trusted: true
          enable_pipeline: true

  # 拒绝请求
  - tag: reject
    type: blackhole
    args:
      rcode: 3

  # 主要序列
  - tag: main_sequence
    type: sequence
    args:
      - exec: query_summary
      
      # 缓存检查
      - exec: $cache
      
      # 拒绝广告域名
      - matches:
          - qname: $geosite:category-ads-all
        exec: $reject
      
      # 国内域名使用国内DNS
      - matches:
          - qname: $geosite:cn
          - qname: $geosite:apple-cn
          - qname: $geosite:google-cn
        exec: $domestic_upstream
      
      # 国外域名使用国外DNS
      - matches:
          - qname: $geosite:geolocation-!cn
          - qname: $geosite:gfw
        exec: $foreign_upstream
      
      # 默认使用国内DNS
      - exec: $domestic_upstream

# 服务器配置
servers:
  - exec: main_sequence
    listeners:
      - protocol: udp
        addr: "0.0.0.0:53"
      - protocol: tcp
        addr: "0.0.0.0:53"
EOF

    chown $SERVICE_USER:$SERVICE_USER $CONFIG_DIR/mosdns/config.yaml
    print_message $GREEN "✅ MosDNS配置文件创建完成"
}

# 创建mihomo配置文件
create_mihomo_config() {
    print_message $BLUE "创建mihomo配置文件..."
    
    cat > $CONFIG_DIR/mihomo/config.yaml << 'EOF'
# mihomo配置文件

# 基础配置
port: 7890
socks-port: 7891
allow-lan: true
bind-address: '*'
mode: rule
log-level: info
external-controller: '0.0.0.0:9090'
external-ui: dashboard
external-ui-name: metacubexd
external-ui-url: "https://github.com/metacubex/metacubexd/archive/refs/heads/gh-pages.zip"

# 实验性功能
experimental:
  ignore-resolve-fail: true
  sniff-tls-sni: true

# 认证
secret: ''

# Tun配置（可选）
tun:
  enable: false
  stack: system
  auto-route: true
  auto-detect-interface: true

# DNS配置
dns:
  enable: true
  listen: 0.0.0.0:1053
  enhanced-mode: fake-ip
  fake-ip-range: 198.18.0.1/16
  fake-ip-filter:
    - '*.lan'
    - '*.local'
    - localhost.ptlogin2.qq.com
    - '+.srv.nintendo.net'
    - '+.stun.playstation.net'
    - xbox.*.microsoft.com
    - '+.xboxlive.com'
  default-nameserver:
    - 223.5.5.5
    - 114.114.114.114
  nameserver:
    - 127.0.0.1:53
  fallback:
    - https://1.1.1.1/dns-query
    - https://dns.google/dns-query
  fallback-filter:
    geoip: true
    geoip-code: CN
    ipcidr:
      - 240.0.0.0/4

# 代理配置（需要用户自己添加）
proxies: []

# 代理组配置
proxy-groups:
  - name: "🚀 节点选择"
    type: select
    proxies:
      - "♻️ 自动选择"
      - "🎯 全球直连"
      - DIRECT

  - name: "♻️ 自动选择"
    type: url-test
    proxies: []
    url: 'http://www.gstatic.com/generate_204'
    interval: 300
    tolerance: 50

  - name: "🌍 国外媒体"
    type: select
    proxies:
      - "🚀 节点选择"
      - "♻️ 自动选择"
      - "🎯 全球直连"

  - name: "📺 国内媒体"
    type: select
    proxies:
      - "🎯 全球直连"
      - "🚀 节点选择"

  - name: "📢 谷歌服务"
    type: select
    proxies:
      - "🚀 节点选择"
      - "🎯 全球直连"

  - name: "🍎 苹果服务"
    type: select
    proxies:
      - "🎯 全球直连"
      - "🚀 节点选择"

  - name: "🎯 全球直连"
    type: select
    proxies:
      - DIRECT

  - name: "🛑 广告拦截"
    type: select
    proxies:
      - REJECT
      - DIRECT

  - name: "🐟 漏网之鱼"
    type: select
    proxies:
      - "🚀 节点选择"
      - "🎯 全球直连"

# 规则配置
rule-providers:
  reject:
    type: http
    behavior: domain
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/reject.txt"
    path: ./ruleset/reject.yaml
    interval: 86400

  icloud:
    type: http
    behavior: domain
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/icloud.txt"
    path: ./ruleset/icloud.yaml
    interval: 86400

  apple:
    type: http
    behavior: domain
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/apple.txt"
    path: ./ruleset/apple.yaml
    interval: 86400

  google:
    type: http
    behavior: domain
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/google.txt"
    path: ./ruleset/google.yaml
    interval: 86400

  proxy:
    type: http
    behavior: domain
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/proxy.txt"
    path: ./ruleset/proxy.yaml
    interval: 86400

  direct:
    type: http
    behavior: domain
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/direct.txt"
    path: ./ruleset/direct.yaml
    interval: 86400

  cncidr:
    type: http
    behavior: ipcidr
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/cncidr.txt"
    path: ./ruleset/cncidr.yaml
    interval: 86400

# 分流规则
rules:
  - RULE-SET,reject,🛑 广告拦截
  - RULE-SET,icloud,🍎 苹果服务
  - RULE-SET,apple,🍎 苹果服务
  - RULE-SET,google,📢 谷歌服务
  - RULE-SET,proxy,🌍 国外媒体
  - RULE-SET,direct,🎯 全球直连
  - RULE-SET,cncidr,🎯 全球直连
  - GEOIP,CN,🎯 全球直连
  - MATCH,🐟 漏网之鱼
EOF

    chown $SERVICE_USER:$SERVICE_USER $CONFIG_DIR/mihomo/config.yaml
    print_message $GREEN "✅ mihomo配置文件创建完成"
}

# 创建systemd服务文件
create_systemd_services() {
    print_message $BLUE "创建systemd服务文件..."
    
    # MosDNS服务
    cat > /etc/systemd/system/mosdns.service << EOF
[Unit]
Description=MosDNS DNS Server
Documentation=https://github.com/IrineSistiana/mosdns
After=network.target
Wants=network.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
ExecStart=$INSTALL_DIR/mosdns/mosdns start -c $CONFIG_DIR/mosdns/config.yaml
ExecReload=/bin/kill -HUP \$MAINPID
Restart=on-failure
RestartSec=5s
LimitNOFILE=1048576
LimitNPROC=1048576

[Install]
WantedBy=multi-user.target
EOF

    # mihomo服务
    cat > /etc/systemd/system/mihomo.service << EOF
[Unit]
Description=mihomo Daemon
Documentation=https://github.com/MetaCubeX/mihomo
After=network.target NetworkManager.service systemd-networkd.service iwd.service
Wants=network.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
ExecStart=$INSTALL_DIR/mihomo/mihomo -d $CONFIG_DIR/mihomo
ExecReload=/bin/kill -HUP \$MAINPID
Restart=on-failure
RestartSec=5s
LimitNOFILE=1048576
LimitNPROC=1048576

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    print_message $GREEN "✅ systemd服务文件创建完成"
}

# 配置防火墙规则
configure_firewall() {
    print_message $BLUE "配置防火墙规则..."
    
    # 检查是否安装了ufw
    if command -v ufw &> /dev/null; then
        # 使用UFW配置
        ufw allow $DNS_PORT/tcp
        ufw allow $DNS_PORT/udp
        ufw allow $HTTP_PORT/tcp
        ufw allow $SOCKS_PORT/tcp
        ufw allow $DASHBOARD_PORT/tcp
        ufw allow $MOSDNS_ALT_PORT/tcp
        ufw allow $MOSDNS_ALT_PORT/udp
        print_message $GREEN "✅ UFW防火墙规则配置完成"
    else
        # 使用iptables配置
        iptables -A INPUT -p tcp --dport $DNS_PORT -j ACCEPT
        iptables -A INPUT -p udp --dport $DNS_PORT -j ACCEPT
        iptables -A INPUT -p tcp --dport $HTTP_PORT -j ACCEPT
        iptables -A INPUT -p tcp --dport $SOCKS_PORT -j ACCEPT
        iptables -A INPUT -p tcp --dport $DASHBOARD_PORT -j ACCEPT
        iptables -A INPUT -p tcp --dport $MOSDNS_ALT_PORT -j ACCEPT
        iptables -A INPUT -p udp --dport $MOSDNS_ALT_PORT -j ACCEPT
        
        # 保存规则
        if command -v iptables-save &> /dev/null; then
            iptables-save > /etc/iptables/rules.v4 2>/dev/null || true
        fi
        print_message $GREEN "✅ iptables防火墙规则配置完成"
    fi
}

# 创建管理脚本
create_management_scripts() {
    print_message $BLUE "创建管理脚本..."
    
    # 创建主管理脚本
    cat > $INSTALL_DIR/homeserver-ctl << 'EOF'
#!/bin/bash

# 家庭服务器管理脚本

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_USER="homeserver"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_message() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

show_status() {
    print_message $BLUE "=== 服务状态 ==="
    echo
    print_message $YELLOW "MosDNS状态:"
    systemctl is-active mosdns || echo "未运行"
    echo
    print_message $YELLOW "mihomo状态:"
    systemctl is-active mihomo || echo "未运行"
    echo
    print_message $YELLOW "端口监听状态:"
    ss -tlnp | grep -E "(53|7890|7891|9090|1053)" || echo "无相关端口监听"
}

start_services() {
    print_message $BLUE "启动所有服务..."
    systemctl start mosdns
    systemctl start mihomo
    print_message $GREEN "✅ 服务启动完成"
    show_status
}

stop_services() {
    print_message $BLUE "停止所有服务..."
    systemctl stop mosdns
    systemctl stop mihomo
    print_message $GREEN "✅ 服务停止完成"
}

restart_services() {
    print_message $BLUE "重启所有服务..."
    systemctl restart mosdns
    systemctl restart mihomo
    print_message $GREEN "✅ 服务重启完成"
    show_status
}

enable_services() {
    print_message $BLUE "启用开机自启..."
    systemctl enable mosdns
    systemctl enable mihomo
    print_message $GREEN "✅ 开机自启已启用"
}

disable_services() {
    print_message $BLUE "禁用开机自启..."
    systemctl disable mosdns
    systemctl disable mihomo
    print_message $GREEN "✅ 开机自启已禁用"
}

show_logs() {
    local service=$1
    if [[ -z "$service" ]]; then
        print_message $YELLOW "请指定服务名称: mosdns 或 mihomo"
        return 1
    fi
    
    print_message $BLUE "显示 $service 日志..."
    journalctl -u $service -f
}

update_rules() {
    print_message $BLUE "更新规则文件..."
    cd /opt/homeserver/data
    
    local files=(
        "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat"
        "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat"
        "https://github.com/Loyalsoldier/clash-rules/releases/latest/download/cncidr.txt"
        "https://github.com/Loyalsoldier/clash-rules/releases/latest/download/direct.txt"
        "https://github.com/Loyalsoldier/clash-rules/releases/latest/download/proxy.txt"
        "https://github.com/Loyalsoldier/clash-rules/releases/latest/download/reject.txt"
    )
    
    for file_url in "${files[@]}"; do
        local filename=$(basename $file_url)
        print_message $YELLOW "更新 $filename..."
        wget -O $filename.tmp $file_url && mv $filename.tmp $filename
    done
    
    chown -R $SERVICE_USER:$SERVICE_USER /opt/homeserver/data
    print_message $GREEN "✅ 规则文件更新完成"
    
    # 重启服务以应用新规则
    restart_services
}

test_dns() {
    print_message $BLUE "测试DNS解析..."
    echo
    print_message $YELLOW "测试国内域名 (baidu.com):"
    nslookup baidu.com 127.0.0.1
    echo
    print_message $YELLOW "测试国外域名 (google.com):"
    nslookup google.com 127.0.0.1
}

test_proxy() {
    print_message $BLUE "测试代理连接..."
    echo
    print_message $YELLOW "测试HTTP代理:"
    curl -I --proxy http://127.0.0.1:7890 http://www.google.com -m 10 || echo "HTTP代理测试失败"
    echo
    print_message $YELLOW "测试SOCKS5代理:"
    curl -I --proxy socks5://127.0.0.1:7891 http://www.google.com -m 10 || echo "SOCKS5代理测试失败"
}

show_help() {
    echo "家庭服务器管理脚本"
    echo
    echo "用法: $0 [命令]"
    echo
    echo "命令:"
    echo "  status      显示服务状态"
    echo "  start       启动所有服务"
    echo "  stop        停止所有服务"
    echo "  restart     重启所有服务"
    echo "  enable      启用开机自启"
    echo "  disable     禁用开机自启"
    echo "  logs <service>  显示服务日志 (mosdns|mihomo)"
    echo "  update      更新规则文件"
    echo "  test-dns    测试DNS解析"
    echo "  test-proxy  测试代理连接"
    echo "  help        显示此帮助信息"
}

# 主逻辑
case "$1" in
    status)
        show_status
        ;;
    start)
        start_services
        ;;
    stop)
        stop_services
        ;;
    restart)
        restart_services
        ;;
    enable)
        enable_services
        ;;
    disable)
        disable_services
        ;;
    logs)
        show_logs $2
        ;;
    update)
        update_rules
        ;;
    test-dns)
        test_dns
        ;;
    test-proxy)
        test_proxy
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        show_help
        exit 1
        ;;
esac
EOF

    chmod +x $INSTALL_DIR/homeserver-ctl
    ln -sf $INSTALL_DIR/homeserver-ctl /usr/local/bin/homeserver-ctl
    
    print_message $GREEN "✅ 管理脚本创建完成"
}

# 创建定时任务
create_cron_jobs() {
    print_message $BLUE "创建定时任务..."
    
    # 创建规则更新任务
    cat > /etc/cron.d/homeserver << EOF
# 家庭服务器定时任务

# 每天凌晨3点更新规则文件
0 3 * * * $SERVICE_USER $INSTALL_DIR/homeserver-ctl update >/dev/null 2>&1

# 每小时清理日志文件
0 * * * * root find /var/log/homeserver -name "*.log" -size +100M -exec truncate -s 0 {} \; >/dev/null 2>&1
EOF

    print_message $GREEN "✅ 定时任务创建完成"
}

# 系统优化
optimize_system() {
    print_message $BLUE "优化系统参数..."
    
    # 网络参数优化
    cat >> /etc/sysctl.conf << EOF

# 家庭服务器网络优化
net.core.rmem_max = 16777216
net.core.wmem_max = 16777216
net.ipv4.tcp_rmem = 4096 16384 16777216
net.ipv4.tcp_wmem = 4096 16384 16777216
net.core.netdev_max_backlog = 5000
net.ipv4.tcp_congestion_control = bbr
net.ipv4.ip_forward = 1
EOF

    # 应用参数
    sysctl -p >/dev/null 2>&1
    
    # 文件描述符限制
    cat >> /etc/security/limits.conf << EOF
$SERVICE_USER soft nofile 65536
$SERVICE_USER hard nofile 65536
EOF

    print_message $GREEN "✅ 系统优化完成"
}

# 显示完成信息
show_completion_info() {
    print_message $GREEN "🎉 家庭服务器部署完成！"
    echo
    print_message $BLUE "服务信息:"
    echo "  - MosDNS DNS服务: 0.0.0.0:53"
    echo "  - mihomo HTTP代理: 0.0.0.0:7890"
    echo "  - mihomo SOCKS5代理: 0.0.0.0:7891"
    echo "  - mihomo Web面板: http://$(hostname -I | awk '{print $1}'):9090"
    echo "  - MosDNS备用DNS: 0.0.0.0:1053"
    echo
    print_message $BLUE "管理命令:"
    echo "  - 服务状态: homeserver-ctl status"
    echo "  - 启动服务: homeserver-ctl start"
    echo "  - 停止服务: homeserver-ctl stop"
    echo "  - 重启服务: homeserver-ctl restart"
    echo "  - 更新规则: homeserver-ctl update"
    echo "  - 测试DNS: homeserver-ctl test-dns"
    echo "  - 测试代理: homeserver-ctl test-proxy"
    echo
    print_message $BLUE "配置文件位置:"
    echo "  - MosDNS: $CONFIG_DIR/mosdns/config.yaml"
    echo "  - mihomo: $CONFIG_DIR/mihomo/config.yaml"
    echo
    print_message $BLUE "日志文件位置:"
    echo "  - MosDNS: $LOG_DIR/mosdns/"
    echo "  - mihomo: journalctl -u mihomo"
    echo
    print_message $YELLOW "⚠️  重要提醒:"
    echo "1. 请在mihomo配置文件中添加您的代理节点信息"
    echo "2. 建议配置客户端DNS指向此服务器: $(hostname -I | awk '{print $1}')"
    echo "3. 代理配置需要手动添加到mihomo配置文件中"
    echo "4. 首次使用请运行: homeserver-ctl start"
}

# 主函数
main() {
    show_header
    
    print_message $GREEN "开始部署家庭服务器..."
    echo
    
    # 执行部署步骤
    check_environment
    setup_directories
    install_dependencies
    install_mosdns
    install_mihomo
    download_rules
    create_mosdns_config
    create_mihomo_config
    create_systemd_services
    configure_firewall
    create_management_scripts
    create_cron_jobs
    optimize_system
    
    echo
    show_completion_info
}

# 检查是否直接运行
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi