# BoomDNS 代理功能说明

## 🚀 **概述**

BoomDNS 现在集成了强大的代理功能，支持多种代理协议和智能路由策略，可以实现类似 Clash、V2Ray、SingBox 等代理工具的功能。

## ✨ **核心特性**

### 1. **多协议支持**
- **HTTP/HTTPS 代理**: 标准 HTTP 代理协议
- **SOCKS5 代理**: 高性能 SOCKS5 协议
- **Shadowsocks**: 支持多种加密方法
- **V2Ray**: 支持 WebSocket、TCP、QUIC 等传输协议
- **Trojan**: 基于 TLS 的代理协议
- **WireGuard**: 现代 VPN 协议
- **Hysteria2**: 基于 QUIC 的高性能代理协议，具有优秀的抗封锁能力

### 2. **智能路由策略**
- **域名分流**: 基于域名的智能分流
- **IP 分流**: 支持 IP CIDR 规则
- **地理位置分流**: 基于 GeoIP 的分流
- **自定义规则**: 灵活的规则配置

### 3. **负载均衡**
- **轮询策略**: 简单的轮询分发
- **延迟优先**: 自动选择延迟最低的节点
- **权重策略**: 基于权重的智能分发
- **故障转移**: 自动故障检测和切换

### 4. **健康监控**
- **自动检测**: 定期检查节点状态
- **延迟测试**: 实时监控节点延迟
- **故障统计**: 记录失败次数和错误信息
- **自动恢复**: 故障节点自动恢复

## 🏗️ **架构设计**

### 1. **核心组件**

```
BoomDNS Server
├── DNS 解析器
├── 代理管理器 (ProxyManager)
│   ├── 节点管理 (ProxyNode)
│   ├── 组管理 (ProxyGroup)
│   ├── 规则管理 (ProxyRule)
│   └── 健康检查器
└── Web 管理界面
```

### 2. **数据流**

```
用户请求 → DNS 解析 → 规则匹配 → 代理选择 → 节点转发 → 响应返回
```

## 📋 **配置说明**

### 1. **代理配置 (config.yaml)**

```yaml
# 代理配置
proxy:
  enabled: true                    # 启用代理功能
  listen_http: ":7890"            # HTTP代理监听地址
  listen_socks: ":7891"           # SOCKS5代理监听地址
  default_strategy: "round-robin" # 默认策略
  test_interval: 300              # 健康检查间隔 (秒)
  test_timeout: 10                # 连接测试超时 (秒)
```

### 2. **代理节点配置**

```yaml
# 代理节点示例
proxy_nodes:
  # Hysteria2 节点示例
  - name: "Hysteria2-香港"
    protocol: "hysteria2"
    address: "hk.example.com"
    port: 443
    enabled: true
    weight: 100
    hysteria2:
      password: "your-hysteria2-password"
      ca: "/path/to/ca.crt"        # CA 证书路径（可选）
      insecure: false              # 是否跳过证书验证
      up_mbps: 100                 # 上行带宽限制 (Mbps)
      down_mbps: 100               # 下行带宽限制 (Mbps)
  
  - name: "香港节点1"
    protocol: "ss"                # 协议类型
    address: "hk1.example.com"    # 服务器地址
    port: 8388                    # 端口
    secret: "your-secret"         # 密钥
    method: "aes-256-gcm"         # 加密方法
    enabled: true                 # 是否启用
    weight: 100                   # 权重
    
  - name: "美国节点1"
    protocol: "v2ray"
    address: "us1.example.com"
    port: 443
    secret: "your-uuid"
    transport: "ws"               # 传输协议
    path: "/path"                 # WebSocket路径
    sni: "example.com"            # TLS SNI
    enabled: true
    weight: 80
```

### 3. **代理组配置**

```yaml
# 代理组示例
proxy_groups:
  - name: "自动选择"
    type: "url-test"              # 组类型
    strategy: "latency"           # 策略
    test_url: "http://www.google.com"
    interval: 300                 # 测试间隔
    timeout: 10                   # 超时时间
    nodes: [1, 2, 3]             # 节点ID列表
    enabled: true
    
  - name: "故障转移"
    type: "fallback"
    strategy: "priority"
    test_url: "http://www.google.com"
    interval: 300
    timeout: 10
    nodes: [1, 2, 3]
    enabled: true
```

### 4. **代理规则配置**

```yaml
# 代理规则示例
proxy_rules:
  - type: "domain"                # 规则类型
    value: "google.com"           # 规则值
    action: "proxy"               # 动作
    proxy_group: "自动选择"        # 代理组
    priority: 100                 # 优先级
    enabled: true
    
  - type: "ip-cidr"
    value: "8.8.8.8/32"
    action: "proxy"
    proxy_group: "自动选择"
    priority: 90
    enabled: true
    
  - type: "geoip"
    value: "CN"
    action: "direct"
    priority: 80
    enabled: true
```

## 🔧 **API 接口**

### 1. **代理节点管理**

```bash
# 获取所有代理节点
GET /api/proxy/nodes

# 创建代理节点
POST /api/proxy/nodes

# 更新代理节点
PUT /api/proxy/nodes/{id}

# 删除代理节点
DELETE /api/proxy/nodes/{id}

# 测试代理节点
POST /api/proxy/nodes/{id}/test
```

### 2. **代理组管理**

```bash
# 获取所有代理组
GET /api/proxy/groups

# 创建代理组
POST /api/proxy/groups

# 更新代理组
PUT /api/proxy/groups/{id}

# 删除代理组
DELETE /api/proxy/groups/{id}
```

### 3. **代理规则管理**

```bash
# 获取所有代理规则
GET /api/proxy/rules

# 创建代理规则
POST /api/proxy/rules

# 更新代理规则
PUT /api/proxy/rules/{id}

# 删除代理规则
DELETE /api/proxy/rules/{id}
```

### 4. **代理状态查询**

```bash
# 获取代理状态
GET /api/proxy/status
```

## 🚀 **使用方法**

### 1. **启动服务**

```bash
# 编译并运行
go build ./cmd/boomdns
./boomdns -config config.yaml
```

### 2. **配置代理**

1. 在 `config.yaml` 中启用代理功能
2. 配置代理节点信息
3. 设置代理组和策略
4. 配置分流规则

### 3. **客户端配置**

#### HTTP 代理
```
代理地址: 127.0.0.1:7890
协议: HTTP
```

#### SOCKS5 代理
```
代理地址: 127.0.0.1:7891
协议: SOCKS5
```

### 4. **浏览器配置**

#### Chrome/Edge
```bash
# 启动时添加参数
--proxy-server="127.0.0.1:7890"
```

#### Firefox
```
设置 → 网络设置 → 连接 → 配置代理访问互联网
手动配置代理:
HTTP 代理: 127.0.0.1
端口: 7890
```

## 📊 **监控和统计**

### 1. **健康检查**
- 自动检测节点状态
- 延迟测试和统计
- 故障次数记录
- 自动故障转移

### 2. **性能指标**
- 连接成功率
- 平均延迟
- 吞吐量统计
- 错误率监控

### 3. **日志记录**
- 连接日志
- 错误日志
- 性能日志
- 审计日志

## 🔒 **安全特性**

### 1. **访问控制**
- 基于 IP 的访问控制
- 用户认证和授权
- API 访问限制

### 2. **加密传输**
- TLS 加密支持
- 多种加密算法
- 证书验证

### 3. **流量保护**
- 流量混淆
- 协议伪装
- 防检测机制

## 🚀 **高级功能**

### 1. **智能分流**
- 基于域名的自动分流
- 地理位置智能识别
- 自定义分流规则

### 2. **负载均衡**
- 多节点负载均衡
- 智能故障转移
- 动态权重调整

### 3. **性能优化**
- 连接池管理
- 缓存优化
- 并发控制

## 📝 **使用示例**

### 1. **基本代理配置**

```yaml
# 启用代理功能
proxy:
  enabled: true
  listen_http: ":7890"
  listen_socks: ":7891"

# 添加代理节点
proxy_nodes:
  - name: "香港节点"
    protocol: "ss"
    address: "hk.example.com"
    port: 8388
    secret: "your-secret"
    method: "aes-256-gcm"
    enabled: true

# 配置代理组
proxy_groups:
  - name: "自动选择"
    type: "url-test"
    strategy: "latency"
    test_url: "http://www.google.com"
    interval: 300
    nodes: [1]
    enabled: true

# 设置分流规则
proxy_rules:
  - type: "domain"
    value: "google.com"
    action: "proxy"
    proxy_group: "自动选择"
    priority: 100
    enabled: true
```

### 2. **高级分流配置**

```yaml
# 多节点配置
proxy_nodes:
  - name: "Hysteria2-香港1"
    protocol: "hysteria2"
    address: "hk1.example.com"
    port: 443
    weight: 100
    hysteria2:
      password: "password1"
      up_mbps: 100
      down_mbps: 100
    
  - name: "Hysteria2-香港2"
    protocol: "hysteria2"
    address: "hk2.example.com"
    port: 443
    weight: 80
    hysteria2:
      password: "password2"
      up_mbps: 80
      down_mbps: 80
    
  - name: "香港节点1"
    protocol: "ss"
    address: "hk1.example.com"
    port: 8388
    secret: "secret1"
    method: "aes-256-gcm"
    weight: 100
    
  - name: "香港节点2"
    protocol: "ss"
    address: "hk2.example.com"
    port: 8388
    secret: "secret2"
    method: "aes-256-gcm"
    weight: 80
    
  - name: "美国节点"
    protocol: "v2ray"
    address: "us.example.com"
    port: 443
    secret: "uuid-here"
    transport: "ws"
    path: "/path"
    weight: 60

# 智能分组
proxy_groups:
  - name: "香港组"
    type: "load-balance"
    strategy: "weight"
    nodes: [1, 2]
    enabled: true
    
  - name: "国外组"
    type: "url-test"
    strategy: "latency"
    test_url: "http://www.google.com"
    interval: 300
    nodes: [3]
    enabled: true

# 智能分流规则
proxy_rules:
  - type: "domain"
    value: "google.com"
    action: "proxy"
    proxy_group: "国外组"
    priority: 100
    
  - type: "domain"
    value: "youtube.com"
    action: "proxy"
    proxy_group: "国外组"
    priority: 100
    
  - type: "domain"
    value: "baidu.com"
    action: "direct"
    priority: 90
    
  - type: "geoip"
    value: "CN"
    action: "direct"
    priority: 80
```

## 🐛 **故障排除**

### 1. **常见问题**

#### 代理无法连接
- 检查节点配置是否正确
- 验证服务器地址和端口
- 检查防火墙设置
- 查看错误日志

#### 分流规则不生效
- 检查规则优先级设置
- 验证域名格式是否正确
- 确认代理组是否启用
- 查看规则匹配日志

#### 性能问题
- 检查节点延迟
- 优化连接池设置
- 调整健康检查间隔
- 监控系统资源

### 2. **调试方法**

#### 启用调试日志
```yaml
logging:
  level: "debug"
  enable_proxy_logs: true
```

#### 查看代理状态
```bash
curl http://localhost:8080/api/proxy/status
```

#### 测试节点连接
```bash
curl -X POST http://localhost:8080/api/proxy/nodes/1/test
```

## 🔮 **未来规划**

### 1. **协议支持**
- [x] Hysteria2 协议支持 ✅
- [ ] VMess 协议支持
- [ ] Trojan-GFW 协议
- [ ] Hysteria 协议
- [ ] Reality 协议

### 2. **功能增强**
- [ ] 图形化配置界面
- [ ] 实时流量监控
- [ ] 智能规则生成
- [ ] 多用户支持

### 3. **性能优化**
- [ ] 多核并发处理
- [ ] 内存池优化
- [ ] 网络栈优化
- [ ] 缓存策略优化

## 📚 **参考资料**

- [Go 网络编程](https://golang.org/pkg/net/)
- [HTTP 代理协议](https://tools.ietf.org/html/rfc7231)
- [SOCKS5 协议](https://tools.ietf.org/html/rfc1928)
- [Shadowsocks 协议](https://shadowsocks.org/en/spec/protocol.html)
- [V2Ray 协议](https://www.v2fly.org/)
- [Hysteria2 协议](https://hysteria.network/)

## 🌟 **Hysteria2 协议详解**

### 1. **协议特点**
- **基于 QUIC**: 使用 QUIC 协议，具有优秀的抗封锁能力
- **高性能**: 支持多路复用，延迟低，吞吐量高
- **TLS 伪装**: 流量看起来像正常的 HTTPS 流量
- **带宽控制**: 支持上行和下行带宽限制
- **证书验证**: 支持自定义 CA 证书和跳过验证选项

### 2. **配置参数说明**
```yaml
hysteria2:
  password: "your-password"        # 必填：Hysteria2 密码
  ca: "/path/to/ca.crt"           # 可选：CA 证书路径
  insecure: false                  # 可选：是否跳过证书验证
  up_mbps: 100                     # 可选：上行带宽限制 (Mbps)
  down_mbps: 100                   # 可选：下行带宽限制 (Mbps)
```

### 3. **使用建议**
- **端口选择**: 建议使用 443 端口，伪装成 HTTPS 流量
- **证书配置**: 生产环境建议配置有效的 TLS 证书
- **带宽设置**: 根据服务器实际带宽设置合理的限制
- **安全考虑**: 避免使用 `insecure: true` 在生产环境

## 🤝 **贡献指南**

欢迎提交 Issue 和 Pull Request 来改进代理功能！

### 开发环境
```bash
# 克隆项目
git clone https://github.com/your-username/boomdns.git
cd boomdns

# 安装依赖
go mod download

# 运行测试
go test ./...

# 编译项目
go build ./cmd/boomdns
```

---

**BoomDNS 代理功能** - 让网络访问更智能、更安全、更高效！
