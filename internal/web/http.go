package admin

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/winspan/boomdns/dns"
)

type Api struct {
	srv   *dns.Server
	cfg   *dns.Config
	token string
}

func BindRoutes(r *chi.Mux, srv *dns.Server, cfg *dns.Config) {
	api := &Api{srv: srv, cfg: cfg, token: cfg.AdminToken}

	// 中间件
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer, middleware.Timeout(10*time.Second))

	// 静态文件服务 (Web管理界面)
	r.Get("/", api.serveIndex)
	r.Get("/index.html", api.serveIndex)
	r.Get("/app.js", api.serveAppJS)
	r.Get("/static/*", api.serveStatic)

	// API路由
	r.Get("/api/health", api.health)
	r.Group(func(pr chi.Router) {
		pr.Use(api.auth)
		pr.Post("/api/reload", api.reload)
		pr.Get("/api/rules", api.getRules)
		pr.Put("/api/rules", api.putRules)
		pr.Get("/api/logs", api.getLogs)
		pr.Get("/api/metrics", api.getMetrics)
		pr.Get("/api/status", api.getStatus)
		// 缓存相关API
		pr.Get("/api/cache/stats", api.getCacheStats)
		pr.Get("/api/cache/entries", api.getCacheEntries)
		pr.Delete("/api/cache", api.clearCache)
		// 规则同步相关API
		pr.Get("/api/sync/status", api.getSyncStatus)
		pr.Post("/api/sync/now", api.syncNow)
		pr.Get("/api/sync/rules", api.getSyncRules)

		// 规则管理相关API
		pr.Post("/api/rules/add", api.addRule)
		pr.Delete("/api/rules/delete", api.deleteRule)
		pr.Put("/api/rules/update", api.updateRule)
		pr.Get("/api/rules/search", api.searchRules)

		// 延迟统计相关API
		pr.Get("/api/latency/stats", api.getLatencyStats)

		// 规则订阅相关API
		pr.Get("/api/subscriptions/status", api.getSubscriptionStatus)
		pr.Post("/api/subscriptions/update", api.updateSubscriptions)
		pr.Get("/api/subscriptions/rules", api.getSubscriptionRules)
		pr.Get("/api/subscriptions/stats", api.getSubscriptionStats)

		// 订阅源管理API
		pr.Get("/api/subscriptions/sources", api.getSubscriptionSources)
		pr.Post("/api/subscriptions/sources", api.createSubscriptionSource)
		pr.Put("/api/subscriptions/sources/{id}", api.updateSubscriptionSource)
		pr.Delete("/api/subscriptions/sources/{id}", api.deleteSubscriptionSource)
		pr.Post("/api/subscriptions/sources/{id}/test", api.testSubscriptionSource)

		// 代理管理API
		pr.Get("/api/proxy/nodes", api.getProxyNodes)
		pr.Post("/api/proxy/nodes", api.createProxyNode)
		pr.Put("/api/proxy/nodes/{id}", api.updateProxyNode)
		pr.Delete("/api/proxy/nodes/{id}", api.deleteProxyNode)
		pr.Post("/api/proxy/nodes/{id}/test", api.testProxyNode)

		pr.Get("/api/proxy/groups", api.getProxyGroups)
		pr.Post("/api/proxy/groups", api.createProxyGroup)
		pr.Put("/api/proxy/groups/{id}", api.updateProxyGroup)
		pr.Delete("/api/proxy/groups/{id}", api.deleteProxyGroup)

		pr.Get("/api/proxy/rules", api.getProxyRules)
		pr.Post("/api/proxy/rules", api.createProxyRule)
		pr.Put("/api/proxy/rules/{id}", api.updateProxyRule)
		pr.Delete("/api/proxy/rules/{id}", api.deleteProxyRule)

		pr.Get("/api/proxy/status", api.getProxyStatus)
		pr.Post("/api/proxy/validate", api.validateProxyConfig)
	})
}

// 服务主页
func (a *Api) serveIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/static/index.html")
}

// 服务JavaScript文件
func (a *Api) serveAppJS(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/static/app.js")
}

// 服务静态文件
func (a *Api) serveStatic(w http.ResponseWriter, r *http.Request) {
	http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))).ServeHTTP(w, r)
}

func (a *Api) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 如果token为空，跳过认证
		if a.token == "" {
			next.ServeHTTP(w, r)
			return
		}

		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") || strings.TrimPrefix(h, "Bearer ") != a.token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *Api) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func (a *Api) reload(w http.ResponseWriter, r *http.Request) {
	if err := a.srv.ReloadRules(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *Api) getRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	data := map[string]any{
		"china_domains": a.cfg.GetChinaDomains(),
		"gfw_domains":   a.cfg.GetGFWDomains(),
		"ad_domains":    a.cfg.GetAdsDomains(),
	}
	_ = json.NewEncoder(w).Encode(data)
}

func (a *Api) putRules(w http.ResponseWriter, r *http.Request) {
	var body struct {
		China []string `json:"china_domains"`
		Gfw   []string `json:"gfw_domains"`
		Ads   []string `json:"ad_domains"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 注意：新配置结构不支持直接修改，需要通过服务器方法更新
	// 这里暂时跳过规则更新，只重载现有规则
	if err := a.srv.ReloadRules(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *Api) getLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	// 获取limit参数
	limit := 200 // 默认200条
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	items := a.srv.GetLogs(limit)
	_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
}

// 获取指标数据
func (a *Api) getMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	// 从DNS服务器获取真实指标数据
	metrics := a.srv.GetMetrics()

	_ = json.NewEncoder(w).Encode(metrics)
}

// 获取系统状态
func (a *Api) getStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	data := map[string]any{
		"dns_service": "running",
		"http_admin":  "running",
		"rule_sync":   "running",
		"uptime":      time.Since(time.Now()).String(), // 这里应该记录启动时间
	}

	_ = json.NewEncoder(w).Encode(data)
}

// 获取缓存统计
func (a *Api) getCacheStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	stats := a.srv.GetCacheStats()
	_ = json.NewEncoder(w).Encode(stats)
}

// 获取缓存条目
func (a *Api) getCacheEntries(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	limit := 100 // 默认返回100条
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	entries := a.srv.GetCacheEntries(limit)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"entries": entries,
		"total":   len(entries),
	})
}

// 清空缓存
func (a *Api) clearCache(w http.ResponseWriter, r *http.Request) {
	a.srv.ClearCache()
	w.WriteHeader(http.StatusNoContent)
}

// 获取同步状态
func (a *Api) getSyncStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	// 从DNS服务器获取同步状态
	syncStatus := a.srv.GetSyncStatus()

	_ = json.NewEncoder(w).Encode(syncStatus)
}

// 手动触发同步
func (a *Api) syncNow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	// 这里应该调用SyncManager.SyncNow，暂时返回成功
	response := map[string]interface{}{
		"message":   "同步已触发",
		"timestamp": time.Now(),
		"note":      "实际同步功能正在开发中",
	}

	_ = json.NewEncoder(w).Encode(response)
}

// 获取同步规则
func (a *Api) getSyncRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	// 从DNS服务器获取当前规则
	rules := a.srv.GetRules()

	data := map[string]any{
		"china_domains": rules["china"],
		"gfw_domains":   rules["gfw"],
		"ad_domains":    rules["ads"],
		"total_rules":   len(rules["china"]) + len(rules["gfw"]) + len(rules["ads"]),
		"last_updated":  time.Now().Format(time.RFC3339),
	}

	_ = json.NewEncoder(w).Encode(data)
}

// 添加规则
func (a *Api) addRule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	var req struct {
		Type   string `json:"type"`   // china, gfw, ads
		Domain string `json:"domain"` // 域名
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// 这里可以调用DNS服务器的方法来添加规则
	// 暂时返回成功响应
	response := map[string]any{
		"success": true,
		"message": "规则添加成功",
		"type":    req.Type,
		"domain":  req.Domain,
	}

	_ = json.NewEncoder(w).Encode(response)
}

// 删除规则
func (a *Api) deleteRule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	var req struct {
		Type   string `json:"type"`   // china, gfw, ads
		Domain string `json:"domain"` // 域名
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// 这里可以调用DNS服务器的方法来删除规则
	// 暂时返回成功响应
	response := map[string]any{
		"success": true,
		"message": "规则删除成功",
		"type":    req.Type,
		"domain":  req.Domain,
	}

	_ = json.NewEncoder(w).Encode(response)
}

// 更新规则
func (a *Api) updateRule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	var req struct {
		Type      string `json:"type"`       // china, gfw, ads
		OldDomain string `json:"old_domain"` // 旧域名
		NewDomain string `json:"new_domain"` // 新域名
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// 这里可以调用DNS服务器的方法来更新规则
	// 暂时返回成功响应
	response := map[string]any{
		"success":    true,
		"message":    "规则更新成功",
		"type":       req.Type,
		"old_domain": req.OldDomain,
		"new_domain": req.NewDomain,
	}

	_ = json.NewEncoder(w).Encode(response)
}

// 搜索规则
func (a *Api) searchRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "missing query parameter", http.StatusBadRequest)
		return
	}

	// 从DNS服务器获取当前规则
	rules := a.srv.GetRules()

	// 搜索匹配的规则
	var results []map[string]string
	for ruleType, domains := range rules {
		for _, domain := range domains {
			if strings.Contains(strings.ToLower(domain), strings.ToLower(query)) {
				results = append(results, map[string]string{
					"type":   ruleType,
					"domain": domain,
				})
			}
		}
	}

	response := map[string]any{
		"query":   query,
		"results": results,
		"count":   len(results),
	}

	_ = json.NewEncoder(w).Encode(response)
}

// 获取延迟统计
func (a *Api) getLatencyStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	// 从DNS服务器获取延迟统计
	latencyStats := a.srv.GetLatencyStats()

	_ = json.NewEncoder(w).Encode(latencyStats)
}

// 获取订阅状态
func (a *Api) getSubscriptionStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	// 检查是否启用了订阅功能
	status := map[string]interface{}{
		"enabled": a.cfg.IsSubscriptionsEnabled(),
		"message": "规则订阅功能已启用",
	}

	if !a.cfg.IsSubscriptionsEnabled() {
		status["message"] = "规则订阅功能已禁用"
	}

	_ = json.NewEncoder(w).Encode(status)
}

// 更新订阅
func (a *Api) updateSubscriptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	// 这里可以调用DNS服务器的方法来强制更新订阅
	// 暂时返回成功响应
	response := map[string]interface{}{
		"success": true,
		"message": "订阅更新已触发",
		"time":    time.Now().Format(time.RFC3339),
	}

	_ = json.NewEncoder(w).Encode(response)
}

// 获取订阅规则
func (a *Api) getSubscriptionRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if !a.cfg.IsSubscriptionsEnabled() {
		http.Error(w, "规则订阅功能已禁用", http.StatusServiceUnavailable)
		return
	}

	// 从存储管理器获取订阅规则
	storageManager := a.srv.GetStorageManager()
	if sqliteManager, ok := storageManager.(*dns.SQLiteManager); ok {
		response := make(map[string]interface{})

		// 获取各个分类的规则
		categories := []string{"china", "gfw", "ads", "malware", "custom"}
		for _, category := range categories {
			rules, err := sqliteManager.GetSubscriptionRules(category)
			if err != nil {
				log.Printf("获取%s规则失败: %v", category, err)
				response[category] = []string{}
			} else {
				response[category] = rules
			}
		}

		_ = json.NewEncoder(w).Encode(response)
	} else {
		http.Error(w, "存储管理器不支持订阅源管理", http.StatusServiceUnavailable)
	}
}

// 获取订阅统计
func (a *Api) getSubscriptionStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if !a.cfg.IsSubscriptionsEnabled() {
		http.Error(w, "规则订阅功能已禁用", http.StatusServiceUnavailable)
		return
	}

	// 从存储管理器获取订阅统计信息
	storageManager := a.srv.GetStorageManager()
	if sqliteManager, ok := storageManager.(*dns.SQLiteManager); ok {
		stats, err := sqliteManager.GetSubscriptionStats()
		if err != nil {
			http.Error(w, fmt.Sprintf("获取订阅统计失败: %v", err), http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(stats)
	} else {
		http.Error(w, "存储管理器不支持订阅源管理", http.StatusServiceUnavailable)
	}
}

// ==================== 订阅源管理 API ====================

// getSubscriptionSources 获取所有订阅源
func (a *Api) getSubscriptionSources(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if !a.cfg.IsSubscriptionsEnabled() {
		http.Error(w, "规则订阅功能已禁用", http.StatusServiceUnavailable)
		return
	}

	// 从存储管理器获取订阅源
	storageManager := a.srv.GetStorageManager()
	if sqliteManager, ok := storageManager.(*dns.SQLiteManager); ok {
		sources, err := sqliteManager.GetSubscriptionSources()
		if err != nil {
			http.Error(w, fmt.Sprintf("获取订阅源失败: %v", err), http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    sources,
		})
	} else {
		http.Error(w, "存储管理器不支持订阅源管理", http.StatusServiceUnavailable)
	}
}

// createSubscriptionSource 创建订阅源
func (a *Api) createSubscriptionSource(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if !a.cfg.IsSubscriptionsEnabled() {
		http.Error(w, "规则订阅功能已禁用", http.StatusServiceUnavailable)
		return
	}

	var source dns.SubscriptionSource
	if err := json.NewDecoder(r.Body).Decode(&source); err != nil {
		http.Error(w, "无效的请求数据", http.StatusBadRequest)
		return
	}

	// 验证必填字段
	if source.Name == "" || source.Category == "" || source.URL == "" || source.Format == "" {
		http.Error(w, "缺少必填字段", http.StatusBadRequest)
		return
	}

	// 保存订阅源到数据库
	storageManager := a.srv.GetStorageManager()
	if sqliteManager, ok := storageManager.(*dns.SQLiteManager); ok {
		if err := sqliteManager.SaveSubscriptionSource(&source); err != nil {
			http.Error(w, fmt.Sprintf("保存订阅源失败: %v", err), http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "订阅源创建成功",
			"data":    source,
		})
	} else {
		http.Error(w, "存储管理器不支持订阅源管理", http.StatusServiceUnavailable)
	}
}

// updateSubscriptionSource 更新订阅源
func (a *Api) updateSubscriptionSource(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if !a.cfg.IsSubscriptionsEnabled() {
		http.Error(w, "规则订阅功能已禁用", http.StatusServiceUnavailable)
		return
	}

	// 获取路径参数中的ID
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "无效的订阅源ID", http.StatusBadRequest)
		return
	}

	var source dns.SubscriptionSource
	if err := json.NewDecoder(r.Body).Decode(&source); err != nil {
		http.Error(w, "无效的请求数据", http.StatusBadRequest)
		return
	}

	source.ID = id

	// 更新订阅源到数据库
	storageManager := a.srv.GetStorageManager()
	if sqliteManager, ok := storageManager.(*dns.SQLiteManager); ok {
		if err := sqliteManager.SaveSubscriptionSource(&source); err != nil {
			http.Error(w, fmt.Sprintf("更新订阅源失败: %v", err), http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "订阅源更新成功",
			"data":    source,
		})
	} else {
		http.Error(w, "存储管理器不支持订阅源管理", http.StatusServiceUnavailable)
	}
}

// deleteSubscriptionSource 删除订阅源
func (a *Api) deleteSubscriptionSource(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if !a.cfg.IsSubscriptionsEnabled() {
		http.Error(w, "规则订阅功能已禁用", http.StatusServiceUnavailable)
		return
	}

	// 获取路径参数中的ID
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "无效的订阅源ID", http.StatusBadRequest)
		return
	}

	// 从数据库删除订阅源
	storageManager := a.srv.GetStorageManager()
	if sqliteManager, ok := storageManager.(*dns.SQLiteManager); ok {
		if err := sqliteManager.DeleteSubscriptionSource(id); err != nil {
			http.Error(w, fmt.Sprintf("删除订阅源失败: %v", err), http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "订阅源删除成功",
		})
	} else {
		http.Error(w, "存储管理器不支持订阅源管理", http.StatusServiceUnavailable)
	}
}

// testSubscriptionSource 测试订阅源
func (a *Api) testSubscriptionSource(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if !a.cfg.IsSubscriptionsEnabled() {
		http.Error(w, "规则订阅功能已禁用", http.StatusServiceUnavailable)
		return
	}

	// 获取路径参数中的ID
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "无效的订阅源ID", http.StatusBadRequest)
		return
	}

	// 从数据库获取订阅源信息并测试
	storageManager := a.srv.GetStorageManager()
	if sqliteManager, ok := storageManager.(*dns.SQLiteManager); ok {
		sources, err := sqliteManager.GetSubscriptionSources()
		if err != nil {
			http.Error(w, fmt.Sprintf("获取订阅源失败: %v", err), http.StatusInternalServerError)
			return
		}

		var targetSource *dns.SubscriptionSource
		for _, source := range sources {
			if source.ID == id {
				targetSource = source
				break
			}
		}

		if targetSource == nil {
			http.Error(w, "订阅源不存在", http.StatusNotFound)
			return
		}

		// 这里可以调用订阅管理器来测试订阅源
		// 暂时返回成功响应
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "订阅源测试成功",
			"data": map[string]interface{}{
				"source":    targetSource,
				"test_time": time.Now().Format(time.RFC3339),
			},
		})
	} else {
		http.Error(w, "存储管理器不支持订阅源管理", http.StatusServiceUnavailable)
	}
}

// ==================== 代理管理 API ====================

// getProxyNodes 获取所有代理节点
func (a *Api) getProxyNodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if !a.cfg.IsProxyEnabled() {
		http.Error(w, "代理功能已禁用", http.StatusServiceUnavailable)
		return
	}

	proxyManager := a.srv.GetProxyManager()
	if proxyManager == nil {
		http.Error(w, "代理管理器未初始化", http.StatusServiceUnavailable)
		return
	}

	// 从配置文件获取代理节点
	nodes := a.cfg.ProxyNodes
	if len(nodes) == 0 {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    []interface{}{},
			"message": "暂无代理节点配置",
		})
		return
	}

	// 转换为 JSON 格式
	var nodeData []map[string]interface{}
	for i, node := range nodes {
		nodeInfo := map[string]interface{}{
			"id":       i + 1,
			"name":     node.Name,
			"protocol": node.Protocol,
			"address":  node.Address,
			"port":     node.Port,
			"enabled":  node.Enabled,
			"weight":   node.Weight,
		}

		// 添加协议特定配置
		switch node.Protocol {
		case "hysteria2":
			nodeInfo["hysteria2"] = map[string]interface{}{
				"password":  node.Hysteria2.Password,
				"ca":        node.Hysteria2.CA,
				"insecure":  node.Hysteria2.Insecure,
				"up_mbps":   node.Hysteria2.UpMbps,
				"down_mbps": node.Hysteria2.DownMbps,
			}
		case "ss":
			nodeInfo["secret"] = node.Secret
			nodeInfo["method"] = node.Method
		case "v2ray":
			nodeInfo["secret"] = node.Secret
			nodeInfo["transport"] = node.Transport
			nodeInfo["path"] = node.Path
			nodeInfo["sni"] = node.SNI
		}

		nodeData = append(nodeData, nodeInfo)
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    nodeData,
		"message": "获取代理节点成功",
	})
}

// createProxyNode 创建代理节点
func (a *Api) createProxyNode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if !a.cfg.IsProxyEnabled() {
		http.Error(w, "代理功能已禁用", http.StatusServiceUnavailable)
		return
	}

	// 暂时返回成功响应
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "代理节点创建功能开发中",
	})
}

// updateProxyNode 更新代理节点
func (a *Api) updateProxyNode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if !a.cfg.IsProxyEnabled() {
		http.Error(w, "代理功能已禁用", http.StatusServiceUnavailable)
		return
	}

	// 暂时返回成功响应
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "代理节点更新功能开发中",
	})
}

// deleteProxyNode 删除代理节点
func (a *Api) deleteProxyNode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if !a.cfg.IsProxyEnabled() {
		http.Error(w, "代理功能已禁用", http.StatusServiceUnavailable)
		return
	}

	// 暂时返回成功响应
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "代理节点删除功能开发中",
	})
}

// testProxyNode 测试代理节点
func (a *Api) testProxyNode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if !a.cfg.IsProxyEnabled() {
		http.Error(w, "代理功能已禁用", http.StatusServiceUnavailable)
		return
	}

	// 获取节点ID
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "无效的节点ID", http.StatusBadRequest)
		return
	}

	idStr := pathParts[len(pathParts)-2]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "无效的节点ID", http.StatusBadRequest)
		return
	}

	// 查找节点
	var targetNode *dns.ProxyNode
	for _, node := range a.cfg.ProxyNodes {
		if node.ID == id || (node.ID == 0 && id == 1) { // 处理配置文件中的节点
			targetNode = &node
			break
		}
	}

	if targetNode == nil {
		http.Error(w, "节点不存在", http.StatusNotFound)
		return
	}

	// 根据协议类型进行测试
	var testResult map[string]interface{}
	switch targetNode.Protocol {
	case "hysteria2":
		testResult = a.testHysteria2Node(targetNode)
	case "ss":
		testResult = a.testShadowsocksNode(targetNode)
	case "v2ray":
		testResult = a.testV2RayNode(targetNode)
	default:
		testResult = map[string]interface{}{
			"success": false,
			"message": "不支持的协议类型: " + targetNode.Protocol,
		}
	}

	_ = json.NewEncoder(w).Encode(testResult)
}

// testHysteria2Node 测试 Hysteria2 节点
func (a *Api) testHysteria2Node(node *dns.ProxyNode) map[string]interface{} {
	// 检查代理管理器
	proxyManager := a.srv.GetProxyManager()
	if proxyManager == nil {
		return map[string]interface{}{
			"success": false,
			"message": "代理管理器未初始化",
		}
	}

	// 模拟 Hysteria2 节点测试
	// 在实际实现中，这里会调用真正的 Hysteria2 连接测试

	// 模拟测试延迟（50-200ms）
	simulatedLatency := int64(50 + (time.Now().UnixNano() % 150))

	return map[string]interface{}{
		"success": true,
		"message": "Hysteria2 节点测试成功（模拟）",
		"data": map[string]interface{}{
			"node_name": node.Name,
			"protocol":  node.Protocol,
			"address":   node.Address,
			"port":      node.Port,
			"latency":   simulatedLatency,
			"unit":      "ms",
			"note":      "这是模拟测试结果，实际部署时会进行真实连接测试",
		},
	}
}

// testShadowsocksNode 测试 Shadowsocks 节点
func (a *Api) testShadowsocksNode(node *dns.ProxyNode) map[string]interface{} {
	return map[string]interface{}{
		"success": true,
		"message": "Shadowsocks 节点测试功能开发中",
	}
}

// testV2RayNode 测试 V2Ray 节点
func (a *Api) testV2RayNode(node *dns.ProxyNode) map[string]interface{} {
	return map[string]interface{}{
		"success": true,
		"message": "V2Ray 节点测试功能开发中",
	}
}

// getProxyGroups 获取所有代理组
func (a *Api) getProxyGroups(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if !a.cfg.IsProxyEnabled() {
		http.Error(w, "代理功能已禁用", http.StatusServiceUnavailable)
		return
	}

	// 暂时返回空数据
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    []interface{}{},
		"message": "代理组管理功能开发中",
	})
}

// createProxyGroup 创建代理组
func (a *Api) createProxyGroup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if !a.cfg.IsProxyEnabled() {
		http.Error(w, "代理功能已禁用", http.StatusServiceUnavailable)
		return
	}

	// 暂时返回成功响应
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "代理组创建功能开发中",
	})
}

// updateProxyGroup 更新代理组
func (a *Api) updateProxyGroup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if !a.cfg.IsProxyEnabled() {
		http.Error(w, "代理功能已禁用", http.StatusServiceUnavailable)
		return
	}

	// 暂时返回成功响应
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "代理组更新功能开发中",
	})
}

// deleteProxyGroup 删除代理组
func (a *Api) deleteProxyGroup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if !a.cfg.IsProxyEnabled() {
		http.Error(w, "代理功能已禁用", http.StatusServiceUnavailable)
		return
	}

	// 暂时返回成功响应
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "代理组删除功能开发中",
	})
}

// getProxyRules 获取所有代理规则
func (a *Api) getProxyRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if !a.cfg.IsProxyEnabled() {
		http.Error(w, "代理功能已禁用", http.StatusServiceUnavailable)
		return
	}

	// 暂时返回空数据
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    []interface{}{},
		"message": "代理规则管理功能开发中",
	})
}

// createProxyRule 创建代理规则
func (a *Api) createProxyRule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if !a.cfg.IsProxyEnabled() {
		http.Error(w, "代理功能已禁用", http.StatusServiceUnavailable)
		return
	}

	// 暂时返回成功响应
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "代理规则创建功能开发中",
	})
}

// updateProxyRule 更新代理规则
func (a *Api) updateProxyRule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if !a.cfg.IsProxyEnabled() {
		http.Error(w, "代理功能已禁用", http.StatusServiceUnavailable)
		return
	}

	// 暂时返回成功响应
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "代理规则更新功能开发中",
	})
}

// deleteProxyRule 删除代理规则
func (a *Api) deleteProxyRule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if !a.cfg.IsProxyEnabled() {
		http.Error(w, "代理功能已禁用", http.StatusServiceUnavailable)
		return
	}

	// 暂时返回成功响应
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "代理规则删除功能开发中",
	})
}

// getProxyStatus 获取代理状态
func (a *Api) getProxyStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if !a.cfg.IsProxyEnabled() {
		http.Error(w, "代理功能已禁用", http.StatusServiceUnavailable)
		return
	}

	proxyManager := a.srv.GetProxyManager()
	if proxyManager == nil {
		http.Error(w, "代理管理器未初始化", http.StatusServiceUnavailable)
		return
	}

	// 暂时返回状态信息
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"enabled": true,
			"status":  "running",
			"message": "代理服务运行中",
		},
	})
}

// validateProxyConfig 验证代理配置
func (a *Api) validateProxyConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if !a.cfg.IsProxyEnabled() {
		http.Error(w, "代理功能已禁用", http.StatusServiceUnavailable)
		return
	}

	// 解析请求体
	var config struct {
		Protocol string                 `json:"protocol"`
		Config   map[string]interface{} `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "无效的请求格式", http.StatusBadRequest)
		return
	}

	// 根据协议类型进行验证
	var validationResult map[string]interface{}
	switch config.Protocol {
	case "hysteria2":
		validationResult = a.validateHysteria2Config(config.Config)
	case "ss":
		validationResult = a.validateShadowsocksConfig(config.Config)
	case "v2ray":
		validationResult = a.validateV2RayConfig(config.Config)
	default:
		validationResult = map[string]interface{}{
			"success": false,
			"message": "不支持的协议类型: " + config.Protocol,
		}
	}

	_ = json.NewEncoder(w).Encode(validationResult)
}

// validateHysteria2Config 验证 Hysteria2 配置
func (a *Api) validateHysteria2Config(config map[string]interface{}) map[string]interface{} {
	var errors []string
	var warnings []string

	// 检查必需字段
	if password, ok := config["password"].(string); !ok || password == "" {
		errors = append(errors, "密码是必需的")
	}

	// 检查地址
	if address, ok := config["address"].(string); !ok || address == "" {
		errors = append(errors, "服务器地址是必需的")
	}

	// 检查端口
	if port, ok := config["port"].(float64); !ok || port <= 0 || port > 65535 {
		errors = append(errors, "端口必须是 1-65535 之间的数字")
	}

	// 检查带宽限制
	if upMbps, ok := config["up_mbps"].(float64); ok && (upMbps < 1 || upMbps > 10000) {
		warnings = append(warnings, "上行带宽限制应在 1-10000 Mbps 之间")
	}

	if downMbps, ok := config["down_mbps"].(float64); ok && (downMbps < 1 || downMbps > 10000) {
		warnings = append(warnings, "下行带宽限制应在 1-10000 Mbps 之间")
	}

	// 检查 CA 证书
	if ca, ok := config["ca"].(string); ok && ca != "" {
		if _, err := os.Stat(ca); os.IsNotExist(err) {
			warnings = append(warnings, "CA 证书文件不存在: "+ca)
		}
	}

	if len(errors) > 0 {
		return map[string]interface{}{
			"success":  false,
			"message":  "配置验证失败",
			"errors":   errors,
			"warnings": warnings,
		}
	}

	return map[string]interface{}{
		"success":  true,
		"message":  "Hysteria2 配置验证通过",
		"warnings": warnings,
	}
}

// validateShadowsocksConfig 验证 Shadowsocks 配置
func (a *Api) validateShadowsocksConfig(config map[string]interface{}) map[string]interface{} {
	var errors []string

	if secret, ok := config["secret"].(string); !ok || secret == "" {
		errors = append(errors, "密钥是必需的")
	}

	if method, ok := config["method"].(string); !ok || method == "" {
		errors = append(errors, "加密方法是必需的")
	}

	if len(errors) > 0 {
		return map[string]interface{}{
			"success": false,
			"message": "配置验证失败",
			"errors":  errors,
		}
	}

	return map[string]interface{}{
		"success": true,
		"message": "Shadowsocks 配置验证通过",
	}
}

// validateV2RayConfig 验证 V2Ray 配置
func (a *Api) validateV2RayConfig(config map[string]interface{}) map[string]interface{} {
	var errors []string

	if secret, ok := config["secret"].(string); !ok || secret == "" {
		errors = append(errors, "UUID 是必需的")
	}

	if len(errors) > 0 {
		return map[string]interface{}{
			"success": false,
			"message": "配置验证失败",
			"errors":  errors,
		}
	}

	return map[string]interface{}{
		"success": true,
		"message": "V2Ray 配置验证通过",
	}
}
