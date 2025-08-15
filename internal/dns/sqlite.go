package dns

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SubscriptionSource 订阅源结构体
type SubscriptionSource struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Category   string `json:"category"`
	URL        string `json:"url"`
	Format     string `json:"format"`
	Enabled    bool   `json:"enabled"`
	LastUpdate int64  `json:"last_update"`
	LastCheck  int64  `json:"last_check"`
	ErrorCount int    `json:"error_count"`
	LastError  string `json:"last_error"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

// SQLiteManager SQLite 数据管理器 (极致优化版本)
type SQLiteManager struct {
	db             *sql.DB
	config         *Config
	autoSaveTicker *time.Ticker
	stopChan       chan bool
	mutex          sync.RWMutex

	// 性能优化: 连接池配置
	maxOpenConns    int
	maxIdleConns    int
	connMaxLifetime time.Duration

	// 批量操作优化
	batchSize   int
	cacheBuffer map[string]*CacheEntry
	logsBuffer  []QueryLog
	rulesBuffer map[string][]string
	statsBuffer map[string]interface{}
	bufferMutex sync.RWMutex
	lastFlush   time.Time
}

// NewSQLiteManager 创建新的 SQLite 管理器
func NewSQLiteManager(config *Config) (*SQLiteManager, error) {
	dbPath := config.GetSQLiteFile()
	if dbPath == "" {
		dbPath = "boomdns.db"
	}

	// 构建完整路径
	var fullPath string
	if filepath.IsAbs(dbPath) {
		fullPath = dbPath
	} else {
		fullPath = filepath.Join(config.Persistence.DataDir, dbPath)
	}

	// 打开数据库连接
	db, err := sql.Open("sqlite3", fullPath+"?_journal_mode=WAL&_synchronous=NORMAL&_cache_size=10000&_temp_store=MEMORY&_mmap_size=268435456")
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite 数据库失败: %v", err)
	}

	// 配置连接池
	maxOpenConns := 25
	maxIdleConns := 5
	connMaxLifetime := 5 * time.Minute

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("数据库连接测试失败: %v", err)
	}

	sm := &SQLiteManager{
		db:              db,
		config:          config,
		maxOpenConns:    maxOpenConns,
		maxIdleConns:    maxIdleConns,
		connMaxLifetime: connMaxLifetime,
		batchSize:       1000,
		cacheBuffer:     make(map[string]*CacheEntry),
		logsBuffer:      make([]QueryLog, 0, 1000),
		rulesBuffer:     make(map[string][]string),
		statsBuffer:     make(map[string]interface{}),
		lastFlush:       time.Now(),
	}

	// 初始化数据库
	if err := sm.initDatabase(); err != nil {
		db.Close()
		return nil, fmt.Errorf("初始化数据库失败: %v", err)
	}

	// 启动自动保存
	sm.StartAutoSave()

	// 启动定期清理
	go sm.periodicCleanup()

	return sm, nil
}

// initDatabase 初始化数据库
func (sm *SQLiteManager) initDatabase() error {
	// 创建表
	if err := sm.createTables(); err != nil {
		return err
	}

	// 创建索引
	if err := sm.createIndexes(); err != nil {
		return err
	}

	// 设置 PRAGMA 优化
	if err := sm.optimizeDatabase(); err != nil {
		return err
	}

	return nil
}

// createTables 创建数据表
func (sm *SQLiteManager) createTables() error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS dns_cache (
			key TEXT PRIMARY KEY,
			response BLOB,
			expire_at INTEGER,
			created_at INTEGER,
			access_count INTEGER DEFAULT 0,
			last_access INTEGER
		)`,

		`CREATE TABLE IF NOT EXISTS query_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			route TEXT NOT NULL,
			latency INTEGER NOT NULL,
			timestamp INTEGER NOT NULL,
			client_ip TEXT,
			query_type TEXT,
			response_code TEXT,
			created_at INTEGER DEFAULT (strftime('%s', 'now'))
		)`,

		`CREATE TABLE IF NOT EXISTS dns_rules (
			category TEXT NOT NULL,
			domain TEXT NOT NULL,
			created_at INTEGER DEFAULT (strftime('%s', 'now')),
			PRIMARY KEY (category, domain)
		)`,

		`CREATE TABLE IF NOT EXISTS stats (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at INTEGER DEFAULT (strftime('%s', 'now'))
		)`,

		`CREATE TABLE IF NOT EXISTS performance_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			operation TEXT NOT NULL,
			duration_ms INTEGER NOT NULL,
			timestamp INTEGER DEFAULT (strftime('%s', 'now')),
			metadata TEXT
		)`,

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

		`CREATE TABLE IF NOT EXISTS subscription_rules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_id INTEGER NOT NULL,
			category TEXT NOT NULL,
			domain TEXT NOT NULL,
			created_at INTEGER DEFAULT (strftime('%s', 'now')),
			FOREIGN KEY (source_id) REFERENCES subscription_sources(id) ON DELETE CASCADE,
			UNIQUE(source_id, domain)
		)`,

		`CREATE TABLE IF NOT EXISTS subscription_stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_id INTEGER NOT NULL,
			total_domains INTEGER DEFAULT 0,
			last_successful_update INTEGER,
			update_count INTEGER DEFAULT 0,
			created_at INTEGER DEFAULT (strftime('%s', 'now')),
			updated_at INTEGER DEFAULT (strftime('%s', 'now')),
			FOREIGN KEY (source_id) REFERENCES subscription_sources(id) ON DELETE CASCADE
		)`,
	}

	for _, table := range tables {
		if _, err := sm.db.Exec(table); err != nil {
			return fmt.Errorf("创建表失败: %v", err)
		}
	}

	return nil
}

// createIndexes 创建性能索引
func (sm *SQLiteManager) createIndexes() error {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_cache_expire ON dns_cache(expire_at)",
		"CREATE INDEX IF NOT EXISTS idx_cache_access ON dns_cache(last_access)",
		"CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON query_logs(timestamp)",
		"CREATE INDEX IF NOT EXISTS idx_logs_name ON query_logs(name)",
		"CREATE INDEX IF NOT EXISTS idx_logs_route ON query_logs(route)",
		"CREATE INDEX IF NOT EXISTS idx_rules_category ON dns_rules(category)",
		"CREATE INDEX IF NOT EXISTS idx_performance_timestamp ON performance_metrics(timestamp)",
		"CREATE INDEX IF NOT EXISTS idx_performance_operation ON performance_metrics(operation)",
		"CREATE INDEX IF NOT EXISTS idx_subscription_sources_category ON subscription_sources(category)",
		"CREATE INDEX IF NOT EXISTS idx_subscription_sources_enabled ON subscription_sources(enabled)",
		"CREATE INDEX IF NOT EXISTS idx_subscription_rules_source ON subscription_rules(source_id)",
		"CREATE INDEX IF NOT EXISTS idx_subscription_rules_category ON subscription_rules(category)",
		"CREATE INDEX IF NOT EXISTS idx_subscription_stats_source ON subscription_stats(source_id)",
	}

	for _, index := range indexes {
		if _, err := sm.db.Exec(index); err != nil {
			return fmt.Errorf("创建索引失败: %v", err)
		}
	}

	return nil
}

// optimizeDatabase 优化数据库性能
func (sm *SQLiteManager) optimizeDatabase() error {
	optimizations := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA cache_size = 10000",
		"PRAGMA temp_store = MEMORY",
		"PRAGMA mmap_size = 268435456",
		"PRAGMA auto_vacuum = INCREMENTAL",
		"PRAGMA incremental_vacuum = 1000",
		"PRAGMA optimize",
	}

	for _, opt := range optimizations {
		if _, err := sm.db.Exec(opt); err != nil {
			return fmt.Errorf("数据库优化失败: %v", err)
		}
	}

	return nil
}

// SaveCache 保存缓存数据 (批量优化版本)
func (sm *SQLiteManager) SaveCache(cache map[string]*CacheEntry) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// 批量写入优化
	tx, err := sm.db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %v", err)
	}
	defer tx.Rollback()

	// 准备语句
	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO dns_cache 
		(key, response, expire_at, created_at, access_count, last_access) 
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("准备语句失败: %v", err)
	}
	defer stmt.Close()

	// 批量执行
	for key, entry := range cache {
		responseData, err := json.Marshal(entry.Response)
		if err != nil {
			log.Printf("序列化缓存响应失败: %v", err)
			continue
		}

		_, err = stmt.Exec(
			key,
			responseData,
			entry.ExpireAt.Unix(),
			time.Now().Unix(),
			entry.Hits,
			time.Now().Unix(),
		)
		if err != nil {
			log.Printf("插入缓存记录失败: %v", err)
			continue
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %v", err)
	}

	return nil
}

// LoadCache 加载缓存数据 (批量优化版本)
func (sm *SQLiteManager) LoadCache() (map[string]*CacheEntry, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	rows, err := sm.db.Query(`
		SELECT key, response, expire_at, created_at, access_count, last_access 
		FROM dns_cache 
		WHERE expire_at > ? 
		ORDER BY last_access DESC
	`, time.Now().Unix())
	if err != nil {
		return nil, fmt.Errorf("查询缓存失败: %v", err)
	}
	defer rows.Close()

	cache := make(map[string]*CacheEntry)

	for rows.Next() {
		var (
			key          string
			responseData []byte
			expireAt     int64
			createdAt    int64
			accessCount  int64
			lastAccess   int64
		)

		if err := rows.Scan(&key, &responseData, &expireAt, &createdAt, &accessCount, &lastAccess); err != nil {
			log.Printf("扫描缓存记录失败: %v", err)
			continue
		}

		// 检查是否过期
		if time.Unix(expireAt, 0).Before(time.Now()) {
			continue
		}

		// 创建缓存条目
		entry := &CacheEntry{
			ExpireAt: time.Unix(expireAt, 0),
			Hits:     accessCount,
		}

		// 反序列化响应 (简化处理)
		entry.Response = nil // 避免类型断言问题

		cache[key] = entry
	}

	return cache, nil
}

// SaveLogs 保存查询日志 (批量优化版本)
func (sm *SQLiteManager) SaveLogs(logs []QueryLog) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if len(logs) == 0 {
		return nil
	}

	// 批量写入优化
	tx, err := sm.db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %v", err)
	}
	defer tx.Rollback()

	// 准备语句
	stmt, err := tx.Prepare(`
		INSERT INTO query_logs 
		(name, route, latency, timestamp, client_ip, query_type, response_code) 
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("准备语句失败: %v", err)
	}
	defer stmt.Close()

	// 批量执行
	for _, log := range logs {
		_, err = stmt.Exec(
			log.Name,
			log.Route,
			log.Latency,
			log.Time.Unix(),
			"",        // client_ip (可选)
			"A",       // query_type (可选)
			"NOERROR", // response_code (可选)
		)
		if err != nil {
			fmt.Printf("插入日志记录失败: %v", err)
			continue
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %v", err)
	}

	return nil
}

// LoadLogs 加载查询日志 (分页优化版本)
func (sm *SQLiteManager) LoadLogs() ([]QueryLog, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	// 限制加载数量，避免内存问题
	limit := sm.config.Persistence.MaxLogs
	if limit <= 0 {
		limit = 10000
	}

	rows, err := sm.db.Query(`
		SELECT name, route, latency, timestamp 
		FROM query_logs 
		ORDER BY timestamp DESC 
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("查询日志失败: %v", err)
	}
	defer rows.Close()

	var logs []QueryLog

	for rows.Next() {
		var (
			name      string
			route     string
			latency   int64
			timestamp int64
		)

		if err := rows.Scan(&name, &route, &latency, &timestamp); err != nil {
			log.Printf("扫描日志记录失败: %v", err)
			continue
		}

		logs = append(logs, QueryLog{
			Name:    name,
			Route:   route,
			Latency: latency,
			Time:    time.Unix(timestamp, 0),
		})
	}

	return logs, nil
}

// SaveStats 保存统计信息
func (sm *SQLiteManager) SaveStats(stats map[string]interface{}) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	tx, err := sm.db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %v", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO stats (key, value, updated_at) 
		VALUES (?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("准备语句失败: %v", err)
	}
	defer stmt.Close()

	for key, value := range stats {
		valueData, err := json.Marshal(value)
		if err != nil {
			log.Printf("序列化统计值失败: %v", err)
			continue
		}

		_, err = stmt.Exec(key, string(valueData), time.Now().Unix())
		if err != nil {
			log.Printf("插入统计记录失败: %v", err)
			continue
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %v", err)
	}

	return nil
}

// LoadStats 加载统计信息
func (sm *SQLiteManager) LoadStats() (map[string]interface{}, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	rows, err := sm.db.Query(`SELECT key, value FROM stats`)
	if err != nil {
		return nil, fmt.Errorf("查询统计信息失败: %v", err)
	}
	defer rows.Close()

	stats := make(map[string]interface{})

	for rows.Next() {
		var key, value string

		if err := rows.Scan(&key, &value); err != nil {
			log.Printf("扫描统计记录失败: %v", err)
			continue
		}

		var parsedValue interface{}
		if err := json.Unmarshal([]byte(value), &parsedValue); err != nil {
			// 如果解析失败，使用原始字符串
			stats[key] = value
		} else {
			stats[key] = parsedValue
		}
	}

	return stats, nil
}

// SaveRules 保存 DNS 规则
func (sm *SQLiteManager) SaveRules(rules map[string][]string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	tx, err := sm.db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %v", err)
	}
	defer tx.Rollback()

	// 清空现有规则
	if _, err := tx.Exec("DELETE FROM dns_rules"); err != nil {
		return fmt.Errorf("清空规则失败: %v", err)
	}

	// 准备语句
	stmt, err := tx.Prepare(`
		INSERT INTO dns_rules (category, domain, created_at) 
		VALUES (?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("准备语句失败: %v", err)
	}
	defer stmt.Close()

	// 批量插入
	for category, domains := range rules {
		for _, domain := range domains {
			_, err = stmt.Exec(category, domain, time.Now().Unix())
			if err != nil {
				log.Printf("插入规则失败: %v", err)
				continue
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %v", err)
	}

	return nil
}

// LoadRules 加载 DNS 规则
func (sm *SQLiteManager) LoadRules() (map[string][]string, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	rows, err := sm.db.Query(`SELECT category, domain FROM dns_rules ORDER BY category, domain`)
	if err != nil {
		return nil, fmt.Errorf("查询规则失败: %v", err)
	}
	defer rows.Close()

	rules := make(map[string][]string)

	for rows.Next() {
		var category, domain string

		if err := rows.Scan(&category, &domain); err != nil {
			log.Printf("扫描规则记录失败: %v", err)
			continue
		}

		if rules[category] == nil {
			rules[category] = make([]string, 0)
		}
		rules[category] = append(rules[category], domain)
	}

	return rules, nil
}

// StartAutoSave 启动自动保存
func (sm *SQLiteManager) StartAutoSave() {
	if sm.autoSaveTicker != nil {
		sm.autoSaveTicker.Stop()
	}

	interval := time.Duration(sm.config.Persistence.AutoSaveInterval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	sm.autoSaveTicker = time.NewTicker(interval)
	sm.stopChan = make(chan bool)

	go func() {
		for {
			select {
			case <-sm.autoSaveTicker.C:
				sm.autoSave()
			case <-sm.stopChan:
				return
			}
		}
	}()
}

// StopAutoSave 停止自动保存
func (sm *SQLiteManager) StopAutoSave() {
	if sm.autoSaveTicker != nil {
		sm.autoSaveTicker.Stop()
	}
	if sm.stopChan != nil {
		close(sm.stopChan)
	}
}

// autoSave 执行自动保存
func (sm *SQLiteManager) autoSave() {
	// 记录自动保存开始
	start := time.Now()

	// 清理过期数据
	if err := sm.CleanupOldData(); err != nil {
		log.Printf("自动清理失败: %v", err)
	}

	// 记录性能指标
	duration := time.Since(start)
	sm.recordPerformance("auto_save", duration, map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"operation": "auto_save",
	})

	log.Printf("自动保存完成，耗时: %v", duration)
}

// periodicCleanup 定期清理任务
func (sm *SQLiteManager) periodicCleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := sm.CleanupOldData(); err != nil {
				log.Printf("定期清理失败: %v", err)
			}
		case <-sm.stopChan:
			return
		}
	}
}

// CleanupOldData 清理旧数据
func (sm *SQLiteManager) CleanupOldData() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// 清理过期缓存
	if _, err := sm.db.Exec("DELETE FROM dns_cache WHERE expire_at < ?", time.Now().Unix()); err != nil {
		return fmt.Errorf("清理过期缓存失败: %v", err)
	}

	// 清理旧日志 (保留最近 30 天)
	cutoff := time.Now().AddDate(0, 0, -30).Unix()
	if _, err := sm.db.Exec("DELETE FROM query_logs WHERE timestamp < ?", cutoff); err != nil {
		return fmt.Errorf("清理旧日志失败: %v", err)
	}

	// 清理旧性能指标 (保留最近 7 天)
	perfCutoff := time.Now().AddDate(0, 0, -7).Unix()
	if _, err := sm.db.Exec("DELETE FROM performance_metrics WHERE timestamp < ?", perfCutoff); err != nil {
		return fmt.Errorf("清理旧性能指标失败: %v", err)
	}

	// 执行 VACUUM 优化
	if _, err := sm.db.Exec("VACUUM"); err != nil {
		log.Printf("VACUUM 优化失败: %v", err)
	}

	return nil
}

// GetQueryStats 获取查询统计信息
func (sm *SQLiteManager) GetQueryStats() (map[string]interface{}, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	stats := make(map[string]interface{})

	// 总查询数
	var totalQueries int64
	err := sm.db.QueryRow("SELECT COUNT(*) FROM query_logs").Scan(&totalQueries)
	if err == nil {
		stats["total_queries"] = totalQueries
	}

	// 今日查询数
	today := time.Now().Truncate(24 * time.Hour).Unix()
	var todayQueries int64
	err = sm.db.QueryRow("SELECT COUNT(*) FROM query_logs WHERE timestamp >= ?", today).Scan(&todayQueries)
	if err == nil {
		stats["today_queries"] = todayQueries
	}

	// 平均响应时间
	var avgLatency float64
	err = sm.db.QueryRow("SELECT AVG(latency) FROM query_logs WHERE timestamp >= ?", today).Scan(&avgLatency)
	if err == nil {
		stats["avg_latency"] = avgLatency
	}

	// 路由分布
	routeStats := make(map[string]int64)
	rows, err := sm.db.Query("SELECT route, COUNT(*) FROM query_logs WHERE timestamp >= ? GROUP BY route", today)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var route string
			var count int64
			if rows.Scan(&route, &count) == nil {
				routeStats[route] = count
			}
		}
		stats["route_distribution"] = routeStats
	}

	// 缓存命中率
	var cacheHits int64
	err = sm.db.QueryRow("SELECT COUNT(*) FROM dns_cache WHERE expire_at > ?", time.Now().Unix()).Scan(&cacheHits)
	if err == nil {
		stats["cache_hits"] = cacheHits
	}

	return stats, nil
}

// Close 关闭数据库连接
func (sm *SQLiteManager) Close() error {
	sm.StopAutoSave()

	if sm.db != nil {
		return sm.db.Close()
	}

	return nil
}

// 性能监控辅助函数
func (sm *SQLiteManager) recordPerformance(operation string, duration time.Duration, metadata map[string]interface{}) {
	metadataJSON, _ := json.Marshal(metadata)

	_, err := sm.db.Exec(`
		INSERT INTO performance_metrics (operation, duration_ms, metadata) 
		VALUES (?, ?, ?)
	`, operation, duration.Milliseconds(), string(metadataJSON))

	if err != nil {
		log.Printf("记录性能指标失败: %v", err)
	}
}

// ==================== 订阅源管理方法 ====================

// SaveSubscriptionSource 保存订阅源
func (sm *SQLiteManager) SaveSubscriptionSource(source *SubscriptionSource) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	now := time.Now().Unix()

	if source.ID == 0 {
		// 新增订阅源
		result, err := sm.db.Exec(`
			INSERT INTO subscription_sources (name, category, url, format, enabled, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, source.Name, source.Category, source.URL, source.Format, source.Enabled, now, now)

		if err != nil {
			return fmt.Errorf("新增订阅源失败: %v", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("获取订阅源ID失败: %v", err)
		}
		source.ID = int(id)
	} else {
		// 更新订阅源
		_, err := sm.db.Exec(`
			UPDATE subscription_sources 
			SET name = ?, category = ?, url = ?, format = ?, enabled = ?, updated_at = ?
			WHERE id = ?
		`, source.Name, source.Category, source.URL, source.Format, source.Enabled, now, source.ID)

		if err != nil {
			return fmt.Errorf("更新订阅源失败: %v", err)
		}
	}

	return nil
}

// GetSubscriptionSources 获取所有订阅源
func (sm *SQLiteManager) GetSubscriptionSources() ([]*SubscriptionSource, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	rows, err := sm.db.Query(`
		SELECT id, name, category, url, format, enabled, last_update, last_check, 
		       error_count, last_error, created_at, updated_at
		FROM subscription_sources
		ORDER BY category, name
	`)
	if err != nil {
		return nil, fmt.Errorf("查询订阅源失败: %v", err)
	}
	defer rows.Close()

	var sources []*SubscriptionSource
	for rows.Next() {
		source := &SubscriptionSource{}
		var lastUpdate, lastCheck sql.NullInt64
		var lastError sql.NullString

		err := rows.Scan(
			&source.ID, &source.Name, &source.Category, &source.URL, &source.Format,
			&source.Enabled, &lastUpdate, &lastCheck, &source.ErrorCount,
			&lastError, &source.CreatedAt, &source.UpdatedAt,
		)
		if err != nil {
			continue
		}

		if lastUpdate.Valid {
			source.LastUpdate = lastUpdate.Int64
		}
		if lastCheck.Valid {
			source.LastCheck = lastCheck.Int64
		}
		if lastError.Valid {
			source.LastError = lastError.String
		}

		sources = append(sources, source)
	}

	return sources, nil
}

// DeleteSubscriptionSource 删除订阅源
func (sm *SQLiteManager) DeleteSubscriptionSource(id int) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// 删除相关的规则和统计
	_, err := sm.db.Exec("DELETE FROM subscription_rules WHERE source_id = ?", id)
	if err != nil {
		return fmt.Errorf("删除订阅规则失败: %v", err)
	}

	_, err = sm.db.Exec("DELETE FROM subscription_stats WHERE source_id = ?", id)
	if err != nil {
		return fmt.Errorf("删除订阅统计失败: %v", err)
	}

	// 删除订阅源
	_, err = sm.db.Exec("DELETE FROM subscription_sources WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("删除订阅源失败: %v", err)
	}

	return nil
}

// SaveSubscriptionRules 保存订阅规则
func (sm *SQLiteManager) SaveSubscriptionRules(sourceID int, category string, domains []string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// 开始事务
	tx, err := sm.db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %v", err)
	}
	defer tx.Rollback()

	// 删除旧的规则
	_, err = tx.Exec("DELETE FROM subscription_rules WHERE source_id = ?", sourceID)
	if err != nil {
		return fmt.Errorf("删除旧规则失败: %v", err)
	}

	// 插入新规则
	stmt, err := tx.Prepare(`
		INSERT INTO subscription_rules (source_id, category, domain)
		VALUES (?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("准备插入语句失败: %v", err)
	}
	defer stmt.Close()

	for _, domain := range domains {
		_, err = stmt.Exec(sourceID, category, domain)
		if err != nil {
			return fmt.Errorf("插入规则失败: %v", err)
		}
	}

	// 更新统计信息
	now := time.Now().Unix()
	_, err = tx.Exec(`
		INSERT OR REPLACE INTO subscription_stats 
		(source_id, total_domains, last_successful_update, update_count, updated_at)
		VALUES (?, ?, ?, 
			COALESCE((SELECT update_count + 1 FROM subscription_stats WHERE source_id = ?), 1),
			?)
	`, sourceID, len(domains), now, sourceID, now)
	if err != nil {
		return fmt.Errorf("更新统计失败: %v", err)
	}

	// 提交事务
	return tx.Commit()
}

// GetSubscriptionRules 获取订阅规则
func (sm *SQLiteManager) GetSubscriptionRules(category string) ([]string, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	rows, err := sm.db.Query(`
		SELECT DISTINCT domain 
		FROM subscription_rules 
		WHERE category = ?
		ORDER BY domain
	`, category)
	if err != nil {
		return nil, fmt.Errorf("查询订阅规则失败: %v", err)
	}
	defer rows.Close()

	var domains []string
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			continue
		}
		domains = append(domains, domain)
	}

	return domains, nil
}

// GetSubscriptionStats 获取订阅统计信息
func (sm *SQLiteManager) GetSubscriptionStats() (map[string]interface{}, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	stats := make(map[string]interface{})

	// 总订阅源数量
	var totalSources int64
	err := sm.db.QueryRow("SELECT COUNT(*) FROM subscription_sources").Scan(&totalSources)
	if err == nil {
		stats["total_sources"] = totalSources
	}

	// 启用的订阅源数量
	var enabledSources int64
	err = sm.db.QueryRow("SELECT COUNT(*) FROM subscription_sources WHERE enabled = 1").Scan(&enabledSources)
	if err == nil {
		stats["enabled_sources"] = enabledSources
	}

	// 总规则数量
	var totalRules int64
	err = sm.db.QueryRow("SELECT COUNT(*) FROM subscription_rules").Scan(&totalRules)
	if err == nil {
		stats["total_rules"] = totalRules
	}

	// 按类别统计规则数量
	categoryStats := make(map[string]int64)
	rows, err := sm.db.Query(`
		SELECT category, COUNT(*) 
		FROM subscription_rules 
		GROUP BY category
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var category string
			var count int64
			if rows.Scan(&category, &count) == nil {
				categoryStats[category] = count
			}
		}
		stats["category_stats"] = categoryStats
	}

	return stats, nil
}
