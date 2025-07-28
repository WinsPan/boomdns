#!/bin/bash

# 测试 PVE Ubuntu 安装脚本的语法

echo "正在检查脚本语法..."

# 检查 bash 语法
if bash -n pve_ubuntu_installer.sh; then
    echo "✅ 脚本语法检查通过"
else
    echo "❌ 脚本语法检查失败"
    exit 1
fi

# 检查脚本是否可执行
if [[ -x pve_ubuntu_installer.sh ]]; then
    echo "✅ 脚本具有执行权限"
else
    echo "❌ 脚本缺少执行权限"
    exit 1
fi

# 检查脚本文件是否存在
if [[ -f pve_ubuntu_installer.sh ]]; then
    echo "✅ 脚本文件存在"
else
    echo "❌ 脚本文件不存在"
    exit 1
fi

echo "🎉 所有检查通过！脚本可以正常使用。"
echo
echo "使用方法:"
echo "sudo ./pve_ubuntu_installer.sh" 