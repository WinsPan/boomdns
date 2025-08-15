package dns

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)



// SubscriptionConfig 订阅配置
type SubscriptionConfig struct {
	Enabled        bool                              `yaml:"enabled"`
	UpdateInterval int                               `yaml:"update_interval"`
	Timeout        int                               `yaml:"timeout"`
	RetryCount     int                               `yaml:"retry_count"`
	UserAgent      string                            `yaml:"user_agent"`
	Sources        map[string][]SubscriptionRuleSource `yaml:"sources"`
}

// SubscriptionManager 规则订阅管理器
type SubscriptionManager struct {
	config     *SubscriptionConfig
	httpClient *http.Client
	cacheDir   string
	mu         sync.RWMutex
	
	// SQLite 存储管理器
	storage StorageManager
	
	// 规则缓存
	rulesCache map[string]map[string][]string // category -> source -> domains
	lastUpdate map[string]time.Time           // source -> last update time
	checksums  map[string]string             // source -> content checksum
}

// NewSubscriptionManager 创建新的订阅管理器
func NewSubscriptionManager(config *SubscriptionConfig, cacheDir string, storage StorageManager) *SubscriptionManager {
	if config == nil {
		config = &SubscriptionConfig{
			Enabled:        true,
			UpdateInterval: 3600,
			Timeout:        300,
			RetryCount:     3,
			UserAgent:      "BoomDNS/1.0",
		}
	}

	// 创建 HTTP 客户端
	httpClient := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
	}

	sm := &SubscriptionManager{
		config:     config,
		httpClient: httpClient,
		cacheDir:   cacheDir,
		storage:    storage,
		rulesCache: make(map[string]map[string][]string),
		lastUpdate: make(map[string]time.Time),
		checksums:  make(map[string]string),
	}

	// 创建缓存目录
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		log.Printf("创建订阅缓存目录失败: %v", err)
	}

	return sm
}

// Start 启动订阅管理器
func (sm *SubscriptionManager) Start() {
	if !sm.config.Enabled {
		log.Println("规则订阅功能已禁用")
		return
	}

	log.Println("启动规则订阅管理器...")
	
	// 立即执行一次更新
	go sm.updateAllRules()
	
	// 启动定时更新
	ticker := time.NewTicker(time.Duration(sm.config.UpdateInterval) * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		go sm.updateAllRules()
	}
}

// updateAllRules 更新所有规则
func (sm *SubscriptionManager) updateAllRules() {
	log.Println("开始更新规则订阅...")
	
	var wg sync.WaitGroup
	for category, sources := range sm.config.Sources {
		for _, source := range sources {
			if !source.Enabled {
				continue
			}
			
			wg.Add(1)
			go func(cat, srcName, srcURL, srcFormat string) {
				defer wg.Done()
				sm.updateRuleSource(cat, srcName, srcURL, srcFormat)
			}(category, source.Name, source.URL, source.Format)
		}
	}
	
	wg.Wait()
	log.Println("规则订阅更新完成")
}

// updateRuleSource 更新单个规则源
func (sm *SubscriptionManager) updateRuleSource(category, name, url, format string) {
	sourceKey := fmt.Sprintf("%s:%s", category, name)
	
	// 检查是否需要更新
	if sm.shouldSkipUpdate(sourceKey, url) {
		return
	}
	
	log.Printf("更新规则源: %s (%s)", name, category)
	
	// 下载规则
	content, err := sm.downloadRule(url)
	if err != nil {
		log.Printf("下载规则失败 %s: %v", name, err)
		return
	}
	
	// 解析规则
	domains, err := sm.parseRule(content, format)
	if err != nil {
		log.Printf("解析规则失败 %s: %v", name, err)
		return
	}
	
	// 更新缓存
	sm.mu.Lock()
	if sm.rulesCache[category] == nil {
		sm.rulesCache[category] = make(map[string][]string)
	}
	sm.rulesCache[category][name] = domains
	sm.lastUpdate[sourceKey] = time.Now()
	sm.checksums[sourceKey] = sm.calculateChecksum(content)
	sm.mu.Unlock()
	
	// 保存到 SQLite
	if sm.storage != nil {
		// 这里需要先获取或创建订阅源记录
		// 暂时保存到文件作为备份
		sm.saveRuleToFile(category, name, domains)
	}
	
	log.Printf("规则源 %s 更新成功，共 %d 个域名", name, len(domains))
}

// shouldSkipUpdate 检查是否应该跳过更新
func (sm *SubscriptionManager) shouldSkipUpdate(sourceKey, url string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	lastUpdate, exists := sm.lastUpdate[sourceKey]
	if !exists {
		return false
	}
	
	// 检查更新间隔
	if time.Since(lastUpdate) < time.Duration(sm.config.UpdateInterval)*time.Second {
		return true
	}
	
	// 检查内容是否变化
	content, err := sm.downloadRule(url)
	if err != nil {
		return true // 下载失败时跳过
	}
	
	newChecksum := sm.calculateChecksum(content)
	return sm.checksums[sourceKey] == newChecksum
}

// downloadRule 下载规则文件
func (sm *SubscriptionManager) downloadRule(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	
	req.Header.Set("User-Agent", sm.config.UserAgent)
	
	resp, err := sm.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}
	
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	return string(content), nil
}

// parseRule 解析不同格式的规则文件
func (sm *SubscriptionManager) parseRule(content, format string) ([]string, error) {
	switch format {
	case "dnsmasq":
		return sm.parseDNSMasq(content)
	case "gfwlist":
		return sm.parseGFWList(content)
	case "hosts":
		return sm.parseHosts(content)
	case "adguard":
		return sm.parseAdGuard(content)
	case "plain":
		return sm.parsePlain(content)
	default:
		return nil, fmt.Errorf("不支持的规则格式: %s", format)
	}
}

// parseDNSMasq 解析 dnsmasq 格式
func (sm *SubscriptionManager) parseDNSMasq(content string) ([]string, error) {
	var domains []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// 格式: server=/example.com/1.1.1.1
		if strings.HasPrefix(line, "server=/") {
			parts := strings.Split(line, "/")
			if len(parts) >= 3 {
				domain := parts[1]
				if domain != "" && domain != "#" {
					domains = append(domains, domain)
				}
			}
		}
	}
	
	return domains, scanner.Err()
}

// parseGFWList 解析 GFWList 格式
func (sm *SubscriptionManager) parseGFWList(content string) ([]string, error) {
	var domains []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "!") || strings.HasPrefix(line, "[") {
			continue
		}
		
		// 跳过注释和特殊规则
		if strings.Contains(line, "|") || strings.Contains(line, "@") {
			continue
		}
		
		// 提取域名
		if strings.HasPrefix(line, "||") {
			domain := strings.TrimPrefix(line, "||")
			if domain != "" {
				domains = append(domains, domain)
			}
		} else if strings.HasPrefix(line, "|") {
			domain := strings.TrimPrefix(line, "|")
			if domain != "" {
				domains = append(domains, domain)
			}
		} else if !strings.Contains(line, "/") && !strings.Contains(line, "*") {
			// 简单域名
			domains = append(domains, line)
		}
	}
	
	return domains, scanner.Err()
}

// parseHosts 解析 hosts 格式
func (sm *SubscriptionManager) parseHosts(content string) ([]string, error) {
	var domains []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// 格式: 127.0.0.1 example.com
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			domain := fields[1]
			if domain != "" && !strings.Contains(domain, "#") {
				domains = append(domains, domain)
			}
		}
	}
	
	return domains, scanner.Err()
}

// parseAdGuard 解析 AdGuard 格式
func (sm *SubscriptionManager) parseAdGuard(content string) ([]string, error) {
	var domains []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "!") || strings.HasPrefix(line, "[") {
			continue
		}
		
		// 跳过注释和特殊规则
		if strings.Contains(line, "|") || strings.Contains(line, "@") {
			continue
		}
		
		// 提取域名
		if strings.HasPrefix(line, "||") {
			domain := strings.TrimPrefix(line, "||")
			if domain != "" {
				domains = append(domains, domain)
			}
		} else if !strings.Contains(line, "/") && !strings.Contains(line, "*") {
			// 简单域名
			domains = append(domains, line)
		}
	}
	
	return domains, scanner.Err()
}

// parsePlain 解析纯文本格式
func (sm *SubscriptionManager) parsePlain(content string) ([]string, error) {
	var domains []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// 简单的域名验证
		if sm.isValidDomain(line) {
			domains = append(domains, line)
		}
	}
	
	return domains, scanner.Err()
}

// isValidDomain 验证域名格式
func (sm *SubscriptionManager) isValidDomain(domain string) bool {
	// 简单的域名格式验证
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	return domainRegex.MatchString(domain)
}

// calculateChecksum 计算内容校验和
func (sm *SubscriptionManager) calculateChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// saveRuleToFile 保存规则到文件
func (sm *SubscriptionManager) saveRuleToFile(category, name string, domains []string) {
	filename := filepath.Join(sm.cacheDir, fmt.Sprintf("%s_%s.txt", category, name))
	
	file, err := os.Create(filename)
	if err != nil {
		log.Printf("创建规则文件失败 %s: %v", filename, err)
		return
	}
	defer file.Close()
	
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	
	// 写入头部信息
	fmt.Fprintf(writer, "# BoomDNS 规则文件\n")
	fmt.Fprintf(writer, "# 类别: %s\n", category)
	fmt.Fprintf(writer, "# 来源: %s\n", name)
	fmt.Fprintf(writer, "# 更新时间: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(writer, "# 域名数量: %d\n", len(domains))
	fmt.Fprintf(writer, "\n")
	
	// 写入域名列表
	for _, domain := range domains {
		fmt.Fprintln(writer, domain)
	}
}

// GetRules 获取指定类别的所有规则
func (sm *SubscriptionManager) GetRules(category string) []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	var allDomains []string
	seen := make(map[string]bool)
	
	if sources, exists := sm.rulesCache[category]; exists {
		for _, domains := range sources {
			for _, domain := range domains {
				if !seen[domain] {
					allDomains = append(allDomains, domain)
					seen[domain] = true
				}
			}
		}
	}
	
	return allDomains
}

// GetRuleStats 获取规则统计信息
func (sm *SubscriptionManager) GetRuleStats() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	stats := make(map[string]interface{})
	
	for category, sources := range sm.rulesCache {
		categoryStats := make(map[string]interface{})
		totalDomains := 0
		
		for sourceName, domains := range sources {
			sourceStats := map[string]interface{}{
				"domains":     len(domains),
				"last_update": sm.lastUpdate[fmt.Sprintf("%s:%s", category, sourceName)],
			}
			categoryStats[sourceName] = sourceStats
			totalDomains += len(domains)
		}
		
		categoryStats["total_domains"] = totalDomains
		stats[category] = categoryStats
	}
	
	return stats
}

// ForceUpdate 强制更新所有规则
func (sm *SubscriptionManager) ForceUpdate() {
	log.Println("强制更新所有规则...")
	
	// 清除更新时间缓存，强制重新下载
	sm.mu.Lock()
	for key := range sm.lastUpdate {
		delete(sm.lastUpdate, key)
	}
	sm.mu.Unlock()
	
	// 执行更新
	go sm.updateAllRules()
}

// Stop 停止订阅管理器
func (sm *SubscriptionManager) Stop() {
	log.Println("停止规则订阅管理器...")
}
