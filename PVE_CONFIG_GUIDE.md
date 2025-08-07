# PVE (Proxmox VE) 一键配置脚本使用指南

## 🚀 快速开始

### 一键安装命令
```bash
wget -q -O install_pve.sh 'https://raw.githubusercontent.com/your-repo/hnet/main/install_pve.sh' && bash install_pve.sh
```

或者直接执行完整配置脚本：
```bash
wget -q -O setup_pve.sh 'https://raw.githubusercontent.com/your-repo/hnet/main/setup_pve.sh' && bash setup_pve.sh
```

## 📋 脚本功能

### 🔧 系统优化
- **软件源配置**: 自动配置中国镜像源，提升下载速度
- **系统更新**: 全自动系统更新和软件包升级
- **性能优化**: CPU调频、内存优化、网络参数调优
- **服务优化**: 禁用不必要的服务，启用BBR拥塞控制

### 🌐 网络配置
- **网桥优化**: 优化虚拟网桥性能
- **防火墙配置**: 自动配置PVE必要端口
- **网络参数调优**: TCP/UDP参数优化

### 💾 存储配置
- **本地存储**: 创建自定义存储目录
- **ISO存储**: 配置ISO镜像存储位置
- **备份存储**: 配置虚拟机备份存储

### 🔒 安全配置
- **SSH优化**: SSH服务安全配置
- **防火墙规则**: 必要端口开放和安全配置
- **访问控制**: 系统访问权限优化

### 🛠️ 管理工具
- **监控工具**: htop, iotop, iftop等系统监控工具
- **网络工具**: 网络诊断和测试工具
- **PVE管理**: 自定义PVE管理命令

### 📦 虚拟机模板
- **Ubuntu模板**: 自动创建Ubuntu云镜像模板
- **模板脚本**: 提供模板创建脚本

## 🎯 使用场景

### 1. 新装PVE系统
```bash
# 全新PVE系统一键配置
wget -q -O install_pve.sh 'https://your-repo/install_pve.sh' && bash install_pve.sh
```

### 2. 现有PVE系统优化
```bash
# 仅执行优化配置
bash setup_pve.sh
```

### 3. 自定义配置
```bash
# 下载脚本后自定义修改
wget https://your-repo/setup_pve.sh
vim setup_pve.sh  # 修改配置
bash setup_pve.sh
```

## ⚙️ 配置详情

### 软件源配置
```bash
# 主要使用中科大镜像源
deb https://mirrors.ustc.edu.cn/debian/ bullseye main contrib non-free
deb https://mirrors.ustc.edu.cn/debian/ bullseye-updates main contrib non-free
deb https://mirrors.ustc.edu.cn/debian-security/ bullseye-security main contrib non-free

# PVE无订阅源
deb http://download.proxmox.com/debian/pve bullseye pve-no-subscription
```

### 系统优化参数
```bash
# 内存优化
vm.swappiness = 10
vm.vfs_cache_pressure = 50

# 网络优化
net.core.rmem_max = 16777216
net.core.wmem_max = 16777216
net.ipv4.tcp_congestion_control = bbr
```

### 防火墙端口
```bash
22/tcp      # SSH
8006/tcp    # PVE Web界面
3128/tcp    # PVE Proxy
5900:5999/tcp # VNC
111         # rpcbind
5404:5405   # corosync
```

## 📊 配置后状态

### Web管理界面
- **访问地址**: `https://your-pve-ip:8006`
- **默认用户**: `root`
- **功能**: 虚拟机管理、监控、配置

### 系统信息查看
```bash
# 查看PVE系统信息
pve-info

# 查看服务状态
systemctl status pve-cluster pve-firewall pvedaemon pveproxy
```

### 性能监控
```bash
# 系统负载
htop

# 磁盘IO
iotop

# 网络流量
iftop

# 磁盘使用
ncdu /
```

## 🔧 后续操作

### 1. 创建虚拟机模板
```bash
# 执行Ubuntu模板创建脚本
/root/create_ubuntu_template.sh
```

### 2. 虚拟机管理
```bash
# 列出所有虚拟机
qm list

# 列出所有容器
pct list

# 启动虚拟机
qm start <vmid>

# 停止虚拟机
qm stop <vmid>
```

### 3. 存储管理
```bash
# 查看存储状态
pvesm status

# 查看存储内容
pvesm list <storage>
```

## 🚨 注意事项

### 系统要求
- **操作系统**: Proxmox VE 6.0+
- **权限**: root用户
- **网络**: 需要互联网连接
- **内存**: 建议4GB+

### 安全提醒
- 脚本会修改系统配置，建议在测试环境先验证
- 执行前会自动备份重要配置文件
- 防火墙配置可能影响网络访问

### 兼容性
- 支持PVE 6.x和7.x版本
- 基于Debian 10/11系统
- 支持x86_64架构

## 🛠️ 故障排除

### 常见问题

1. **软件源配置失败**
```bash
# 手动恢复原始软件源
cp /etc/apt/sources.list.backup /etc/apt/sources.list
apt update
```

2. **网络配置问题**
```bash
# 检查网络接口
ip addr show
# 重启网络服务
systemctl restart networking
```

3. **防火墙问题**
```bash
# 临时禁用防火墙
ufw disable
# 重新配置防火墙
ufw --force reset
```

### 日志查看
```bash
# 脚本执行日志
tail -f /var/log/pve_setup.log

# 系统日志
journalctl -f

# PVE服务日志
journalctl -u pvedaemon -f
```

## 📞 支持

如遇问题请检查：
1. 系统日志: `/var/log/pve_setup.log`
2. 配置备份: `/root/pve_backup_*`
3. PVE官方文档: https://pve.proxmox.com/wiki/Main_Page

---

**注意**: 请在使用前仔细阅读脚本内容，确保符合您的环境要求。