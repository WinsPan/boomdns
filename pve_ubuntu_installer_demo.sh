#!/bin/bash

# PVE Ubuntu Server 安装脚本 - 演示版本
# 作者: AI Assistant
# 版本: 1.0 (演示版)

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 默认配置
DEFAULT_VM_ID=100
DEFAULT_VM_NAME="ubuntu-server"
DEFAULT_MEMORY=2048
DEFAULT_CORES=2
DEFAULT_DISK_SIZE="20G"
DEFAULT_IP_ADDRESS="192.168.1.100"
DEFAULT_GATEWAY="192.168.1.1"
DEFAULT_DNS="8.8.8.8"
DEFAULT_USERNAME="ubuntu"
DEFAULT_PASSWORD=""
DEFAULT_SSH_KEY=""
DEFAULT_UBUNTU_VERSION="22.04"

# 当前配置
VM_ID=$DEFAULT_VM_ID
VM_NAME=$DEFAULT_VM_NAME
MEMORY=$DEFAULT_MEMORY
CORES=$DEFAULT_CORES
DISK_SIZE=$DEFAULT_DISK_SIZE
IP_ADDRESS=$DEFAULT_IP_ADDRESS
GATEWAY=$DEFAULT_GATEWAY
DNS=$DEFAULT_DNS
USERNAME=$DEFAULT_USERNAME
PASSWORD=$DEFAULT_PASSWORD
SSH_KEY=$DEFAULT_SSH_KEY
UBUNTU_VERSION=$DEFAULT_UBUNTU_VERSION

# 演示模式标志
DEMO_MODE=true

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
    echo "    PVE Ubuntu Server 安装脚本 (演示版)"
    echo "=========================================="
    echo
}

# 显示当前配置
show_current_config() {
    print_message $BLUE "当前配置:"
    echo "VM ID: $VM_ID"
    echo "VM 名称: $VM_NAME"
    echo "内存: ${MEMORY}MB"
    echo "CPU 核心数: $CORES"
    echo "磁盘大小: $DISK_SIZE"
    echo "IP 地址: $IP_ADDRESS"
    echo "网关: $GATEWAY"
    echo "DNS: $DNS"
    echo "用户名: $USERNAME"
    echo "密码: ${PASSWORD:-未设置}"
    echo "SSH 密钥: ${SSH_KEY:-未设置}"
    echo "Ubuntu 版本: $UBUNTU_VERSION"
    echo
}

# 输入验证函数
validate_ip() {
    local ip=$1
    if [[ $ip =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
        IFS='.' read -r -a ip_parts <<< "$ip"
        for part in "${ip_parts[@]}"; do
            if [[ $part -lt 0 || $part -gt 255 ]]; then
                return 1
            fi
        done
        return 0
    else
        return 1
    fi
}

validate_vm_id() {
    local id=$1
    if [[ $id =~ ^[0-9]+$ ]] && [[ $id -ge 100 && $id -le 999999 ]]; then
        return 0
    else
        return 1
    fi
}

# 配置菜单
configure_menu() {
    while true; do
        show_header
        show_current_config
        
        print_message $YELLOW "配置菜单:"
        echo "1. 设置 VM ID"
        echo "2. 设置 VM 名称"
        echo "3. 设置内存大小"
        echo "4. 设置 CPU 核心数"
        echo "5. 设置磁盘大小"
        echo "6. 设置网络配置"
        echo "7. 设置用户信息"
        echo "8. 设置 SSH 密钥"
        echo "9. 设置 Ubuntu 版本"
        echo "10. 重置为默认配置"
        echo "11. 开始安装 (演示模式)"
        echo "0. 退出"
        echo
        
        read -p "请选择操作 (0-11): " choice
        
        case $choice in
            1)
                read -p "请输入 VM ID (100-999999): " new_vm_id
                if validate_vm_id "$new_vm_id"; then
                    VM_ID=$new_vm_id
                    print_message $GREEN "VM ID 已更新为: $VM_ID"
                else
                    print_message $RED "无效的 VM ID，请输入 100-999999 之间的数字"
                fi
                read -p "按回车键继续..."
                ;;
            2)
                read -p "请输入 VM 名称: " new_vm_name
                if [[ -n "$new_vm_name" ]]; then
                    VM_NAME=$new_vm_name
                    print_message $GREEN "VM 名称已更新为: $VM_NAME"
                else
                    print_message $RED "VM 名称不能为空"
                fi
                read -p "按回车键继续..."
                ;;
            3)
                read -p "请输入内存大小 (MB): " new_memory
                if [[ $new_memory =~ ^[0-9]+$ ]] && [[ $new_memory -ge 512 ]]; then
                    MEMORY=$new_memory
                    print_message $GREEN "内存大小已更新为: ${MEMORY}MB"
                else
                    print_message $RED "无效的内存大小，请输入至少 512MB"
                fi
                read -p "按回车键继续..."
                ;;
            4)
                read -p "请输入 CPU 核心数: " new_cores
                if [[ $new_cores =~ ^[0-9]+$ ]] && [[ $new_cores -ge 1 ]]; then
                    CORES=$new_cores
                    print_message $GREEN "CPU 核心数已更新为: $CORES"
                else
                    print_message $RED "无效的 CPU 核心数，请输入至少 1"
                fi
                read -p "按回车键继续..."
                ;;
            5)
                read -p "请输入磁盘大小 (如: 20G, 50G): " new_disk_size
                if [[ -n "$new_disk_size" ]]; then
                    DISK_SIZE=$new_disk_size
                    print_message $GREEN "磁盘大小已更新为: $DISK_SIZE"
                else
                    print_message $RED "磁盘大小不能为空"
                fi
                read -p "按回车键继续..."
                ;;
            6)
                print_message $YELLOW "网络配置:"
                read -p "请输入 IP 地址: " new_ip
                if validate_ip "$new_ip"; then
                    IP_ADDRESS=$new_ip
                    print_message $GREEN "IP 地址已更新为: $IP_ADDRESS"
                else
                    print_message $RED "无效的 IP 地址格式"
                fi
                
                read -p "请输入网关地址: " new_gateway
                if validate_ip "$new_gateway"; then
                    GATEWAY=$new_gateway
                    print_message $GREEN "网关已更新为: $GATEWAY"
                else
                    print_message $RED "无效的网关地址格式"
                fi
                
                read -p "请输入 DNS 服务器: " new_dns
                if validate_ip "$new_dns"; then
                    DNS=$new_dns
                    print_message $GREEN "DNS 已更新为: $DNS"
                else
                    print_message $RED "无效的 DNS 地址格式"
                fi
                read -p "按回车键继续..."
                ;;
            7)
                read -p "请输入用户名: " new_username
                if [[ -n "$new_username" ]]; then
                    USERNAME=$new_username
                    print_message $GREEN "用户名已更新为: $USERNAME"
                else
                    print_message $RED "用户名不能为空"
                fi
                
                echo -n "请输入密码: "
                read -s new_password
                echo
                if [[ -n "$new_password" ]]; then
                    PASSWORD=$new_password
                    print_message $GREEN "密码已设置"
                else
                    print_message $RED "密码不能为空"
                fi
                read -p "按回车键继续..."
                ;;
            8)
                read -p "请输入 SSH 公钥文件路径 (可选): " new_ssh_key
                if [[ -n "$new_ssh_key" ]]; then
                    if [[ -f "$new_ssh_key" ]]; then
                        SSH_KEY=$new_ssh_key
                        print_message $GREEN "SSH 密钥已更新为: $SSH_KEY"
                    else
                        print_message $RED "SSH 密钥文件不存在"
                    fi
                else
                    SSH_KEY=""
                    print_message $GREEN "SSH 密钥已清除"
                fi
                read -p "按回车键继续..."
                ;;
            9)
                print_message $YELLOW "可用的 Ubuntu 版本:"
                echo "1. Ubuntu 20.04 LTS"
                echo "2. Ubuntu 22.04 LTS"
                echo "3. Ubuntu 24.04 LTS"
                read -p "请选择 Ubuntu 版本 (1-3): " version_choice
                case $version_choice in
                    1) UBUNTU_VERSION="20.04";;
                    2) UBUNTU_VERSION="22.04";;
                    3) UBUNTU_VERSION="24.04";;
                    *) print_message $RED "无效选择，保持当前版本: $UBUNTU_VERSION";;
                esac
                print_message $GREEN "Ubuntu 版本已更新为: $UBUNTU_VERSION"
                read -p "按回车键继续..."
                ;;
            10)
                VM_ID=$DEFAULT_VM_ID
                VM_NAME=$DEFAULT_VM_NAME
                MEMORY=$DEFAULT_MEMORY
                CORES=$DEFAULT_CORES
                DISK_SIZE=$DEFAULT_DISK_SIZE
                IP_ADDRESS=$DEFAULT_IP_ADDRESS
                GATEWAY=$DEFAULT_GATEWAY
                DNS=$DEFAULT_DNS
                USERNAME=$DEFAULT_USERNAME
                PASSWORD=$DEFAULT_PASSWORD
                SSH_KEY=$DEFAULT_SSH_KEY
                UBUNTU_VERSION=$DEFAULT_UBUNTU_VERSION
                print_message $GREEN "配置已重置为默认值"
                read -p "按回车键继续..."
                ;;
            11)
                if [[ -z "$PASSWORD" ]]; then
                    print_message $RED "错误: 密码未设置，请先设置密码"
                    read -p "按回车键继续..."
                    continue
                fi
                install_ubuntu_demo
                return 0
                ;;
            0)
                print_message $YELLOW "退出脚本"
                exit 0
                ;;
            *)
                print_message $RED "无效选择，请重新输入"
                read -p "按回车键继续..."
                ;;
        esac
    done
}

# 演示模式安装
install_ubuntu_demo() {
    print_message $GREEN "开始演示模式安装 Ubuntu Server..."
    
    # 确认安装
    print_message $YELLOW "安装配置确认:"
    show_current_config
    read -p "确认开始演示安装? (y/N): " confirm
    
    if [[ $confirm != "y" && $confirm != "Y" ]]; then
        print_message $YELLOW "安装已取消"
        return 0
    fi
    
    print_message $BLUE "=== 演示模式安装流程 ==="
    echo
    
    # 步骤1: 检查环境
    print_message $BLUE "步骤 1: 检查 PVE 环境..."
    sleep 1
    print_message $YELLOW "⚠️  演示模式: 跳过 PVE 环境检查"
    print_message $GREEN "✅ 环境检查完成"
    echo
    
    # 步骤2: 下载镜像
    print_message $BLUE "步骤 2: 下载 Ubuntu $UBUNTU_VERSION 镜像..."
    sleep 1
    print_message $YELLOW "⚠️  演示模式: 跳过镜像下载"
    print_message $GREEN "✅ 镜像下载完成"
    echo
    
    # 步骤3: 创建 cloud-init 配置
    print_message $BLUE "步骤 3: 创建 cloud-init 配置..."
    sleep 1
    print_message $YELLOW "⚠️  演示模式: 跳过配置文件创建"
    print_message $GREEN "✅ cloud-init 配置完成"
    echo
    
    # 步骤4: 创建虚拟机
    print_message $BLUE "步骤 4: 创建虚拟机..."
    sleep 1
    print_message $YELLOW "⚠️  演示模式: 跳过虚拟机创建"
    print_message $GREEN "✅ 虚拟机创建完成"
    echo
    
    # 步骤5: 配置 cloud-init
    print_message $BLUE "步骤 5: 配置 cloud-init..."
    sleep 1
    print_message $YELLOW "⚠️  演示模式: 跳过 cloud-init 配置"
    print_message $GREEN "✅ cloud-init 配置完成"
    echo
    
    # 步骤6: 启动虚拟机
    print_message $BLUE "步骤 6: 启动虚拟机..."
    sleep 1
    print_message $YELLOW "⚠️  演示模式: 跳过虚拟机启动"
    print_message $GREEN "✅ 虚拟机启动完成"
    echo
    
    # 显示安装结果
    print_message $GREEN "🎉 Ubuntu Server 演示安装完成!"
    echo
    print_message $BLUE "安装配置摘要:"
    echo "VM ID: $VM_ID"
    echo "VM 名称: $VM_NAME"
    echo "内存: ${MEMORY}MB"
    echo "CPU 核心数: $CORES"
    echo "磁盘大小: $DISK_SIZE"
    echo "IP 地址: $IP_ADDRESS"
    echo "网关: $GATEWAY"
    echo "DNS: $DNS"
    echo "用户名: $USERNAME"
    echo "密码: $PASSWORD"
    echo "Ubuntu 版本: $UBUNTU_VERSION"
    echo
    print_message $YELLOW "注意: 这是演示模式，实际安装需要在 PVE 环境中运行"
    echo
    print_message $BLUE "在真实的 PVE 环境中，您可以使用以下命令连接到虚拟机:"
    echo "ssh $USERNAME@$IP_ADDRESS"
    echo
    print_message $YELLOW "要使用真实版本，请将脚本复制到 PVE 节点并运行:"
    echo "sudo ./pve_ubuntu_installer.sh"
    
    return 0
}

# 主函数
main() {
    # 显示欢迎信息
    show_header
    print_message $GREEN "欢迎使用 PVE Ubuntu Server 安装脚本 (演示版)"
    echo
    print_message $YELLOW "⚠️  演示模式说明:"
    echo "- 此版本用于演示和测试配置功能"
    echo "- 不会实际创建虚拟机或下载镜像"
    echo "- 真实安装需要在 PVE 环境中运行"
    echo
    print_message $BLUE "功能特性:"
    echo "- 支持 Ubuntu 20.04/22.04/24.04 LTS"
    echo "- 自动配置网络设置"
    echo "- 支持 SSH 密钥认证"
    echo "- 自动安装常用软件包"
    echo "- 菜单式配置界面"
    echo
    
    # 显示当前配置
    show_current_config
    
    # 进入配置菜单
    configure_menu
}

# 运行主函数
main "$@" 