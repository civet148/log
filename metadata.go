package log

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"
)

// LogMetadata 日志元数据结构
type LogMetadata struct {
	Timestamp time.Time              `json:"@timestamp"`
	Level     string                 `json:"level"`
	ProcessID int                    `json:"process_id,omitempty"`
	RoutineID uint64                 `json:"routine_id,omitempty"`
	Caller    string                 `json:"caller,omitempty"`
	Message   string                 `json:"msg"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// JSONLogEntry JSON专用日志条目
type JSONLogEntry struct {
	Timestamp string                 `json:"@timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"msg"`
	Value     interface{}            `json:"value,omitempty"`
	ProcessID int                    `json:"process_id,omitempty"`
	RoutineID uint64                 `json:"routine_id,omitempty"`
	Caller    string                 `json:"caller,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// SerializationContext 序列化上下文
type SerializationContext struct {
	OriginalType      reflect.Type
	SerializationTime time.Time
	Size              int
	Error             error
}

// newLogMetadata 创建新的日志元数据
func newLogMetadata(level int, message string, opts *logOptions) *LogMetadata {
	metadata := &LogMetadata{
		Timestamp: time.Now(),
		Level:     LevelNames[level],
		Message:   message,
		Fields:    opts.fields,
	}

	// 设置进程ID
	if opts.showProcess {
		metadata.ProcessID = os.Getpid()
	}

	// 设置协程ID
	if opts.showRoutine {
		metadata.RoutineID = getGoroutineID()
	}

	// 设置调用者信息
	if opts.showCaller && opts.skipCallerNum > 0 {
		metadata.Caller = getCaller(opts.skipCallerNum)
	}

	return metadata
}

// newJSONLogEntry 创建新的JSON日志条目
func newJSONLogEntry(args []interface{}, opts *logOptions) *JSONLogEntry {
	entry := &JSONLogEntry{
		Timestamp: time.Now().Format(DateTimeFormat),
		Level:     "JSON",
		Fields:    make(map[string]interface{}),
	}

	// 设置进程ID
	if opts.showProcess {
		entry.ProcessID = os.Getpid()
	}

	// 设置协程ID
	if opts.showRoutine {
		entry.RoutineID = getGoroutineID()
	}

	// 设置调用者信息
	if opts.showCaller && opts.skipCallerNum > 0 {
		entry.Caller = getCaller(opts.skipCallerNum)
	}

	// 处理参数
	if len(args) == 0 {
		return entry
	}

	// 统一将所有参数序列化为JSON字符串存储在Msg字段中
	if len(args) == 1 {
		arg := args[0]
		switch v := arg.(type) {
		case string:
			entry.Message = v
		default:
			// 将对象序列化为JSON字符串
			if jsonData, err := safeJSONMarshal(v); err == nil {
				entry.Message = string(jsonData)
			} else {
				// 序列化失败时使用字符串表示
				entry.Message = fmt.Sprintf("%+v", v)
			}
		}
	} else {
		// 多个参数，将整个数组序列化为JSON字符串
		if jsonData, err := safeJSONMarshal(args); err == nil {
			entry.Message = string(jsonData)
		} else {
			// 序列化失败时使用字符串表示
			entry.Message = fmt.Sprintf("%+v", args)
		}
	}

	return entry
}

// getGoroutineID 获取当前协程ID
func getGoroutineID() uint64 {
	buf := make([]byte, 64)
	buf = buf[:runtime.Stack(buf, false)]
	stack := string(buf)

	// 解析协程ID，格式类似 "goroutine 123 [running]:"
	if strings.HasPrefix(stack, "goroutine ") {
		end := strings.Index(stack, " [")
		if end > 10 {
			idStr := stack[10:end]
			var id uint64
			fmt.Sscanf(idStr, "%d", &id)
			return id
		}
	}
	return 0
}

// getCaller 获取调用者信息
func getCaller(skip int) string {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown"
	}

	// 只保留文件名，不包含完整路径
	if idx := strings.LastIndex(file, "/"); idx >= 0 {
		file = file[idx+1:]
	}

	return fmt.Sprintf("%s:%d %s()", file, line, getFuncName(pc))
}

// 截取函数名称
func getFuncName(pc uintptr) (name string) {

	n := runtime.FuncForPC(pc).Name()
	ns := strings.Split(n, ".")
	name = ns[len(ns)-1]
	return
}

// MarshalJSON 自定义JSON序列化
func (entry *JSONLogEntry) MarshalJSON() ([]byte, error) {
	// 创建一个临时结构体来控制序列化
	temp := make(map[string]interface{})

	temp["@timestamp"] = entry.Timestamp
	temp["level"] = entry.Level

	if entry.Value != nil {
		temp["value"] = entry.Value
	}

	if entry.ProcessID != 0 {
		temp["process_id"] = entry.ProcessID
	}

	if entry.RoutineID != 0 {
		temp["routine_id"] = entry.RoutineID
	}

	if entry.Caller != "" {
		temp["caller"] = entry.Caller
	}

	if len(entry.Fields) > 0 {
		temp["fields"] = entry.Fields
	}

	if entry.Message != "" {
		temp["msg"] = entry.Message
	}

	return json.Marshal(temp)
}

// UnmarshalJSON 自定义JSON反序列化
func (entry *JSONLogEntry) UnmarshalJSON(data []byte) error {
	temp := make(map[string]interface{})
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	if v, ok := temp["@timestamp"].(string); ok {
		entry.Timestamp = v
	}

	if v, ok := temp["level"].(string); ok {
		entry.Level = v
	}

	// 向后兼容性：同时支持旧的data和args字段
	if v, ok := temp["data"]; ok {
		if jsonData, err := safeJSONMarshal(v); err == nil {
			entry.Message = string(jsonData)
		}
	}

	if v, ok := temp["args"]; ok {
		if jsonData, err := safeJSONMarshal(v); err == nil {
			entry.Message = string(jsonData)
		}
	}

	if v, ok := temp["msg"].(string); ok {
		entry.Message = v
	}

	if v, ok := temp["value"]; ok {
		entry.Value = v
	}

	if v, ok := temp["process_id"].(float64); ok {
		entry.ProcessID = int(v)
	}

	if v, ok := temp["routine_id"].(float64); ok {
		entry.RoutineID = uint64(v)
	}

	if v, ok := temp["caller"].(string); ok {
		entry.Caller = v
	}

	if v, ok := temp["fields"].(map[string]interface{}); ok {
		entry.Fields = v
	}

	return nil
}

// safeJSONMarshal 安全的JSON序列化，处理可能的序列化错误
func safeJSONMarshal(v interface{}) ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			// 序列化过程中发生panic，记录错误
		}
	}()

	// 尝试标准序列化
	data, err := json.Marshal(v)
	if err != nil {
		// 序列化失败，尝试转换为字符串表示
		fallback := fmt.Sprintf("%+v", v)
		return json.Marshal(map[string]interface{}{
			"serialization_error":   err.Error(),
			"string_representation": fallback,
			"type":                  reflect.TypeOf(v).String(),
		})
	}

	return data, nil
}

// processJSONArgs 处理JSON参数，支持多种类型
func processJSONArgs(args []interface{}) interface{} {
	if len(args) == 0 {
		return nil
	}

	if len(args) == 1 {
		return processJSONSingleArg(args[0])
	}

	// 多个参数，包装为数组
	return args
}

// processJSONSingleArg 处理单个JSON参数
func processJSONSingleArg(arg interface{}) interface{} {
	if arg == nil {
		return nil
	}

	rt := reflect.TypeOf(arg)

	// 处理特殊类型
	switch rt.Kind() {
	case reflect.Func:
		return map[string]interface{}{
			"type":      "function",
			"signature": rt.String(),
		}
	case reflect.Chan:
		return map[string]interface{}{
			"type":         "channel",
			"element_type": rt.Elem().String(),
		}
	case reflect.Ptr:
		rv := reflect.ValueOf(arg)
		if rv.IsNil() {
			return nil
		}
		return processJSONSingleArg(rv.Elem().Interface())
	case reflect.Interface:
		rv := reflect.ValueOf(arg)
		if rv.IsNil() {
			return nil
		}
		return processJSONSingleArg(rv.Elem().Interface())
	}

	// 检查是否实现了json.Marshaler接口
	if marshaler, ok := arg.(json.Marshaler); ok {
		data, err := marshaler.MarshalJSON()
		if err == nil {
			var result interface{}
			if json.Unmarshal(data, &result) == nil {
				return result
			}
		}
	}

	// 处理时间类型
	if t, ok := arg.(time.Time); ok {
		return t.Format(DateTimeFormat)
	}

	// 处理错误类型
	if err, ok := arg.(error); ok {
		return map[string]interface{}{
			"error": err.Error(),
			"type":  reflect.TypeOf(err).String(),
		}
	}

	return arg
}
