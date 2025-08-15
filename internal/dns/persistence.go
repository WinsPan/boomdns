package dns

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PersistenceManager 数据持久化管理器
type PersistenceManager struct {
	config *Config
	mu     sync.RWMutex

	// 数据文件路径
	cacheFile string
	logsFile  string
	statsFile string
	rulesFile string

	// 自动保存定时器
	autoSaveTicker *time.Ticker
	stopChan       chan struct{}
}

// NewPersistenceManager 创建新的持久化管理器
func NewPersistenceManager(config *Config) *PersistenceManager {
	pm := &PersistenceManager{
		config: config,
	}

	// 确保数据目录存在
	dataDir := config.GetDataDir()
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Printf("创建数据目录失败: %v\n", err)
		return pm
	}

	// 设置文件路径
	pm.cacheFile = filepath.Join(dataDir, config.Persistence.Database.CacheFile)
	pm.logsFile = filepath.Join(dataDir, config.Persistence.Database.LogsFile)
	pm.statsFile = filepath.Join(dataDir, config.Persistence.Database.StatsFile)
	pm.rulesFile = filepath.Join(dataDir, config.Persistence.Database.RulesFile)

	// 启动自动保存
	if config.IsPersistenceEnabled() {
		pm.StartAutoSave()
	}

	return pm
}

// StartAutoSave 启动自动保存
func (pm *PersistenceManager) StartAutoSave() {
	interval := pm.config.GetAutoSaveInterval()
	pm.autoSaveTicker = time.NewTicker(interval)
	pm.stopChan = make(chan struct{})

	go func() {
		for {
			select {
			case <-pm.autoSaveTicker.C:
				pm.autoSave()
			case <-pm.stopChan:
				return
			}
		}
	}()

	fmt.Printf("数据持久化已启动，自动保存间隔: %v\n", interval)
}

// StopAutoSave 停止自动保存
func (pm *PersistenceManager) StopAutoSave() {
	if pm.autoSaveTicker != nil {
		pm.autoSaveTicker.Stop()
	}
	if pm.stopChan != nil {
		close(pm.stopChan)
	}
}

// SaveCache 保存缓存数据
func (pm *PersistenceManager) SaveCache(cache map[string]*CacheEntry) error {
	if !pm.config.IsPersistenceEnabled() {
		return nil
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	data := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"entries":   cache,
	}

	return pm.saveToFile(pm.cacheFile, data)
}

// LoadCache 加载缓存数据
func (pm *PersistenceManager) LoadCache() (map[string]*CacheEntry, error) {
	if !pm.config.IsPersistenceEnabled() {
		return make(map[string]*CacheEntry), nil
	}

	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var data struct {
		Timestamp int64                  `json:"timestamp"`
		Entries   map[string]*CacheEntry `json:"entries"`
	}

	if err := pm.loadFromFile(pm.cacheFile, &data); err != nil {
		return make(map[string]*CacheEntry), nil
	}

	// 清理过期缓存
	now := time.Now()
	validEntries := make(map[string]*CacheEntry)
	for key, entry := range data.Entries {
		if entry.ExpireAt.After(now) {
			validEntries[key] = entry
		}
	}

	fmt.Printf("从文件加载缓存: %d 个有效条目\n", len(validEntries))
	return validEntries, nil
}

// SaveLogs 保存查询日志
func (pm *PersistenceManager) SaveLogs(logs []QueryLog) error {
	if !pm.config.IsPersistenceEnabled() {
		return nil
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 限制日志数量
	maxLogs := pm.config.GetMaxLogs()
	if len(logs) > maxLogs {
		logs = logs[len(logs)-maxLogs:]
	}

	data := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"count":     len(logs),
		"logs":      logs,
	}

	return pm.saveToFile(pm.logsFile, data)
}

// LoadLogs 加载查询日志
func (pm *PersistenceManager) LoadLogs() ([]QueryLog, error) {
	if !pm.config.IsPersistenceEnabled() {
		return make([]QueryLog, 0), nil
	}

	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var data struct {
		Timestamp int64      `json:"timestamp"`
		Count     int        `json:"count"`
		Logs      []QueryLog `json:"logs"`
	}

	if err := pm.loadFromFile(pm.logsFile, &data); err != nil {
		return make([]QueryLog, 0), nil
	}

	fmt.Printf("从文件加载日志: %d 条\n", len(data.Logs))
	return data.Logs, nil
}

// SaveStats 保存统计信息
func (pm *PersistenceManager) SaveStats(stats map[string]interface{}) error {
	if !pm.config.IsPersistenceEnabled() {
		return nil
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	data := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"stats":     stats,
	}

	return pm.saveToFile(pm.statsFile, data)
}

// LoadStats 加载统计信息
func (pm *PersistenceManager) LoadStats() (map[string]interface{}, error) {
	if !pm.config.IsPersistenceEnabled() {
		return make(map[string]interface{}), nil
	}

	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var data struct {
		Timestamp int64                  `json:"timestamp"`
		Stats     map[string]interface{} `json:"stats"`
	}

	if err := pm.loadFromFile(pm.statsFile, &data); err != nil {
		return make(map[string]interface{}), nil
	}

	fmt.Printf("从文件加载统计信息\n")
	return data.Stats, nil
}

// SaveRules 保存规则数据
func (pm *PersistenceManager) SaveRules(rules map[string][]string) error {
	if !pm.config.IsPersistenceEnabled() {
		return nil
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	data := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"rules":     rules,
	}

	return pm.saveToFile(pm.rulesFile, data)
}

// LoadRules 加载规则数据
func (pm *PersistenceManager) LoadRules() (map[string][]string, error) {
	if !pm.config.IsPersistenceEnabled() {
		return make(map[string][]string), nil
	}

	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var data struct {
		Timestamp int64               `json:"timestamp"`
		Rules     map[string][]string `json:"rules"`
	}

	if err := pm.loadFromFile(pm.rulesFile, &data); err != nil {
		return make(map[string][]string), nil
	}

	fmt.Printf("从文件加载规则数据\n")
	return data.Rules, nil
}

// autoSave 自动保存所有数据
func (pm *PersistenceManager) autoSave() {
	fmt.Println("执行自动保存...")

	// 这里需要从Server获取数据，暂时跳过
	// 实际实现中应该通过接口或回调获取数据
}

// saveToFile 保存数据到文件
func (pm *PersistenceManager) saveToFile(filename string, data interface{}) error {
	// 创建临时文件
	tempFile := filename + ".tmp"

	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer file.Close()

	// 编码数据
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("编码数据失败: %v", err)
	}

	// 原子性重命名
	if err := os.Rename(tempFile, filename); err != nil {
		return fmt.Errorf("重命名文件失败: %v", err)
	}

	return nil
}

// loadFromFile 从文件加载数据
func (pm *PersistenceManager) loadFromFile(filename string, data interface{}) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(data)
}

// Close 关闭持久化管理器
func (pm *PersistenceManager) Close() error {
	pm.StopAutoSave()
	return nil
}

// CleanupOldData 清理旧数据（文件存储版本）
func (pm *PersistenceManager) CleanupOldData() error {
	// 文件存储不需要特殊清理
	return nil
}

// GetQueryStats 获取查询统计信息（文件存储版本）
func (pm *PersistenceManager) GetQueryStats() (map[string]interface{}, error) {
	// 文件存储暂时返回空统计
	return make(map[string]interface{}), nil
}
