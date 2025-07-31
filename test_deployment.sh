#!/bin/bash

# 家庭服务器部署测试脚本
# 用于验证MosDNS和mihomo部署是否正确

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

print_header() {
    echo "=========================================="
    echo "    家庭服务器部署测试"
    echo "=========================================="
    echo
}

# 测试系统环境
test_environment() {
    print_message $BLUE "1. 测试系统环境..."
    
    # 检查操作系统
    if [[ -f /etc/lsb-release ]] && grep -q "Ubuntu" /etc/lsb-release; then
        print_message $GREEN "✅ Ubuntu系统检测正常"
    else
        print_message $RED "❌ 非Ubuntu系统"
        return 1
    fi
    
    # 检查网络连接
    if ping -c 1 8.8.8.8 &> /dev/null; then
        print_message $GREEN "✅ 网络连接正常"
    else
        print_message $RED "❌ 网络连接失败"
        return 1
    fi
    
    echo
}

# 测试文件和目录
test_files() {
    print_message $BLUE "2. 测试文件和目录..."
    
    local dirs=(
        "/opt/homeserver"
        "/etc/homeserver"
        "/var/log/homeserver"
        "/opt/homeserver/mosdns"
        "/opt/homeserver/mihomo"
    )
    
    local files=(
        "/opt/homeserver/mosdns/mosdns"
        "/opt/homeserver/mihomo/mihomo"
        "/etc/homeserver/mosdns/config.yaml"
        "/etc/homeserver/mihomo/config.yaml"
        "/usr/local/bin/homeserver-ctl"
    )
    
    # 检查目录
    for dir in "${dirs[@]}"; do
        if [[ -d "$dir" ]]; then
            print_message $GREEN "✅ 目录存在: $dir"
        else
            print_message $RED "❌ 目录缺失: $dir"
        fi
    done
    
    # 检查文件
    for file in "${files[@]}"; do
        if [[ -f "$file" ]]; then
            print_message $GREEN "✅ 文件存在: $file"
        else
            print_message $RED "❌ 文件缺失: $file"
        fi
    done
    
    echo
}

# 测试服务状态
test_services() {
    print_message $BLUE "3. 测试服务状态..."
    
    # 检查systemd服务文件
    local services=("mosdns" "mihomo")
    
    for service in "${services[@]}"; do
        if [[ -f "/etc/systemd/system/${service}.service" ]]; then
            print_message $GREEN "✅ 服务文件存在: ${service}.service"
            
            # 检查服务状态
            if systemctl is-enabled ${service} &> /dev/null; then
                print_message $GREEN "✅ 服务已启用: $service"
            else
                print_message $YELLOW "⚠️ 服务未启用: $service"
            fi
            
            if systemctl is-active ${service} &> /dev/null; then
                print_message $GREEN "✅ 服务正在运行: $service"
            else
                print_message $YELLOW "⚠️ 服务未运行: $service"
            fi
        else
            print_message $RED "❌ 服务文件缺失: ${service}.service"
        fi
    done
    
    echo
}

# 测试端口监听
test_ports() {
    print_message $BLUE "4. 测试端口监听..."
    
    local ports=(
        "53:DNS服务"
        "7890:HTTP代理"
        "7891:SOCKS5代理"
        "9090:Web管理面板"
        "1053:备用DNS"
    )
    
    for port_info in "${ports[@]}"; do
        local port=$(echo $port_info | cut -d: -f1)
        local desc=$(echo $port_info | cut -d: -f2)
        
        if ss -tlnp | grep -q ":${port} "; then
            print_message $GREEN "✅ 端口监听正常: $port ($desc)"
        else
            print_message $YELLOW "⚠️ 端口未监听: $port ($desc)"
        fi
    done
    
    echo
}

# 测试DNS解析
test_dns() {
    print_message $BLUE "5. 测试DNS解析..."
    
    # 测试本地DNS服务
    if nslookup baidu.com 127.0.0.1 &> /dev/null; then
        print_message $GREEN "✅ 本地DNS解析正常 (baidu.com)"
    else
        print_message $RED "❌ 本地DNS解析失败 (baidu.com)"
    fi
    
    if nslookup google.com 127.0.0.1 &> /dev/null; then
        print_message $GREEN "✅ 本地DNS解析正常 (google.com)"
    else
        print_message $RED "❌ 本地DNS解析失败 (google.com)"
    fi
    
    # 测试DNS响应时间
    local response_time=$(dig @127.0.0.1 github.com +stats | grep "Query time" | awk '{print $4}')
    if [[ -n "$response_time" ]]; then
        print_message $GREEN "✅ DNS响应时间: ${response_time}ms"
    else
        print_message $YELLOW "⚠️ 无法获取DNS响应时间"
    fi
    
    echo
}

# 测试代理连接
test_proxy() {
    print_message $BLUE "6. 测试代理连接..."
    
    # 测试HTTP代理
    if curl -x http://127.0.0.1:7890 --connect-timeout 5 -s http://httpbin.org/ip &> /dev/null; then
        print_message $GREEN "✅ HTTP代理连接正常"
    else
        print_message $YELLOW "⚠️ HTTP代理连接失败（可能需要配置代理节点）"
    fi
    
    # 测试SOCKS5代理
    if curl --socks5 127.0.0.1:7891 --connect-timeout 5 -s http://httpbin.org/ip &> /dev/null; then
        print_message $GREEN "✅ SOCKS5代理连接正常"
    else
        print_message $YELLOW "⚠️ SOCKS5代理连接失败（可能需要配置代理节点）"
    fi
    
    echo
}

# 测试管理命令
test_management() {
    print_message $BLUE "7. 测试管理命令..."
    
    # 检查管理脚本
    if command -v homeserver-ctl &> /dev/null; then
        print_message $GREEN "✅ 管理命令可用: homeserver-ctl"
        
        # 测试命令功能
        if homeserver-ctl help &> /dev/null; then
            print_message $GREEN "✅ 帮助命令正常"
        else
            print_message $RED "❌ 帮助命令失败"
        fi
    else
        print_message $RED "❌ 管理命令不可用: homeserver-ctl"
    fi
    
    echo
}

# 测试配置文件
test_config() {
    print_message $BLUE "8. 测试配置文件..."
    
    # 测试MosDNS配置
    if /opt/homeserver/mosdns/mosdns start -c /etc/homeserver/mosdns/config.yaml --dry-run &> /dev/null; then
        print_message $GREEN "✅ MosDNS配置文件正确"
    else
        print_message $RED "❌ MosDNS配置文件有误"
    fi
    
    # 测试mihomo配置
    if /opt/homeserver/mihomo/mihomo -t -d /etc/homeserver/mihomo &> /dev/null; then
        print_message $GREEN "✅ mihomo配置文件正确"
    else
        print_message $RED "❌ mihomo配置文件有误"
    fi
    
    echo
}

# 生成测试报告
generate_report() {
    print_message $BLUE "9. 生成测试报告..."
    
    local report_file="/tmp/homeserver_test_report.txt"
    
    cat > $report_file << EOF
家庭服务器部署测试报告
生成时间: $(date)
服务器信息: $(hostname) - $(hostname -I | awk '{print $1}')

=== 系统信息 ===
操作系统: $(lsb_release -d | cut -f2)
内核版本: $(uname -r)
系统负载: $(uptime | awk -F'load average:' '{print $2}')

=== 服务状态 ===
MosDNS: $(systemctl is-active mosdns 2>/dev/null || echo "未知")
mihomo: $(systemctl is-active mihomo 2>/dev/null || echo "未知")

=== 端口监听 ===
$(ss -tlnp | grep -E ":(53|7890|7891|9090|1053) " || echo "无相关端口监听")

=== 进程信息 ===
$(ps aux | grep -E "(mosdns|mihomo)" | grep -v grep || echo "无相关进程")

=== 网络连接测试 ===
DNS解析测试 (baidu.com): $(nslookup baidu.com 127.0.0.1 >/dev/null 2>&1 && echo "成功" || echo "失败")
DNS解析测试 (google.com): $(nslookup google.com 127.0.0.1 >/dev/null 2>&1 && echo "成功" || echo "失败")

=== 磁盘使用 ===
$(df -h | grep -E "(homeserver|opt|etc|var)")

=== 内存使用 ===
$(free -h)
EOF

    print_message $GREEN "✅ 测试报告已生成: $report_file"
    print_message $BLUE "可以使用以下命令查看完整报告:"
    echo "cat $report_file"
    
    echo
}

# 显示下一步操作
show_next_steps() {
    print_message $BLUE "10. 下一步操作建议..."
    
    echo "如果测试通过，建议进行以下操作："
    echo
    print_message $GREEN "1. 启动服务:"
    echo "   sudo homeserver-ctl start"
    echo
    print_message $GREEN "2. 启用开机自启:"
    echo "   sudo homeserver-ctl enable"
    echo
    print_message $GREEN "3. 配置客户端:"
    echo "   - 设置DNS服务器为: $(hostname -I | awk '{print $1}')"
    echo "   - 配置代理服务器: $(hostname -I | awk '{print $1}'):7890"
    echo
    print_message $GREEN "4. 访问管理面板:"
    echo "   http://$(hostname -I | awk '{print $1}'):9090"
    echo
    print_message $YELLOW "5. 配置代理节点:"
    echo "   编辑 /etc/homeserver/mihomo/config.yaml 添加您的代理节点"
    echo
    print_message $BLUE "6. 查看详细配置说明:"
    echo "   cat CLIENT_CONFIG.md"
    
    echo
}

# 主函数
main() {
    print_header
    
    # 检查是否为root用户
    if [[ $EUID -ne 0 ]]; then
        print_message $YELLOW "⚠️ 建议使用root权限运行以获得完整测试结果"
        echo
    fi
    
    # 运行所有测试
    test_environment
    test_files
    test_services
    test_ports
    test_dns
    test_proxy
    test_management
    test_config
    generate_report
    show_next_steps
    
    print_message $GREEN "🎉 测试完成！"
}

# 执行主函数
main "$@"