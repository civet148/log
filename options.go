package log

import (
	"fmt"
	"strconv"
	"strings"
)

type SizeMB = int

const (
	LevelTrace = 0
	LevelDebug = 1
	LevelInfo  = 2
	LevelWarn  = 3
	LevelError = 4
	LevelFatal = 5
	LevelPanic = 6
)

const (
	DefaultFileSize   = 1024 //MB
	DefaultMaxBackups = 31
)

type logOptions struct {
	level          int            //日志级别
	logFilePath    string         //日志输出文件路径
	logFileSize    SizeMB         //文件日志分割大小(MB)
	maxBackups     int            //日志文件最大备份数
	disableConsole bool           //是否禁止终端屏幕输出
	showProcess    bool           //是否显示进程ID
	showRoutine    bool           //是否显示协程ID
	showCaller     bool           //是否显示调用者信息
	showColor      bool           //是否显示颜色
	jsonFormatter  bool           //是否JSON格式输出
	skipCallerNum  int            //跳过调用者数
	fields         map[string]any //日志字段
}

var options = &logOptions{
	level:          LevelInfo,
	showColor:      true,
	showProcess:    false,
	showRoutine:    true,
	showCaller:     true,
	disableConsole: false,
	logFileSize:    DefaultFileSize,
	maxBackups:     DefaultMaxBackups,
	skipCallerNum:  4,
	jsonFormatter:  false,
	fields:         make(map[string]any),
}

func SetLevel(level any) {
	opf := WithLevel(level)
	opf(options)
}

func SetFileSize(size SizeMB) {
	opf := WithFileSize(size)
	opf(options)
}

func SetMaxBackups(maxBackups int) {
	opf := WithMaxBackups(maxBackups)
	opf(options)
}

func SetJsonFormatter() {
	opf := WithJsonFormatter()
	opf(options)
}

func SetFileName(filePath string) {
	options.logFilePath = filePath
}

func DisableColor() {
	options.showColor = false
}

func EnableProcess() {
	options.showProcess = true
}

func DisableRouine() {
	options.showRoutine = false
}

func DisableCaller() {
	options.showProcess = false
}

func DisableConsole() {
	options.disableConsole = true
}

/* --------------------------------------------------------------------------------------------------- */

type Option func(*logOptions)

func WithLevel(level any) Option {
	return func(o *logOptions) {
		var nLevel int
		switch level.(type) {
		case string:
			strLevel := strings.ToLower(level.(string))
			switch strLevel {
			case "t", "trace":
				nLevel = LevelTrace
			case "d", "debug":
				nLevel = LevelDebug
			case "i", "info":
				nLevel = LevelInfo
			case "w", "warn", "warning":
				nLevel = LevelWarn
			case "e", "error":
				nLevel = LevelError
			case "f", "fatal":
				nLevel = LevelFatal
			}
		case int8, int16, int, int32, int64, uint8, uint16, uint, uint32, uint64:
			nLevel, _ = strconv.Atoi(fmt.Sprintf("%v", level))
		default:
			nLevel = LevelInfo
		}
		o.level = nLevel
	}
}

// WithJsonFormatter 设置所有级别都采用JSON格式输出
func WithJsonFormatter() Option {
	return func(o *logOptions) {
		o.jsonFormatter = true
	}
}

func WithFileSize(size SizeMB) Option {
	return func(o *logOptions) {
		if size > 0 {
			o.logFileSize = size
		}
	}
}

func WithMaxBackups(maxBackups int) Option {
	return func(o *logOptions) {
		if maxBackups > 0 {
			o.maxBackups = maxBackups
		}
	}
}

func WithDisableConsole() Option {
	return func(o *logOptions) {
		o.disableConsole = true
	}
}

func WithDisableCaller() Option {
	return func(o *logOptions) {
		o.showCaller = false
	}
}

func WithDisableProcess() Option {
	return func(o *logOptions) {
		o.showProcess = false
	}
}

func WithDisableRoutine() Option {
	return func(o *logOptions) {
		o.showRoutine = false
	}
}

func WithDisableColor() Option {
	return func(o *logOptions) {
		o.showColor = false
	}
}

func WithLogFile(logFile string) Option {
	return func(o *logOptions) {
		o.logFilePath = logFile
	}
}

func WithSkipCallerNum(num int) Option {
	return func(o *logOptions) {
		o.skipCallerNum = num //default 4
	}
}
