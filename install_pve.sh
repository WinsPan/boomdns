#!/bin/bash

# PVE一键安装配置脚本
# 参考格式: wget -q -O install_pve.sh 'https://your-repo/install_pve.sh' && bash install_pve.sh

# 配置变量
REPO_BASE_URL="https://raw.githubusercontent.com/your-repo/hnet/main"
SCRIPT_NAME="setup_pve.sh"
TEMP_DIR="/tmp/pve_setup"

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

# 显示标题
show_header() {
    clear
    echo "============================================="
    echo "    PVE一键配置安装器"
    echo "============================================="
    echo
}

# 检查系统
check_system() {
    print_message $BLUE "检查系统环境..."
    
    if [ "$EUID" -ne 0 ]; then
        print_message $RED "❌ 请使用root权限运行"
        exit 1
    fi
    
    if [ ! -f /etc/pve/version ]; then
        print_message $RED "❌ 当前系统不是Proxmox VE"
        exit 1
    fi
    
    print_message $GREEN "✅ 系统检查通过"
}

# 下载并执行主脚本
download_and_run() {
    print_message $BLUE "下载PVE配置脚本..."
    
    # 创建临时目录
    mkdir -p $TEMP_DIR
    cd $TEMP_DIR
    
    # 下载主配置脚本
    if wget -q -O $SCRIPT_NAME "$REPO_BASE_URL/$SCRIPT_NAME"; then
        print_message $GREEN "✅ 脚本下载成功"
    else
        print_message $RED "❌ 脚本下载失败"
        exit 1
    fi
    
    # 给脚本执行权限
    chmod +x $SCRIPT_NAME
    
    # 执行主脚本
    print_message $BLUE "开始执行PVE配置..."
    ./$SCRIPT_NAME
    
    # 清理临时文件
    cd /
    rm -rf $TEMP_DIR
}

# 主函数
main() {
    show_header
    check_system
    download_and_run
    
    print_message $GREEN "🎉 PVE一键配置完成！"
}

# 执行主函数
main "$@"