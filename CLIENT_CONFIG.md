# 客户端配置指南

## 网络架构说明

```
Internet
    ↓
ROS主路由 (192.168.1.1)
    ↓
PVE Host (192.168.1.10)
    ↓
Ubuntu VM (192.168.1.100) - MosDNS + mihomo
    ├── DNS服务: 53端口
    ├── HTTP代理: 7890端口
    ├── SOCKS5代理: 7891端口
    └── Web管理: 9090端口
```

## DNS配置

### 1. 路由器配置（推荐）
在ROS主路由中配置DNS服务器：
```
/ip dns set servers=192.168.1.100
```

这样所有设备都会自动使用分流DNS服务。

### 2. 设备单独配置

#### Windows设备
1. 打开网络设置
2. 更改适配器选项
3. 右键网络连接 → 属性
4. 选择"Internet协议版本4(TCP/IPv4)" → 属性
5. 选择"使用下面的DNS服务器地址"
6. 首选DNS服务器：`192.168.1.100`
7. 备用DNS服务器：`223.5.5.5`

#### macOS设备
1. 系统偏好设置 → 网络
2. 选择当前网络连接 → 高级
3. DNS标签页
4. 添加DNS服务器：`192.168.1.100`

#### iOS/iPadOS设备
1. 设置 → Wi-Fi
2. 点击已连接网络的"i"图标
3. 配置DNS → 手动
4. 添加服务器：`192.168.1.100`

#### Android设备
1. 设置 → 网络和Internet → Wi-Fi
2. 长按已连接网络 → 修改网络
3. 高级选项 → IP设置 → 静态
4. DNS 1：`192.168.1.100`

## 代理配置

### 1. 自动代理配置（推荐）

#### PAC文件配置
创建自动代理配置文件，让设备自动选择直连或代理：

```javascript
function FindProxyForURL(url, host) {
    // 本地地址直连
    if (isPlainHostName(host) ||
        shExpMatch(host, "*.local") ||
        isInNet(host, "10.0.0.0", "255.0.0.0") ||
        isInNet(host, "172.16.0.0", "255.240.0.0") ||
        isInNet(host, "192.168.0.0", "255.255.0.0") ||
        isInNet(host, "127.0.0.0", "255.255.255.0"))
        return "DIRECT";
    
    // 国内网站直连
    if (shExpMatch(host, "*.cn") ||
        shExpMatch(host, "*.baidu.com") ||
        shExpMatch(host, "*.qq.com") ||
        shExpMatch(host, "*.taobao.com") ||
        shExpMatch(host, "*.tmall.com") ||
        shExpMatch(host, "*.jd.com") ||
        shExpMatch(host, "*.weibo.com") ||
        shExpMatch(host, "*.sina.com.cn") ||
        shExpMatch(host, "*.163.com") ||
        shExpMatch(host, "*.126.com") ||
        shExpMatch(host, "*.sohu.com") ||
        shExpMatch(host, "*.youku.com") ||
        shExpMatch(host, "*.tudou.com") ||
        shExpMatch(host, "*.iqiyi.com") ||
        shExpMatch(host, "*.bilibili.com"))
        return "DIRECT";
    
    // 其他网站使用代理
    return "PROXY 192.168.1.100:7890; SOCKS5 192.168.1.100:7891; DIRECT";
}
```

将此文件保存为 `proxy.pac`，并在浏览器中配置自动代理脚本URL。

#### 浏览器配置
- **Chrome**: 设置 → 高级 → 系统 → 打开代理设置
- **Firefox**: 设置 → 网络设置 → 自动代理配置URL
- **Safari**: 系统偏好设置 → 网络 → 高级 → 代理

### 2. 手动代理配置

#### HTTP代理
- 服务器：`192.168.1.100`
- 端口：`7890`

#### SOCKS5代理
- 服务器：`192.168.1.100`
- 端口：`7891`

### 3. 应用专用代理

#### Clash客户端配置
```yaml
# Clash客户端配置
mixed-port: 7890
allow-lan: true
mode: rule
log-level: info

external-controller: 192.168.1.100:9090

proxies:
  - name: "HomeServer"
    type: http
    server: 192.168.1.100
    port: 7890

proxy-groups:
  - name: "PROXY"
    type: select
    proxies:
      - "HomeServer"
      - "DIRECT"

rules:
  - GEOIP,CN,DIRECT
  - MATCH,PROXY
```

#### V2rayN配置
1. 添加服务器
2. 协议：HTTP
3. 地址：`192.168.1.100`
4. 端口：`7890`

## 设备配置示例

### 1. 路由器统一配置（最佳方案）

#### MikroTik RouterOS配置
```bash
# 设置DNS服务器
/ip dns set servers=192.168.1.100

# 设置DHCP客户端DNS
/ip dhcp-server network set [find] dns-server=192.168.1.100

# 创建防火墙规则允许DNS查询
/ip firewall filter add chain=input protocol=udp dst-port=53 src-address=192.168.1.0/24 action=accept
```

### 2. 企业级配置

#### 路由表配置
```bash
# 在家庭服务器上配置路由规则
# 添加到 /etc/systemd/network/99-default.link

[Match]
OriginalName=*

[Link]
NamePolicy=kernel database onboard slot path
MACAddressPolicy=persistent
```

#### iptables规则
```bash
# 透明代理配置（高级用户）
iptables -t nat -A OUTPUT -p tcp --dport 80,443 -j REDIRECT --to-port 7890
iptables -t nat -A PREROUTING -p tcp --dport 80,443 -j REDIRECT --to-port 7890
```

## 性能优化建议

### 1. DNS缓存优化
```bash
# 在客户端启用DNS缓存
# Windows
ipconfig /flushdns
netsh int ip set dns "本地连接" static 192.168.1.100

# macOS
sudo dscacheutil -flushcache
sudo killall -HUP mDNSResponder

# Linux
sudo systemctl restart systemd-resolved
```

### 2. 代理连接优化
- 使用HTTP/2协议
- 启用连接复用
- 配置合适的超时时间
- 使用本地SOCKS5代理

## 测试和验证

### 1. DNS分流测试
```bash
# 测试国内域名解析
nslookup baidu.com 192.168.1.100

# 测试国外域名解析
nslookup google.com 192.168.1.100

# 检查DNS响应时间
dig @192.168.1.100 github.com +stats
```

### 2. 代理分流测试
```bash
# 测试HTTP代理
curl -x http://192.168.1.100:7890 http://httpbin.org/ip

# 测试SOCKS5代理
curl --socks5 192.168.1.100:7891 http://httpbin.org/ip

# 检查IP归属地
curl -x http://192.168.1.100:7890 http://ip-api.com/json
```

### 3. 网络连通性测试
```bash
# 测试直连速度
curl -o /dev/null -s -w "%{time_total}\n" http://baidu.com

# 测试代理速度
curl -x http://192.168.1.100:7890 -o /dev/null -s -w "%{time_total}\n" http://google.com
```

## 故障排除

### 1. DNS解析问题
- 检查防火墙规则
- 验证DNS服务状态
- 查看MosDNS日志
- 测试上游DNS连通性

### 2. 代理连接问题
- 检查mihomo服务状态
- 验证代理节点可用性
- 查看连接日志
- 测试网络连通性

### 3. 性能问题
- 监控系统资源使用
- 检查网络延迟
- 优化规则配置
- 调整缓存设置

## 监控和维护

### 1. 性能监控
- 使用Web管理界面查看状态
- 定期检查日志文件
- 监控系统资源使用
- 测试分流效果

### 2. 定期维护
- 更新规则文件
- 检查服务状态
- 清理日志文件
- 备份配置文件

### 3. 安全注意事项
- 定期更新软件版本
- 配置访问控制
- 监控异常连接
- 备份重要配置

## 高级配置

### 1. 多设备负载均衡
```yaml
# mihomo负载均衡配置
proxy-groups:
  - name: "LoadBalance"
    type: load-balance
    proxies:
      - "Server1"
      - "Server2"
    strategy: round-robin
```

### 2. 故障自动切换
```yaml
# 故障切换配置
proxy-groups:
  - name: "Fallback"
    type: fallback
    proxies:
      - "Primary"
      - "Secondary"
    url: 'http://www.gstatic.com/generate_204'
    interval: 300
```

### 3. 自定义规则
```yaml
# 自定义分流规则
rules:
  - DOMAIN-SUFFIX,company.com,DIRECT
  - DOMAIN-KEYWORD,google,PROXY
  - IP-CIDR,192.168.0.0/16,DIRECT
  - GEOIP,CN,DIRECT
  - MATCH,PROXY
```

这个配置指南涵盖了从基础设置到高级配置的所有内容，帮助你在各种设备上正确配置和使用家庭服务器的分流功能。