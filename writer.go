package log

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// fileWriterAdapter 适配器，将FileWriter转换为io.Writer
type fileWriterAdapter struct {
	fw *FileWriter
}

func (fwa *fileWriterAdapter) Write(p []byte) (n int, err error) {
	fwa.fw.mu.Lock()
	defer fwa.fw.mu.Unlock()
	return fwa.fw.rotator.Write(p)
}

// Writer 日志写入器接口
type Writer interface {
	Write(level int, formattedMessage string) error
	Close() error
}

// OutputManager 输出管理器
type OutputManager struct {
	consoleWriter *ConsoleWriter
	fileWriter    *FileWriter
	jsonWriter    *JSONWriter
	config        *logOptions
	mu            sync.RWMutex
}

// ConsoleWriter 终端输出器
type ConsoleWriter struct {
	writer    io.Writer
	formatter Formatter
	mu        sync.Mutex
}

// FileWriter 文件输出器
type FileWriter struct {
	rotator   *FileRotator
	formatter Formatter
	mu        sync.RWMutex
}

// JSONWriter JSON输出器
type JSONWriter struct {
	outputs   []io.Writer
	formatter *JSONFormatter
	mu        sync.RWMutex
}

// BufferedWriter 缓冲写入器
type BufferedWriter struct {
	writer     io.Writer
	buffer     *bufio.Writer
	bufferSize int
	mu         sync.Mutex
}

// NewOutputManager 创建输出管理器
func NewOutputManager(config *logOptions) (*OutputManager, error) {
	manager := &OutputManager{
		config: config,
	}

	var err error

	// 创建终端输出器
	if !config.disableConsole {
		manager.consoleWriter, err = NewConsoleWriter(os.Stdout, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create console writer: %w", err)
		}
	}

	// 创建文件输出器
	if config.logFilePath != "" {
		manager.fileWriter, err = NewFileWriter(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create file writer: %w", err)
		}
	}

	// 创建JSON输出器
	var jsonOutputs []io.Writer
	if !config.disableConsole {
		jsonOutputs = append(jsonOutputs, os.Stdout)
	}
	if config.logFilePath != "" && manager.fileWriter != nil {
		jsonOutputs = append(jsonOutputs, &fileWriterAdapter{manager.fileWriter})
	}

	if len(jsonOutputs) > 0 {
		manager.jsonWriter = NewJSONWriter(jsonOutputs)
	}

	return manager, nil
}

// Write 写入日志到所有配置的输出器
func (om *OutputManager) Write(level int, message string, metadata *LogMetadata) error {
	om.mu.RLock()
	defer om.mu.RUnlock()

	var errors []error

	// 如果设置了jsonFormatter选项，则所有级别都采用JSON格式输出
	if om.config.jsonFormatter {
		// 创建JSON格式化器
		jsonFormatter := NewJSONFormatter(false)

		// 格式化为JSON
		formattedMessage := jsonFormatter.Format(level, message, metadata)

		// 写入终端
		if om.consoleWriter != nil {
			om.consoleWriter.mu.Lock()
			_, err := fmt.Fprintln(om.consoleWriter.writer, formattedMessage)
			om.consoleWriter.mu.Unlock()
			if err != nil {
				errors = append(errors, fmt.Errorf("console writer error: %w", err))
			}
		}

		// 写入文件
		if om.fileWriter != nil {
			om.fileWriter.mu.Lock()
			_, err := om.fileWriter.rotator.Write([]byte(formattedMessage + "\n"))
			om.fileWriter.mu.Unlock()
			if err != nil {
				errors = append(errors, fmt.Errorf("file writer error: %w", err))
			}
		}

		return combineErrors(errors)
	}

	// JSON级别特殊处理
	if om.config.jsonFormatter {
		if om.jsonWriter != nil {
			if err := om.jsonWriter.WriteJSON(metadata.Message); err != nil {
				errors = append(errors, fmt.Errorf("json writer error: %w", err))
			}
		}
		return combineErrors(errors)
	}

	// 写入终端
	if om.consoleWriter != nil {
		if err := om.consoleWriter.Write(level, message, metadata); err != nil {
			errors = append(errors, fmt.Errorf("console writer error: %w", err))
		}
	}

	// 写入文件
	if om.fileWriter != nil {
		if err := om.fileWriter.Write(level, message, metadata); err != nil {
			errors = append(errors, fmt.Errorf("file writer error: %w", err))
		}
	}

	return combineErrors(errors)
}

// Close 关闭所有输出器
func (om *OutputManager) Close() error {
	om.mu.Lock()
	defer om.mu.Unlock()

	var errors []error

	if om.consoleWriter != nil {
		if err := om.consoleWriter.Close(); err != nil {
			errors = append(errors, err)
		}
	}

	if om.fileWriter != nil {
		if err := om.fileWriter.Close(); err != nil {
			errors = append(errors, err)
		}
	}

	if om.jsonWriter != nil {
		if err := om.jsonWriter.Close(); err != nil {
			errors = append(errors, err)
		}
	}

	return combineErrors(errors)
}

// NewConsoleWriter 创建终端输出器
func NewConsoleWriter(writer io.Writer, config *logOptions) (*ConsoleWriter, error) {
	var formatter Formatter

	if config.showColor {
		formatter = NewColorFormatter(true, config.showProcess, config.showRoutine, config.showCaller, config.showStack)
	} else {
		formatter = NewPlainFormatter(config.showProcess, config.showRoutine, config.showCaller, config.showStack)
	}

	return &ConsoleWriter{
		writer:    writer,
		formatter: formatter,
	}, nil
}

// Write 写入到终端
func (cw *ConsoleWriter) Write(level int, message string, metadata *LogMetadata) error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	formattedMessage := cw.formatter.Format(level, message, metadata)
	_, err := fmt.Fprintln(cw.writer, formattedMessage)
	return err
}

// Close 关闭终端输出器
func (cw *ConsoleWriter) Close() error {
	// 终端输出器通常不需要显式关闭
	return nil
}

// NewFileWriter 创建文件输出器
func NewFileWriter(config *logOptions) (*FileWriter, error) {
	rotator, err := NewFileRotator(config.logFilePath, config.logFileSize, config.maxBackups)
	if err != nil {
		return nil, err
	}

	formatter := NewPlainFormatter(config.showProcess, config.showRoutine, config.showCaller, config.showStack)

	return &FileWriter{
		rotator:   rotator,
		formatter: formatter,
	}, nil
}

// Write 写入到文件
func (fw *FileWriter) Write(level int, message string, metadata *LogMetadata) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	formattedMessage := fw.formatter.Format(level, message, metadata)
	_, err := fw.rotator.Write([]byte(formattedMessage + "\n"))
	return err
}

// Close 关闭文件输出器
func (fw *FileWriter) Close() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.rotator != nil {
		return fw.rotator.Close()
	}
	return nil
}

// NewJSONWriter 创建JSON输出器
func NewJSONWriter(outputs []io.Writer) *JSONWriter {
	return &JSONWriter{
		outputs:   outputs,
		formatter: NewJSONFormatter(false), // 不使用美化输出
	}
}

// WriteJSON 写入JSON数据
func (jw *JSONWriter) WriteJSON(data interface{}) error {
	jw.mu.Lock()
	defer jw.mu.Unlock()

	// 根据数据类型创建JSON条目
	var jsonEntry *JSONLogEntry

	switch v := data.(type) {
	case *JSONLogEntry:
		jsonEntry = v
	case []interface{}:
		jsonEntry = newJSONLogEntry(v, options)
	case string:
		// 如果是字符串，创建一个简单的JSON条目
		jsonEntry = &JSONLogEntry{
			Timestamp: time.Now().Format(DateTimeFormat),
			Level:     "JSON",
			Message:   v,
		}
	default:
		jsonEntry = newJSONLogEntry([]interface{}{v}, options)
	}

	// 格式化JSON
	formattedJSON := jw.formatter.FormatJSONEntry(jsonEntry)

	// 写入到所有输出
	var errors []error
	for _, output := range jw.outputs {
		if _, err := fmt.Fprintln(output, formattedJSON); err != nil {
			errors = append(errors, err)
		}
	}

	return combineErrors(errors)
}

// Write 实现Writer接口
func (jw *JSONWriter) Write(level int, formattedMessage string) error {
	// JSON Writer主要通过WriteJSON方法使用
	return jw.WriteJSON(formattedMessage)
}

// Close 关闭JSON输出器
func (jw *JSONWriter) Close() error {
	// JSON输出器通常不需要显式关闭，因为它使用的是其他Writer
	return nil
}

// NewBufferedWriter 创建缓冲写入器
func NewBufferedWriter(writer io.Writer, bufferSize int) *BufferedWriter {
	if bufferSize <= 0 {
		bufferSize = 4096 // 默认4KB缓冲区
	}

	return &BufferedWriter{
		writer:     writer,
		buffer:     bufio.NewWriterSize(writer, bufferSize),
		bufferSize: bufferSize,
	}
}

// Write 写入到缓冲区
func (bw *BufferedWriter) Write(data []byte) (int, error) {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	return bw.buffer.Write(data)
}

// WriteString 写入字符串到缓冲区
func (bw *BufferedWriter) WriteString(s string) (int, error) {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	return bw.buffer.WriteString(s)
}

// Flush 刷新缓冲区
func (bw *BufferedWriter) Flush() error {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	return bw.buffer.Flush()
}

// Close 关闭缓冲写入器
func (bw *BufferedWriter) Close() error {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	// 刷新缓冲区
	if err := bw.buffer.Flush(); err != nil {
		return err
	}

	// 如果底层writer实现了io.Closer，则关闭它
	if closer, ok := bw.writer.(io.Closer); ok {
		return closer.Close()
	}

	return nil
}

// Size 返回缓冲区大小
func (bw *BufferedWriter) Size() int {
	return bw.bufferSize
}

// Available 返回缓冲区可用空间
func (bw *BufferedWriter) Available() int {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	return bw.buffer.Available()
}

// Buffered 返回缓冲区中的数据大小
func (bw *BufferedWriter) Buffered() int {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	return bw.buffer.Buffered()
}

// combineErrors 合并多个错误
func combineErrors(errors []error) error {
	if len(errors) == 0 {
		return nil
	}

	if len(errors) == 1 {
		return errors[0]
	}

	var errorMessages []string
	for _, err := range errors {
		errorMessages = append(errorMessages, err.Error())
	}

	return fmt.Errorf("multiple errors: %v", errorMessages)
}

// SyncWriter 同步写入器，确保写入操作的原子性
type SyncWriter struct {
	writer io.Writer
	mu     sync.Mutex
}

// NewSyncWriter 创建同步写入器
func NewSyncWriter(writer io.Writer) *SyncWriter {
	return &SyncWriter{
		writer: writer,
	}
}

// Write 同步写入
func (sw *SyncWriter) Write(p []byte) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	return sw.writer.Write(p)
}

// WriteString 同步写入字符串
func (sw *SyncWriter) WriteString(s string) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	if stringWriter, ok := sw.writer.(io.StringWriter); ok {
		return stringWriter.WriteString(s)
	}

	return sw.writer.Write([]byte(s))
}

// MultiWriter 多重写入器，同时写入到多个目标
type MultiWriter struct {
	writers []io.Writer
}

// NewMultiWriter 创建多重写入器
func NewMultiWriter(writers ...io.Writer) *MultiWriter {
	return &MultiWriter{
		writers: writers,
	}
}

// Write 写入到所有目标
func (mw *MultiWriter) Write(p []byte) (n int, err error) {
	for _, writer := range mw.writers {
		if n, err = writer.Write(p); err != nil {
			return n, err
		}
	}
	return len(p), nil
}

// Add 添加新的写入器
func (mw *MultiWriter) Add(writer io.Writer) {
	mw.writers = append(mw.writers, writer)
}

// Remove 移除写入器
func (mw *MultiWriter) Remove(writer io.Writer) {
	for i, w := range mw.writers {
		if w == writer {
			mw.writers = append(mw.writers[:i], mw.writers[i+1:]...)
			break
		}
	}
}
