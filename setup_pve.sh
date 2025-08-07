#!/bin/bash

# PVE (Proxmox Virtual Environment) 自动配置脚本
# 功能: 自动配置PVE系统，包括软件源、系统优化、网络配置等
# 作者: AI Assistant
# 版本: 1.0
# 使用方法: wget -q -O setup_pve.sh https://your-repo/setup_pve.sh && bash setup_pve.sh

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 脚本配置
SCRIPT_VERSION="1.0"
LOG_FILE="/var/log/pve_setup.log"
BACKUP_DIR="/root/pve_backup_$(date +%Y%m%d_%H%M%S)"

# 打印带颜色的消息
print_message() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $message" >> $LOG_FILE
}

# 显示脚本标题
show_header() {
    clear
    echo "============================================="
    echo "    PVE (Proxmox VE) 自动配置脚本 v$SCRIPT_VERSION"
    echo "============================================="
    echo
    print_message $CYAN "开始PVE系统配置..."
    echo
}

# 检查系统环境
check_environment() {
    print_message $BLUE "检查系统环境..."
    
    # 检查是否为root用户
    if [ "$EUID" -ne 0 ]; then
        print_message $RED "❌ 请使用root权限运行此脚本"
        exit 1
    fi
    
    # 检查是否为PVE系统
    if [ ! -f /etc/pve/version ]; then
        print_message $RED "❌ 当前系统不是Proxmox VE"
        exit 1
    fi
    
    # 获取PVE版本信息
    PVE_VERSION=$(pveversion | head -n1)
    print_message $GREEN "✅ 检测到PVE系统: $PVE_VERSION"
    
    # 检查网络连接
    if ! ping -c 1 8.8.8.8 &> /dev/null; then
        print_message $RED "❌ 网络连接异常，请检查网络设置"
        exit 1
    fi
    
    print_message $GREEN "✅ 系统环境检查通过"
}

# 创建备份目录
create_backup() {
    print_message $BLUE "创建配置备份..."
    
    mkdir -p $BACKUP_DIR
    
    # 备份重要配置文件
    cp -r /etc/apt/sources.list* $BACKUP_DIR/ 2>/dev/null
    cp -r /etc/pve/local $BACKUP_DIR/ 2>/dev/null
    cp /etc/network/interfaces $BACKUP_DIR/ 2>/dev/null
    cp /etc/hosts $BACKUP_DIR/ 2>/dev/null
    
    print_message $GREEN "✅ 配置文件已备份到: $BACKUP_DIR"
}

# 配置中国软件源
configure_sources() {
    print_message $BLUE "配置中国软件源..."
    
    # 备份原始sources.list
    cp /etc/apt/sources.list /etc/apt/sources.list.backup
    
    # 获取系统版本代号
    CODENAME=$(lsb_release -cs)
    
    # 配置Debian软件源
    cat > /etc/apt/sources.list << EOF
# Debian $CODENAME 中国镜像源
deb https://mirrors.ustc.edu.cn/debian/ $CODENAME main contrib non-free
deb https://mirrors.ustc.edu.cn/debian/ $CODENAME-updates main contrib non-free
deb https://mirrors.ustc.edu.cn/debian/ $CODENAME-backports main contrib non-free
deb https://mirrors.ustc.edu.cn/debian-security/ $CODENAME-security main contrib non-free

# 阿里云镜像源备用
# deb https://mirrors.aliyun.com/debian/ $CODENAME main contrib non-free
# deb https://mirrors.aliyun.com/debian/ $CODENAME-updates main contrib non-free
# deb https://mirrors.aliyun.com/debian-security/ $CODENAME-security main contrib non-free
EOF

    # 配置PVE企业源（免费版本注释掉）
    if [ -f /etc/apt/sources.list.d/pve-enterprise.list ]; then
        sed -i 's/^deb/#deb/g' /etc/apt/sources.list.d/pve-enterprise.list
        print_message $YELLOW "已注释PVE企业版软件源"
    fi
    
    # 添加PVE无订阅源
    echo "deb http://download.proxmox.com/debian/pve $CODENAME pve-no-subscription" > /etc/apt/sources.list.d/pve-no-subscription.list
    
    # 配置Ceph源
    if [ -f /etc/apt/sources.list.d/ceph.list ]; then
        sed -i 's/^deb/#deb/g' /etc/apt/sources.list.d/ceph.list
        echo "deb https://mirrors.ustc.edu.cn/proxmox/debian/ceph-quincy $CODENAME no-subscription" >> /etc/apt/sources.list.d/ceph.list
    fi
    
    print_message $GREEN "✅ 软件源配置完成"
}

# 系统更新和优化
system_update() {
    print_message $BLUE "更新系统软件包..."
    
    # 更新软件包列表
    apt update
    
    # 升级系统
    print_message $YELLOW "开始系统升级，这可能需要几分钟..."
    apt upgrade -y
    
    # 安装常用工具
    print_message $BLUE "安装常用工具..."
    apt install -y \
        curl wget git vim htop iotop \
        net-tools dnsutils telnet \
        lsof tree unzip zip \
        build-essential \
        software-properties-common \
        apt-transport-https \
        ca-certificates \
        gnupg2
    
    # 清理不需要的软件包
    apt autoremove -y
    apt autoclean
    
    print_message $GREEN "✅ 系统更新完成"
}

# PVE系统优化
optimize_pve() {
    print_message $BLUE "PVE系统优化..."
    
    # CPU管理器优化
    echo 'performance' | tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor > /dev/null 2>&1
    
    # 内存优化
    cat >> /etc/sysctl.conf << EOF

# PVE内存优化
vm.swappiness = 10
vm.vfs_cache_pressure = 50
vm.dirty_background_ratio = 5
vm.dirty_ratio = 10

# 网络优化
net.core.rmem_default = 262144
net.core.rmem_max = 16777216
net.core.wmem_default = 262144
net.core.wmem_max = 16777216
net.ipv4.tcp_rmem = 4096 65536 16777216
net.ipv4.tcp_wmem = 4096 65536 16777216
net.core.netdev_max_backlog = 5000
net.ipv4.tcp_congestion_control = bbr
EOF

    # 应用内核参数
    sysctl -p
    
    # 禁用不必要的服务
    systemctl disable rpcbind.service 2>/dev/null
    systemctl disable nfs-common.service 2>/dev/null
    
    # 配置时区
    timedatectl set-timezone Asia/Shanghai
    
    print_message $GREEN "✅ PVE系统优化完成"
}

# 配置存储
configure_storage() {
    print_message $BLUE "配置存储..."
    
    # 显示当前存储配置
    print_message $CYAN "当前存储配置:"
    pvesm status
    
    # 创建本地存储目录
    mkdir -p /opt/vm-data/{iso,template,backup}
    
    # 配置本地存储
    cat >> /etc/pve/storage.cfg << EOF

# 自定义本地存储
dir: local-data
        path /opt/vm-data
        content iso,vztmpl,backup
        shared 0
EOF

    print_message $GREEN "✅ 存储配置完成"
}

# 网络优化配置
configure_network() {
    print_message $BLUE "网络配置优化..."
    
    # 获取主网卡名称
    MAIN_INTERFACE=$(ip route | grep default | awk '{print $5}' | head -n1)
    
    # 配置网桥优化
    cat >> /etc/sysctl.conf << EOF

# 网桥优化
net.bridge.bridge-nf-call-ip6tables = 0
net.bridge.bridge-nf-call-iptables = 0
net.bridge.bridge-nf-call-arptables = 0
EOF

    # 加载br_netfilter模块
    echo 'br_netfilter' >> /etc/modules
    modprobe br_netfilter
    
    print_message $GREEN "✅ 网络配置完成，主网卡: $MAIN_INTERFACE"
}

# 安全配置
configure_security() {
    print_message $BLUE "配置系统安全..."
    
    # 配置SSH安全
    if [ -f /etc/ssh/sshd_config ]; then
        # 备份SSH配置
        cp /etc/ssh/sshd_config /etc/ssh/sshd_config.backup
        
        # 优化SSH配置
        sed -i 's/#PermitRootLogin yes/PermitRootLogin yes/' /etc/ssh/sshd_config
        sed -i 's/#PasswordAuthentication yes/PasswordAuthentication yes/' /etc/ssh/sshd_config
        
        # 重启SSH服务
        systemctl restart sshd
    fi
    
    # 配置防火墙
    print_message $BLUE "配置防火墙规则..."
    
    # 允许PVE必要端口
    ufw --force enable
    ufw allow 22/tcp    # SSH
    ufw allow 8006/tcp  # PVE Web界面
    ufw allow 3128/tcp  # PVE Proxy
    ufw allow 5900:5999/tcp # VNC
    ufw allow 111       # rpcbind
    ufw allow 5404:5405 # corosync
    
    print_message $GREEN "✅ 安全配置完成"
}

# 安装额外工具
install_tools() {
    print_message $BLUE "安装额外管理工具..."
    
    # 安装Docker (可选)
    read -p "是否安装Docker? (y/n): " install_docker
    if [ "$install_docker" = "y" ] || [ "$install_docker" = "Y" ]; then
        curl -fsSL https://mirrors.ustc.edu.cn/docker-ce/linux/debian/gpg | apt-key add -
        echo "deb [arch=amd64] https://mirrors.ustc.edu.cn/docker-ce/linux/debian $(lsb_release -cs) stable" > /etc/apt/sources.list.d/docker.list
        apt update
        apt install -y docker-ce docker-ce-cli containerd.io
        systemctl enable docker
        systemctl start docker
        print_message $GREEN "✅ Docker安装完成"
    fi
    
    # 安装监控工具
    apt install -y htop iotop iftop ncdu
    
    # 安装PVE管理脚本
    cat > /usr/local/bin/pve-info << 'EOF'
#!/bin/bash
echo "=== PVE系统信息 ==="
echo "PVE版本: $(pveversion)"
echo "系统负载: $(uptime | awk -F'load average:' '{ print $2 }')"
echo "内存使用: $(free -h | grep Mem | awk '{print $3"/"$2}')"
echo "磁盘使用: $(df -h / | tail -1 | awk '{print $3"/"$2" ("$5")"}')"
echo "运行中的虚拟机:"
qm list | grep running
echo "运行中的容器:"
pct list | grep running
EOF
    chmod +x /usr/local/bin/pve-info
    
    print_message $GREEN "✅ 额外工具安装完成"
}

# 创建虚拟机模板
create_vm_template() {
    print_message $BLUE "创建常用虚拟机模板..."
    
    # 创建Ubuntu模板脚本
    cat > /root/create_ubuntu_template.sh << 'EOF'
#!/bin/bash
# Ubuntu虚拟机模板创建脚本

VMID=9000
VM_NAME="ubuntu-template"
STORAGE="local-lvm"

# 下载Ubuntu云镜像
wget https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img -O /tmp/jammy-server-cloudimg-amd64.img

# 创建虚拟机
qm create $VMID --name $VM_NAME --memory 2048 --cores 2 --net0 virtio,bridge=vmbr0

# 导入磁盘镜像
qm importdisk $VMID /tmp/jammy-server-cloudimg-amd64.img $STORAGE

# 设置磁盘
qm set $VMID --scsihw virtio-scsi-pci --scsi0 $STORAGE:vm-$VMID-disk-0

# 设置云初始化
qm set $VMID --ide2 $STORAGE:cloudinit
qm set $VMID --boot c --bootdisk scsi0
qm set $VMID --serial0 socket --vga serial0

# 转换为模板
qm template $VMID

echo "Ubuntu模板创建完成，VMID: $VMID"
EOF
    chmod +x /root/create_ubuntu_template.sh
    
    print_message $GREEN "✅ 虚拟机模板脚本创建完成"
    print_message $YELLOW "运行 /root/create_ubuntu_template.sh 创建Ubuntu模板"
}

# 显示配置摘要
show_summary() {
    clear
    print_message $GREEN "============================================="
    print_message $GREEN "    PVE配置完成！"
    print_message $GREEN "============================================="
    echo
    print_message $CYAN "配置摘要:"
    print_message $WHITE "• 软件源已配置为中国镜像"
    print_message $WHITE "• 系统已更新到最新版本"
    print_message $WHITE "• PVE系统已优化"
    print_message $WHITE "• 网络和存储已配置"
    print_message $WHITE "• 安全设置已优化"
    print_message $WHITE "• 管理工具已安装"
    echo
    print_message $CYAN "访问信息:"
    print_message $WHITE "• PVE Web界面: https://$(hostname -I | awk '{print $1}'):8006"
    print_message $WHITE "• 系统信息查看: pve-info"
    print_message $WHITE "• 配置备份位置: $BACKUP_DIR"
    echo
    print_message $YELLOW "建议下一步操作:"
    print_message $WHITE "1. 重启系统使所有配置生效"
    print_message $WHITE "2. 登录Web界面配置虚拟机"
    print_message $WHITE "3. 创建虚拟机模板: /root/create_ubuntu_template.sh"
    echo
    print_message $BLUE "日志文件: $LOG_FILE"
    echo
}

# 主函数
main() {
    show_header
    check_environment
    create_backup
    configure_sources
    system_update
    optimize_pve
    configure_storage
    configure_network
    configure_security
    install_tools
    create_vm_template
    show_summary
    
    print_message $GREEN "🎉 PVE配置脚本执行完成！"
    print_message $YELLOW "建议重启系统: reboot"
}

# 错误处理
set -e
trap 'print_message $RED "❌ 脚本执行出错，请检查日志: $LOG_FILE"' ERR

# 启动脚本
main "$@"