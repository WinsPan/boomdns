#!/usr/bin/env bash

# BoomDNS 数据库更新脚本
set -euo pipefail

# 配置
readonly DB_PATH="data/boomdns.db"

# 创建代理相关表
create_proxy_tables() {
    echo "创建代理相关表..."
    
    sqlite3 "$DB_PATH" << 'EOF'
-- 代理节点表
CREATE TABLE IF NOT EXISTS proxy_nodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    protocol TEXT NOT NULL,
    address TEXT NOT NULL,
    port INTEGER NOT NULL,
    enabled BOOLEAN DEFAULT 1,
    weight INTEGER DEFAULT 100,
    latency INTEGER DEFAULT -1,
    last_check INTEGER,
    fail_count INTEGER DEFAULT 0,
    created_at INTEGER DEFAULT (strftime('%s', 'now')),
    updated_at INTEGER DEFAULT (strftime('%s', 'now'))
);

-- 代理节点配置表
CREATE TABLE IF NOT EXISTS proxy_node_configs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id INTEGER NOT NULL,
    config_key TEXT NOT NULL,
    config_value TEXT NOT NULL,
    created_at INTEGER DEFAULT (strftime('%s', 'now')),
    FOREIGN KEY (node_id) REFERENCES proxy_nodes(id) ON DELETE CASCADE,
    UNIQUE(node_id, config_key)
);

-- 代理组表
CREATE TABLE IF NOT EXISTS proxy_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    strategy TEXT NOT NULL,
    test_url TEXT,
    interval INTEGER DEFAULT 300,
    timeout INTEGER DEFAULT 10,
    enabled BOOLEAN DEFAULT 1,
    created_at INTEGER DEFAULT (strftime('%s', 'now')),
    updated_at INTEGER DEFAULT (strftime('%s', 'now'))
);

-- 代理组成员表
CREATE TABLE IF NOT EXISTS proxy_group_members (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id INTEGER NOT NULL,
    node_id INTEGER NOT NULL,
    priority INTEGER DEFAULT 0,
    created_at INTEGER DEFAULT (strftime('%s', 'now')),
    FOREIGN KEY (group_id) REFERENCES proxy_groups(id) ON DELETE CASCADE,
    FOREIGN KEY (node_id) REFERENCES proxy_nodes(id) ON DELETE CASCADE,
    UNIQUE(group_id, node_id)
);

-- 代理规则表
CREATE TABLE IF NOT EXISTS proxy_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL,
    value TEXT NOT NULL,
    action TEXT NOT NULL,
    proxy_group TEXT,
    priority INTEGER DEFAULT 100,
    enabled BOOLEAN DEFAULT 1,
    created_at INTEGER DEFAULT (strftime('%s', 'now')),
    updated_at INTEGER DEFAULT (strftime('%s', 'now'))
);

-- 代理使用统计表
CREATE TABLE IF NOT EXISTS proxy_usage_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id INTEGER NOT NULL,
    bytes_sent INTEGER DEFAULT 0,
    bytes_received INTEGER DEFAULT 0,
    connections INTEGER DEFAULT 0,
    last_used INTEGER,
    created_at INTEGER DEFAULT (strftime('%s', 'now')),
    FOREIGN KEY (node_id) REFERENCES proxy_nodes(id) ON DELETE CASCADE
);
EOF

    echo "代理表创建完成"
}

# 创建索引
create_indexes() {
    echo "创建索引..."
    
    sqlite3 "$DB_PATH" << 'EOF'
CREATE INDEX IF NOT EXISTS idx_proxy_nodes_protocol ON proxy_nodes(protocol);
CREATE INDEX IF NOT EXISTS idx_proxy_nodes_enabled ON proxy_nodes(enabled);
CREATE INDEX IF NOT EXISTS idx_proxy_groups_type ON proxy_groups(type);
CREATE INDEX IF NOT EXISTS idx_proxy_rules_type ON proxy_rules(type);
CREATE INDEX IF NOT EXISTS idx_proxy_rules_priority ON proxy_rules(priority);
EOF

    echo "索引创建完成"
}

# 插入示例数据
insert_sample_data() {
    echo "插入示例数据..."
    
    sqlite3 "$DB_PATH" << 'EOF'
-- 插入示例代理节点
INSERT OR IGNORE INTO proxy_nodes (name, protocol, address, port, enabled, weight) VALUES 
('Hysteria2-香港', 'hysteria2', 'hk.example.com', 443, 1, 100),
('SS-香港', 'ss', 'hk-ss.example.com', 8388, 1, 80),
('V2Ray-美国', 'v2ray', 'us.example.com', 443, 1, 60);

-- 插入示例代理组
INSERT OR IGNORE INTO proxy_groups (name, type, strategy, test_url, interval, timeout, enabled) VALUES 
('自动选择', 'url-test', 'latency', 'http://www.google.com', 300, 10, 1),
('故障转移', 'fallback', 'latency', 'http://www.google.com', 300, 10, 1);

-- 插入示例代理规则
INSERT OR IGNORE INTO proxy_rules (type, value, action, proxy_group, priority, enabled) VALUES 
('domain', 'google.com', 'proxy', '自动选择', 100, 1),
('domain', 'youtube.com', 'proxy', '自动选择', 100, 1),
('domain', 'github.com', 'proxy', '自动选择', 100, 1),
('domain', 'baidu.com', 'direct', NULL, 200, 1);
EOF

    echo "示例数据插入完成"
}

# 验证数据库
verify_database() {
    echo "验证数据库结构..."
    
    echo "数据库表:"
    sqlite3 "$DB_PATH" ".tables"
    
    echo ""
    echo "代理相关表:"
    sqlite3 "$DB_PATH" "SELECT name FROM sqlite_master WHERE type='table' AND name LIKE '%proxy%';"
    
    echo ""
    echo "数据统计:"
    echo "代理节点数量: $(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM proxy_nodes;")"
    echo "代理组数量: $(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM proxy_groups;")"
    echo "代理规则数量: $(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM proxy_rules;")"
}

# 主函数
main() {
    echo "开始更新 BoomDNS 数据库..."
    
    create_proxy_tables
    create_indexes
    insert_sample_data
    verify_database
    
    echo "数据库更新完成！"
}

main "$@"
