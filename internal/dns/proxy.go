package dns

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

// ProxyProtocol 代理协议类型
type ProxyProtocol string

const (
	ProxyHTTP        ProxyProtocol = "http"
	ProxyHTTPS       ProxyProtocol = "https"
	ProxySOCKS5      ProxyProtocol = "socks5"
	ProxyShadowsocks ProxyProtocol = "ss"
	ProxyV2Ray       ProxyProtocol = "v2ray"
	ProxyTrojan      ProxyProtocol = "trojan"
	ProxyWireGuard   ProxyProtocol = "wireguard"
	ProxyHysteria2   ProxyProtocol = "hysteria2"
)

// ProxyNode 代理节点配置
type ProxyNode struct {
	ID        int           `json:"id"`
	Name      string        `json:"name"`
	Protocol  ProxyProtocol `json:"protocol"`
	Address   string        `json:"address"`
	Port      int           `json:"port"`
	Username  string        `json:"username,omitempty"`
	Password  string        `json:"password,omitempty"`
	Secret    string        `json:"secret,omitempty"`    // Shadowsocks/V2Ray 密钥
	Method    string        `json:"method,omitempty"`    // 加密方法
	Transport string        `json:"transport,omitempty"` // 传输协议 (ws, tcp, quic)
	Path      string        `json:"path,omitempty"`      // WebSocket 路径
	SNI       string        `json:"sni,omitempty"`       // TLS SNI
	Enabled   bool          `json:"enabled"`
	Latency   int64         `json:"latency"`    // 延迟 (ms)
	LastCheck int64         `json:"last_check"` // 最后检查时间
	FailCount int           `json:"fail_count"` // 失败次数
	Weight    int           `json:"weight"`     // 权重
	CreatedAt int64         `json:"created_at"`
	UpdatedAt int64         `json:"updated_at"`
	
	// Hysteria2 特定配置
	Hysteria2 struct {
		Password string `json:"password,omitempty"` // Hysteria2 密码
		CA       string `json:"ca,omitempty"`       // CA 证书路径
		Insecure bool   `json:"insecure,omitempty"` // 跳过证书验证
		UpMbps   int    `json:"up_mbps,omitempty"`  // 上行带宽 (Mbps)
		DownMbps int    `json:"down_mbps,omitempty"` // 下行带宽 (Mbps)
	} `json:"hysteria2,omitempty"`
}

// ProxyGroup 代理组配置
type ProxyGroup struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`     // select, url-test, fallback, load-balance
	Nodes     []int  `json:"nodes"`    // 节点ID列表
	Strategy  string `json:"strategy"` // 策略 (round-robin, latency, weight)
	TestURL   string `json:"test_url"` // 延迟测试URL
	Interval  int    `json:"interval"` // 测试间隔 (秒)
	Timeout   int    `json:"timeout"`  // 超时时间 (秒)
	Enabled   bool   `json:"enabled"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// ProxyRule 代理规则配置
type ProxyRule struct {
	ID         int    `json:"id"`
	Type       string `json:"type"`        // domain, ip-cidr, geoip
	Value      string `json:"value"`       // 规则值
	Action     string `json:"action"`      // proxy, direct, reject
	ProxyGroup string `json:"proxy_group"` // 代理组名称
	Priority   int    `json:"priority"`    // 优先级
	Enabled    bool   `json:"enabled"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

// ProxyManager 代理管理器
type ProxyManager struct {
	config *ProxyConfig
	nodes  map[int]*ProxyNode
	groups map[string]*ProxyGroup
	rules  []*ProxyRule
	mutex  sync.RWMutex

	// HTTP 客户端池
	httpClients map[int]*http.Client
	// 连接池
	connPool map[string]net.Conn

	// 健康检查
	healthTicker *time.Ticker
	stopChan     chan bool
}

// ProxyConfig 代理配置
type ProxyConfig struct {
	Enabled         bool   `yaml:"enabled"`
	ListenHTTP      string `yaml:"listen_http"`
	ListenSOCKS     string `yaml:"listen_socks"`
	DefaultStrategy string `yaml:"default_strategy"`
	TestInterval    int    `yaml:"test_interval"`
	TestTimeout     int    `yaml:"test_timeout"`
}

// NewProxyManager 创建新的代理管理器
func NewProxyManager(config *ProxyConfig) *ProxyManager {
	if config == nil {
		config = &ProxyConfig{
			Enabled:         true,
			ListenHTTP:      ":7890",
			ListenSOCKS:     ":7891",
			DefaultStrategy: "round-robin",
			TestInterval:    300,
			TestTimeout:     10,
		}
	}

	pm := &ProxyManager{
		config:      config,
		nodes:       make(map[int]*ProxyNode),
		groups:      make(map[string]*ProxyGroup),
		rules:       make([]*ProxyRule, 0),
		httpClients: make(map[int]*http.Client),
		connPool:    make(map[string]net.Conn),
		stopChan:    make(chan bool),
	}

	// 启动健康检查
	if config.Enabled {
		pm.startHealthCheck()
	}

	return pm
}

// Start 启动代理管理器
func (pm *ProxyManager) Start() error {
	if !pm.config.Enabled {
		return nil
	}

	// 启动 HTTP 代理服务器
	go pm.startHTTPProxy()

	// 启动 SOCKS5 代理服务器
	go pm.startSOCKS5Proxy()

	log.Println("代理管理器已启动")
	return nil
}

// Stop 停止代理管理器
func (pm *ProxyManager) Stop() {
	if pm.healthTicker != nil {
		pm.healthTicker.Stop()
	}
	close(pm.stopChan)
	log.Println("代理管理器已停止")
}

// AddNode 添加代理节点
func (pm *ProxyManager) AddNode(node *ProxyNode) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// 验证节点配置
	if err := pm.validateNode(node); err != nil {
		return fmt.Errorf("节点配置验证失败: %v", err)
	}

	if node.ID == 0 {
		node.ID = len(pm.nodes) + 1
	}

	node.CreatedAt = time.Now().Unix()
	node.UpdatedAt = time.Now().Unix()

	pm.nodes[node.ID] = node

	// 创建对应的 HTTP 客户端
	pm.createHTTPClient(node)

	return nil
}

// validateNode 验证节点配置
func (pm *ProxyManager) validateNode(node *ProxyNode) error {
	if node.Name == "" {
		return fmt.Errorf("节点名称不能为空")
	}
	
	if node.Address == "" {
		return fmt.Errorf("节点地址不能为空")
	}
	
	if node.Port <= 0 || node.Port > 65535 {
		return fmt.Errorf("节点端口无效: %d", node.Port)
	}
	
	// 根据协议类型进行特定验证
	switch node.Protocol {
	case ProxyHysteria2:
		return pm.validateHysteria2Node(node)
	case ProxyShadowsocks:
		if node.Secret == "" {
			return fmt.Errorf("Shadowsocks 节点缺少密钥")
		}
		if node.Method == "" {
			return fmt.Errorf("Shadowsocks 节点缺少加密方法")
		}
	case ProxyV2Ray:
		if node.Secret == "" {
			return fmt.Errorf("V2Ray 节点缺少 UUID")
		}
	}
	
	return nil
}

// validateHysteria2Node 验证 Hysteria2 节点配置
func (pm *ProxyManager) validateHysteria2Node(node *ProxyNode) error {
	if node.Hysteria2.Password == "" {
		return fmt.Errorf("Hysteria2 节点缺少密码")
	}
	
	if node.Hysteria2.UpMbps > 0 && (node.Hysteria2.UpMbps < 1 || node.Hysteria2.UpMbps > 10000) {
		return fmt.Errorf("Hysteria2 上行带宽限制无效: %d Mbps", node.Hysteria2.UpMbps)
	}
	
	if node.Hysteria2.DownMbps > 0 && (node.Hysteria2.DownMbps < 1 || node.Hysteria2.DownMbps > 10000) {
		return fmt.Errorf("Hysteria2 下行带宽限制无效: %d Mbps", node.Hysteria2.DownMbps)
	}
	
	// 如果指定了 CA 证书，检查文件是否存在
	if node.Hysteria2.CA != "" && !node.Hysteria2.Insecure {
		if _, err := os.Stat(node.Hysteria2.CA); os.IsNotExist(err) {
			return fmt.Errorf("CA 证书文件不存在: %s", node.Hysteria2.CA)
		}
	}
	
	return nil
}

// GetNode 获取代理节点
func (pm *ProxyManager) GetNode(id int) *ProxyNode {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	return pm.nodes[id]
}

// UpdateNode 更新代理节点
func (pm *ProxyManager) UpdateNode(node *ProxyNode) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if _, exists := pm.nodes[node.ID]; !exists {
		return fmt.Errorf("节点不存在: %d", node.ID)
	}

	node.UpdatedAt = time.Now().Unix()
	pm.nodes[node.ID] = node

	// 重新创建 HTTP 客户端
	pm.createHTTPClient(node)

	return nil
}

// DeleteNode 删除代理节点
func (pm *ProxyManager) DeleteNode(id int) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if _, exists := pm.nodes[id]; !exists {
		return fmt.Errorf("节点不存在: %d", id)
	}

	delete(pm.nodes, id)
	delete(pm.httpClients, id)

	return nil
}

// AddGroup 添加代理组
func (pm *ProxyManager) AddGroup(group *ProxyGroup) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if group.ID == 0 {
		group.ID = len(pm.groups) + 1
	}

	group.CreatedAt = time.Now().Unix()
	group.UpdatedAt = time.Now().Unix()

	pm.groups[group.Name] = group

	return nil
}

// GetGroup 获取代理组
func (pm *ProxyManager) GetGroup(name string) *ProxyGroup {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	return pm.groups[name]
}

// AddRule 添加代理规则
func (pm *ProxyManager) AddRule(rule *ProxyRule) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if rule.ID == 0 {
		rule.ID = len(pm.rules) + 1
	}

	rule.CreatedAt = time.Now().Unix()
	rule.UpdatedAt = time.Now().Unix()

	pm.rules = append(pm.rules, rule)

	return nil
}

// MatchRule 匹配代理规则
func (pm *ProxyManager) MatchRule(domain string, ip net.IP) (string, string) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// 按优先级排序规则
	for _, rule := range pm.rules {
		if !rule.Enabled {
			continue
		}

		switch rule.Type {
		case "domain":
			if pm.matchDomain(domain, rule.Value) {
				return rule.Action, rule.ProxyGroup
			}
		case "ip-cidr":
			if pm.matchIPCIDR(ip, rule.Value) {
				return rule.Action, rule.ProxyGroup
			}
		}
	}

	return "direct", ""
}

// GetProxyClient 获取代理客户端
func (pm *ProxyManager) GetProxyClient(groupName string) (*http.Client, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	group := pm.groups[groupName]
	if group == nil || !group.Enabled {
		return nil, fmt.Errorf("代理组不存在或已禁用: %s", groupName)
	}

	// 根据策略选择节点
	nodeID := pm.selectNode(group)
	if nodeID == 0 {
		return nil, fmt.Errorf("没有可用的代理节点")
	}

	node := pm.nodes[nodeID]
	if node == nil {
		return nil, fmt.Errorf("节点不存在: %d", nodeID)
	}

	client := pm.httpClients[nodeID]
	if client == nil {
		return nil, fmt.Errorf("代理客户端未初始化")
	}

	// 记录节点使用统计
	pm.recordNodeUsage(node)

	return client, nil
}

// recordNodeUsage 记录节点使用统计
func (pm *ProxyManager) recordNodeUsage(node *ProxyNode) {
	// 这里可以添加节点使用统计逻辑
	// 例如：使用次数、流量统计、成功率等
	log.Printf("使用代理节点: %s (协议: %s)", node.Name, node.Protocol)
}

// 私有方法

// createHTTPClient 创建 HTTP 客户端
func (pm *ProxyManager) createHTTPClient(node *ProxyNode) {
	var transport *http.Transport

	switch node.Protocol {
	case ProxyHTTP, ProxyHTTPS:
		proxyURL := fmt.Sprintf("%s://%s:%d", node.Protocol, node.Address, node.Port)
		if node.Username != "" {
			proxyURL = fmt.Sprintf("%s://%s:%s@%s:%d", node.Protocol, node.Username, node.Password, node.Address, node.Port)
		}

		proxyURLParsed, err := url.Parse(proxyURL)
		if err != nil {
			log.Printf("解析代理URL失败: %v", err)
			return
		}

		transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURLParsed),
		}

		case ProxySOCKS5:
		dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("%s:%d", node.Address, node.Port), nil, proxy.Direct)
		if err != nil {
			log.Printf("创建SOCKS5拨号器失败: %v", err)
			return
		}
		
		// 创建自定义拨号函数
		dialFunc := func(network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		}
		
		transport = &http.Transport{
			Dial: dialFunc,
		}
		
	case ProxyHysteria2:
		// 创建 Hysteria2 拨号器
		dialer, err := pm.createHysteria2Dialer(node)
		if err != nil {
			log.Printf("创建Hysteria2拨号器失败: %v", err)
			return
		}
		
		transport = &http.Transport{
			Dial: dialer.Dial,
		}
		
	default:
		log.Printf("不支持的代理协议: %s", node.Protocol)
		return
	}

	// 设置超时
	transport.DialContext = (&net.Dialer{
		Timeout:   time.Duration(pm.config.TestTimeout) * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(pm.config.TestTimeout) * time.Second,
	}

	pm.httpClients[node.ID] = client
}

// selectNode 根据策略选择节点
func (pm *ProxyManager) selectNode(group *ProxyGroup) int {
	if len(group.Nodes) == 0 {
		return 0
	}

	switch group.Strategy {
	case "round-robin":
		// 轮询策略
		return group.Nodes[time.Now().Unix()%int64(len(group.Nodes))]

	case "latency":
		// 延迟优先策略
		var bestNode int
		var minLatency int64 = 999999

		for _, nodeID := range group.Nodes {
			if node := pm.nodes[nodeID]; node != nil && node.Enabled && node.Latency > 0 {
				if node.Latency < minLatency {
					minLatency = node.Latency
					bestNode = nodeID
				}
			}
		}
		return bestNode

	case "weight":
		// 权重策略
		var totalWeight int
		for _, nodeID := range group.Nodes {
			if node := pm.nodes[nodeID]; node != nil && node.Enabled {
				totalWeight += node.Weight
			}
		}

		if totalWeight == 0 {
			return 0
		}

		// 简单的权重选择
		rand := time.Now().Unix() % int64(totalWeight)
		currentWeight := 0
		for _, nodeID := range group.Nodes {
			if node := pm.nodes[nodeID]; node != nil && node.Enabled {
				currentWeight += node.Weight
				if int64(currentWeight) > rand {
					return nodeID
				}
			}
		}

	default:
		// 默认返回第一个可用节点
		for _, nodeID := range group.Nodes {
			if node := pm.nodes[nodeID]; node != nil && node.Enabled {
				return nodeID
			}
		}
	}

	return 0
}

// matchDomain 匹配域名规则
func (pm *ProxyManager) matchDomain(domain, pattern string) bool {
	// 简单的域名匹配，支持通配符
	if pattern == "" || domain == "" {
		return false
	}

	// 精确匹配
	if pattern == domain {
		return true
	}

	// 通配符匹配
	if len(pattern) > 1 && pattern[0] == '.' {
		return len(domain) >= len(pattern) && domain[len(domain)-len(pattern):] == pattern
	}

	return false
}

// matchIPCIDR 匹配IP CIDR规则
func (pm *ProxyManager) matchIPCIDR(ip net.IP, cidr string) bool {
	// 这里需要实现CIDR匹配逻辑
	// 暂时返回false
	return false
}

// startHealthCheck 启动健康检查
func (pm *ProxyManager) startHealthCheck() {
	pm.healthTicker = time.NewTicker(time.Duration(pm.config.TestInterval) * time.Second)

	go func() {
		for {
			select {
			case <-pm.healthTicker.C:
				pm.checkAllNodes()
			case <-pm.stopChan:
				return
			}
		}
	}()
}

// checkAllNodes 检查所有节点
func (pm *ProxyManager) checkAllNodes() {
	pm.mutex.RLock()
	nodes := make([]*ProxyNode, 0, len(pm.nodes))
	for _, node := range pm.nodes {
		if node.Enabled {
			nodes = append(nodes, node)
		}
	}
	pm.mutex.RUnlock()

	for _, node := range nodes {
		go pm.checkNode(node)
	}
}

// checkNode 检查单个节点
func (pm *ProxyManager) checkNode(node *ProxyNode) {
	start := time.Now()

	// 根据协议类型选择不同的健康检查方法
	switch node.Protocol {
	case ProxyHysteria2:
		pm.checkHysteria2Node(node, start)
	default:
		pm.checkStandardNode(node, start)
	}
}

// checkHysteria2Node 检查 Hysteria2 节点健康状态
func (pm *ProxyManager) checkHysteria2Node(node *ProxyNode, start time.Time) {
	// 创建 Hysteria2 拨号器
	dialer, err := pm.createHysteria2Dialer(node)
	if err != nil {
		log.Printf("创建 Hysteria2 拨号器失败: %v", err)
		pm.updateNodeStatus(node.ID, -1, true)
		return
	}
	
	// 测试连接
	conn, err := dialer.Dial("tcp", "www.google.com:80")
	if err != nil {
		log.Printf("Hysteria2 节点 %s 连接测试失败: %v", node.Name, err)
		pm.updateNodeStatus(node.ID, -1, true)
		return
	}
	defer conn.Close()
	
	latency := time.Since(start).Milliseconds()
	pm.updateNodeStatus(node.ID, latency, false)
	
	log.Printf("Hysteria2 节点 %s 健康检查通过，延迟: %dms", node.Name, latency)
}

// checkStandardNode 检查标准协议节点健康状态
func (pm *ProxyManager) checkStandardNode(node *ProxyNode, start time.Time) {
	client := pm.httpClients[node.ID]
	if client == nil {
		return
	}

	// 测试连接
	resp, err := client.Get("http://www.google.com")
	if err != nil {
		pm.updateNodeStatus(node.ID, -1, true)
		return
	}
	defer resp.Body.Close()

	latency := time.Since(start).Milliseconds()
	pm.updateNodeStatus(node.ID, latency, false)
}

// updateNodeStatus 更新节点状态
func (pm *ProxyManager) updateNodeStatus(nodeID int, latency int64, failed bool) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if node, exists := pm.nodes[nodeID]; exists {
		if failed {
			node.FailCount++
			node.Latency = -1
		} else {
			node.Latency = latency
			node.FailCount = 0
		}
		node.LastCheck = time.Now().Unix()
	}
}

// startHTTPProxy 启动HTTP代理服务器
func (pm *ProxyManager) startHTTPProxy() {
	// 这里实现HTTP代理服务器
	log.Printf("HTTP代理服务器启动在 %s", pm.config.ListenHTTP)
}

// startSOCKS5Proxy 启动SOCKS5代理服务器
func (pm *ProxyManager) startSOCKS5Proxy() {
	// 这里实现SOCKS5代理服务器
	log.Printf("SOCKS5代理服务器启动在 %s", pm.config.ListenSOCKS)
}

// createHysteria2Dialer 创建 Hysteria2 拨号器
func (pm *ProxyManager) createHysteria2Dialer(node *ProxyNode) (proxy.Dialer, error) {
	// 创建 Hysteria2 拨号器
	dialer := &hysteria2Dialer{
		node: node,
		pm:   pm,
	}
	
	// 验证 Hysteria2 配置
	if node.Hysteria2.Password == "" {
		return nil, fmt.Errorf("Hysteria2 节点 %s 缺少密码配置", node.Name)
	}
	
	// 记录节点信息
	log.Printf("创建 Hysteria2 拨号器: %s (%s:%d)", node.Name, node.Address, node.Port)
	
	return dialer, nil
}

// hysteria2Dialer Hysteria2 拨号器
type hysteria2Dialer struct {
	node *ProxyNode
	pm   *ProxyManager
}

// Dial 实现 proxy.Dialer 接口
func (h *hysteria2Dialer) Dial(network, addr string) (net.Conn, error) {
	// 实现 Hysteria2 连接逻辑
	// 由于 Go 标准库不直接支持 QUIC，这里使用 TCP 连接模拟
	// 在实际部署中，可以集成 quic-go 库实现真正的 QUIC 连接
	
	// 创建到 Hysteria2 服务器的连接
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", h.node.Address, h.node.Port))
	if err != nil {
		return nil, fmt.Errorf("连接 Hysteria2 服务器失败: %v", err)
	}
	
	// 记录连接信息
	log.Printf("Hysteria2 连接建立: %s -> %s:%d", addr, h.node.Address, h.node.Port)
	
	// 这里应该实现 Hysteria2 协议握手
	// 暂时返回原始连接，实际使用时需要实现完整的协议栈
	
	return conn, nil
}
