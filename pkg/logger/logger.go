package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Level 日志级别
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String 返回日志级别的字符串表示
func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger 日志记录器
type Logger struct {
	level      Level
	output     io.Writer
	format     string
	maxSize    int
	maxBackups int
	maxAge     int
	file       *os.File
	prefix     string
}

// Config 日志配置
type Config struct {
	Level      Level  `yaml:"level"`
	Format     string `yaml:"format"`
	Output     string `yaml:"output"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
	Prefix     string `yaml:"prefix"`
}

// NewLogger 创建新的日志记录器
func NewLogger(config *Config) (*Logger, error) {
	logger := &Logger{
		level:      config.Level,
		format:     config.Format,
		maxSize:    config.MaxSize,
		maxBackups: config.MaxBackups,
		maxAge:     config.MaxAge,
		prefix:     config.Prefix,
	}

	// 设置输出
	if err := logger.setOutput(config.Output); err != nil {
		return nil, err
	}

	return logger, nil
}

// setOutput 设置日志输出
func (l *Logger) setOutput(output string) error {
	switch output {
	case "stdout":
		l.output = os.Stdout
	case "stderr":
		l.output = os.Stderr
	case "file":
		return l.setFileOutput()
	default:
		// 尝试作为文件路径处理
		return l.setFileOutput(output)
	}
	return nil
}

// setFileOutput 设置文件输出
func (l *Logger) setFileOutput(filePath ...string) error {
	path := "logs/boomdns.log"
	if len(filePath) > 0 && filePath[0] != "" {
		path = filePath[0]
	}

	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 打开文件
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %v", err)
	}

	l.file = file
	l.output = file

	// 设置日志轮转
	go l.rotateLog()

	return nil
}

// rotateLog 日志轮转
func (l *Logger) rotateLog() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		if l.file != nil {
			// 检查文件大小
			if info, err := l.file.Stat(); err == nil {
				if info.Size() > int64(l.maxSize*1024*1024) {
					l.rotate()
				}
			}
		}
	}
}

// rotate 执行日志轮转
func (l *Logger) rotate() {
	if l.file == nil {
		return
	}

	// 关闭当前文件
	l.file.Close()

	// 重命名旧文件
	oldPath := l.file.Name()
	backupPath := oldPath + "." + time.Now().Format("2006-01-02-15-04-05")
	os.Rename(oldPath, backupPath)

	// 打开新文件
	file, err := os.OpenFile(oldPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		l.file = file
		l.output = file
	}

	// 清理旧文件
	l.cleanOldLogs()
}

// cleanOldLogs 清理旧日志文件
func (l *Logger) cleanOldLogs() {
	if l.file == nil {
		return
	}

	dir := filepath.Dir(l.file.Name())
	files, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	// 按修改时间排序
	var logFiles []os.FileInfo
	for _, file := range files {
		if info, err := file.Info(); err == nil {
			if filepath.Ext(file.Name()) == ".log" {
				logFiles = append(logFiles, info)
			}
		}
	}

	// 删除超过最大备份数量的文件
	if len(logFiles) > l.maxBackups {
		// 这里可以实现更复杂的清理逻辑
		// 暂时简单删除最旧的文件
	}
}

// formatMessage 格式化日志消息
func (l *Logger) formatMessage(level Level, message string) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	switch l.format {
	case "json":
		return fmt.Sprintf(`{"timestamp":"%s","level":"%s","prefix":"%s","message":"%s"}`,
			timestamp, level.String(), l.prefix, message)
	case "text":
		return fmt.Sprintf("[%s] %s [%s] %s",
			timestamp, level.String(), l.prefix, message)
	default:
		return fmt.Sprintf("[%s] %s [%s] %s",
			timestamp, level.String(), l.prefix, message)
	}
}

// log 记录日志
func (l *Logger) log(level Level, message string) {
	if level < l.level {
		return
	}

	formattedMessage := l.formatMessage(level, message)

	if l.output != nil {
		fmt.Fprintln(l.output, formattedMessage)
	}
}

// Debug 记录调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, fmt.Sprintf(format, args...))
}

// Info 记录信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, fmt.Sprintf(format, args...))
}

// Warn 记录警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, fmt.Sprintf(format, args...))
}

// Error 记录错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, fmt.Sprintf(format, args...))
}

// Fatal 记录致命错误日志并退出
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(FATAL, fmt.Sprintf(format, args...))
	os.Exit(1)
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

// SetPrefix 设置日志前缀
func (l *Logger) SetPrefix(prefix string) {
	l.prefix = prefix
}

// Close 关闭日志记录器
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// GetLevel 获取日志级别
func (l *Logger) GetLevel() Level {
	return l.level
}

// IsDebug 检查是否为调试级别
func (l *Logger) IsDebug() bool {
	return l.level <= DEBUG
}

// IsInfo 检查是否为信息级别
func (l *Logger) IsInfo() bool {
	return l.level <= INFO
}

// IsWarn 检查是否为警告级别
func (l *Logger) IsWarn() bool {
	return l.level <= WARN
}

// IsError 检查是否为错误级别
func (l *Logger) IsError() bool {
	return l.level <= ERROR
}
