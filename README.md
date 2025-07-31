# 基于PVE的家庭服务器分流方案

## 项目简介

这是一个基于Proxmox VE (PVE) 虚拟化平台的家庭服务器方案，使用 **MosDNS** 和 **mihomo** 在Ubuntu虚拟机中实现智能DNS分流和代理分流，为家庭网络提供高效、稳定的网络加速和内容访问优化。

## 系统架构

```
Internet
    ↓
ROS主路由 (已实现)
    ↓
PVE虚拟化平台
    ↓
Ubuntu VM (192.168.1.100)
├── MosDNS (DNS分流) - 端口53
├── mihomo (代理分流) - 端口7890/7891
└── Web管理面板 - 端口9090
```

## 核心特性

### 🚀 智能分流
- **DNS分流**: 国内域名使用国内DNS，国外域名使用国外DNS
- **代理分流**: 根据规则自动选择直连或代理访问
- **规则引擎**: 支持域名、IP、地理位置等多维度规则

### 🛠️ 易于部署
- **一键部署**: 全自动安装配置脚本
- **服务管理**: 完整的systemd服务管理
- **Web界面**: 直观的管理面板

### 📊 性能优化
- **高效缓存**: DNS查询缓存和连接复用
- **负载均衡**: 多上游DNS服务器负载均衡
- **故障切换**: 自动故障检测和切换

### 🔧 易于维护
- **自动更新**: 定时更新规则文件
- **日志管理**: 完整的日志记录和轮转
- **监控告警**: 服务状态监控

## 快速开始

### 1. 环境要求
- PVE 6.0+ 虚拟化平台
- Ubuntu 20.04+ 虚拟机
- 2GB+ 内存，20GB+ 存储
- 网络连接正常

### 2. 一键部署
```bash
# 下载部署脚本
wget https://raw.githubusercontent.com/your-repo/hnet/main/deploy_home_server.sh

# 给脚本执行权限
chmod +x deploy_home_server.sh

# 以root权限运行部署脚本
sudo ./deploy_home_server.sh
```

### 3. 启动服务
```bash
# 启动所有服务
homeserver-ctl start

# 查看服务状态
homeserver-ctl status

# 启用开机自启
homeserver-ctl enable
```

## 组件说明

### MosDNS - DNS分流服务
- **功能**: 智能DNS解析和分流
- **端口**: 53 (主要), 1053 (备用)
- **特性**:
  - 国内外DNS分离解析
  - 广告域名拦截
  - DNS缓存加速
  - 支持DoH/DoT上游

### mihomo - 代理分流服务
- **功能**: Clash兼容的代理客户端
- **端口**: 7890 (HTTP), 7891 (SOCKS5), 9090 (管理面板)
- **特性**:
  - 多协议支持
  - 规则引擎
  - 负载均衡
  - 故障切换

## 管理命令

### 服务管理
```bash
homeserver-ctl status      # 查看服务状态
homeserver-ctl start       # 启动所有服务
homeserver-ctl stop        # 停止所有服务
homeserver-ctl restart     # 重启所有服务
homeserver-ctl enable      # 启用开机自启
homeserver-ctl disable     # 禁用开机自启
```

### 维护操作
```bash
homeserver-ctl update      # 更新规则文件
homeserver-ctl logs mosdns # 查看MosDNS日志
homeserver-ctl logs mihomo # 查看mihomo日志
```

### 测试功能
```bash
homeserver-ctl test-dns    # 测试DNS解析
homeserver-ctl test-proxy  # 测试代理连接
```

## 客户端配置

### 1. 路由器配置（推荐）
在主路由器中配置DNS服务器为服务器IP，实现全网自动分流：
```bash
# MikroTik RouterOS示例
/ip dns set servers=192.168.1.100
```

### 2. 设备单独配置
详细的客户端配置请参考：[CLIENT_CONFIG.md](CLIENT_CONFIG.md)

## 配置文件

### 目录结构
```
/opt/homeserver/          # 程序安装目录
├── mosdns/              # MosDNS程序
├── mihomo/              # mihomo程序
├── data/                # 数据文件
└── rules/               # 规则文件

/etc/homeserver/         # 配置文件目录
├── mosdns/             # MosDNS配置
└── mihomo/             # mihomo配置

/var/log/homeserver/     # 日志文件目录
├── mosdns/             # MosDNS日志
└── mihomo/             # mihomo日志
```

### 主要配置文件
- **MosDNS配置**: `/etc/homeserver/mosdns/config.yaml`
- **mihomo配置**: `/etc/homeserver/mihomo/config.yaml`

## 性能监控

### Web管理面板
访问 `http://服务器IP:9090` 查看mihomo管理面板，可以：
- 查看连接状态
- 监控流量统计
- 管理代理规则
- 测试节点延迟

### 系统监控
```bash
# 查看端口监听状态
ss -tlnp | grep -E "(53|7890|7891|9090)"

# 查看服务运行状态
systemctl status mosdns mihomo

# 查看系统资源使用
htop
```

## 故障排除

### 常见问题

1. **DNS解析失败**
   ```bash
   # 检查MosDNS服务状态
   systemctl status mosdns
   
   # 查看MosDNS日志
   journalctl -u mosdns -f
   
   # 测试上游DNS
   nslookup google.com 8.8.8.8
   ```

2. **代理连接失败**
   ```bash
   # 检查mihomo服务状态
   systemctl status mihomo
   
   # 查看mihomo日志
   journalctl -u mihomo -f
   
   # 检查配置文件
   /opt/homeserver/mihomo/mihomo -t -d /etc/homeserver/mihomo
   ```

3. **规则不生效**
   ```bash
   # 更新规则文件
   homeserver-ctl update
   
   # 重启服务
   homeserver-ctl restart
   ```

### 诊断工具
```bash
# DNS解析测试
dig @127.0.0.1 google.com
nslookup baidu.com 127.0.0.1

# 代理连接测试
curl -x http://127.0.0.1:7890 http://httpbin.org/ip
curl --socks5 127.0.0.1:7891 http://httpbin.org/ip

# 网络连通性测试
ping 8.8.8.8
traceroute google.com
```

## 安全考虑

### 网络安全
- 防火墙规则配置
- 端口访问控制
- 内网服务隔离

### 数据安全
- 配置文件权限控制
- 敏感信息保护
- 定期备份配置

### 访问控制
```bash
# 限制管理面板访问
iptables -A INPUT -p tcp --dport 9090 -s 192.168.1.0/24 -j ACCEPT
iptables -A INPUT -p tcp --dport 9090 -j DROP
```

## 高级配置

### 自定义规则
可以根据需要修改配置文件，添加自定义分流规则：

```yaml
# mihomo自定义规则示例
rules:
  - DOMAIN-SUFFIX,company.com,DIRECT
  - DOMAIN-KEYWORD,google,PROXY
  - IP-CIDR,10.0.0.0/8,DIRECT
  - GEOIP,CN,DIRECT
  - MATCH,PROXY
```

### 负载均衡
```yaml
# 多节点负载均衡
proxy-groups:
  - name: "LoadBalance"
    type: load-balance
    proxies:
      - "Server1"
      - "Server2"
    strategy: round-robin
```

## 更新维护

### 自动更新
系统已配置定时任务，每天自动更新规则文件。也可以手动执行：
```bash
homeserver-ctl update
```

### 版本更新
```bash
# 备份配置
cp -r /etc/homeserver /etc/homeserver.backup

# 重新运行部署脚本
sudo ./deploy_home_server.sh

# 恢复自定义配置
# (如需要)
```

## 贡献指南

欢迎提交问题报告和改进建议：
1. Fork 本项目
2. 创建特性分支
3. 提交更改
4. 发起 Pull Request

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 相关链接

- [MosDNS项目](https://github.com/IrineSistiana/mosdns)
- [mihomo项目](https://github.com/MetaCubeX/mihomo)
- [Proxmox VE官网](https://www.proxmox.com/en/proxmox-ve)

## 支持

如果这个项目对你有帮助，请给个 ⭐ Star！

如果遇到问题，请查看：
1. [问题解答](https://github.com/your-repo/hnet/issues)
2. [详细文档](https://github.com/your-repo/hnet/wiki)
3. 提交新的 Issue

---

**注意**: 请确保遵守当地法律法规，合理使用网络资源。