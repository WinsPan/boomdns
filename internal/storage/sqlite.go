package storage

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteManager SQLite 存储管理器
type SQLiteManager struct {
	db *sql.DB
}

// NewSQLiteManager 创建新的 SQLite 管理器
func NewSQLiteManager(dbPath string) (*SQLiteManager, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %v", err)
	}

	// 设置连接参数
	db.SetMaxOpenConns(1) // SQLite 只支持单个连接
	db.SetMaxIdleConns(1)

	// 创建表
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("创建表失败: %v", err)
	}

	return &SQLiteManager{db: db}, nil
}

// createTables 创建数据库表
func createTables(db *sql.DB) error {
	tables := []string{
		// DNS 缓存表
		`CREATE TABLE IF NOT EXISTS dns_cache (
			domain TEXT PRIMARY KEY,
			response BLOB NOT NULL,
			ttl INTEGER NOT NULL,
			created_at INTEGER DEFAULT (strftime('%s', 'now')),
			expires_at INTEGER NOT NULL
		)`,
		
		// DNS 规则表
		`CREATE TABLE IF NOT EXISTS dns_rules (
			category TEXT NOT NULL,
			domain TEXT NOT NULL,
			created_at INTEGER DEFAULT (strftime('%s', 'now')),
			PRIMARY KEY (category, domain)
		)`,
		
		// 查询日志表
		`CREATE TABLE IF NOT EXISTS query_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			domain TEXT NOT NULL,
			client_ip TEXT NOT NULL,
			query_type TEXT NOT NULL,
			response_time INTEGER,
			upstream TEXT,
			cache_hit BOOLEAN DEFAULT 0,
			created_at INTEGER DEFAULT (strftime('%s', 'now'))
		)`,
		
		// 统计信息表
		`CREATE TABLE IF NOT EXISTS stats (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at INTEGER DEFAULT (strftime('%s', 'now'))
		)`,
		
		// 性能指标表
		`CREATE TABLE IF NOT EXISTS performance_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			metric_name TEXT NOT NULL,
			metric_value REAL NOT NULL,
			timestamp INTEGER DEFAULT (strftime('%s', 'now'))
		)`,
		
		// 订阅源表
		`CREATE TABLE IF NOT EXISTS subscription_sources (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			category TEXT NOT NULL,
			url TEXT NOT NULL,
			format TEXT NOT NULL,
			enabled BOOLEAN DEFAULT 1,
			last_update INTEGER,
			last_check INTEGER,
			error_count INTEGER DEFAULT 0,
			last_error TEXT,
			created_at INTEGER DEFAULT (strftime('%s', 'now')),
			updated_at INTEGER DEFAULT (strftime('%s', 'now')),
			UNIQUE(name, category)
		)`,
		
		// 订阅规则表
		`CREATE TABLE IF NOT EXISTS subscription_rules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_id INTEGER NOT NULL,
			category TEXT NOT NULL,
			domain TEXT NOT NULL,
			created_at INTEGER DEFAULT (strftime('%s', 'now')),
			FOREIGN KEY (source_id) REFERENCES subscription_sources(id) ON DELETE CASCADE,
			UNIQUE(source_id, domain)
		)`,
		
		// 订阅统计表
		`CREATE TABLE IF NOT EXISTS subscription_stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_id INTEGER NOT NULL,
			last_success INTEGER,
			last_failure INTEGER,
			success_count INTEGER DEFAULT 0,
			failure_count INTEGER DEFAULT 0,
			total_rules INTEGER DEFAULT 0,
			updated_at INTEGER DEFAULT (strftime('%s', 'now')),
			FOREIGN KEY (source_id) REFERENCES subscription_sources(id) ON DELETE CASCADE
		)`,
		
		// 代理节点表
		`CREATE TABLE IF NOT EXISTS proxy_nodes (
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
		)`,
		
		// 代理节点配置表（存储协议特定配置）
		`CREATE TABLE IF NOT EXISTS proxy_node_configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			node_id INTEGER NOT NULL,
			config_key TEXT NOT NULL,
			config_value TEXT NOT NULL,
			created_at INTEGER DEFAULT (strftime('%s', 'now')),
			FOREIGN KEY (node_id) REFERENCES proxy_nodes(id) ON DELETE CASCADE,
			UNIQUE(node_id, config_key)
		)`,
		
		// 代理组表
		`CREATE TABLE IF NOT EXISTS proxy_groups (
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
		)`,
		
		// 代理组成员表
		`CREATE TABLE IF NOT EXISTS proxy_group_members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			group_id INTEGER NOT NULL,
			node_id INTEGER NOT NULL,
			priority INTEGER DEFAULT 0,
			created_at INTEGER DEFAULT (strftime('%s', 'now')),
			FOREIGN KEY (group_id) REFERENCES proxy_groups(id) ON DELETE CASCADE,
			FOREIGN KEY (node_id) REFERENCES proxy_nodes(id) ON DELETE CASCADE,
			UNIQUE(group_id, node_id)
		)`,
		
		// 代理规则表
		`CREATE TABLE IF NOT EXISTS proxy_rules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			value TEXT NOT NULL,
			action TEXT NOT NULL,
			proxy_group TEXT,
			priority INTEGER DEFAULT 100,
			enabled BOOLEAN DEFAULT 1,
			created_at INTEGER DEFAULT (strftime('%s', 'now')),
			updated_at INTEGER DEFAULT (strftime('%s', 'now'))
		)`,
		
		// 代理使用统计表
		`CREATE TABLE IF NOT EXISTS proxy_usage_stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			node_id INTEGER NOT NULL,
			bytes_sent INTEGER DEFAULT 0,
			bytes_received INTEGER DEFAULT 0,
			connections INTEGER DEFAULT 0,
			last_used INTEGER,
			created_at INTEGER DEFAULT (strftime('%s', 'now')),
			FOREIGN KEY (node_id) REFERENCES proxy_nodes(id) ON DELETE CASCADE
		)`,
	}

	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			return fmt.Errorf("创建表失败: %v", err)
		}
	}

	// 创建索引
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_rules_category ON dns_rules(category)",
		"CREATE INDEX IF NOT EXISTS idx_cache_expires ON dns_cache(expires_at)",
		"CREATE INDEX IF NOT EXISTS idx_logs_domain ON query_logs(domain)",
		"CREATE INDEX IF NOT EXISTS idx_logs_created ON query_logs(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_subscription_sources_category ON subscription_sources(category)",
		"CREATE INDEX IF NOT EXISTS idx_subscription_sources_enabled ON subscription_sources(enabled)",
		"CREATE INDEX IF NOT EXISTS idx_subscription_rules_source ON subscription_rules(source_id)",
		"CREATE INDEX IF NOT EXISTS idx_subscription_rules_category ON subscription_rules(category)",
		"CREATE INDEX IF NOT EXISTS idx_proxy_nodes_protocol ON proxy_nodes(protocol)",
		"CREATE INDEX IF NOT EXISTS idx_proxy_nodes_enabled ON proxy_nodes(enabled)",
		"CREATE INDEX IF NOT EXISTS idx_proxy_groups_type ON proxy_groups(type)",
		"CREATE INDEX IF NOT EXISTS idx_proxy_rules_type ON proxy_rules(type)",
		"CREATE INDEX IF NOT EXISTS idx_proxy_rules_priority ON proxy_rules(priority)",
	}

	for _, index := range indexes {
		if _, err := db.Exec(index); err != nil {
			return fmt.Errorf("创建索引失败: %v", err)
		}
	}

	return nil
}

// Close 关闭数据库连接
func (sm *SQLiteManager) Close() error {
	return sm.db.Close()
}

// GetDB 获取数据库连接
func (sm *SQLiteManager) GetDB() *sql.DB {
	return sm.db
}

// 实现其他必要的接口方法...
// 这里可以根据需要添加更多的方法
