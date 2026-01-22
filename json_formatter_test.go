package log

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// TestWithJsonFormatterOption 测试WithJsonFormatter选项
func TestWithJsonFormatterOption(t *testing.T) {
	// 创建内存写入器用于捕获输出
	var buf bytes.Buffer

	// 创建带JsonFormatter选项的logger
	logger, err := NewLogger(
		WithJsonFormatter(),
	)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// 替换logger的输出管理器，使用内存写入器
	memWriter := &testMemoryWriter{buffer: &buf}
	logger.outputManager.consoleWriter = &ConsoleWriter{
		writer:    memWriter,
		formatter: NewJSONFormatter(false),
	}

	// 测试Info日志
	logger.Info("Test info message")

	// 测试Json日志（应该被序列化为普通消息）
	logger.Json(map[string]interface{}{
		"user_id": 123,
		"action":  "login",
	})

	// 验证输出是JSON格式
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) < 2 {
		t.Fatalf("Expected at least 2 lines, got %d", len(lines))
	}

	// 验证Info日志是JSON格式
	var infoLog map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &infoLog); err != nil {
		t.Errorf("Info log is not valid JSON: %v", err)
	}

	// 验证Json日志是JSON格式
	var jsonLog map[string]interface{}
	if err := json.Unmarshal([]byte(lines[1]), &jsonLog); err != nil {
		t.Errorf("Json log is not valid JSON: %v", err)
	}

	// 验证必要字段
	if infoLog["level"] != "INFO" {
		t.Errorf("Expected level INFO, got %v", infoLog["level"])
	}

	if jsonLog["level"] != "INFO" {
		t.Errorf("Expected level INFO for Json log, got %v", jsonLog["level"])
	}

	// 验证Json日志的内容是序列化的JSON字符串
	if msg, ok := jsonLog["message"]; ok {
		var serializedData map[string]interface{}
		if err := json.Unmarshal([]byte(msg.(string)), &serializedData); err != nil {
			t.Errorf("Json log message is not serialized JSON: %v", err)
		}
		if userID, ok := serializedData["user_id"]; !ok || userID != float64(123) {
			t.Errorf("Expected user_id 123 in serialized data, got %v", serializedData)
		}
	}
}

// TestJsonMethodAsNormalMessage 测试Json方法将参数序列化为普通消息
func TestJsonMethodAsNormalMessage(t *testing.T) {
	// 创建内存写入器用于捕获输出
	var buf bytes.Buffer

	// 创建带JsonFormatter选项的logger
	logger, err := NewLogger(
		WithJsonFormatter(),
	)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// 替换logger的输出管理器，使用内存写入器
	memWriter := &testMemoryWriter{buffer: &buf}
	logger.outputManager.consoleWriter = &ConsoleWriter{
		writer:    memWriter,
		formatter: NewJSONFormatter(false),
	}

	// 测试不同类型的Json参数
	testCases := []struct {
		name string
		args []interface{}
	}{
		{
			name: "Map data",
			args: []interface{}{map[string]interface{}{"key": "value", "num": 42}},
		},
		{
			name: "Multiple args",
			args: []interface{}{"string", 123, true},
		},
		{
			name: "Struct data",
			args: []interface{}{struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}{"Alice", 30}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf.Reset() // 清空缓冲区
			logger.Json(tc.args...)

			output := buf.String()
			if output == "" {
				t.Fatal("No output captured")
			}

			// 验证输出是JSON格式
			var logEntry map[string]interface{}
			if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
				t.Errorf("Output is not valid JSON: %v", err)
				t.Logf("Output: %s", output)
				return
			}

			// 验证消息字段包含序列化的数据
			if message, ok := logEntry["message"]; ok {
				// 验证message字段是有效的JSON字符串
				var serializedData interface{}
				if err := json.Unmarshal([]byte(message.(string)), &serializedData); err != nil {
					t.Errorf("Message field is not valid JSON: %v", err)
					t.Logf("Message content: %s", message)
				}
			} else {
				t.Error("No message field in log entry")
				t.Logf("Log entry: %+v", logEntry)
			}
		})
	}
}

// TestWithoutJsonFormatterOption 测试不使用JsonFormatter选项的情况
func TestWithoutJsonFormatterOption(t *testing.T) {
	// 创建内存写入器用于捕获输出
	var buf bytes.Buffer

	// 创建不带JsonFormatter选项的logger
	logger, err := NewLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// 替换logger的输出管理器，使用内存写入器
	memWriter := &testMemoryWriter{buffer: &buf}
	logger.outputManager.consoleWriter = &ConsoleWriter{
		writer:    memWriter,
		formatter: NewPlainFormatter(false, false, false),
	}

	// 测试Info日志
	logger.Info("Test info message")

	// 验证输出不是JSON格式
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// 过滤掉空行
	nonEmptyLines := []string{}
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines = append(nonEmptyLines, line)
		}
	}

	if len(nonEmptyLines) < 1 {
		t.Fatalf("Expected at least 1 non-empty line, got %d. Output: %q", len(nonEmptyLines), output)
	}

	// 验证Info日志不是JSON格式
	var infoLog map[string]interface{}
	if err := json.Unmarshal([]byte(nonEmptyLines[0]), &infoLog); err == nil {
		t.Error("Info log should not be JSON format without JsonFormatter option")
	}

	// 验证Info日志是普通文本格式
	if !strings.Contains(nonEmptyLines[0], "[INFO]") {
		t.Errorf("Info log should contain [INFO] level. Got: %s", nonEmptyLines[0])
	}

	if !strings.Contains(nonEmptyLines[0], "Test info message") {
		t.Errorf("Info log should contain the message. Got: %s", nonEmptyLines[0])
	}
}

// testMemoryWriter 用于测试的内存写入器
type testMemoryWriter struct {
	buffer *bytes.Buffer
}

func (mw *testMemoryWriter) Write(p []byte) (n int, err error) {
	return mw.buffer.Write(p)
}

func (mw *testMemoryWriter) String() string {
	return mw.buffer.String()
}

func (mw *testMemoryWriter) Reset() {
	mw.buffer.Reset()
}
