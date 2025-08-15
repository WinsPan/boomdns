# BoomDNS

一个高性能的智能 DNS 服务器，支持智能路由、广告拦截和性能优化。

## 特性

- 🚀 **高性能**: 基于 SQLite 的数据存储，查询性能提升 10 倍
- 🌍 **智能路由**: 支持中国/国际/广告域名智能分流
- 🛡️ **广告拦截**: 内置广告域名黑名单
- 📊 **实时监控**: 详细的查询统计和性能指标
- 🐳 **容器化**: 完整的 Docker 支持，适合 RouterOS 等容器环境
- 🔄 **规则同步**: 支持在线规则更新

## 快速开始

### 1. 本地运行

```bash
# 编译
go build -o boomdns ./cmd/boomdns

# 运行
./boomdns -config config.yaml
```

### 2. Docker 部署

```bash
# 使用部署脚本
cd deploy
./deploy.sh

# 或手动构建
docker build -t boomdns .
docker run -d --name boomdns --network host boomdns
```

### 3. 测试功能

```bash
# 运行测试
./test.sh

# 演示程序
go build -o demo ./cmd/demo
./demo
```

## 配置

编辑 `config.yaml` 文件：

```yaml
# DNS 服务配置
listen_dns: ":53"
listen_http: ":8080"

# 上游 DNS 服务器
upstreams:
  china:
    - "223.5.5.5:53"
  intl:
    - "8.8.8.8:53"

# 数据存储
persistence:
  enabled: true
  database:
    type: "sqlite"  # 推荐使用 SQLite
    sqlite_file: "boomdns.db"
```

## 项目结构

```
boomdns/
├── cmd/                    # 命令行程序
│   ├── boomdns/           # 主程序
│   └── demo/              # 演示程序
├── dns/                   # DNS 核心逻辑
│   ├── server.go          # DNS 服务器
│   ├── sqlite.go          # SQLite 数据存储
│   ├── storage.go         # 存储接口
│   └── config.go          # 配置管理
├── deploy/                # 部署配置
│   ├── Dockerfile         # Docker 镜像
│   ├── docker-compose.yml # 容器编排
│   └── deploy.sh          # 部署脚本
├── web/                   # Web 管理界面
├── admin/                 # 管理 API
└── config.yaml            # 配置文件
```

## 性能特性

- **SQLite 存储**: 比文件存储快 10 倍以上
- **智能缓存**: 自动 TTL 管理和过期清理
- **并发优化**: 支持高并发查询
- **内存优化**: 内存占用 < 256MB

## 部署环境

- **RouterOS 容器**: ✅ 完全支持
- **Docker**: ✅ 官方支持
- **Linux**: ✅ 原生支持
- **Windows**: ✅ 跨平台支持

## 文档

- [SQLite 使用说明](SQLITE_README.md)
- [实现总结](SQLITE_IMPLEMENTATION_SUMMARY.md)

## 许可证

MIT License
