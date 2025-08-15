package dns

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type SyncManager struct {
	cfg    *Config
	server *Server
	httpc  *http.Client

	// 同步状态跟踪
	mu        sync.RWMutex
	lastSync  time.Time
	syncStats struct {
		totalSyncs      int64
		successfulSyncs int64
		failedSyncs     int64
		lastError       string
	}

	// 规则来源统计
	ruleSources map[string]*RuleSource
}

// RuleSource 规则来源信息
type RuleSource struct {
	URL          string        `json:"url"`
	Type         string        `json:"type"` // gfwlist, chinalist, adlist
	LastSync     time.Time     `json:"last_sync"`
	LastSuccess  time.Time     `json:"last_success"`
	LastError    string        `json:"last_error"`
	RuleCount    int           `json:"rule_count"`
	Status       string        `json:"status"` // success, error, pending
	ResponseTime time.Duration `json:"response_time"`
}

func NewSyncManager(cfg *Config, server *Server) *SyncManager {
	sm := &SyncManager{
		cfg:         cfg,
		server:      server,
		httpc:       &http.Client{Timeout: 15 * time.Second},
		ruleSources: make(map[string]*RuleSource),
	}

	// 初始化规则来源
	sm.initRuleSources()

	return sm
}

// initRuleSources 初始化规则来源
func (m *SyncManager) initRuleSources() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 初始化规则来源
	if m.cfg.Sync.Enabled && m.cfg.Sync.Sources != nil {
		// 初始化GFW列表来源
		if url, ok := m.cfg.Sync.Sources["gfw"]; ok && url != "" {
			key := "gfwlist:" + url
			m.ruleSources[key] = &RuleSource{
				URL:       url,
				Type:      "gfwlist",
				Status:    "pending",
				RuleCount: 0,
			}
		}

		// 初始化中国列表来源
		if url, ok := m.cfg.Sync.Sources["china"]; ok && url != "" {
			key := "chinalist:" + url
			m.ruleSources[key] = &RuleSource{
				URL:       url,
				Type:      "chinalist",
				Status:    "pending",
				RuleCount: 0,
			}
		}

		// 初始化广告列表来源
		if url, ok := m.cfg.Sync.Sources["ads"]; ok && url != "" {
			key := "adlist:" + url
			m.ruleSources[key] = &RuleSource{
				URL:       url,
				Type:      "adlist",
				Status:    "pending",
				RuleCount: 0,
			}
		}
	}
}

// GetSyncStatus 获取同步状态
func (m *SyncManager) GetSyncStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 计算成功率
	var successRate float64
	if m.syncStats.totalSyncs > 0 {
		successRate = float64(m.syncStats.successfulSyncs) / float64(m.syncStats.totalSyncs) * 100
	}

	return map[string]interface{}{
		"last_sync":        m.lastSync,
		"total_syncs":      m.syncStats.totalSyncs,
		"successful_syncs": m.syncStats.successfulSyncs,
		"failed_syncs":     m.syncStats.failedSyncs,
		"success_rate":     successRate,
		"last_error":       m.syncStats.lastError,
		"rule_sources":     m.ruleSources,
	}
}

// GetRuleCounts 获取各类规则数量
func (m *SyncManager) GetRuleCounts() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	counts := make(map[string]interface{})

	// 统计各类规则数量
	gfwCount := 0
	chinaCount := 0
	adCount := 0

	for _, source := range m.ruleSources {
		switch source.Type {
		case "gfwlist":
			gfwCount += source.RuleCount
		case "chinalist":
			chinaCount += source.RuleCount
		case "adlist":
			adCount += source.RuleCount
		}
	}

	counts["gfw_rules"] = gfwCount
	counts["china_rules"] = chinaCount
	counts["ad_rules"] = adCount
	counts["total_rules"] = gfwCount + chinaCount + adCount

	return counts
}

// updateSourceStatus 更新规则来源状态
func (m *SyncManager) updateSourceStatus(key, status, error string, ruleCount int, responseTime time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if source, exists := m.ruleSources[key]; exists {
		source.Status = status
		source.LastSync = time.Now()
		source.ResponseTime = responseTime

		if status == "success" {
			source.LastSuccess = time.Now()
			source.LastError = ""
			source.RuleCount = ruleCount
		} else {
			source.LastError = error
		}
	}
}

func (m *SyncManager) Start(ctx context.Context) {
	// 使用配置的同步间隔
	iv := m.cfg.GetSyncInterval()
	ticker := time.NewTicker(iv)
	defer ticker.Stop()

	// 初始同步
	_ = m.SyncNow(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = m.SyncNow(ctx)
		}
	}
}

func (m *SyncManager) SyncNow(ctx context.Context) error {
	m.mu.Lock()
	m.lastSync = time.Now()
	m.syncStats.totalSyncs++
	m.mu.Unlock()

	var chinaSet = map[string]struct{}{}
	var gfwSet = map[string]struct{}{}
	var adSet = map[string]struct{}{}

	// 跟踪同步结果
	success := true
	lastError := ""

	// 检查是否启用同步
	if !m.cfg.Sync.Enabled || m.cfg.Sync.Sources == nil {
		return nil
	}

	// china lists (dnsmasq format or plain domains)
	if url, ok := m.cfg.Sync.Sources["china"]; ok && strings.TrimSpace(url) != "" {
		start := time.Now()
		b, err := m.fetch(ctx, url)
		responseTime := time.Since(start)

		if err != nil {
			success = false
			lastError = err.Error()
			m.updateSourceStatus("chinalist:"+url, "error", err.Error(), 0, responseTime)
		} else {
			domains := parseDomainsFromChinaList(string(b))
			for d := range domains {
				chinaSet[d] = struct{}{}
			}
			m.updateSourceStatus("chinalist:"+url, "success", "", len(domains), responseTime)
		}
	}

	// gfwlist (base64-encoded rules)
	if url, ok := m.cfg.Sync.Sources["gfw"]; ok && strings.TrimSpace(url) != "" {
		start := time.Now()
		b64, err := m.fetch(ctx, url)
		responseTime := time.Since(start)

		if err != nil {
			success = false
			lastError = err.Error()
			m.updateSourceStatus("gfwlist:"+url, "error", err.Error(), 0, responseTime)
		} else {
			raw, err := base64.StdEncoding.DecodeString(string(b64))
			if err != nil {
				success = false
				lastError = err.Error()
				m.updateSourceStatus("gfwlist:"+url, "error", err.Error(), 0, responseTime)
			} else {
				domains := parseDomainsFromGFWList(string(raw))
				for d := range domains {
					gfwSet[d] = struct{}{}
				}
				m.updateSourceStatus("gfwlist:"+url, "success", "", len(domains), responseTime)
			}
		}
	}

	// ad lists (hosts/address or plain domains)
	if url, ok := m.cfg.Sync.Sources["ads"]; ok && strings.TrimSpace(url) != "" {
		start := time.Now()
		b, err := m.fetch(ctx, url)
		responseTime := time.Since(start)

		if err != nil {
			success = false
			lastError = err.Error()
			m.updateSourceStatus("adlist:"+url, "error", err.Error(), 0, responseTime)
		} else {
			domains := parseDomainsGeneric(string(b))
			for d := range domains {
				adSet[d] = struct{}{}
			}
			m.updateSourceStatus("adlist:"+url, "success", "", len(domains), responseTime)
		}
	}

	// convert to sorted slices
	china := setToSlice(chinaSet)
	gfw := setToSlice(gfwSet)
	ads := setToSlice(adSet)

	if len(china) == 0 && len(gfw) == 0 && len(ads) == 0 {
		success = false
		lastError = "rule sync got empty sets"
	} else {
		// update server rules atomically
		m.server.SetRules(china, gfw, ads)
	}

	// 更新同步统计
	m.mu.Lock()
	if success {
		m.syncStats.successfulSyncs++
	} else {
		m.syncStats.failedSyncs++
		m.syncStats.lastError = lastError
	}
	m.mu.Unlock()

	if !success {
		return errors.New(lastError)
	}
	return nil
}

func (m *SyncManager) fetch(ctx context.Context, url string) ([]byte, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := m.httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, errors.New(resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func setToSlice(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for d := range set {
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}

// parsers

var reDomain = regexp.MustCompile(`(?i)([a-z0-9][a-z0-9-]*\.)+[a-z]{2,}`)

func normalizeDomain(d string) (string, bool) {
	d = strings.ToLower(strings.TrimSpace(d))
	if d == "" {
		return "", false
	}
	// strip leading dots and scheme patterns
	d = strings.TrimPrefix(d, ".")
	d = strings.TrimPrefix(d, "||")
	d = strings.TrimPrefix(d, "|")
	d = strings.TrimPrefix(d, "@")
	if !reDomain.MatchString(d) {
		m := reDomain.FindString(d)
		if m == "" {
			return "", false
		}
		d = m
	}
	return "." + d, true
}

func parseDomainsFromChinaList(s string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// dnsmasq: server=/example.com/114.114.114.114
		if strings.HasPrefix(line, "server=/") {
			seg := strings.TrimPrefix(line, "server=/")
			idx := strings.Index(seg, "/")
			if idx > 0 {
				seg = seg[:idx]
			}
			if d, ok := normalizeDomain(seg); ok {
				out[d] = struct{}{}
			}
			continue
		}
		if d, ok := normalizeDomain(line); ok {
			out[d] = struct{}{}
		}
	}
	return out
}

func parseDomainsFromGFWList(s string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "!") || strings.HasPrefix(line, "@@") {
			continue
		}
		line = strings.TrimPrefix(line, "||")
		line = strings.TrimPrefix(line, "|")
		line = strings.TrimPrefix(line, ".")
		if d, ok := normalizeDomain(line); ok {
			out[d] = struct{}{}
		}
	}
	return out
}

func parseDomainsGeneric(s string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// hosts: 0.0.0.0 domain
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			if d, ok := normalizeDomain(fields[1]); ok {
				out[d] = struct{}{}
			}
			continue
		}
		// address=/domain/
		if strings.HasPrefix(line, "address=/") {
			seg := strings.TrimPrefix(line, "address=/")
			idx := strings.Index(seg, "/")
			if idx > 0 {
				seg = seg[:idx]
			}
			if d, ok := normalizeDomain(seg); ok {
				out[d] = struct{}{}
			}
			continue
		}
		if d, ok := normalizeDomain(line); ok {
			out[d] = struct{}{}
		}
	}
	return out
}
