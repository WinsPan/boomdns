package dns

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"time"

	mdns "github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus"
)

// CacheEntry DNS缓存条目
type CacheEntry struct {
	Response *mdns.Msg
	ExpireAt time.Time
	Hits     int64 // 命中次数
}

// Server DNS服务器
type Server struct {
	cfg *Config
	mu  sync.RWMutex
	// 最近查询日志（环形缓冲）
	logs []QueryLog

	// 预编译的规则列表（已标准化为小写、带或不带前导点的一致形式）
	compiledChina []string
	compiledGfw   []string
	compiledAds   []string

	// 上游健康状态（简单熔断）
	healthMu       sync.Mutex
	upstreamHealth map[string]*healthState

	// DNS缓存
	cacheMu    sync.RWMutex
	cache      map[string]*CacheEntry
	cacheStats struct {
		hits   int64
		misses int64
		size   int64
	}

	// 延迟统计相关字段
	latencyStats struct {
		mu           sync.RWMutex
		totalQueries int64
		totalLatency time.Duration
		minLatency   time.Duration
		maxLatency   time.Duration
		avgLatency   time.Duration
		routeStats   map[string]*routeLatencyStats
	}

	// 持久化管理器
	persistence StorageManager

	// 规则订阅管理器
	subscriptionManager *SubscriptionManager

	// 代理管理器
	proxyManager *ProxyManager
}

func NewServer(cfg *Config) (*Server, error) {
	srv := &Server{
		cfg:            cfg,
		upstreamHealth: make(map[string]*healthState),
		cache:          make(map[string]*CacheEntry),
	}

	// 初始化延迟统计
	srv.latencyStats.routeStats = make(map[string]*routeLatencyStats)

	// 初始化持久化管理器
	var err error
	srv.persistence, err = NewStorageManager(cfg)
	if err != nil {
		return nil, fmt.Errorf("初始化存储管理器失败: %v", err)
	}

	// 从持久化存储加载数据
	if cfg.IsPersistenceEnabled() {
		srv.loadPersistedData()
	}

	// 初始化规则订阅管理器
	if cfg.IsSubscriptionsEnabled() {
		subscriptionConfig := &SubscriptionConfig{
			Enabled:        cfg.Subscriptions.Enabled,
			UpdateInterval: cfg.Subscriptions.UpdateInterval,
			Timeout:        cfg.Subscriptions.Timeout,
			RetryCount:     cfg.Subscriptions.RetryCount,
			UserAgent:      cfg.Subscriptions.UserAgent,
			Sources:        cfg.Subscriptions.Sources,
		}
		srv.subscriptionManager = NewSubscriptionManager(subscriptionConfig, filepath.Join(cfg.GetDataDir(), "subscriptions"), srv.persistence)
		go srv.subscriptionManager.Start()
	}

	// 初始化代理管理器
	if cfg.IsProxyEnabled() {
		proxyConfig := &ProxyConfig{
			Enabled:         cfg.Proxy.Enabled,
			ListenHTTP:      cfg.GetProxyListenHTTP(),
			ListenSOCKS:     cfg.GetProxyListenSOCKS(),
			DefaultStrategy: cfg.GetProxyDefaultStrategy(),
			TestInterval:    cfg.GetProxyTestInterval(),
			TestTimeout:     cfg.GetProxyTestTimeout(),
		}
		srv.proxyManager = NewProxyManager(proxyConfig)

		// 从配置文件加载代理节点
		if len(cfg.ProxyNodes) > 0 {
			for _, node := range cfg.ProxyNodes {
				if err := srv.proxyManager.AddNode(&node); err != nil {
					log.Printf("添加代理节点失败: %v", err)
				}
			}
			log.Printf("从配置文件加载了 %d 个代理节点", len(cfg.ProxyNodes))
		}

		// 从配置文件加载代理组
		if len(cfg.ProxyGroups) > 0 {
			for _, group := range cfg.ProxyGroups {
				if err := srv.proxyManager.AddGroup(&group); err != nil {
					log.Printf("添加代理组失败: %v", err)
				}
			}
			log.Printf("从配置文件加载了 %d 个代理组", len(cfg.ProxyGroups))
		}

		// 从配置文件加载代理规则
		if len(cfg.ProxyRules) > 0 {
			for _, rule := range cfg.ProxyRules {
				if err := srv.proxyManager.AddRule(&rule); err != nil {
					log.Printf("添加代理规则失败: %v", err)
				}
			}
			log.Printf("从配置文件加载了 %d 个代理规则", len(cfg.ProxyRules))
		}

		if err := srv.proxyManager.Start(); err != nil {
			log.Printf("启动代理管理器失败: %v", err)
		}
	}

	_ = srv.ReloadRules()

	// 启动缓存清理协程
	go srv.cacheCleaner()

	// HTTP API 服务器由 admin 包处理

	return srv, nil
}

// loadPersistedData 从持久化存储加载数据
func (s *Server) loadPersistedData() {
	// 加载缓存数据
	if cache, err := s.persistence.LoadCache(); err == nil {
		s.cacheMu.Lock()
		s.cache = cache
		s.cacheMu.Unlock()
		fmt.Printf("从持久化存储加载缓存: %d 个条目\n", len(cache))
	}

	// 加载查询日志
	if logs, err := s.persistence.LoadLogs(); err == nil {
		s.mu.Lock()
		s.logs = logs
		s.mu.Unlock()
		fmt.Printf("从持久化存储加载日志: %d 条\n", len(logs))
	}

	// 加载统计信息
	if _, err := s.persistence.LoadStats(); err == nil {
		// 这里可以恢复延迟统计等数据
		fmt.Printf("从持久化存储加载统计信息\n")
	}

	// 加载规则数据
	if rules, err := s.persistence.LoadRules(); err == nil {
		// 更新规则配置
		if china, ok := rules["china"]; ok {
			s.cfg.Domains.China = china
		}
		if gfw, ok := rules["gfw"]; ok {
			s.cfg.Domains.GFW = gfw
		}
		if ads, ok := rules["ads"]; ok {
			s.cfg.Domains.Ads = ads
		}
		fmt.Printf("从持久化存储加载规则数据\n")
	}
}

// SaveData 保存所有数据到持久化存储
func (s *Server) SaveData() error {
	if s.persistence == nil || !s.cfg.IsPersistenceEnabled() {
		return nil
	}

	// 保存缓存数据
	if err := s.persistence.SaveCache(s.cache); err != nil {
		fmt.Printf("保存缓存数据失败: %v\n", err)
	}

	// 保存查询日志
	if err := s.persistence.SaveLogs(s.logs); err != nil {
		fmt.Printf("保存查询日志失败: %v\n", err)
	}

	// 保存统计信息
	stats := s.GetMetrics()
	if err := s.persistence.SaveStats(stats); err != nil {
		fmt.Printf("保存统计信息失败: %v\n", err)
	}

	// 保存规则数据
	rules := map[string][]string{
		"china": s.cfg.GetChinaDomains(),
		"gfw":   s.cfg.GetGFWDomains(),
		"ads":   s.cfg.GetAdsDomains(),
	}
	if err := s.persistence.SaveRules(rules); err != nil {
		fmt.Printf("保存规则数据失败: %v\n", err)
	}

	fmt.Println("所有数据已保存到持久化存储")
	return nil
}

func (s *Server) ReloadRules() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 加载配置文件中的规则
	chinaDomains := s.cfg.GetChinaDomains()
	gfwDomains := s.cfg.GetGFWDomains()
	adsDomains := s.cfg.GetAdsDomains()

	// 如果启用了订阅，合并订阅规则
	if s.subscriptionManager != nil {
		subscriptionChina := s.subscriptionManager.GetRules("china")
		subscriptionGFW := s.subscriptionManager.GetRules("gfw")
		subscriptionAds := s.subscriptionManager.GetRules("ads")

		// 合并规则，去重
		chinaDomains = mergeAndDeduplicate(chinaDomains, subscriptionChina)
		gfwDomains = mergeAndDeduplicate(gfwDomains, subscriptionGFW)
		adsDomains = mergeAndDeduplicate(adsDomains, subscriptionAds)

		log.Printf("规则重载完成 - 中国: %d (配置: %d, 订阅: %d), GFW: %d (配置: %d, 订阅: %d), 广告: %d (配置: %d, 订阅: %d)",
			len(chinaDomains), len(s.cfg.GetChinaDomains()), len(subscriptionChina),
			len(gfwDomains), len(s.cfg.GetGFWDomains()), len(subscriptionGFW),
			len(adsDomains), len(s.cfg.GetAdsDomains()), len(subscriptionAds))
	}

	s.compiledChina = normalizeSuffixes(chinaDomains)
	s.compiledGfw = normalizeSuffixes(gfwDomains)
	s.compiledAds = normalizeSuffixes(adsDomains)

	return nil
}

// SetRules 原子更新规则（由 SyncManager 或 API 调用）
func (s *Server) SetRules(china, gfw, ads []string) {
	s.mu.Lock()
	// 注意：这里需要更新配置，但新配置结构不支持直接修改
	// 可以考虑添加配置更新方法或使用配置管理器
	s.mu.Unlock()
	_ = s.ReloadRules()
}

func (s *Server) ServeUDP(conn *net.UDPConn) {
	srv := &mdns.Server{Handler: mdns.HandlerFunc(s.handle), PacketConn: conn}
	if err := srv.ActivateAndServe(); err != nil {
		log.Printf("udp serve err: %v", err)
	}
}

func (s *Server) ServeTCP(ln net.Listener) {
	srv := &mdns.Server{Handler: mdns.HandlerFunc(s.handle), Listener: ln}
	if err := srv.ActivateAndServe(); err != nil {
		log.Printf("tcp serve err: %v", err)
	}
}

func (s *Server) handle(w mdns.ResponseWriter, r *mdns.Msg) {
	if len(r.Question) == 0 {
		_ = w.WriteMsg(new(mdns.Msg))
		return
	}
	q := r.Question[0]
	name := strings.TrimSuffix(strings.ToLower(q.Name), ".")
	qtype := mdns.TypeToString[q.Qtype]

	// 首先尝试从缓存获取
	if cachedResp, hit := s.getFromCache(name, qtype); hit {
		s.addLog(name, "cache", 0) // 缓存命中，延迟为0
		queryCounter.WithLabelValues("cache").Inc()
		_ = w.WriteMsg(cachedResp)
		return
	}

	// 缓存未命中，记录统计
	s.cacheMu.Lock()
	s.cacheStats.misses++
	s.cacheMu.Unlock()

	// 分流：广告 -> adguard；gfw -> intl；china -> china；其他：先 china 失败再 intl
	var upstreams []string
	decision := ""
	if s.match(name, s.compiledAds) && len(s.cfg.GetAdguardUpstreams()) > 0 {
		upstreams = s.cfg.GetAdguardUpstreams()
		decision = "adguard"
	} else if s.match(name, s.compiledGfw) {
		upstreams = s.cfg.GetIntlUpstreams()
		decision = "intl"
	} else if s.match(name, s.compiledChina) {
		upstreams = s.cfg.GetChinaUpstreams()
		decision = "china"
	} else {
		// fallback：china -> intl
		startTime := time.Now()
		if resp, err := s.forward(context.Background(), r, s.cfg.GetChinaUpstreams(), "china"); err == nil && hasAnswer(resp) {
			// 计算延迟并更新统计
			latency := time.Since(startTime)
			s.updateLatencyStats("china", latency)

			s.addLog(name, "china", latency)
			queryCounter.WithLabelValues("china").Inc()
			// 缓存响应
			s.setCache(name, qtype, resp)
			_ = w.WriteMsg(resp)
			return
		}
		upstreams = s.cfg.GetIntlUpstreams()
		decision = "intl"
	}

	// 记录开始时间用于计算延迟
	startTime := time.Now()

	resp, err := s.forward(context.Background(), r, upstreams, decision)
	if err != nil {
		s.writeServFail(w, r)
		return
	}

	// 计算延迟并更新统计
	latency := time.Since(startTime)
	s.updateLatencyStats(decision, latency)

	s.addLog(name, decision, latency)
	queryCounter.WithLabelValues(decision).Inc()

	// 缓存响应
	s.setCache(name, qtype, resp)

	_ = w.WriteMsg(resp)
}

func hasAnswer(m *mdns.Msg) bool { return m != nil && (len(m.Answer) > 0 || len(m.Ns) > 0) }

func (s *Server) writeServFail(w mdns.ResponseWriter, req *mdns.Msg) {
	m := new(mdns.Msg)
	m.SetRcode(req, mdns.RcodeServerFailure)
	_ = w.WriteMsg(m)
}

func (s *Server) match(name string, suffixes []string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sfx := range suffixes {
		sfx = strings.ToLower(strings.TrimSpace(sfx))
		if sfx == "" {
			continue
		}
		if strings.HasSuffix(name, strings.TrimPrefix(sfx, ".")) {
			return true
		}
	}
	return false
}

func (s *Server) forward(ctx context.Context, req *mdns.Msg, ups []string, target string) (*mdns.Msg, error) {
	var lastErr error
	for _, addr := range ups {
		netw, endpoint := upstreamDialParams(addr)
		if !s.isUpstreamAvailable(netw, endpoint) {
			upstreamSkippedUnhealthy.WithLabelValues(target).Inc()
			continue
		}
		c := &mdns.Client{Net: netw, Timeout: 3 * time.Second}
		start := time.Now()
		// 这里未直接支持 socks5，建议使用 mihomo 暴露本地 DNS 端口，或在系统层做 socks5 透明转发
		resp, _, err := c.Exchange(req, endpoint)
		upstreamLatency.WithLabelValues(target).Observe(time.Since(start).Seconds())
		if err == nil && resp != nil {
			s.recordSuccess(netw, endpoint)
			return resp, nil
		}
		s.recordFailure(netw, endpoint, target, err)
		upstreamFailures.WithLabelValues(target).Inc()
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("no upstream")
	}
	return nil, lastErr
}

func netProto(addr string) string {
	if strings.HasPrefix(addr, "tls://") || strings.HasPrefix(addr, "https://") {
		return "tcp"
	}
	return "udp"
}

// upstreamDialParams 解析上游地址，支持 tls:// 和 https:// 前缀的 TCP 连接（仅透传到 miekg/dns，不实现 DoH）。
func upstreamDialParams(address string) (network, endpoint string) {
	if strings.HasPrefix(address, "tls://") || strings.HasPrefix(address, "https://") {
		// miekg/dns 使用 "tcp" 网络，endpoint 保留主机:端口
		a := strings.TrimPrefix(strings.TrimPrefix(address, "tls://"), "https://")
		return "tcp", a
	}
	return "udp", address
}

type healthState struct {
	failures     int
	trippedUntil time.Time
}

// 简单熔断：连续失败 N 次后在 M 时间内跳过该上游
const (
	circuitFailThreshold = 3
	circuitOpenDuration  = 30 * time.Second
)

func (s *Server) isUpstreamAvailable(network, endpoint string) bool {
	s.healthMu.Lock()
	defer s.healthMu.Unlock()
	st := s.upstreamHealth[network+"|"+endpoint]
	if st == nil {
		return true
	}
	if time.Now().Before(st.trippedUntil) {
		return false
	}
	return true
}

func (s *Server) recordFailure(network, endpoint, target string, err error) {
	s.healthMu.Lock()
	defer s.healthMu.Unlock()
	key := network + "|" + endpoint
	st := s.upstreamHealth[key]
	if st == nil {
		st = &healthState{}
		s.upstreamHealth[key] = st
	}
	st.failures++
	if st.failures >= circuitFailThreshold {
		st.trippedUntil = time.Now().Add(circuitOpenDuration)
		st.failures = 0
		upstreamCircuitOpened.WithLabelValues(target).Inc()
	}
}

func (s *Server) recordSuccess(network, endpoint string) {
	s.healthMu.Lock()
	defer s.healthMu.Unlock()
	key := network + "|" + endpoint
	if st := s.upstreamHealth[key]; st != nil {
		st.failures = 0
		st.trippedUntil = time.Time{}
	}
}

type QueryLog struct {
	Time    time.Time `json:"time"`
	Name    string    `json:"name"`
	Route   string    `json:"route"`
	Latency int64     `json:"latency"` // 延迟，单位毫秒
}

func (s *Server) addLog(name, route string, latency time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	const max = 1000
	s.logs = append(s.logs, QueryLog{Time: time.Now(), Name: name, Route: route, Latency: latency.Milliseconds()})
	if len(s.logs) > max {
		s.logs = s.logs[len(s.logs)-max:]
	}
}

func (s *Server) GetLogs(limit int) []QueryLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > len(s.logs) {
		limit = len(s.logs)
	}
	out := make([]QueryLog, limit)
	copy(out, s.logs[len(s.logs)-limit:])
	return out
}

// normalizeSuffixes 将规则统一为小写去空白的后缀匹配形式
func normalizeSuffixes(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			continue
		}
		// 与 normalizeDomain 一致，确保有前导点，便于一致匹配
		if !strings.HasPrefix(s, ".") {
			if d, ok := normalizeDomain(s); ok {
				s = d
			} else {
				continue
			}
		}
		out = append(out, s)
	}
	return out
}

var (
	queryCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "boomdns_queries_total",
			Help: "Total DNS queries by route decision",
		},
		[]string{"route"},
	)
	upstreamLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "boomdns_upstream_request_duration_seconds",
			Help:    "DNS upstream request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"target"},
	)
	upstreamFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "boomdns_upstream_failures_total",
			Help: "Total DNS upstream request failures",
		},
		[]string{"target"},
	)
	upstreamCircuitOpened = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "boomdns_upstream_circuit_opened_total",
			Help: "Times of upstream circuit breaker opened",
		},
		[]string{"target"},
	)
	upstreamSkippedUnhealthy = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "boomdns_upstream_skipped_unhealthy_total",
			Help: "Upstream attempts skipped due to temporary unhealthy state",
		},
		[]string{"target"},
	)
)

func init() {
	prometheus.MustRegister(queryCounter, upstreamLatency, upstreamFailures, upstreamCircuitOpened, upstreamSkippedUnhealthy)
}

// cacheCleaner 定期清理过期缓存
func (s *Server) cacheCleaner() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanExpiredCache()

		// 每5分钟保存一次数据
		if time.Now().Minute()%5 == 0 {
			if err := s.SaveData(); err != nil {
				fmt.Printf("自动保存数据失败: %v\n", err)
			}
		}
	}
}

// cleanExpiredCache 清理过期的缓存条目
func (s *Server) cleanExpiredCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	for key, entry := range s.cache {
		if now.After(entry.ExpireAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(s.cache, key)
	}

	s.cacheStats.size = int64(len(s.cache))
}

// generateCacheKey 生成缓存键
func (s *Server) generateCacheKey(qname, qtype string) string {
	return qname + ":" + qtype
}

// getFromCache 从缓存获取DNS响应
func (s *Server) getFromCache(qname, qtype string) (*mdns.Msg, bool) {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	key := s.generateCacheKey(qname, qtype)
	entry, exists := s.cache[key]

	if !exists {
		return nil, false
	}

	// 检查是否过期
	if time.Now().After(entry.ExpireAt) {
		return nil, false
	}

	// 检查响应是否有效
	if entry.Response == nil {
		return nil, false
	}

	// 增加命中次数
	entry.Hits++
	s.cacheStats.hits++

	// 返回缓存的响应副本
	response := entry.Response.Copy()
	return response, true
}

// setCache 设置缓存
func (s *Server) setCache(qname, qtype string, response *mdns.Msg) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	key := s.generateCacheKey(qname, qtype)

	// 计算TTL（使用响应中的最小TTL）
	ttl := time.Duration(300) * time.Second // 默认5分钟
	if len(response.Answer) > 0 {
		if answer, ok := response.Answer[0].(*mdns.A); ok {
			ttl = time.Duration(answer.Hdr.Ttl) * time.Second
		}
	}

	entry := &CacheEntry{
		Response: response.Copy(),
		ExpireAt: time.Now().Add(ttl),
		Hits:     0,
	}

	s.cache[key] = entry
	s.cacheStats.size = int64(len(s.cache))
}

// GetCacheStats 获取缓存统计信息
func (s *Server) GetCacheStats() map[string]interface{} {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	// 计算缓存命中率
	var hitRate float64
	total := s.cacheStats.hits + s.cacheStats.misses
	if total > 0 {
		hitRate = float64(s.cacheStats.hits) / float64(total) * 100
	}

	return map[string]interface{}{
		"hits":     s.cacheStats.hits,
		"misses":   s.cacheStats.misses,
		"hit_rate": hitRate,
		"size":     s.cacheStats.size,
		"entries":  len(s.cache),
	}
}

// GetCacheEntries 获取缓存条目列表
func (s *Server) GetCacheEntries(limit int) []map[string]interface{} {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	entries := make([]map[string]interface{}, 0, len(s.cache))
	now := time.Now()

	for key, entry := range s.cache {
		if len(entries) >= limit {
			break
		}

		// 解析key获取域名和类型
		parts := strings.Split(key, ":")
		qname := parts[0]
		qtype := parts[1]

		entries = append(entries, map[string]interface{}{
			"domain":   qname,
			"type":     qtype,
			"hits":     entry.Hits,
			"expires":  entry.ExpireAt,
			"ttl_left": entry.ExpireAt.Sub(now).String(),
			"expired":  now.After(entry.ExpireAt),
		})
	}

	// 按命中次数排序
	// 这里简化处理，实际可以添加排序逻辑

	return entries
}

// ClearCache 清空缓存
func (s *Server) ClearCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	s.cache = make(map[string]*CacheEntry)
	s.cacheStats.hits = 0
	s.cacheStats.misses = 0
	s.cacheStats.size = 0
}

// GetRules 获取当前规则
func (s *Server) GetRules() map[string][]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string][]string{
		"china": s.compiledChina,
		"gfw":   s.compiledGfw,
		"ads":   s.compiledAds,
	}
}

// GetSyncStatus 获取同步状态
func (s *Server) GetSyncStatus() map[string]interface{} {
	return map[string]interface{}{
		"status": "running",
		"note":   "详细同步状态请通过SyncManager获取",
	}
}

// GetMetrics 获取当前的指标数据
func (s *Server) GetMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// 获取查询统计（从日志中统计）
	s.mu.RLock()
	routeCounts := make(map[string]int)
	total := len(s.logs)

	for _, log := range s.logs {
		routeCounts[log.Route]++
	}
	s.mu.RUnlock()

	metrics["total_queries"] = total
	metrics["china_queries"] = routeCounts["china"]
	metrics["intl_queries"] = routeCounts["intl"]
	metrics["adguard_queries"] = routeCounts["adguard"]
	metrics["cache_queries"] = routeCounts["cache"]

	// 获取缓存统计
	cacheStats := s.GetCacheStats()
	metrics["cache_stats"] = cacheStats

	// 获取延迟统计
	latencyStats := s.GetLatencyStats()
	metrics["latency_stats"] = latencyStats

	// 获取上游健康状态
	upstreamHealth := make(map[string]interface{})
	s.healthMu.Lock()
	for key, state := range s.upstreamHealth {
		upstreamHealth[key] = map[string]interface{}{
			"healthy":       time.Now().After(state.trippedUntil),
			"failures":      state.failures,
			"tripped_until": state.trippedUntil,
		}
	}
	s.healthMu.Unlock()

	metrics["upstream_health"] = upstreamHealth

	return metrics
}

// 路由延迟统计
type routeLatencyStats struct {
	Count        int64         `json:"count"`
	TotalLatency time.Duration `json:"total_latency"`
	MinLatency   time.Duration `json:"min_latency"`
	MaxLatency   time.Duration `json:"max_latency"`
	AvgLatency   time.Duration `json:"avg_latency"`
	LastUpdated  time.Time     `json:"last_updated"`
}

// 更新延迟统计
func (s *Server) updateLatencyStats(route string, latency time.Duration) {
	s.latencyStats.mu.Lock()
	defer s.latencyStats.mu.Unlock()

	// 更新总体统计
	s.latencyStats.totalQueries++
	s.latencyStats.totalLatency += latency

	// 更新最小延迟
	if s.latencyStats.minLatency == 0 || latency < s.latencyStats.minLatency {
		s.latencyStats.minLatency = latency
	}

	// 更新最大延迟
	if latency > s.latencyStats.maxLatency {
		s.latencyStats.maxLatency = latency
	}

	// 计算平均延迟
	s.latencyStats.avgLatency = s.latencyStats.totalLatency / time.Duration(s.latencyStats.totalQueries)

	// 更新路由统计
	if s.latencyStats.routeStats[route] == nil {
		s.latencyStats.routeStats[route] = &routeLatencyStats{}
	}

	routeStat := s.latencyStats.routeStats[route]
	routeStat.Count++
	routeStat.TotalLatency += latency

	// 更新路由最小延迟
	if routeStat.MinLatency == 0 || latency < routeStat.MinLatency {
		routeStat.MinLatency = latency
	}

	// 更新路由最大延迟
	if latency > routeStat.MaxLatency {
		routeStat.MaxLatency = latency
	}

	// 计算路由平均延迟
	routeStat.AvgLatency = routeStat.TotalLatency / time.Duration(routeStat.Count)
	routeStat.LastUpdated = time.Now()
}

// GetLatencyStats 获取延迟统计信息
func (s *Server) GetLatencyStats() map[string]interface{} {
	s.latencyStats.mu.RLock()
	defer s.latencyStats.mu.RUnlock()

	stats := map[string]interface{}{
		"total_queries":    s.latencyStats.totalQueries,
		"total_latency_ms": s.latencyStats.totalLatency.Milliseconds(),
		"min_latency_ms":   s.latencyStats.minLatency.Milliseconds(),
		"max_latency_ms":   s.latencyStats.maxLatency.Milliseconds(),
		"avg_latency_ms":   s.latencyStats.avgLatency.Milliseconds(),
		"route_stats":      make(map[string]interface{}),
	}

	// 添加路由统计
	for route, routeStat := range s.latencyStats.routeStats {
		stats["route_stats"].(map[string]interface{})[route] = map[string]interface{}{
			"count":            routeStat.Count,
			"total_latency_ms": routeStat.TotalLatency.Milliseconds(),
			"min_latency_ms":   routeStat.MinLatency.Milliseconds(),
			"max_latency_ms":   routeStat.MaxLatency.Milliseconds(),
			"avg_latency_ms":   routeStat.AvgLatency.Milliseconds(),
			"last_updated":     routeStat.LastUpdated.Format(time.RFC3339),
		}
	}

	return stats
}

// mergeAndDeduplicate 合并两个字符串切片并去重
func mergeAndDeduplicate(a, b []string) []string {
	seen := make(map[string]bool)
	var result []string

	// 添加第一个切片中的元素
	for _, item := range a {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	// 添加第二个切片中的元素
	for _, item := range b {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// GetStorageManager 获取存储管理器（公共方法）
func (s *Server) GetStorageManager() StorageManager {
	return s.persistence
}

// GetProxyManager 获取代理管理器（公共方法）
func (s *Server) GetProxyManager() *ProxyManager {
	return s.proxyManager
}
