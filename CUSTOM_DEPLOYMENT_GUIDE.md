# 自定义三层DNS架构部署指南

## 架构概述

您的DNS分流架构设计非常优秀，实现了专业级的分层处理：

```
客户端 DNS查询
    ↓
MosDNS (10.0.0.4:53) - DNS分流入口
    ├── 国内域名 → AdGuardHome (10.0.0.5:53) - 广告拦截 + 国内优化
    ├── 国外域名 → mihomo (10.0.0.6:1053) - 代理DNS
    └── 本地域名 → AdGuardHome (10.0.0.5:53) - 本地网络服务
```

## 核心优势

### 🎯 **精准分流**
- **国内域名**: AdGuardHome提供广告拦截和国内DNS优化
- **国外域名**: mihomo通过代理进行DNS解析，避免污染
- **本地域名**: 路由器、NAS、PVE等本地服务快速解析

### 🛡️ **多重保护**
- **广告拦截**: AdGuardHome处理国内域名时自动拦截广告
- **DNS防污染**: 国外域名通过mihomo代理解析
- **故障切换**: 自动检测上游服务健康状态

### ⚡ **性能优化**
- **智能缓存**: 4096条目缓存，提升响应速度
- **健康检查**: 自动检测AdGuardHome和mihomo状态
- **并发查询**: 多路并发提升解析效率

## 配置文件特性

### 1. **三层服务器配置**

```yaml
# AdGuardHome DNS服务器 - 处理国内域名
- tag: adguardhome_upstream
  type: forward
  args:
    upstreams:
      - addr: "10.0.0.5:53"

# mihomo DNS服务器 - 处理国外域名  
- tag: mihomo_upstream
  type: forward
  args:
    upstreams:
      - addr: "10.0.0.6:1053"
```

### 2. **智能分流逻辑**

- **明确分类**: 国内域名直接走AdGuardHome
- **代理域名**: GFW列表、国际流媒体走mihomo
- **智能判断**: 未明确分类的域名先试AdGuardHome，根据返回IP智能选择
- **本地优先**: 本地服务域名直接走AdGuardHome

### 3. **健康检查机制**

```yaml
# 自动检测服务状态，故障时切换到备用DNS
- tag: check_adguardhome
  type: sequence
  args:
    - exec: $adguardhome_upstream
    - matches:
        - "!rcode 2"        # 非SERVFAIL响应
      exec: accept
    - exec: $fallback_upstream  # 故障时使用备用
```

## 部署步骤

### 1. **准备配置文件**

```bash
# 复制自定义配置文件
sudo cp mosdns-custom-config.yaml /etc/homeserver/mosdns/config.yaml

# 复制本地域名规则
sudo cp local-domains.txt /opt/homeserver/rules/local.txt

# 设置权限
sudo chown homeserver:homeserver /etc/homeserver/mosdns/config.yaml
sudo chown homeserver:homeserver /opt/homeserver/rules/local.txt
```

### 2. **验证上游服务**

在部署前，确保上游服务正常运行：

```bash
# 测试AdGuardHome (10.0.0.5:53)
nslookup baidu.com 10.0.0.5

# 测试mihomo DNS (10.0.0.6:1053)  
nslookup google.com 10.0.0.6 -port=1053

# 检查端口监听
nmap -p 53 10.0.0.5
nmap -p 1053 10.0.0.6
```

### 3. **配置MosDNS服务器**

```bash
# 在10.0.0.4服务器上启动MosDNS
sudo systemctl start mosdns
sudo systemctl enable mosdns

# 检查服务状态
sudo systemctl status mosdns

# 检查端口监听
sudo ss -tulnp | grep :53
```

### 4. **验证分流效果**

```bash
# 测试国内域名解析（应该走AdGuardHome）
nslookup baidu.com 10.0.0.4

# 测试国外域名解析（应该走mihomo）
nslookup google.com 10.0.0.4

# 测试本地域名解析
nslookup pve.local 10.0.0.4

# 检查解析路径（查看日志）
sudo tail -f /var/log/homeserver/mosdns/mosdns.log
```

## 网络配置

### ROS路由器配置

在您的ROS路由器中配置DNS：

```bash
# 设置MosDNS为主DNS服务器
/ip dns set servers=10.0.0.4

# 配置静态DNS记录（可选）
/ip dns static add name=pve.local address=10.0.0.4
/ip dns static add name=adguard.local address=10.0.0.5
/ip dns static add name=mihomo.local address=10.0.0.6
```

### 客户端配置

各设备可以直接使用自动获取的DNS，或手动设置：

- **主DNS**: 10.0.0.4 (MosDNS)
- **备用DNS**: 10.0.0.5 (AdGuardHome)

## 监控和维护

### 1. **性能监控**

```bash
# 查看MosDNS状态
sudo systemctl status mosdns

# 查看缓存统计
grep "cache" /var/log/homeserver/mosdns/mosdns.log | tail -20

# 监控解析延迟
while true; do 
  time nslookup baidu.com 10.0.0.4 > /dev/null 
  sleep 1
done
```

### 2. **故障排除**

```bash
# 检查上游服务连通性
ping 10.0.0.5  # AdGuardHome
ping 10.0.0.6  # mihomo

# 检查DNS解析路径
dig @10.0.0.4 baidu.com +trace
dig @10.0.0.4 google.com +trace

# 查看详细日志
sudo journalctl -u mosdns -f
```

### 3. **规则更新**

```bash
# 更新远程规则列表
sudo ./update-mosdns-rules.sh

# 重启服务应用新规则
sudo systemctl restart mosdns
```

## 自定义配置

### 1. **添加本地域名**

编辑 `/opt/homeserver/rules/local.txt`：

```bash
# 添加您的自定义本地域名
echo "mynas.local" >> /opt/homeserver/rules/local.txt
echo "*.homelab" >> /opt/homeserver/rules/local.txt

# 重启服务
sudo systemctl restart mosdns
```

### 2. **调整上游服务器**

如果需要修改上游服务器地址，编辑配置文件：

```yaml
# 修改AdGuardHome地址
- addr: "10.0.0.5:53"

# 修改mihomo地址  
- addr: "10.0.0.6:1053"
```

### 3. **优化性能参数**

根据您的网络环境调整：

```yaml
# 增加缓存大小（如果内存充足）
size: 8192

# 调整并发数（如果网络较慢）
concurrent: 1

# 修改TTL范围
minimal_ttl: 300
maximum_ttl: 3600
```

## 高级特性

### 1. **智能分流算法**

配置文件实现了智能分流：
- 明确的国内域名直接走AdGuardHome
- 明确的代理域名直接走mihomo  
- 未分类域名先试AdGuardHome，根据返回IP智能判断

### 2. **故障切换机制**

- 自动检测AdGuardHome和mihomo健康状态
- 服务故障时自动切换到备用DNS（223.5.5.5、114.114.114.114）
- 保证DNS服务高可用性

### 3. **本地域名优化**

- 路由器管理界面、PVE、NAS等本地服务
- 直接使用AdGuardHome解析，无需远程查询
- 提升本地服务访问速度

这个配置充分利用了您现有的三个服务，实现了专业级的DNS分流方案，既保证了解析速度，又提供了广告拦截和防污染功能！
