# PVE家庭服务器分流方案设计

## 系统架构概览

```
Internet
    ↓
[ROS主路由] (已实现)
    ↓
[PVE虚拟化平台]
    ├── [Ubuntu VM] - MosDNS + mihomo
    ├── [其他服务VM]
    └── [存储/应用服务]
```

## 核心组件分析

### 1. 当前环境
- **ROS主路由**: 已实现，作为网络入口
- **PVE平台**: 虚拟化主机
- **目标**: 在Ubuntu VM中实现DNS分流和代理分流

### 2. 分流组件选择

#### MosDNS (DNS分流)
- **功能**: 智能DNS解析和分流
- **优势**: 
  - 支持多种规则引擎
  - 可配置上游DNS服务器
  - 支持域名分类解析
  - 低延迟，高性能

#### mihomo (代理分流)
- **功能**: Clash内核的增强版本
- **优势**:
  - 兼容Clash配置
  - 支持多种代理协议
  - 规则引擎强大
  - Web管理界面

## 详细方案设计

### 网络拓扑

```
Internet
    ↓
ROS Router (192.168.1.1)
    ↓
PVE Host (192.168.1.10)
    ↓
Ubuntu VM (192.168.1.100)
├── MosDNS (53/UDP)
├── mihomo (7890/HTTP, 7891/SOCKS5)
└── Dashboard (9090/HTTP)
```

### 分流策略

#### DNS分流逻辑
1. **国内域名** → 国内DNS (114.114.114.114, 223.5.5.5)
2. **国外域名** → 国外DNS (1.1.1.1, 8.8.8.8)
3. **代理域名** → 通过代理解析
4. **直连域名** → 直连解析

#### 代理分流逻辑
1. **国内流量** → 直连
2. **国外流量** → 代理
3. **特定服务** → 指定代理节点
4. **广告域名** → 拒绝连接

## 实施步骤

### 阶段1: 环境准备
1. 在PVE中创建Ubuntu VM
2. 配置网络接口
3. 安装基础依赖

### 阶段2: MosDNS部署
1. 下载安装MosDNS
2. 配置DNS分流规则
3. 设置上游DNS服务器
4. 启动并测试服务

### 阶段3: mihomo部署
1. 下载安装mihomo
2. 配置代理规则
3. 设置订阅更新
4. 启动Web管理界面

### 阶段4: 系统集成
1. 配置系统服务
2. 设置开机自启
3. 配置防火墙规则
4. 测试分流效果

### 阶段5: 客户端配置
1. 设置设备DNS指向
2. 配置代理设置
3. 验证分流效果
4. 性能优化

## 配置文件模板

### MosDNS配置结构
```yaml
# 日志配置
log:
  level: info
  file: "./mosdns.log"

# 数据源配置
data_sources:
  - tag: geosite
    type: v2ray_geodata
    args:
      file: "./geosite.dat"
  
  - tag: geoip
    type: v2ray_geodata
    args:
      file: "./geoip.dat"

# 插件配置
plugins:
  # 缓存
  - tag: cache
    type: cache
    args:
      size: 1024
      lazy_cache_ttl: 86400

  # 上游DNS服务器
  - tag: domestic_upstream
    type: forward
    args:
      concurrent: 2
      upstreams:
        - addr: "223.5.5.5:53"
        - addr: "114.114.114.114:53"

  - tag: foreign_upstream
    type: forward
    args:
      concurrent: 2
      upstreams:
        - addr: "1.1.1.1:53"
        - addr: "8.8.8.8:53"

# 分流规则
  - tag: main_sequence
    type: sequence
    args:
      - exec: query_summary
      - matches:
          - qname: $geosite:cn
        exec: $domestic_upstream
      - matches:
          - qname: $geosite:geolocation-!cn
        exec: $foreign_upstream
      - exec: $domestic_upstream

# 服务器配置
servers:
  - exec: main_sequence
    listeners:
      - protocol: udp
        addr: "0.0.0.0:53"
      - protocol: tcp
        addr: "0.0.0.0:53"
```

### mihomo配置结构
```yaml
# 基础配置
port: 7890
socks-port: 7891
allow-lan: true
mode: rule
log-level: info
external-controller: '0.0.0.0:9090'

# DNS配置
dns:
  enable: true
  listen: 0.0.0.0:1053
  enhanced-mode: fake-ip
  fake-ip-range: 198.18.0.1/16
  nameserver:
    - 127.0.0.1:53
  fallback:
    - https://1.1.1.1/dns-query
    - https://dns.google/dns-query

# 代理配置
proxies: []

# 代理组配置
proxy-groups:
  - name: "PROXY"
    type: select
    proxies:
      - "自动选择"
      - "手动选择"
  
  - name: "自动选择"
    type: url-test
    proxies: []
    url: 'http://www.gstatic.com/generate_204'
    interval: 300

# 规则配置
rules:
  - GEOIP,CN,DIRECT
  - GEOSITE,CN,DIRECT
  - GEOSITE,GFW,PROXY
  - MATCH,DIRECT
```

## 性能优化建议

### 1. 硬件配置
- **CPU**: 2核心以上
- **内存**: 2GB以上
- **存储**: 20GB SSD
- **网络**: 千兆网卡

### 2. 系统优化
- 调整内核参数
- 优化网络栈
- 配置BBR拥塞控制
- 设置合适的文件描述符限制

### 3. 应用优化
- 合理设置缓存大小
- 优化DNS解析策略
- 调整代理并发数
- 配置健康检查

## 监控和维护

### 1. 日志管理
- MosDNS查询日志
- mihomo连接日志
- 系统资源监控
- 错误告警机制

### 2. 性能监控
- DNS解析延迟
- 代理连接成功率
- 带宽使用情况
- 系统负载监控

### 3. 自动化维护
- 规则自动更新
- 订阅定期刷新
- 日志轮转清理
- 服务健康检查

## 安全考虑

### 1. 网络安全
- 防火墙规则配置
- 端口访问控制
- 内网隔离设置
- DDoS防护

### 2. 数据安全
- 配置文件加密
- 敏感信息保护
- 访问权限控制
- 审计日志记录

## 故障排除

### 1. 常见问题
- DNS解析失败
- 代理连接中断
- 规则不生效
- 性能下降

### 2. 诊断工具
- nslookup/dig
- curl测试
- tcpdump抓包
- 性能分析工具

### 3. 恢复策略
- 配置备份
- 服务重启
- 回滚机制
- 应急预案

## 扩展功能

### 1. 高级特性
- 负载均衡
- 故障切换
- 智能路由
- 流量统计

### 2. 集成服务
- AdGuard Home
- Pi-hole
- Nginx代理
- 监控面板

这个方案提供了完整的分流解决方案，既保证了性能，又提供了灵活的配置选项。接下来我可以帮你实现具体的部署脚本和配置文件。