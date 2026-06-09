package log

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

var LevelNames = []string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR", "FATAL", "PANIC", "JSON"}

// Logger 日志接口
type Logger interface {
	Json(args ...any)
	Trace(args ...any)
	Debug(args ...any)
	Info(args ...any)
	Warn(args ...any)
	Error(args ...any) error
	Fatal(args ...any)
	Panic(args ...any)
	Tracef(format string, args ...any)
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any) error
	Fatalf(format string, args ...any)
	Panicf(format string, args ...any)
	WithFields(fields ...any)
}

// LoggerImpl Logger接口的实现
type LoggerImpl struct {
	outputManager *OutputManager
	options       *logOptions
	isClosed      int32
	mu            sync.RWMutex
}

// 全局默认logger实例
var defaultLogger *LoggerImpl
var once sync.Once

// getDefaultLogger 获取默认logger实例（单例模式）
func getDefaultLogger() *LoggerImpl {
	once.Do(func() {
		var err error
		defaultLogger, err = NewLogger(WithSkipCallerNum(5))
		if err != nil {
			// 如果创建失败，创建一个最基本的logger
			defaultLogger = &LoggerImpl{
				options: options,
			}
			// 尝试创建一个基本的输出管理器
			if manager, err := NewOutputManager(options); err == nil {
				defaultLogger.outputManager = manager
			}
		}
	})
	return defaultLogger
}

// NewLogger 创建新的Logger实例
func NewLogger(opts ...Option) (*LoggerImpl, error) {
	// 复制全局配置
	loggerOptions := &logOptions{
		level:          options.level,
		logFilePath:    options.logFilePath,
		logFileSize:    options.logFileSize,
		maxBackups:     options.maxBackups,
		disableConsole: options.disableConsole,
		showProcess:    options.showProcess,
		showRoutine:    options.showRoutine,
		showCaller:     options.showCaller,
		showColor:      options.showColor,
		showStack:      options.showStack,
		skipCallerNum:  options.skipCallerNum,
		jsonFormatter:  options.jsonFormatter,
		fields:         options.fields,
	}

	// 应用选项
	for _, opt := range opts {
		opt(loggerOptions)
	}

	// 创建输出管理器
	outputManager, err := NewOutputManager(loggerOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create output manager: %w", err)
	}

	return &LoggerImpl{
		outputManager: outputManager,
		options:       loggerOptions,
	}, nil
}

// 包级别的便捷函数，使用默认logger

// Json 输出JSON格式日志
func Json(args ...any) {
	getDefaultLogger().Json(args...)
}

// Trace 输出TRACE级别日志
func Trace(args ...any) {
	getDefaultLogger().Trace(args...)
}

// Debug 输出DEBUG级别日志
func Debug(args ...any) {
	getDefaultLogger().Debug(args...)
}

// Info 输出INFO级别日志
func Info(args ...any) {
	getDefaultLogger().Info(args...)
}

// Warn 输出WARN级别日志
func Warn(args ...any) {
	getDefaultLogger().Warn(args...)
}

// Error 输出ERROR级别日志
func Error(args ...any) error {
	return getDefaultLogger().Error(args...)
}

// Fatal 输出FATAL级别日志
func Fatal(args ...any) {
	getDefaultLogger().Fatal(args...)
}

// Panic 输出PANIC级别日志
func Panic(args ...any) {
	getDefaultLogger().Panic(args...)
}

// Tracef 输出格式化TRACE级别日志
func Tracef(format string, args ...any) {
	getDefaultLogger().Tracef(format, args...)
}

// Debugf 输出格式化DEBUG级别日志
func Debugf(format string, args ...any) {
	getDefaultLogger().Debugf(format, args...)
}

// Infof 输出格式化INFO级别日志
func Infof(format string, args ...any) {
	getDefaultLogger().Infof(format, args...)
}

// Warnf 输出格式化WARN级别日志
func Warnf(format string, args ...any) {
	getDefaultLogger().Warnf(format, args...)
}

// Errorf 输出格式化ERROR级别日志
func Errorf(format string, args ...any) error {
	return getDefaultLogger().Errorf(format, args...)
}

// Fatalf 输出格式化FATAL级别日志
func Fatalf(format string, args ...any) {
	getDefaultLogger().Fatalf(format, args...)
}

// Panicf 输出格式化PANIC级别日志
func Panicf(format string, args ...any) {
	getDefaultLogger().Panicf(format, args...)
}

// LoggerImpl的方法实现

func (l *LoggerImpl) WithFields(fields ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, v := range fields {
		if i%2 == 0 && i+1 < len(fields) {
			l.options.fields[strings.TrimSpace(fmt.Sprint(v))] = fields[i+1]
		}
	}
}

// Json 输出JSON格式日志
func (l *LoggerImpl) Json(args ...any) {
	if l.isClosed != 0 || !l.isLevelEnabled(LevelInfo) {
		return
	}
	var message = l.marshalJson(args...)
	l.logWithLevel(LevelInfo, message)
}

func (l *LoggerImpl) marshalJson(args ...any) (message string) {
	if len(args) == 1 {
		// 单个参数
		if jsonData, err := json.Marshal(args[0]); err == nil {
			message = string(jsonData)
		} else {
			message = fmt.Sprintf("%+v", args[0])
		}
	} else {
		// 多个参数序列化为JSON数组
		if jsonData, err := json.Marshal(args); err == nil {
			message = string(jsonData)
		} else {
			message = fmt.Sprintf("%+v", args)
		}
	}
	return message
}

// Trace 输出TRACE级别日志
func (l *LoggerImpl) Trace(args ...any) {
	if l.isClosed != 0 || !l.isLevelEnabled(LevelTrace) {
		return
	}

	message := fmt.Sprint(args...)
	l.logWithLevel(LevelTrace, message)
}

// Debug 输出DEBUG级别日志
func (l *LoggerImpl) Debug(args ...any) {
	if l.isClosed != 0 || !l.isLevelEnabled(LevelDebug) {
		return
	}

	message := fmt.Sprint(args...)
	l.logWithLevel(LevelDebug, message)
}

// Info 输出INFO级别日志
func (l *LoggerImpl) Info(args ...any) {
	if l.isClosed != 0 || !l.isLevelEnabled(LevelInfo) {
		return
	}

	message := fmt.Sprint(args...)
	l.logWithLevel(LevelInfo, message)
}

// Warn 输出WARN级别日志
func (l *LoggerImpl) Warn(args ...any) {
	if l.isClosed != 0 || !l.isLevelEnabled(LevelWarn) {
		return
	}

	message := fmt.Sprint(args...)
	l.logWithLevel(LevelWarn, message)
}

// Error 输出ERROR级别日志并返回错误
func (l *LoggerImpl) Error(args ...any) error {
	var message string
	if l.isClosed == 0 && l.isLevelEnabled(LevelError) {
		message = l.logWithLevel(LevelError, "", args...)
	}
	return fmt.Errorf(message)
}

// Fatal 输出FATAL级别日志并返回错误
func (l *LoggerImpl) Fatal(args ...any) {
	message := fmt.Sprint(args...)
	if l.isClosed == 0 && l.isLevelEnabled(LevelFatal) {
		l.logWithLevel(LevelFatal, message)
	}
}

// Panic 输出PANIC级别日志并触发panic
func (l *LoggerImpl) Panic(args ...any) {
	message := fmt.Sprint(args...)

	if l.isClosed == 0 && l.isLevelEnabled(LevelPanic) {
		l.logWithLevel(LevelPanic, message)
	}

	panic(message)
}

// Tracef 输出格式化TRACE级别日志
func (l *LoggerImpl) Tracef(format string, args ...any) {
	if l.isClosed != 0 || !l.isLevelEnabled(LevelTrace) {
		return
	}

	l.logWithLevel(LevelTrace, format, args...)
}

// Debugf 输出格式化DEBUG级别日志
func (l *LoggerImpl) Debugf(format string, args ...any) {
	if l.isClosed != 0 || !l.isLevelEnabled(LevelDebug) {
		return
	}

	l.logWithLevel(LevelDebug, format, args...)
}

// Infof 输出格式化INFO级别日志
func (l *LoggerImpl) Infof(format string, args ...any) {
	if l.isClosed != 0 || !l.isLevelEnabled(LevelInfo) {
		return
	}
	l.logWithLevel(LevelInfo, format, args...)
}

// Warnf 输出格式化WARN级别日志并返回错误
func (l *LoggerImpl) Warnf(format string, args ...any) {
	if l.isClosed == 0 && l.isLevelEnabled(LevelWarn) {
		l.logWithLevel(LevelWarn, format, args...)
	}
}

// Errorf 输出格式化ERROR级别日志并返回错误
func (l *LoggerImpl) Errorf(format string, args ...any) error {
	var message string
	if l.isClosed == 0 && l.isLevelEnabled(LevelError) {
		message = l.logWithLevel(LevelError, format, args...)
	}
	return fmt.Errorf("%s", message)
}

// Fatalf 输出格式化FATAL级别日志
func (l *LoggerImpl) Fatalf(format string, args ...any) {
	if l.isClosed == 0 && l.isLevelEnabled(LevelFatal) {
		l.logWithLevel(LevelFatal, format, args...)
		os.Exit(-99)
	}
}

// Panicf 输出格式化PANIC级别日志并触发panic
func (l *LoggerImpl) Panicf(format string, args ...any) {
	var message string
	if l.isClosed == 0 && l.isLevelEnabled(LevelPanic) {
		message = l.logWithLevel(LevelPanic, format, args...)
	}
	panic(message)
}

// logWithLevel 核心日志写入方法
func (l *LoggerImpl) logWithLevel(level int, format any, args ...any) string {
	if l.outputManager == nil {
		return ""
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	var message string
	switch format.(type) {
	case string:
		message = format.(string)
		if format != "" {
			message = fmt.Sprintf(message, args...)
		} else {
			message = fmt.Sprint(args...)
		}
	case error:
		err := format.(error)
		message = err.Error()
	}
	// 创建日志元数据
	var metadata *LogMetadata
	if l.options.jsonFormatter && len(args) > 0 {
		// JSON级别特殊处理
		metadata = newLogMetadata(level, "", l.options)
		// 将参数序列化为JSON字符串存储在msg字段中
		if len(args) == 1 {
			arg := args[0]
			switch v := arg.(type) {
			case string:
				metadata.Message = v
			default:
				if jsonData, err := json.Marshal(v); err == nil {
					metadata.Message = string(jsonData)
				} else {
					metadata.Message = fmt.Sprintf("%+v", v)
				}
			}
		} else {
			// 多个参数序列化为JSON数组
			if jsonData, err := json.Marshal(args); err == nil {
				metadata.Message = string(jsonData)
			} else {
				metadata.Message = fmt.Sprintf("%+v", args)
			}
		}
	} else {
		metadata = newLogMetadata(level, message, l.options)
	}

	// 写入日志
	if err := l.outputManager.Write(level, message, metadata); err != nil {
		// 错误处理：输出到stderr，但不影响程序继续执行
		fmt.Fprintf(os.Stderr, "Logger output message error: %v\n", err)
	}
	return message
}

// isLevelEnabled 检查日志级别是否启用
func (l *LoggerImpl) isLevelEnabled(level int) bool {
	return level >= l.options.level
}

// Close 关闭logger
func (l *LoggerImpl) Close() error {
	if !atomic.CompareAndSwapInt32(&l.isClosed, 0, 1) {
		return nil // 已经关闭
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.outputManager != nil {
		return l.outputManager.Close()
	}
	return nil
}

// SetLevel 设置日志级别
func (l *LoggerImpl) SetLevel(level int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level >= LevelTrace {
		l.options.level = level
	}
}

// GetLevel 获取当前日志级别
func (l *LoggerImpl) GetLevel() int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.options.level
}

// SetOptions 更新logger配置
func (l *LoggerImpl) SetOptions(opts ...Option) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 应用选项
	for _, opt := range opts {
		opt(l.options)
	}

	// 重新创建输出管理器
	if l.outputManager != nil {
		l.outputManager.Close()
	}

	var err error
	l.outputManager, err = NewOutputManager(l.options)
	return err
}

// GetLevelName 获取级别名称
func GetLevelName(level int) string {
	if level >= 0 && level < len(LevelNames) {
		return LevelNames[level]
	}
	return "UNKNOWN"
}

// ParseLevel 解析级别字符串
func ParseLevel(levelStr string) int {
	levelStr = strings.ToUpper(strings.TrimSpace(levelStr))
	for i, name := range LevelNames {
		if name == levelStr {
			return i
		}
	}
	// 支持别名
	switch levelStr {
	case "WARNING":
		return LevelWarn
	default:
		return LevelInfo // 默认级别
	}
}

// Flush 强制刷新所有缓冲区
func (l *LoggerImpl) Flush() error {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.outputManager == nil {
		return nil
	}

	// 如果输出管理器有文件写入器，同步文件
	if l.outputManager.fileWriter != nil && l.outputManager.fileWriter.rotator != nil {
		return l.outputManager.fileWriter.rotator.Sync()
	}

	return nil
}

// 包级别的配置和管理函数

// SetGlobalLevel 设置全局日志级别
func SetGlobalLevel(level int) {
	getDefaultLogger().SetLevel(level)
}

// GetGlobalLevel 获取全局日志级别
func GetGlobalLevel() int {
	return getDefaultLogger().GetLevel()
}

// Flush 刷新全局logger
func Flush() error {
	return getDefaultLogger().Flush()
}

// Close 关闭全局logger
func Close() error {
	if defaultLogger != nil {
		return defaultLogger.Close()
	}
	return nil
}
