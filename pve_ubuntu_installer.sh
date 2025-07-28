#!/bin/bash

# PVE Ubuntu Server 安装脚本
# 作者: AI Assistant
# 版本: 1.0

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
    echo "    PVE Ubuntu Server 安装脚本"
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
        echo "11. 开始安装"
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
                install_ubuntu
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

# 检查 PVE 环境
check_pve_environment() {
    print_message $BLUE "检查 PVE 环境..."
    
    # 检查是否在 PVE 节点上
    if [[ ! -f /etc/pve/version ]]; then
        print_message $RED "错误: 此脚本需要在 PVE 节点上运行"
        exit 1
    fi
    
    # 检查 qm 命令
    if ! command -v qm &> /dev/null; then
        print_message $RED "错误: qm 命令不可用"
        exit 1
    fi
    
    # 检查存储
    local storages=$(qm config --current 1 2>/dev/null | grep -o 'storage=[^,]*' | cut -d= -f2 | head -1)
    if [[ -z "$storages" ]]; then
        print_message $YELLOW "警告: 未找到可用的存储，将使用默认存储"
    fi
    
    print_message $GREEN "PVE 环境检查通过"
}

# 下载 Ubuntu 镜像
download_ubuntu_image() {
    print_message $BLUE "下载 Ubuntu $UBUNTU_VERSION 镜像..."
    
    local image_url=""
    case $UBUNTU_VERSION in
        "20.04")
            image_url="https://cloud-images.ubuntu.com/releases/20.04/release/ubuntu-20.04-server-cloudimg-amd64.img"
            ;;
        "22.04")
            image_url="https://cloud-images.ubuntu.com/releases/22.04/release/ubuntu-22.04-server-cloudimg-amd64.img"
            ;;
        "24.04")
            image_url="https://cloud-images.ubuntu.com/releases/24.04/release/ubuntu-24.04-server-cloudimg-amd64.img"
            ;;
        *)
            print_message $RED "不支持的 Ubuntu 版本: $UBUNTU_VERSION"
            return 1
            ;;
    esac
    
    local image_name="ubuntu-${UBUNTU_VERSION}-server-cloudimg-amd64.img"
    local image_path="/var/lib/vz/template/cache/$image_name"
    
    if [[ -f "$image_path" ]]; then
        print_message $GREEN "镜像已存在: $image_path"
        return 0
    fi
    
    print_message $YELLOW "开始下载镜像..."
    if wget -O "$image_path" "$image_url"; then
        print_message $GREEN "镜像下载完成: $image_path"
        return 0
    else
        print_message $RED "镜像下载失败"
        return 1
    fi
}

# 创建 cloud-init 配置
create_cloud_init_config() {
    print_message $BLUE "创建 cloud-init 配置..."
    
    local cloud_init_dir="/var/lib/vz/snippets"
    local config_file="$cloud_init_dir/ubuntu-${VM_ID}-cloud-init.yml"
    
    # 创建 snippets 目录
    mkdir -p "$cloud_init_dir"
    
    # 生成 SSH 密钥配置
    local ssh_key_content=""
    if [[ -n "$SSH_KEY" && -f "$SSH_KEY" ]]; then
        ssh_key_content=$(cat "$SSH_KEY")
    fi
    
    # 创建 cloud-init 配置
    cat > "$config_file" << EOF
#cloud-config
hostname: $VM_NAME
manage_etc_hosts: true

users:
  - name: $USERNAME
    sudo: ALL=(ALL) NOPASSWD:ALL
    shell: /bin/bash
    password: $PASSWORD
    lock_passwd: false
    ssh_pwauth: true
EOF

    if [[ -n "$ssh_key_content" ]]; then
        cat >> "$config_file" << EOF
    ssh_authorized_keys:
      - $ssh_key_content
EOF
    fi

    cat >> "$config_file" << EOF

# 配置网络
network:
  version: 2
  ethernets:
    eth0:
      dhcp4: false
      addresses:
        - $IP_ADDRESS/24
      gateway4: $GATEWAY
      nameservers:
        addresses:
          - $DNS

# 更新系统
package_update: true
package_upgrade: true

# 安装常用软件包
packages:
  - qemu-guest-agent
  - curl
  - wget
  - vim
  - htop

# 运行命令
runcmd:
  - systemctl enable qemu-guest-agent
  - systemctl start qemu-guest-agent
EOF

    print_message $GREEN "cloud-init 配置已创建: $config_file"
    return 0
}

# 创建虚拟机
create_virtual_machine() {
    print_message $BLUE "创建虚拟机..."
    
    # 检查 VM ID 是否已存在
    if qm list | grep -q " $VM_ID "; then
        print_message $RED "错误: VM ID $VM_ID 已存在"
        return 1
    fi
    
    # 获取存储
    local storage=$(qm config --current 1 2>/dev/null | grep -o 'storage=[^,]*' | cut -d= -f2 | head -1)
    if [[ -z "$storage" ]]; then
        storage="local"
    fi
    
    # 创建虚拟机
    print_message $YELLOW "创建虚拟机 ID: $VM_ID, 名称: $VM_NAME"
    qm create $VM_ID \
        --name "$VM_NAME" \
        --memory $MEMORY \
        --cores $CORES \
        --sockets 1 \
        --net0 name=eth0,bridge=vmbr0,model=virtio \
        --scsihw virtio-scsi-pci
    
    if [[ $? -ne 0 ]]; then
        print_message $RED "创建虚拟机失败"
        return 1
    fi
    
    # 导入镜像
    local image_name="ubuntu-${UBUNTU_VERSION}-server-cloudimg-amd64.img"
    local image_path="/var/lib/vz/template/cache/$image_name"
    
    print_message $YELLOW "导入 Ubuntu 镜像..."
    qm importdisk $VM_ID "$image_path" $storage
    
    if [[ $? -ne 0 ]]; then
        print_message $RED "导入镜像失败"
        return 1
    fi
    
    # 附加磁盘
    qm set $VM_ID --scsi0 $storage:vm-$VM_ID-disk-0
    
    # 设置启动磁盘
    qm set $VM_ID --boot c --bootdisk scsi0
    
    # 设置 cloud-init
    qm set $VM_ID --ide2 $storage:cloudinit
    
    # 设置串行控制台
    qm set $VM_ID --serial0 socket --vga serial0
    
    # 设置内存
    qm set $VM_ID --memory $MEMORY
    
    # 设置 CPU
    qm set $VM_ID --cores $CORES
    
    print_message $GREEN "虚拟机创建完成"
    return 0
}

# 配置 cloud-init
configure_cloud_init() {
    print_message $BLUE "配置 cloud-init..."
    
    local cloud_init_dir="/var/lib/vz/snippets"
    local config_file="$cloud_init_dir/ubuntu-${VM_ID}-cloud-init.yml"
    
    # 复制配置文件到 cloud-init
    qm set $VM_ID --cicustom "user=local:snippets/ubuntu-${VM_ID}-cloud-init.yml"
    
    print_message $GREEN "cloud-init 配置完成"
    return 0
}

# 启动虚拟机
start_virtual_machine() {
    print_message $BLUE "启动虚拟机..."
    
    qm start $VM_ID
    
    if [[ $? -eq 0 ]]; then
        print_message $GREEN "虚拟机启动成功"
        print_message $YELLOW "VM ID: $VM_ID"
        print_message $YELLOW "VM 名称: $VM_NAME"
        print_message $YELLOW "IP 地址: $IP_ADDRESS"
        print_message $YELLOW "用户名: $USERNAME"
        print_message $YELLOW "密码: $PASSWORD"
        echo
        print_message $BLUE "您可以使用以下命令连接到虚拟机:"
        echo "ssh $USERNAME@$IP_ADDRESS"
        echo
        print_message $YELLOW "注意: 首次启动可能需要几分钟时间完成 cloud-init 配置"
    else
        print_message $RED "虚拟机启动失败"
        return 1
    fi
    
    return 0
}

# 安装 Ubuntu
install_ubuntu() {
    print_message $GREEN "开始安装 Ubuntu Server..."
    
    # 检查环境
    check_pve_environment
    
    # 确认安装
    print_message $YELLOW "安装配置确认:"
    show_current_config
    read -p "确认开始安装? (y/N): " confirm
    
    if [[ $confirm != "y" && $confirm != "Y" ]]; then
        print_message $YELLOW "安装已取消"
        return 0
    fi
    
    # 下载镜像
    if ! download_ubuntu_image; then
        print_message $RED "镜像下载失败，安装终止"
        return 1
    fi
    
    # 创建 cloud-init 配置
    if ! create_cloud_init_config; then
        print_message $RED "创建 cloud-init 配置失败，安装终止"
        return 1
    fi
    
    # 创建虚拟机
    if ! create_virtual_machine; then
        print_message $RED "创建虚拟机失败，安装终止"
        return 1
    fi
    
    # 配置 cloud-init
    if ! configure_cloud_init; then
        print_message $RED "配置 cloud-init 失败，安装终止"
        return 1
    fi
    
    # 启动虚拟机
    if ! start_virtual_machine; then
        print_message $RED "启动虚拟机失败，安装终止"
        return 1
    fi
    
    print_message $GREEN "Ubuntu Server 安装完成!"
    return 0
}

# 主函数
main() {
    # 检查是否为 root 用户
    if [[ $EUID -ne 0 ]]; then
        print_message $RED "此脚本需要 root 权限运行"
        exit 1
    fi
    
    # 显示欢迎信息
    show_header
    print_message $GREEN "欢迎使用 PVE Ubuntu Server 安装脚本"
    echo
    print_message $YELLOW "此脚本将帮助您在 PVE 系统中安装 Ubuntu Server"
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