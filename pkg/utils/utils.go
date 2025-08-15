package utils

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// StringUtils 字符串工具函数
type StringUtils struct{}

// IsEmpty 检查字符串是否为空
func (s *StringUtils) IsEmpty(str string) bool {
	return strings.TrimSpace(str) == ""
}

// IsNotEmpty 检查字符串是否非空
func (s *StringUtils) IsNotEmpty(str string) bool {
	return !s.IsEmpty(str)
}

// Truncate 截断字符串到指定长度
func (s *StringUtils) Truncate(str string, maxLen int) string {
	if len(str) <= maxLen {
		return str
	}
	return str[:maxLen] + "..."
}

// Reverse 反转字符串
func (s *StringUtils) Reverse(str string) string {
	runes := []rune(str)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// NetworkUtils 网络工具函数
type NetworkUtils struct{}

// IsValidIP 检查是否为有效的 IP 地址
func (n *NetworkUtils) IsValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// IsValidPort 检查是否为有效的端口号
func (n *NetworkUtils) IsValidPort(port int) bool {
	return port > 0 && port <= 65535
}

// IsValidDomain 检查是否为有效的域名
func (n *NetworkUtils) IsValidDomain(domain string) bool {
	// 简单的域名验证
	pattern := `^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`
	matched, _ := regexp.MatchString(pattern, domain)
	return matched
}

// ResolveIP 解析域名到 IP 地址
func (n *NetworkUtils) ResolveIP(domain string) ([]string, error) {
	ips, err := net.LookupHost(domain)
	if err != nil {
		return nil, err
	}
	return ips, nil
}

// FileUtils 文件工具函数
type FileUtils struct{}

// EnsureDir 确保目录存在
func (f *FileUtils) EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// FileExists 检查文件是否存在
func (f *FileUtils) FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// IsDir 检查路径是否为目录
func (f *FileUtils) IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// GetFileSize 获取文件大小
func (f *FileUtils) GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// CopyFile 复制文件
func (f *FileUtils) CopyFile(src, dst string) error {
	// 读取源文件
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// 确保目标目录存在
	if err := f.EnsureDir(filepath.Dir(dst)); err != nil {
		return err
	}

	// 写入目标文件
	return os.WriteFile(dst, data, 0644)
}

// TimeUtils 时间工具函数
type TimeUtils struct{}

// FormatTime 格式化时间
func (t *TimeUtils) FormatTime(tm time.Time, format string) string {
	if format == "" {
		format = "2006-01-02 15:04:05"
	}
	return tm.Format(format)
}

// ParseTime 解析时间字符串
func (t *TimeUtils) ParseTime(timeStr, format string) (time.Time, error) {
	if format == "" {
		format = "2006-01-02 15:04:05"
	}
	return time.Parse(format, timeStr)
}

// GetTimestamp 获取时间戳
func (t *TimeUtils) GetTimestamp() int64 {
	return time.Now().Unix()
}

// GetTimestampNano 获取纳秒时间戳
func (t *TimeUtils) GetTimestampNano() int64 {
	return time.Now().UnixNano()
}

// Sleep 睡眠指定时间
func (t *TimeUtils) Sleep(duration time.Duration) {
	time.Sleep(duration)
}

// CryptoUtils 加密工具函数
type CryptoUtils struct{}

// MD5Hash 计算 MD5 哈希
func (c *CryptoUtils) MD5Hash(data string) string {
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// SHA256Hash 计算 SHA256 哈希
func (c *CryptoUtils) SHA256Hash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// ValidateUtils 验证工具函数
type ValidateUtils struct{}

// IsEmail 验证邮箱格式
func (v *ValidateUtils) IsEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

// IsPhone 验证手机号格式（中国）
func (v *ValidateUtils) IsPhone(phone string) bool {
	pattern := `^1[3-9]\d{9}$`
	matched, _ := regexp.MatchString(pattern, phone)
	return matched
}

// IsURL 验证 URL 格式
func (v *ValidateUtils) IsURL(url string) bool {
	pattern := `^(https?|ftp)://[^\s/$.?#].[^\s]*$`
	matched, _ := regexp.MatchString(pattern, url)
	return matched
}

// IsNumeric 验证是否为数字
func (v *ValidateUtils) IsNumeric(str string) bool {
	_, err := strconv.Atoi(str)
	return err == nil
}

// JSONUtils JSON 工具函数
type JSONUtils struct{}

// ToJSON 转换为 JSON 字符串
func (j *JSONUtils) ToJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ToPrettyJSON 转换为格式化的 JSON 字符串
func (j *JSONUtils) ToPrettyJSON(v interface{}) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON 从 JSON 字符串解析
func (j *JSONUtils) FromJSON(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}

// SliceUtils 切片工具函数
type SliceUtils struct{}

// Contains 检查切片是否包含元素
func (s *SliceUtils) Contains(slice []interface{}, item interface{}) bool {
	for _, element := range slice {
		if element == item {
			return true
		}
	}
	return false
}

// Remove 从切片中移除元素
func (s *SliceUtils) Remove(slice []interface{}, item interface{}) []interface{} {
	result := make([]interface{}, 0)
	for _, element := range slice {
		if element != item {
			result = append(result, element)
		}
	}
	return result
}

// Unique 去重切片
func (s *SliceUtils) Unique(slice []interface{}) []interface{} {
	seen := make(map[interface{}]bool)
	result := make([]interface{}, 0)

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// MapUtils Map 工具函数
type MapUtils struct{}

// GetString 获取字符串值
func (m *MapUtils) GetString(data map[string]interface{}, key string, defaultValue string) string {
	if value, exists := data[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

// GetInt 获取整数值
func (m *MapUtils) GetInt(data map[string]interface{}, key string, defaultValue int) int {
	if value, exists := data[key]; exists {
		switch v := value.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
	}
	return defaultValue
}

// GetBool 获取布尔值
func (m *MapUtils) GetBool(data map[string]interface{}, key string, defaultValue bool) bool {
	if value, exists := data[key]; exists {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// 全局工具实例
var (
	String   = &StringUtils{}
	Network  = &NetworkUtils{}
	File     = &FileUtils{}
	Time     = &TimeUtils{}
	Crypto   = &CryptoUtils{}
	Validate = &ValidateUtils{}
	JSON     = &JSONUtils{}
	Slice    = &SliceUtils{}
	Map      = &MapUtils{}
)

// 便捷函数
func IsEmpty(str string) bool {
	return String.IsEmpty(str)
}

func IsNotEmpty(str string) bool {
	return String.IsNotEmpty(str)
}

func IsValidIP(ip string) bool {
	return Network.IsValidIP(ip)
}

func IsValidPort(port int) bool {
	return Network.IsValidPort(port)
}

func IsValidDomain(domain string) bool {
	return Network.IsValidDomain(domain)
}

func EnsureDir(path string) error {
	return File.EnsureDir(path)
}

func FileExists(path string) bool {
	return File.FileExists(path)
}

func MD5Hash(data string) string {
	return Crypto.MD5Hash(data)
}

func SHA256Hash(data string) string {
	return Crypto.SHA256Hash(data)
}

func IsEmail(email string) bool {
	return Validate.IsEmail(email)
}

func IsPhone(phone string) bool {
	return Validate.IsPhone(phone)
}

func IsURL(url string) bool {
	return Validate.IsURL(url)
}

func ToJSON(v interface{}) (string, error) {
	return JSON.ToJSON(v)
}

func FromJSON(data string, v interface{}) error {
	return JSON.FromJSON(data, v)
}
