package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 应用配置结构
type Config struct {
	// 基础配置
	App struct {
		Name        string `yaml:"name"`
		Version     string `yaml:"version"`
		Environment string `yaml:"environment"`
		Debug       bool   `yaml:"debug"`
	} `yaml:"app"`

	// 服务器配置
	Server struct {
		DNS  string `yaml:"dns"`
		HTTP string `yaml:"http"`
	} `yaml:"server"`

	// 数据库配置
	Database struct {
		Type       string `yaml:"type"`
		SQLiteFile string `yaml:"sqlite_file"`
		MaxConn    int    `yaml:"max_conn"`
		Timeout    int    `yaml:"timeout"`
	} `yaml:"database"`

	// 日志配置
	Logging struct {
		Level      string `yaml:"level"`
		Format     string `yaml:"format"`
		Output     string `yaml:"output"`
		MaxSize    int    `yaml:"max_size"`
		MaxBackups int    `yaml:"max_backups"`
		MaxAge     int    `yaml:"max_age"`
	} `yaml:"logging"`

	// 监控配置
	Monitoring struct {
		Enabled bool   `yaml:"enabled"`
		Port    string `yaml:"port"`
		Path    string `yaml:"path"`
	} `yaml:"monitoring"`

	// 安全配置
	Security struct {
		AdminToken string   `yaml:"admin_token"`
		AllowedIPs []string `yaml:"allowed_ips"`
		RateLimit  int      `yaml:"rate_limit"`
	} `yaml:"security"`
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	// 如果未指定配置文件，使用默认路径
	if configPath == "" {
		configPath = getDefaultConfigPath()
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("配置文件不存在: %s", configPath)
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 解析 YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	// 设置默认值
	setDefaults(&config)

	// 验证配置
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %v", err)
	}

	return &config, nil
}

// getDefaultConfigPath 获取默认配置文件路径
func getDefaultConfigPath() string {
	// 按优先级查找配置文件
	paths := []string{
		"configs/config.yaml",
		"config.yaml",
		"configs/config.dev.yaml",
		"configs/config.prod.yaml",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return "configs/config.yaml"
}

// setDefaults 设置默认配置值
func setDefaults(config *Config) {
	// 应用默认值
	if config.App.Name == "" {
		config.App.Name = "BoomDNS"
	}
	if config.App.Version == "" {
		config.App.Version = "1.0.0"
	}
	if config.App.Environment == "" {
		config.App.Environment = "development"
	}

	// 服务器默认值
	if config.Server.DNS == "" {
		config.Server.DNS = ":53"
	}
	if config.Server.HTTP == "" {
		config.Server.HTTP = ":8080"
	}

	// 数据库默认值
	if config.Database.Type == "" {
		config.Database.Type = "sqlite"
	}
	if config.Database.SQLiteFile == "" {
		config.Database.SQLiteFile = "data/boomdns.db"
	}
	if config.Database.MaxConn == 0 {
		config.Database.MaxConn = 10
	}
	if config.Database.Timeout == 0 {
		config.Database.Timeout = 30
	}

	// 日志默认值
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if config.Logging.Format == "" {
		config.Logging.Format = "json"
	}
	if config.Logging.Output == "" {
		config.Logging.Output = "logs/boomdns.log"
	}
	if config.Logging.MaxSize == 0 {
		config.Logging.MaxSize = 100
	}
	if config.Logging.MaxBackups == 0 {
		config.Logging.MaxBackups = 3
	}
	if config.Logging.MaxAge == 0 {
		config.Logging.MaxAge = 28
	}

	// 监控默认值
	if config.Monitoring.Port == "" {
		config.Monitoring.Port = ":9090"
	}
	if config.Monitoring.Path == "" {
		config.Monitoring.Path = "/metrics"
	}

	// 安全默认值
	if config.Security.AdminToken == "" {
		config.Security.AdminToken = "boomdns-secret-token-2024"
	}
	if config.Security.RateLimit == 0 {
		config.Security.RateLimit = 1000
	}
}

// validateConfig 验证配置
func validateConfig(config *Config) error {
	// 验证服务器配置
	if config.Server.DNS == "" {
		return fmt.Errorf("DNS 服务器端口不能为空")
	}
	if config.Server.HTTP == "" {
		return fmt.Errorf("HTTP 服务器端口不能为空")
	}

	// 验证数据库配置
	if config.Database.Type == "" {
		return fmt.Errorf("数据库类型不能为空")
	}
	if config.Database.SQLiteFile == "" {
		return fmt.Errorf("SQLite 文件路径不能为空")
	}

	// 验证日志配置
	if config.Logging.Level == "" {
		return fmt.Errorf("日志级别不能为空")
	}
	if !isValidLogLevel(config.Logging.Level) {
		return fmt.Errorf("无效的日志级别: %s", config.Logging.Level)
	}

	// 验证安全配置
	if config.Security.AdminToken == "" {
		return fmt.Errorf("管理员令牌不能为空")
	}

	return nil
}

// isValidLogLevel 验证日志级别
func isValidLogLevel(level string) bool {
	validLevels := []string{"debug", "info", "warn", "error", "fatal"}
	level = strings.ToLower(level)
	for _, valid := range validLevels {
		if level == valid {
			return true
		}
	}
	return false
}

// SaveConfig 保存配置到文件
func SaveConfig(config *Config, configPath string) error {
	// 如果未指定路径，使用默认路径
	if configPath == "" {
		configPath = getDefaultConfigPath()
	}

	// 确保目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}

	// 序列化配置
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	// 写入文件
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	return nil
}

// GetConfigPath 获取配置文件路径
func GetConfigPath() string {
	return getDefaultConfigPath()
}

// IsDevelopment 检查是否为开发环境
func (c *Config) IsDevelopment() bool {
	return c.App.Environment == "development"
}

// IsProduction 检查是否为生产环境
func (c *Config) IsProduction() bool {
	return c.App.Environment == "production"
}

// IsDebug 检查是否启用调试模式
func (c *Config) IsDebug() bool {
	return c.App.Debug
}
