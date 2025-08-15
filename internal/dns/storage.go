package dns

// 移除未使用的导入

// StorageManager 数据存储管理器接口
type StorageManager interface {
	// 缓存管理
	SaveCache(cache map[string]*CacheEntry) error
	LoadCache() (map[string]*CacheEntry, error)
	
	// 日志管理
	SaveLogs(logs []QueryLog) error
	LoadLogs() ([]QueryLog, error)
	
	// 统计信息管理
	SaveStats(stats map[string]interface{}) error
	LoadStats() (map[string]interface{}, error)
	
	// 规则管理
	SaveRules(rules map[string][]string) error
	LoadRules() (map[string][]string, error)
	
	// 生命周期管理
	StartAutoSave()
	StopAutoSave()
	Close() error
	
	// 数据清理
	CleanupOldData() error
	
	// 统计查询
	GetQueryStats() (map[string]interface{}, error)
}

// NewStorageManager 根据配置创建相应的存储管理器
func NewStorageManager(config *Config) (StorageManager, error) {
	if config.IsSQLiteEnabled() {
		return NewSQLiteManager(config)
	}
	
	// 回退到文件存储
	return NewPersistenceManager(config), nil
}
