package dns

import (
	"time"
)

// SubscriptionRuleSource 订阅规则源配置
type SubscriptionRuleSource struct {
	Name    string `yaml:"name"`    // 规则源名称
	URL     string `yaml:"url"`     // 规则源URL
	Format  string `yaml:"format"`  // 规则格式
	Enabled bool   `yaml:"enabled"` // 是否启用
}

// 注意：ProxyNode、ProxyGroup、ProxyRule 类型已在 dns/proxy.go 中定义

// Config DNS服务器配置
type Config struct {
	ListenDNS  string `yaml:"listen_dns"`
	ListenHTTP string `yaml:"listen_http"`
	AdminToken string `yaml:"admin_token"`

	// 上游DNS服务器配置
	Upstreams struct {
		China   []string `yaml:"china"`
		Intl    []string `yaml:"intl"`
		Adguard []string `yaml:"adguard"`
	} `yaml:"upstreams"`

	// 域名规则配置
	Domains struct {
		China []string `yaml:"china"`
		GFW   []string `yaml:"gfw"`
		Ads   []string `yaml:"ads"`
	} `yaml:"domains"`

	// 规则同步配置
	Sync struct {
		Enabled  bool              `yaml:"enabled"`
		Interval int               `yaml:"interval"`
		Sources  map[string]string `yaml:"sources"`
	} `yaml:"sync"`

	// 规则订阅配置
	Subscriptions struct {
		Enabled        bool                                `yaml:"enabled"`
		UpdateInterval int                                 `yaml:"update_interval"`
		Timeout        int                                 `yaml:"timeout"`
		RetryCount     int                                 `yaml:"retry_count"`
		UserAgent      string                              `yaml:"user_agent"`
		Sources        map[string][]SubscriptionRuleSource `yaml:"sources"`
	} `yaml:"subscriptions"`

	// 代理配置
	Proxy struct {
		Enabled         bool   `yaml:"enabled"`
		ListenHTTP      string `yaml:"listen_http"`
		ListenSOCKS     string `yaml:"listen_socks"`
		DefaultStrategy string `yaml:"default_strategy"`
		TestInterval    int    `yaml:"test_interval"`
		TestTimeout     int    `yaml:"test_timeout"`
	} `yaml:"proxy"`

	// 代理节点配置
	ProxyNodes []ProxyNode `yaml:"proxy_nodes,omitempty"`

	// 代理组配置
	ProxyGroups []ProxyGroup `yaml:"proxy_groups,omitempty"`

	// 代理规则配置
	ProxyRules []ProxyRule `yaml:"proxy_rules,omitempty"`

	// 数据持久化配置
	Persistence struct {
		Enabled  bool   `yaml:"enabled"`
		DataDir  string `yaml:"data_dir"`
		Database struct {
			Type       string `yaml:"type"`
			SQLiteFile string `yaml:"sqlite_file"`
			CacheFile  string `yaml:"cache_file"`
			LogsFile   string `yaml:"logs_file"`
			StatsFile  string `yaml:"stats_file"`
			RulesFile  string `yaml:"rules_file"`
		} `yaml:"database"`
		AutoSaveInterval int `yaml:"auto_save_interval"`
		MaxLogs          int `yaml:"max_logs"`
		MaxCacheEntries  int `yaml:"max_cache_entries"`
	} `yaml:"persistence"`
}

// GetChinaUpstreams 获取中国上游DNS服务器
func (c *Config) GetChinaUpstreams() []string {
	return c.Upstreams.China
}

// GetIntlUpstreams 获取国际上游DNS服务器
func (c *Config) GetIntlUpstreams() []string {
	return c.Upstreams.Intl
}

// GetAdguardUpstreams 获取AdGuard上游DNS服务器
func (c *Config) GetAdguardUpstreams() []string {
	return c.Upstreams.Adguard
}

// GetChinaDomains 获取中国域名列表
func (c *Config) GetChinaDomains() []string {
	return c.Domains.China
}

// GetGFWDomains 获取GFW域名列表
func (c *Config) GetGFWDomains() []string {
	return c.Domains.GFW
}

// GetAdsDomains 获取广告域名列表
func (c *Config) GetAdsDomains() []string {
	return c.Domains.Ads
}

// GetSyncInterval 获取同步间隔
func (c *Config) GetSyncInterval() time.Duration {
	if c.Sync.Interval <= 0 {
		return time.Hour // 默认1小时
	}
	return time.Duration(c.Sync.Interval) * time.Second
}

// IsPersistenceEnabled 是否启用数据持久化
func (c *Config) IsPersistenceEnabled() bool {
	return c.Persistence.Enabled
}

// GetDataDir 获取数据目录
func (c *Config) GetDataDir() string {
	if c.Persistence.DataDir == "" {
		return "./data"
	}
	return c.Persistence.DataDir
}

// IsSubscriptionsEnabled 是否启用规则订阅
func (c *Config) IsSubscriptionsEnabled() bool {
	return c.Subscriptions.Enabled
}

// GetSubscriptionsUpdateInterval 获取订阅更新间隔
func (c *Config) GetSubscriptionsUpdateInterval() time.Duration {
	if c.Subscriptions.UpdateInterval <= 0 {
		return time.Hour // 默认1小时
	}
	return time.Duration(c.Subscriptions.UpdateInterval) * time.Second
}

// GetSubscriptionsTimeout 获取订阅请求超时
func (c *Config) GetSubscriptionsTimeout() time.Duration {
	if c.Subscriptions.Timeout <= 0 {
		return 5 * time.Minute // 默认5分钟
	}
	return time.Duration(c.Subscriptions.Timeout) * time.Second
}

// GetSubscriptionsRetryCount 获取订阅重试次数
func (c *Config) GetSubscriptionsRetryCount() int {
	if c.Subscriptions.RetryCount <= 0 {
		return 3 // 默认3次
	}
	return c.Subscriptions.RetryCount
}

// GetSubscriptionsUserAgent 获取订阅用户代理
func (c *Config) GetSubscriptionsUserAgent() string {
	if c.Subscriptions.UserAgent == "" {
		return "BoomDNS/1.0"
	}
	return c.Subscriptions.UserAgent
}

// GetAutoSaveInterval 获取自动保存间隔
func (c *Config) GetAutoSaveInterval() time.Duration {
	if c.Persistence.AutoSaveInterval <= 0 {
		return 5 * time.Minute // 默认5分钟
	}
	return time.Duration(c.Persistence.AutoSaveInterval) * time.Second
}

// GetMaxLogs 获取最大日志条数
func (c *Config) GetMaxLogs() int {
	if c.Persistence.MaxLogs <= 0 {
		return 10000
	}
	return c.Persistence.MaxLogs
}

// GetMaxCacheEntries 获取最大缓存条目数
func (c *Config) GetMaxCacheEntries() int {
	if c.Persistence.MaxCacheEntries <= 0 {
		return 10000
	}
	return c.Persistence.MaxCacheEntries
}

// GetDatabaseType 获取数据库类型
func (c *Config) GetDatabaseType() string {
	if c.Persistence.Database.Type == "" {
		return "sqlite" // 默认使用 SQLite
	}
	return c.Persistence.Database.Type
}

// IsSQLiteEnabled 是否启用 SQLite
func (c *Config) IsSQLiteEnabled() bool {
	return c.GetDatabaseType() == "sqlite"
}

// GetSQLiteFile 获取 SQLite 数据库文件路径
func (c *Config) GetSQLiteFile() string {
	if c.Persistence.Database.SQLiteFile == "" {
		return "boomdns.db"
	}
	return c.Persistence.Database.SQLiteFile
}

// 代理配置辅助方法
// IsProxyEnabled 是否启用代理功能
func (c *Config) IsProxyEnabled() bool {
	return c.Proxy.Enabled
}

// GetProxyListenHTTP 获取HTTP代理监听地址
func (c *Config) GetProxyListenHTTP() string {
	if c.Proxy.ListenHTTP == "" {
		return ":7890"
	}
	return c.Proxy.ListenHTTP
}

// GetProxyListenSOCKS 获取SOCKS5代理监听地址
func (c *Config) GetProxyListenSOCKS() string {
	if c.Proxy.ListenSOCKS == "" {
		return ":7891"
	}
	return c.Proxy.ListenSOCKS
}

// GetProxyDefaultStrategy 获取默认代理策略
func (c *Config) GetProxyDefaultStrategy() string {
	if c.Proxy.DefaultStrategy == "" {
		return "round-robin"
	}
	return c.Proxy.DefaultStrategy
}

// GetProxyTestInterval 获取代理测试间隔
func (c *Config) GetProxyTestInterval() int {
	if c.Proxy.TestInterval <= 0 {
		return 300
	}
	return c.Proxy.TestInterval
}

// GetProxyTestTimeout 获取代理测试超时
func (c *Config) GetProxyTestTimeout() int {
	if c.Proxy.TestTimeout <= 0 {
		return 10
	}
	return c.Proxy.TestTimeout
}
