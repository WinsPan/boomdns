#!/bin/bash

# MosDNS 定时更新任务安装脚本

set -euo pipefail

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

# 检查运行权限
check_privileges() {
    if [[ $EUID -ne 0 ]]; then
        log_error "请以root权限运行此脚本"
        exit 1
    fi
}

# 检查更新脚本是否存在
check_update_script() {
    local script_path="/usr/local/bin/update-mosdns-rules.sh"
    
    if [[ ! -f "$script_path" ]]; then
        log_info "更新脚本不存在，正在安装..."
        
        # 检查当前目录是否有更新脚本
        if [[ -f "./update-mosdns-rules.sh" ]]; then
            cp "./update-mosdns-rules.sh" "$script_path"
            chmod +x "$script_path"
            log_success "更新脚本已安装到: $script_path"
        else
            log_error "找不到 update-mosdns-rules.sh 脚本文件"
            log_error "请确保脚本文件存在于当前目录"
            exit 1
        fi
    else
        log_info "更新脚本已存在: $script_path"
    fi
}

# 安装日志轮转配置
setup_logrotate() {
    local logrotate_conf="/etc/logrotate.d/mosdns"
    
    log_info "配置日志轮转..."
    
    cat > "$logrotate_conf" << 'EOF'
# MosDNS 日志轮转配置
/var/log/homeserver/mosdns/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 0644 homeserver homeserver
    postrotate
        systemctl reload mosdns 2>/dev/null || true
    endscript
}
EOF
    
    log_success "日志轮转配置已创建: $logrotate_conf"
}

# 创建定时任务
setup_cron() {
    log_info "设置定时更新任务..."
    
    # 检查是否已存在相关的cron任务
    if crontab -l 2>/dev/null | grep -q "update-mosdns-rules.sh"; then
        log_warn "检测到已存在的MosDNS更新任务"
        read -p "是否要替换现有任务？(y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "保留现有任务，跳过cron设置"
            return 0
        fi
        
        # 移除现有任务
        crontab -l 2>/dev/null | grep -v "update-mosdns-rules.sh" | crontab -
        log_info "已移除现有任务"
    fi
    
    # 显示可选的定时任务选项
    echo
    log_info "请选择更新频率："
    echo "1) 每天凌晨2点更新"
    echo "2) 每6小时更新一次"
    echo "3) 每12小时更新一次"
    echo "4) 每周一凌晨2点更新"
    echo "5) 自定义cron表达式"
    echo "6) 跳过定时任务设置"
    echo
    
    read -p "请选择 (1-6): " -n 1 -r choice
    echo
    
    local cron_expr=""
    local description=""
    
    case $choice in
        1)
            cron_expr="0 2 * * *"
            description="每天凌晨2点"
            ;;
        2)
            cron_expr="0 */6 * * *"
            description="每6小时"
            ;;
        3)
            cron_expr="0 */12 * * *"
            description="每12小时"
            ;;
        4)
            cron_expr="0 2 * * 1"
            description="每周一凌晨2点"
            ;;
        5)
            read -p "请输入cron表达式 (例如: 0 2 * * *): " cron_expr
            description="自定义"
            ;;
        6)
            log_info "跳过定时任务设置"
            return 0
            ;;
        *)
            log_error "无效选择"
            return 1
            ;;
    esac
    
    if [[ -z "$cron_expr" ]]; then
        log_error "cron表达式不能为空"
        return 1
    fi
    
    # 添加新的cron任务
    local temp_cron=$(mktemp)
    crontab -l 2>/dev/null > "$temp_cron" || true
    
    cat >> "$temp_cron" << EOF

# MosDNS 规则自动更新任务 - $description
$cron_expr /usr/local/bin/update-mosdns-rules.sh >/dev/null 2>&1
EOF
    
    crontab "$temp_cron"
    rm -f "$temp_cron"
    
    log_success "定时任务已设置: $description 执行更新"
    log_info "cron表达式: $cron_expr"
}

# 创建systemd定时器（可选方案）
setup_systemd_timer() {
    read -p "是否要创建systemd定时器作为备选方案？(y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        return 0
    fi
    
    log_info "创建systemd定时器..."
    
    # 创建service文件
    cat > /etc/systemd/system/mosdns-update.service << 'EOF'
[Unit]
Description=Update MosDNS Rules
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
User=root
ExecStart=/usr/local/bin/update-mosdns-rules.sh
StandardOutput=journal
StandardError=journal
EOF

    # 创建timer文件
    cat > /etc/systemd/system/mosdns-update.timer << 'EOF'
[Unit]
Description=Update MosDNS Rules Timer
Requires=mosdns-update.service

[Timer]
# 每天凌晨2点30分运行
OnCalendar=*-*-* 02:30:00
# 系统启动后15分钟运行一次
OnBootSec=15min
# 如果错过了计划时间，启动后立即运行
Persistent=true

[Install]
WantedBy=timers.target
EOF

    # 重载systemd并启用定时器
    systemctl daemon-reload
    systemctl enable mosdns-update.timer
    systemctl start mosdns-update.timer
    
    log_success "systemd定时器已创建并启用"
    log_info "查看定时器状态: systemctl status mosdns-update.timer"
    log_info "查看下次运行时间: systemctl list-timers mosdns-update.timer"
}

# 测试更新脚本
test_update_script() {
    log_info "测试更新脚本..."
    
    if /usr/local/bin/update-mosdns-rules.sh --test; then
        log_success "更新脚本测试通过"
    else
        log_error "更新脚本测试失败"
        log_error "请检查脚本配置和网络连接"
        return 1
    fi
}

# 显示任务状态
show_status() {
    log_info "定时任务状态:"
    
    echo
    echo "=== Cron任务 ==="
    if crontab -l 2>/dev/null | grep -q "update-mosdns-rules.sh"; then
        crontab -l 2>/dev/null | grep "update-mosdns-rules.sh" | while read line; do
            echo "  $line"
        done
    else
        echo "  未设置cron任务"
    fi
    
    echo
    echo "=== Systemd定时器 ==="
    if systemctl is-enabled mosdns-update.timer >/dev/null 2>&1; then
        echo "  状态: $(systemctl is-active mosdns-update.timer)"
        echo "  启用: $(systemctl is-enabled mosdns-update.timer)"
        systemctl list-timers mosdns-update.timer --no-pager 2>/dev/null | tail -n +2 || true
    else
        echo "  未设置systemd定时器"
    fi
    
    echo
    echo "=== 最近的更新日志 ==="
    if [[ -f "/var/log/homeserver/mosdns/update-rules.log" ]]; then
        tail -n 5 "/var/log/homeserver/mosdns/update-rules.log" 2>/dev/null || echo "  日志文件为空"
    else
        echo "  日志文件不存在"
    fi
}

# 显示帮助信息
show_help() {
    cat << EOF
MosDNS 定时更新任务安装脚本

用法: $0 [选项]

选项:
  -h, --help      显示此帮助信息
  -s, --status    显示当前任务状态
  -t, --test      仅测试更新脚本
  --remove        移除所有定时任务

功能:
  - 安装MosDNS规则自动更新脚本
  - 设置cron定时任务
  - 可选创建systemd定时器
  - 配置日志轮转
  - 测试脚本功能

示例:
  $0              # 完整安装
  $0 --status     # 查看状态
  $0 --test       # 测试脚本
  $0 --remove     # 移除任务
EOF
}

# 移除定时任务
remove_tasks() {
    log_info "移除MosDNS定时更新任务..."
    
    # 移除cron任务
    if crontab -l 2>/dev/null | grep -q "update-mosdns-rules.sh"; then
        crontab -l 2>/dev/null | grep -v "update-mosdns-rules.sh" | crontab -
        log_success "已移除cron任务"
    fi
    
    # 移除systemd定时器
    if systemctl is-enabled mosdns-update.timer >/dev/null 2>&1; then
        systemctl stop mosdns-update.timer
        systemctl disable mosdns-update.timer
        rm -f /etc/systemd/system/mosdns-update.{service,timer}
        systemctl daemon-reload
        log_success "已移除systemd定时器"
    fi
    
    # 移除日志轮转配置
    if [[ -f "/etc/logrotate.d/mosdns" ]]; then
        rm -f "/etc/logrotate.d/mosdns"
        log_success "已移除日志轮转配置"
    fi
    
    log_success "所有定时任务已移除"
}

# 主函数
main() {
    log_info "MosDNS 定时更新任务安装脚本"
    log_info "============================="
    echo
    
    check_privileges
    check_update_script
    setup_logrotate
    setup_cron
    setup_systemd_timer
    
    echo
    log_info "测试更新脚本功能..."
    if test_update_script; then
        echo
        log_success "安装完成！"
        log_info "MosDNS规则将根据设定的时间自动更新"
        log_info "可以使用 'sudo $0 --status' 查看任务状态"
    else
        log_warn "安装完成，但测试失败"
        log_warn "请检查网络连接和配置"
    fi
}

# 解析命令行参数
case "${1:-}" in
    -h|--help)
        show_help
        exit 0
        ;;
    -s|--status)
        check_privileges
        show_status
        exit 0
        ;;
    -t|--test)
        check_privileges
        check_update_script
        test_update_script
        exit $?
        ;;
    --remove)
        check_privileges
        remove_tasks
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
