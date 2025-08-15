# BoomDNS SQLite 数据存储

## 概述

BoomDNS 现在支持使用 SQLite 作为数据存储后端，相比原来的 JSON 文件存储，SQLite 提供了更好的性能、数据一致性和查询能力。

## 特性

- **高性能**: 支持索引和 SQL 查询，比文件存储快很多
- **数据一致性**: 支持 ACID 事务，确保数据完整性
- **并发安全**: 内置并发控制，支持多进程/多线程访问
- **自动清理**: 自动清理过期缓存和旧日志
- **统计查询**: 支持复杂的统计查询和分析
- **向后兼容**: 可以无缝切换到 SQLite，不影响现有功能

## 配置

在 `config.yaml` 中启用 SQLite：

```yaml
persistence:
  enabled: true
  data_dir: "./data"
  database:
    type: "sqlite"  # 使用 SQLite
    sqlite_file: "boomdns.db"  # 数据库文件名
    # 文件存储配置（当 type 为 "file" 时使用）
    cache_file: "cache.json"
    logs_file: "logs.json"
    stats_file: "stats.json"
    rules_file: "rules.json"
  auto_save_interval: 300  # 自动保存间隔（秒）
  max_logs: 10000          # 最大日志条数
  max_cache_entries: 10000 # 最大缓存条目数
```

## 数据库结构

### 1. DNS 缓存表 (dns_cache)

```sql
CREATE TABLE dns_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    domain TEXT NOT NULL,           -- 域名
    query_type TEXT NOT NULL,       -- 查询类型 (A, AAAA, MX等)
    response TEXT NOT NULL,         -- DNS响应数据 (JSON格式)
    ttl INTEGER NOT NULL,           -- TTL值
    expire_at INTEGER NOT NULL,     -- 过期时间戳
    created_at INTEGER NOT NULL,    -- 创建时间戳
    updated_at INTEGER NOT NULL,    -- 更新时间戳
    UNIQUE(domain, query_type)      -- 唯一约束
);
```

### 2. 查询日志表 (query_logs)

```sql
CREATE TABLE query_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,             -- 查询的域名
    route TEXT NOT NULL,            -- 路由决策 (china, intl, gfw等)
    latency INTEGER NOT NULL,       -- 响应延迟 (毫秒)
    timestamp INTEGER NOT NULL      -- 查询时间戳
);
```

### 3. DNS 规则表 (dns_rules)

```sql
CREATE TABLE dns_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_type TEXT NOT NULL,        -- 规则类型 (china, gfw, ads等)
    domain TEXT NOT NULL,           -- 域名
    action TEXT NOT NULL,           -- 动作 (block, redirect等)
    target TEXT,                    -- 目标地址
    priority INTEGER NOT NULL DEFAULT 0,  -- 优先级
    enabled BOOLEAN NOT NULL DEFAULT 1,   -- 是否启用
    created_at INTEGER NOT NULL,    -- 创建时间戳
    updated_at INTEGER NOT NULL,    -- 更新时间戳
    UNIQUE(rule_type, domain)       -- 唯一约束
);
```

### 4. 统计信息表 (stats)

```sql
CREATE TABLE stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    metric_name TEXT NOT NULL,      -- 指标名称
    metric_value TEXT NOT NULL,     -- 指标值 (JSON格式)
    timestamp INTEGER NOT NULL,     -- 时间戳
    UNIQUE(metric_name, timestamp)  -- 唯一约束
);
```

## 索引

为了提高查询性能，系统自动创建以下索引：

- `idx_cache_domain`: 缓存域名索引
- `idx_cache_expire`: 缓存过期时间索引
- `idx_logs_name`: 日志域名索引
- `idx_logs_timestamp`: 日志时间索引
- `idx_rules_type_domain`: 规则类型和域名索引
- `idx_stats_metric`: 统计指标索引

## 使用示例

### 1. 查看缓存数据

```bash
# 连接到数据库
sqlite3 data/boomdns.db

# 查看所有表
.tables

# 查看缓存数据
SELECT domain, query_type, ttl, datetime(expire_at, 'unixepoch') as expire_time 
FROM dns_cache 
WHERE expire_at > strftime('%s', 'now');
```

### 2. 查看查询统计

```bash
# 今日查询数
SELECT COUNT(*) FROM query_logs 
WHERE timestamp >= strftime('%s', 'now', 'start of day');

# 平均响应时间
SELECT AVG(latency) FROM query_logs;

# 按路由分组的查询统计
SELECT route, COUNT(*) as count, AVG(latency) as avg_latency 
FROM query_logs 
GROUP BY route;
```

### 3. 查看规则数据

```bash
# 查看所有启用的规则
SELECT rule_type, domain, action, priority 
FROM dns_rules 
WHERE enabled = 1 
ORDER BY rule_type, priority;

# 查看特定类型的规则
SELECT domain FROM dns_rules WHERE rule_type = 'china' AND enabled = 1;
```

## 性能优化

### 1. 数据库参数

```go
// SQLite 建议单连接
db.SetMaxOpenConns(1)
db.SetMaxIdleConns(1)
```

### 2. 定期清理

系统会自动清理：
- 过期的缓存条目
- 7天前的查询日志
- 旧的统计数据

### 3. 批量操作

所有数据操作都使用事务，提高写入性能：

```go
tx, err := db.Begin()
// ... 批量插入数据
err = tx.Commit()
```

## 迁移指南

### 从文件存储迁移到 SQLite

1. 备份现有数据
2. 修改配置文件，设置 `database.type: "sqlite"`
3. 重启服务，系统会自动创建数据库和表
4. 验证数据是否正确加载

### 从 SQLite 迁移到文件存储

1. 修改配置文件，设置 `database.type: "file"`
2. 重启服务，系统会使用文件存储
3. 数据会保存为 JSON 文件

## 故障排除

### 1. 数据库锁定

如果遇到数据库锁定错误，检查：
- 是否有多个进程同时访问数据库
- 数据库文件权限是否正确
- 磁盘空间是否充足

### 2. 性能问题

如果查询性能较差，检查：
- 索引是否正确创建
- 是否有长时间运行的事务
- 数据库文件大小是否过大

### 3. 数据损坏

如果数据损坏，可以：
- 删除数据库文件，系统会重新创建
- 检查磁盘健康状态
- 从备份恢复数据

## RouterOS 容器支持

SQLite 完全支持在 RouterOS 的容器环境中运行：

- **资源占用少**: 内存和磁盘占用都很小
- **无外部依赖**: 只需要 SQLite 库文件
- **跨平台支持**: 支持 ARM、x86 等架构
- **容器友好**: 适合资源受限的容器环境

## 测试

运行测试脚本验证 SQLite 功能：

```bash
chmod +x test-sqlite.sh
./test-sqlite.sh
```

## 总结

SQLite 为 BoomDNS 提供了企业级的数据存储能力，特别适合：
- 高并发环境
- 需要复杂查询的场景
- 对数据一致性要求高的应用
- 资源受限的容器环境

通过合理的配置和优化，SQLite 可以显著提升 BoomDNS 的性能和可靠性。
