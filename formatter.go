package log

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	DateTimeFormat = "2006-01-02 15:04:05.000"
)

const (
	fieldTitle = "#FIELDS#"
)

// Formatter 格式化器接口
type Formatter interface {
	Format(level int, message string, metadata *LogMetadata) string
}

// ColorFormatter 彩色格式化器
type ColorFormatter struct {
	colorMap    map[int]string
	resetColor  string
	showColor   bool
	showProcess bool
	showRoutine bool
	showCaller  bool
	showStack   bool
}

// PlainFormatter 纯文本格式化器
type PlainFormatter struct {
	showProcess bool
	showRoutine bool
	showCaller  bool
	showStack   bool
}

// JSONFormatter JSON格式化器
type JSONFormatter struct {
	prettyPrint bool
	timeFormat  string
}

// 颜色代码常量
const (
	ColorReset = "\033[0m"
	ColorTrace = "\033[37m" // 白色
	ColorDebug = "\033[36m" // 青色
	ColorInfo  = "\033[32m" // 绿色
	ColorWarn  = "\033[33m" // 黄色
	ColorError = "\033[31m" // 红色
	ColorFatal = "\033[35m" // 紫色
	ColorPanic = "\033[41m" // 红色背景
	ColorJSON  = "\033[34m" // 蓝色
)

// NewColorFormatter 创建彩色格式化器
func NewColorFormatter(showColor, showProcess, showRoutine, showCaller, showStack bool) *ColorFormatter {
	formatter := &ColorFormatter{
		colorMap: map[int]string{
			LevelTrace: ColorTrace,
			LevelDebug: ColorDebug,
			LevelInfo:  ColorInfo,
			LevelWarn:  ColorWarn,
			LevelError: ColorError,
			LevelFatal: ColorFatal,
			LevelPanic: ColorPanic,
		},
		resetColor:  ColorReset,
		showColor:   showColor && isTerminalColorSupported(),
		showProcess: showProcess,
		showRoutine: showRoutine,
		showCaller:  showCaller,
		showStack:   showStack,
	}
	return formatter
}

// Format 格式化日志消息
func (f *ColorFormatter) Format(level int, message string, metadata *LogMetadata) string {
	var parts []string

	// 时间戳
	timeStr := metadata.Timestamp.Format(DateTimeFormat)
	parts = append(parts, timeStr)

	// 日志级别
	levelStr := fmt.Sprintf("[%s]", metadata.Level)
	if f.showColor {
		if colorCode, ok := f.colorMap[level]; ok {
			levelStr = colorCode + levelStr + f.resetColor
		}
	}
	parts = append(parts, levelStr)

	// 进程ID
	if f.showProcess && metadata.ProcessID != 0 {
		parts = append(parts, fmt.Sprintf("[PID:%d]", metadata.ProcessID))
	}

	// 协程ID
	if f.showRoutine && metadata.RoutineID != 0 {
		parts = append(parts, fmt.Sprintf("[GID:%d]", metadata.RoutineID))
	}

	// 调用者信息
	if f.showCaller && metadata.Caller != "" {
		parts = append(parts, fmt.Sprintf("[CALLER:%s]", metadata.Caller))
	}

	// 消息内容
	parts = append(parts, message)
	if len(metadata.Fields) > 0 {
		parts = append(parts, formatFields(metadata.Fields))
	}

	// 打印堆栈
	if f.showStack {
		parts = append(parts, fmt.Sprintf("#STACK# %s", getCallStack(level)))
	}

	return strings.Join(parts, " ")
}

// getColorCode 获取级别对应的颜色代码
func (f *ColorFormatter) getColorCode(level int) string {
	if !f.showColor {
		return ""
	}
	if colorCode, ok := f.colorMap[level]; ok {
		return colorCode
	}
	return ""
}

// NewPlainFormatter 创建纯文本格式化器
func NewPlainFormatter(showProcess, showRoutine, showCaller, showStack bool) *PlainFormatter {
	return &PlainFormatter{
		showProcess: showProcess,
		showRoutine: showRoutine,
		showCaller:  showCaller,
		showStack:   showStack,
	}
}

// Format 格式化日志消息（纯文本）
func (f *PlainFormatter) Format(level int, message string, metadata *LogMetadata) string {
	var parts []string

	// 时间戳
	timeStr := metadata.Timestamp.Format(DateTimeFormat)
	parts = append(parts, timeStr)

	// 日志级别
	levelStr := fmt.Sprintf("[%s]", metadata.Level)
	parts = append(parts, levelStr)

	// 进程ID
	if f.showProcess && metadata.ProcessID != 0 {
		parts = append(parts, fmt.Sprintf("[PID:%d]", metadata.ProcessID))
	}

	// 协程ID
	if f.showRoutine && metadata.RoutineID != 0 {
		parts = append(parts, fmt.Sprintf("[GID:%d]", metadata.RoutineID))
	}

	// 调用者信息
	if f.showCaller && metadata.Caller != "" {
		parts = append(parts, fmt.Sprintf("[CALLER:%s]", metadata.Caller))
	}

	// 消息内容
	parts = append(parts, message)
	if len(metadata.Fields) > 0 {
		parts = append(parts, formatFields(metadata.Fields))
	}

	// 打印堆栈
	if f.showStack {
		parts = append(parts, fmt.Sprintf("#STACK# %s", getCallStack(level)))
	}

	return strings.Join(parts, " ")
}

func formatFields(fields map[string]any) string {
	data, _ := json.Marshal(fields)
	return fmt.Sprintf("%s %s", fieldTitle, data)
}

// NewJSONFormatter 创建JSON格式化器
func NewJSONFormatter(prettyPrint bool) *JSONFormatter {
	return &JSONFormatter{
		prettyPrint: prettyPrint,
		timeFormat:  DateTimeFormat,
	}
}

// Format 格式化为JSON格式
func (f *JSONFormatter) Format(level int, message string, metadata *LogMetadata) string {
	// 创建JSON日志条目
	entry := map[string]interface{}{
		"@timestamp": metadata.Timestamp.Format(f.timeFormat),
		"level":      metadata.Level,
		"msg":        message,
	}

	// 添加可选字段
	if metadata.ProcessID != 0 {
		entry["process_id"] = metadata.ProcessID
	}

	if metadata.RoutineID != 0 {
		entry["routine_id"] = metadata.RoutineID
	}

	if metadata.Caller != "" {
		entry["caller"] = metadata.Caller
	}

	if metadata.Message != "" {
		entry["msg"] = metadata.Message
	}
	if metadata.ShowStack {
		entry["stack"] = getCallStack(level)
	}
	for k, v := range metadata.Fields {
		entry[k] = v
	}

	// 序列化
	var data []byte

	if f.prettyPrint {
		data, _ = json.MarshalIndent(entry, "", " ")
	} else {
		data, _ = json.Marshal(entry)
	}

	return string(data)
}

// FormatJSONEntry 格式化JSON日志条目
func (f *JSONFormatter) FormatJSONEntry(entry *JSONLogEntry) string {
	var data []byte

	if f.prettyPrint {
		data, _ = json.MarshalIndent(entry, "", " ")
	} else {
		data, _ = json.Marshal(entry)
	}
	return string(data)
}

// isTerminalColorSupported 检查终端是否支持颜色
func isTerminalColorSupported() bool {
	// 检查TERM环境变量
	term := os.Getenv("TERM")
	if term == "" {
		return false
	}

	// 检查是否为常见的支持颜色的终端
	colorTerms := []string{
		"xterm", "xterm-color", "xterm-256color",
		"screen", "screen-256color",
		"tmux", "tmux-256color",
		"rxvt", "rxvt-unicode",
		"linux", "cygwin",
		"putty", "konsole",
		"gnome", "mate",
	}

	termLower := strings.ToLower(term)
	for _, colorTerm := range colorTerms {
		if strings.Contains(termLower, colorTerm) {
			return true
		}
	}

	// 检查NO_COLOR环境变量
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// 检查FORCE_COLOR环境变量
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}

	// 默认情况下，如果不是Windows且TERM不为空，则支持颜色
	return true
}

// escapeJSONString 转义JSON字符串中的特殊字符
func escapeJSONString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}

// validateJSONOutput 验证JSON输出格式
func validateJSONOutput(jsonStr string) bool {
	var temp interface{}
	return json.Unmarshal([]byte(jsonStr), &temp) == nil
}

// compactJSON 压缩JSON输出（移除不必要的空白）
func compactJSON(jsonStr string) string {
	var temp interface{}
	if err := json.Unmarshal([]byte(jsonStr), &temp); err != nil {
		return jsonStr // 解析失败，返回原字符串
	}

	data, err := json.Marshal(temp)
	if err != nil {
		return jsonStr // 序列化失败，返回原字符串
	}

	return string(data)
}

// formatTimestamp 格式化时间戳
func formatTimestamp(t time.Time, format string) string {
	if format == "" {
		format = DateTimeFormat
	}
	return t.Format(format)
}

// createFormatterChain 创建格式化器链，支持多种格式化器组合
func createFormatterChain(formatters ...Formatter) Formatter {
	return &FormatterChain{formatters: formatters}
}

// FormatterChain 格式化器链
type FormatterChain struct {
	formatters []Formatter
}

// Format 使用格式化器链处理日志
func (fc *FormatterChain) Format(level int, message string, metadata *LogMetadata) string {
	result := message
	for _, formatter := range fc.formatters {
		result = formatter.Format(level, result, metadata)
	}
	return result
}

// getCallStack 获取调用栈信息(从当前堆栈上一层往上最多取n层函数，用分号分隔)
func getCallStack(level int) (stacks string) {

	start := 6
	depth := 5
	var stackList []string

	if level > LevelWarn {
		depth = 10
	}

	for i := start; i < start+depth; i++ {
		pc, _, _, ok := runtime.Caller(i)
		if !ok {
			break
		}

		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		name := fn.Name()
		if strings.Contains(name, "/") {
			name = name[strings.LastIndex(name, "/")+1:]
		}
		stackList = append(stackList, name)
	}

	return strings.Join(stackList, ";")
}
