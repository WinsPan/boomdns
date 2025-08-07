# MosDNS 配置指南

## 配置概述

这个 MosDNS 配置文件是为您的家庭服务器分流方案特别设计的，实现了智能的 DNS 分流和查询优化。

## 主要特性

### 🚀 智能分流
- **国内域名**: 使用阿里DNS、114DNS、腾讯DNS
- **国外域名**: 使用 Cloudflare DoH、Google DoH
- **本地域名**: 直接使用国内DNS解析
- **广告域名**: 返回 NXDOMAIN 实现拦截

### 📊 性能优化
- **缓存策略**: 2048条目缓存，24小时懒惰缓存
- **并发查询**: 国内3并发，国外2并发
- **TTL优化**: 最小5分钟，最大1小时
- **缓存持久化**: 定期保存缓存到磁盘

### 🔒 安全特性
- **DoH支持**: 国外DNS使用HTTPS加密
- **广告拦截**: 基于geosite和自定义规则
- **安全DNS**: 可选的恶意软件拦截

## 目录结构

配置文件依赖以下目录结构：

```
/opt/homeserver/
├── data/
│   ├── geosite.dat          # 域名地理数据库
│   ├── geoip.dat            # IP地理数据库
│   └── cache.dump           # DNS缓存持久化文件
└── rules/
    ├── reject.txt           # 广告域名黑名单
    └── direct.txt           # 国内域名白名单

/var/log/homeserver/
└── mosdns/
    └── mosdns.log          # MosDNS运行日志

/etc/homeserver/
└── mosdns/
    └── config.yaml         # MosDNS主配置文件
```

## 安装配置

### 1. 创建目录结构
```bash
sudo mkdir -p /opt/homeserver/{data,rules}
sudo mkdir -p /var/log/homeserver/mosdns
sudo mkdir -p /etc/homeserver/mosdns
```

### 2. 复制配置文件
```bash
# 复制主配置文件
sudo cp mosdns-config.yaml /etc/homeserver/mosdns/config.yaml

# 设置权限
sudo chown -R homeserver:homeserver /opt/homeserver
sudo chown -R homeserver:homeserver /var/log/homeserver
sudo chown -R homeserver:homeserver /etc/homeserver
```

### 3. 下载数据文件
```bash
# 下载geosite和geoip数据库
wget -O /opt/homeserver/data/geosite.dat https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat
wget -O /opt/homeserver/data/geoip.dat https://github.com/v2fly/geoip/releases/latest/download/geoip.dat
```

### 4. 创建规则文件
```bash
# 创建基础规则文件（参考 mosdns-rules-example.txt）
sudo touch /opt/homeserver/rules/reject.txt
sudo touch /opt/homeserver/rules/direct.txt
```

## 配置说明

### DNS 分流逻辑

1. **查询类型过滤**: 只处理 A 和 AAAA 记录查询
2. **本地域名**: `.local`, `.lan`, `.localhost` 等直接使用国内DNS
3. **广告拦截**: 匹配广告域名列表，返回 NXDOMAIN
4. **缓存检查**: 优先从缓存获取结果
5. **国内域名**: 使用国内DNS服务器组
6. **国外域名**: 使用国外DNS服务器组（DoH）
7. **默认策略**: 未匹配域名使用国内DNS

### 上游DNS服务器

#### 国内DNS组
- **阿里DNS**: 223.5.5.5 (主要)
- **114DNS**: 114.114.114.114 (备用)
- **腾讯DNS**: 119.29.29.29 (备用)

#### 国外DNS组
- **Cloudflare DoH**: https://1.1.1.1/dns-query
- **Google DoH**: https://8.8.8.8/dns-query

#### 安全DNS组（可选）
- **Quad9**: https://dns.quad9.net/dns-query
- **Cloudflare安全版**: https://security.cloudflare-dns.com/dns-query

### 性能参数调优

#### 缓存配置
```yaml
cache:
  size: 2048              # 根据内存调整（建议1024-4096）
  lazy_cache_ttl: 86400   # 24小时（可调整为12-48小时）
  dump_interval: 3600     # 1小时持久化（可调整为30分钟-2小时）
```

#### 并发配置
```yaml
domestic_upstream:
  concurrent: 3           # 国内DNS并发数（建议2-5）

foreign_upstream:
  concurrent: 2           # 国外DNS并发数（建议1-3）
```

#### TTL配置
```yaml
modify_ttl:
  minimal_ttl: 300        # 5分钟最小TTL
  maximum_ttl: 3600       # 1小时最大TTL
```

## 服务管理

### 使用systemd管理
```bash
# 启动服务
sudo systemctl start mosdns

# 停止服务
sudo systemctl stop mosdns

# 重启服务
sudo systemctl restart mosdns

# 查看状态
sudo systemctl status mosdns

# 开机自启
sudo systemctl enable mosdns
```

### 查看日志
```bash
# 实时日志
sudo journalctl -u mosdns -f

# 历史日志
sudo journalctl -u mosdns --since "1 hour ago"

# 查看配置文件日志
sudo tail -f /var/log/homeserver/mosdns/mosdns.log
```

## 测试验证

### DNS解析测试
```bash
# 测试国内域名解析
nslookup baidu.com 127.0.0.1

# 测试国外域名解析
nslookup google.com 127.0.0.1

# 测试广告域名拦截
nslookup doubleclick.net 127.0.0.1

# 使用dig测试
dig @127.0.0.1 github.com
```

### 性能测试
```bash
# 测试解析速度
time nslookup baidu.com 127.0.0.1

# 测试缓存效果（第二次应该更快）
time nslookup baidu.com 127.0.0.1
```

## 故障排除

### 常见问题

1. **DNS解析失败**
   ```bash
   # 检查服务状态
   sudo systemctl status mosdns
   
   # 检查端口监听
   sudo ss -tulnp | grep :53
   
   # 检查配置文件语法
   mosdns verify -c /etc/homeserver/mosdns/config.yaml
   ```

2. **上游DNS无法访问**
   ```bash
   # 测试上游DNS连通性
   ping 223.5.5.5
   ping 1.1.1.1
   
   # 测试DoH连接
   curl -H "Accept: application/dns-json" "https://1.1.1.1/dns-query?name=google.com&type=A"
   ```

3. **规则不生效**
   ```bash
   # 检查规则文件
   ls -la /opt/homeserver/rules/
   
   # 检查geosite数据
   ls -la /opt/homeserver/data/
   
   # 重新下载geosite数据
   wget -O /opt/homeserver/data/geosite.dat https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat
   ```

### 调试模式

修改配置文件中的日志级别为debug：
```yaml
log:
  level: debug
```

重启服务后查看详细日志。

## 自定义配置

### 远程列表自动更新

配置文件已支持从远程URL自动下载和更新规则列表：

#### 内置的远程规则源

1. **广告拦截列表**:
   - AdGuard 广告拦截列表
   - EasyList 广告拦截
   - 反广告联盟列表
   - 国内广告拦截列表
   - 更新频率：每小时更新一次

2. **国内直连列表**:
   - Loyalsoldier 直连域名
   - 国内常用域名加速列表
   - Apple 中国域名
   - 更新频率：每24小时更新一次

3. **代理域名列表**:
   - GFW 列表
   - 国际流媒体域名
   - Telegram 域名
   - 更新频率：每24小时更新一次

#### 手动更新规则

使用提供的自动更新脚本：

```bash
# 给脚本执行权限
chmod +x update-mosdns-rules.sh

# 执行更新
sudo ./update-mosdns-rules.sh

# 测试模式（不重启服务）
sudo ./update-mosdns-rules.sh --test
```

#### 设置定时更新

创建cron定时任务：

```bash
# 编辑root的crontab
sudo crontab -e

# 添加定时任务（每天凌晨2点更新）
0 2 * * * /path/to/update-mosdns-rules.sh >/dev/null 2>&1

# 或者每6小时更新一次
0 */6 * * * /path/to/update-mosdns-rules.sh >/dev/null 2>&1
```

### 添加自定义规则

1. **添加广告域名**:
   编辑 `/opt/homeserver/rules/reject.txt`，添加要拦截的域名

2. **添加直连域名**:
   编辑 `/opt/homeserver/rules/direct.txt`，添加要直连的域名

3. **添加代理域名**:
   编辑 `/opt/homeserver/rules/proxy.txt`，添加要通过代理访问的域名

4. **修改上游DNS**:
   在配置文件中修改 `upstreams` 部分

5. **自定义远程规则源**:
   在配置文件的 `urls` 部分添加自定义的远程列表地址

### 性能调优建议

1. **内存充足**:
   - 增加缓存大小到4096或更多
   - 减少dump_interval到30分钟

2. **网络较慢**:
   - 减少并发数到1-2
   - 增加查询超时时间

3. **高并发环境**:
   - 增加domestic_upstream并发数到5
   - 添加更多上游DNS服务器

## 维护和更新

### 定期维护任务

1. **更新geosite数据库**（建议每周）:
   ```bash
   wget -O /opt/homeserver/data/geosite.dat.new https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat
   mv /opt/homeserver/data/geosite.dat.new /opt/homeserver/data/geosite.dat
   sudo systemctl restart mosdns
   ```

2. **清理日志文件**（建议每月）:
   ```bash
   sudo journalctl --vacuum-time=30d
   sudo logrotate /etc/logrotate.d/mosdns
   ```

3. **备份配置**（建议每月）:
   ```bash
   sudo tar -czf /backup/mosdns-config-$(date +%Y%m%d).tar.gz /etc/homeserver/mosdns/ /opt/homeserver/rules/
   ```

### 自动化脚本

可以创建cron任务自动执行维护：
```bash
# 编辑crontab
sudo crontab -e

# 添加定时任务
0 2 * * 0 /usr/local/bin/update-geosite.sh  # 每周日凌晨2点更新geosite
0 3 1 * * /usr/local/bin/cleanup-logs.sh    # 每月1日凌晨3点清理日志
```

这个配置文件为您的家庭服务器提供了企业级的DNS解析能力，结合mihomo代理可以实现完整的网络分流方案。
