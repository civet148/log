package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestLogger 测试主要Logger功能
func TestLogger(t *testing.T) {
	// 创建测试目录
	testDir := filepath.Join(os.TempDir(), "logex_test")
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	// 测试基本Logger创建
	logger, err := NewLogger(
		WithLevel(LevelDebug),
		WithLogFile(filepath.Join(testDir, "test.log")),
		WithFileSize(1), // 1MB
		WithMaxBackups(3),
	)

	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// 测试各种日志级别
	logger.Debug("Debug message")
	logger.Info("Info message")
	logger.Warn("Warning message")

	// 测试格式化日志
	logger.Debugf("Debug with format: %s", "test")
	logger.Infof("Info with format: %d", 42)

	// 测试错误日志
	err = logger.Error("Error message")
	if err == nil {
		t.Error("Error method should return an error")
	}

	// 测试JSON日志
	logger.Json(map[string]interface{}{
		"user_id": 123,
		"action":  "login",
		"success": true,
	})

	// 刷新日志
	if err := logger.Flush(); err != nil {
		t.Errorf("Failed to flush logger: %v", err)
	}

	// 验证日志文件是否创建
	logFile := filepath.Join(testDir, "test.log")
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

// TestLoggerLevels 测试日志级别过滤
func TestLoggerLevels(t *testing.T) {
	testDir := filepath.Join(os.TempDir(), "logex_level_test")
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	// 创建INFO级别的logger
	logger, err := NewLogger(
		WithLevel(LevelInfo),
		WithLogFile(filepath.Join(testDir, "level_test.log")),
	)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// 测试各级别日志
	logger.Trace("This should not appear") // 不应该输出
	logger.Debug("This should not appear") // 不应该输出
	logger.Info("This should appear")      // 应该输出
	logger.Warn("This should appear")      // 应该输出
	logger.Error("This should appear")     // 应该输出

	logger.Flush()

	// 读取日志文件内容
	content, err := os.ReadFile(filepath.Join(testDir, "level_test.log"))
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)

	// 验证级别过滤
	if strings.Contains(contentStr, "TRACE") {
		t.Error("TRACE log should not appear with INFO level")
	}
	if strings.Contains(contentStr, "DEBUG") {
		t.Error("DEBUG log should not appear with INFO level")
	}
	if !strings.Contains(contentStr, "INFO") {
		t.Error("INFO log should appear with INFO level")
	}
	if !strings.Contains(contentStr, "WARN") {
		t.Error("WARN log should appear with INFO level")
	}
	if !strings.Contains(contentStr, "ERROR") {
		t.Error("ERROR log should appear with INFO level")
	}
}

// TestFileRotation 测试文件切割功能
func TestFileRotation(t *testing.T) {
	testDir := filepath.Join(os.TempDir(), "logex_rotation_test")
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	logFile := filepath.Join(testDir, "rotation_test.log")

	// 创建小尺寸的日志文件用于测试切割
	rotator, err := NewFileRotator(logFile, 1, 2) // 1MB, 2个备份
	if err != nil {
		t.Fatalf("Failed to create file rotator: %v", err)
	}
	defer rotator.Close()

	// 写入大量数据触发切割
	largeMessage := strings.Repeat("This is a test message for file rotation.", 1000)
	for i := 0; i < 50; i++ {
		message := fmt.Sprintf("%s Line %d\n", largeMessage, i)
		if _, err := rotator.Write([]byte(message)); err != nil {
			t.Errorf("Failed to write to rotator: %v", err)
		}
	}

	// 强制切割
	if err := rotator.ForceRotate(); err != nil {
		t.Errorf("Failed to force rotate: %v", err)
	}

	// 检查备份文件
	backupFiles, err := rotator.GetBackupFileNames()
	if err != nil {
		t.Errorf("Failed to get backup files: %v", err)
	}

	if len(backupFiles) == 0 {
		t.Error("No backup files were created")
	}

	t.Logf("Created backup files: %v", backupFiles)
}

// TestJSONFormatting 测试JSON格式化
func TestJSONFormatting(t *testing.T) {
	// 测试JSON格式化器
	formatter := NewJSONFormatter(false)

	metadata := &LogMetadata{
		Timestamp: time.Now(),
		Level:     "INFO",
		ProcessID: 123,
		Caller:    "test.go:10",
	}

	formatted := formatter.Format(LevelInfo, "Test message", metadata)

	// 验证JSON格式
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(formatted), &jsonData); err != nil {
		t.Errorf("Failed to parse formatted JSON: %v", err)
	}

	// 验证必要字段
	if jsonData["level"] != "INFO" {
		t.Errorf("Expected level INFO, got %v", jsonData["level"])
	}
	if jsonData["msg"] != "Test message" {
		t.Errorf("Expected message 'Test message', got %v", jsonData["msg"])
	}
	if jsonData["process_id"] != float64(123) {
		t.Errorf("Expected process_id 123, got %v", jsonData["process_id"])
	}
}

// TestJSONLogEntry 测试JSON日志条目
func TestJSONLogEntry(t *testing.T) {
	// 测试不同类型的JSON日志
	testCases := []struct {
		name string
		args []interface{}
	}{
		{
			name: "String message",
			args: []interface{}{"Simple string message"},
		},
		{
			name: "Struct data",
			args: []interface{}{
				map[string]interface{}{
					"user_id": 123,
					"action":  "login",
					"time":    time.Now(),
				},
			},
		},
		{
			name: "Multiple args",
			args: []interface{}{"user", 123, true, map[string]string{"key": "value"}},
		},
		{
			name: "Slice data",
			args: []interface{}{[]string{"item1", "item2", "item3"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := newJSONLogEntry(tc.args, &logOptions{
				showProcess: true,
				showRoutine: false,
				showCaller:  true,
			})

			// 测试序列化
			data, err := json.Marshal(entry)
			if err != nil {
				t.Errorf("Failed to marshal JSON entry: %v", err)
			}

			// 测试反序列化
			var restored JSONLogEntry
			if err := json.Unmarshal(data, &restored); err != nil {
				t.Errorf("Failed to unmarshal JSON entry: %v", err)
			}

			// 验证基本字段
			if restored.Level != "JSON" {
				t.Errorf("Expected level JSON, got %s", restored.Level)
			}

		})
	}
}

// TestConcurrentLogging 测试并发日志记录
func TestConcurrentLogging(t *testing.T) {
	testDir := filepath.Join(os.TempDir(), "logex_concurrent_test")
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	logger, err := NewLogger(
		WithLevel(LevelDebug),
		WithLogFile(filepath.Join(testDir, "concurrent_test.log")),
	)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// 并发写入测试
	const numGoroutines = 10
	const messagesPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				logger.Infof("Goroutine %d message %d", goroutineID, j)
				logger.Json(map[string]interface{}{
					"goroutine_id": goroutineID,
					"message_id":   j,
					"timestamp":    time.Now(),
				})
			}
		}(i)
	}

	wg.Wait()
	logger.Flush()

	// 验证日志文件
	logFile := filepath.Join(testDir, "concurrent_test.log")
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}

	// 读取并计算日志条数
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	nonEmptyLines := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines++
		}
	}

	expectedLines := numGoroutines * messagesPerGoroutine * 2 // Info + Json for each
	if nonEmptyLines < expectedLines {
		t.Errorf("Expected at least %d log lines, got %d", expectedLines, nonEmptyLines)
	}

	t.Logf("Successfully logged %d lines from %d concurrent goroutines", nonEmptyLines, numGoroutines)
}

// TestColorFormatter 测试颜色格式化器
func TestColorFormatter(t *testing.T) {
	formatter := NewColorFormatter(true, true, true, true)

	metadata := &LogMetadata{
		Timestamp: time.Now(),
		Level:     "INFO",
		ProcessID: 123,
		RoutineID: 456,
		Caller:    "test.go:10",
	}

	formatted := formatter.Format(LevelInfo, "Test message", metadata)

	// 验证包含颜色代码
	if !strings.Contains(formatted, "\033[") {
		t.Error("Formatted message should contain color codes")
	}

	// 验证包含必要信息
	if !strings.Contains(formatted, "[INFO]") {
		t.Error("Formatted message should contain level")
	}
	if !strings.Contains(formatted, "[PID:123]") {
		t.Error("Formatted message should contain process ID")
	}
	if !strings.Contains(formatted, "[GID:456]") {
		t.Error("Formatted message should contain goroutine ID")
	}
	if !strings.Contains(formatted, "[test.go:10]") {
		t.Error("Formatted message should contain caller info")
	}
	if !strings.Contains(formatted, "Test message") {
		t.Error("Formatted message should contain original message")
	}
}

// TestMemoryWriter 测试内存写入器（用于测试）
type MemoryWriter struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (mw *MemoryWriter) Write(p []byte) (n int, err error) {
	mw.mu.Lock()
	defer mw.mu.Unlock()
	return mw.buffer.Write(p)
}

func (mw *MemoryWriter) String() string {
	mw.mu.Lock()
	defer mw.mu.Unlock()
	return mw.buffer.String()
}

func (mw *MemoryWriter) Reset() {
	mw.mu.Lock()
	defer mw.mu.Unlock()
	mw.buffer.Reset()
}

// TestConsoleWriter 测试终端写入器
func TestConsoleWriter(t *testing.T) {
	memWriter := &MemoryWriter{}

	config := &logOptions{
		showColor:   false,
		showProcess: true,
		showRoutine: false,
		showCaller:  true,
	}

	consoleWriter, err := NewConsoleWriter(memWriter, config)
	if err != nil {
		t.Fatalf("Failed to create console writer: %v", err)
	}

	metadata := &LogMetadata{
		Timestamp: time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
		Level:     "INFO",
		ProcessID: 123,
		Caller:    "test.go:10",
	}

	err = consoleWriter.Write(LevelInfo, "Test console message", metadata)
	if err != nil {
		t.Errorf("Failed to write to console: %v", err)
	}

	output := memWriter.String()
	if !strings.Contains(output, "Test console message") {
		t.Error("Output should contain the test message")
	}
	if !strings.Contains(output, "[INFO]") {
		t.Error("Output should contain log level")
	}
	if !strings.Contains(output, "[PID:123]") {
		t.Error("Output should contain process ID")
	}

	t.Logf("Console output: %s", output)
}

// TestErrorHandling 测试错误处理
func TestErrorHandling(t *testing.T) {
	// 测试无效配置
	testCases := []struct {
		name   string
		config func() *logOptions
	}{
		{
			name: "Invalid log file directory",
			config: func() *logOptions {
				return &logOptions{
					logFilePath: "/invalid/path/test.log",
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := tc.config()
			_, err := NewOutputManager(config)
			if err == nil {
				t.Error("Expected error for invalid configuration")
			}
			t.Logf("Expected error: %v", err)
		})
	}
}

// TestPerformancePool 测试对象池性能
func TestPerformancePool(t *testing.T) {
	// 测试LogMetadata池
	metadata1 := globalPool.GetLogMetadata()
	if metadata1 == nil {
		t.Error("Failed to get metadata from pool")
	}

	metadata1.Level = "TEST"
	metadata1.Message = "Test message"

	globalPool.PutLogMetadata(metadata1)

	metadata2 := globalPool.GetLogMetadata()
	if metadata2 == nil {
		t.Error("Failed to get second metadata from pool")
	}

	// 验证对象被重置
	if metadata2.Level != "" || metadata2.Message != "" {
		t.Error("Metadata object was not properly reset")
	}

	globalPool.PutLogMetadata(metadata2)

	// 测试缓冲区池
	buffer1 := globalPool.GetBuffer()
	buffer1.WriteString("test")
	globalPool.PutBuffer(buffer1)

	buffer2 := globalPool.GetBuffer()
	if buffer2.Len() != 0 {
		t.Error("Buffer was not properly reset")
	}
	globalPool.PutBuffer(buffer2)
}

// TestLoggerConfiguration 测试Logger配置
func TestLoggerConfiguration(t *testing.T) {
	logger, err := NewLogger()
	if err != nil {
		t.Fatalf("Failed to create default logger: %v", err)
	}
	defer logger.Close()

	// 测试级别设置
	logger.SetLevel(LevelWarn)
	if logger.GetLevel() != LevelWarn {
		t.Errorf("Expected level %d, got %d", LevelWarn, logger.GetLevel())
	}

	// 测试级别名称解析
	level := ParseLevel("DEBUG")
	if level != LevelDebug {
		t.Errorf("Expected level %d for DEBUG, got %d", LevelDebug, level)
	}

	levelName := GetLevelName(LevelInfo)
	if levelName != "INFO" {
		t.Errorf("Expected level name INFO, got %s", levelName)
	}
}

// BenchmarkLogger 性能基准测试
func BenchmarkLogger(b *testing.B) {
	logger, err := NewLogger(
		WithLevel(LevelInfo),
		WithLogFile("/dev/null"), // 丢弃输出以测试纯性能
	)
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("Benchmark test message")
		}
	})
}

// BenchmarkJSONLogging JSON日志性能测试
func BenchmarkJSONLogging(b *testing.B) {
	logger, err := NewLogger(
		WithLevel(LevelInfo),
		WithLogFile("/dev/null"),
	)
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	testData := map[string]interface{}{
		"user_id":   123,
		"action":    "test",
		"timestamp": time.Now(),
		"details": map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Json(testData)
		}
	})
}
