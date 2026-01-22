package log

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FileRotator 文件切割器
type FileRotator struct {
	maxSize      SizeMB
	maxBackups   int
	currentSize  int64
	baseFileName string
	currentFile  *os.File
	mu           sync.RWMutex
	policy       *RotationPolicy
}

// RotationPolicy 切割策略
type RotationPolicy struct {
	SizeLimit     SizeMB
	BackupLimit   int
	TimeLimit     time.Duration
	CompressFiles bool
}

// BackupFile 备份文件信息
type BackupFile struct {
	Name     string
	Path     string
	Size     int64
	ModTime  time.Time
	Index    int
}

// NewFileRotator 创建文件切割器
func NewFileRotator(baseFileName string, maxSize SizeMB, maxBackups int) (*FileRotator, error) {
	if baseFileName == "" {
		return nil, fmt.Errorf("base file name cannot be empty")
	}
	
	if maxSize <= 0 {
		maxSize = DefaultFileSize
	}
	
	if maxBackups < 0 {
		maxBackups = DefaultMaxBackups
	}
	
	// 确保目录存在
	dir := filepath.Dir(baseFileName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}
	
	rotator := &FileRotator{
		maxSize:      maxSize,
		maxBackups:   maxBackups,
		baseFileName: baseFileName,
		policy: &RotationPolicy{
			SizeLimit:   maxSize,
			BackupLimit: maxBackups,
		},
	}
	
	// 打开当前日志文件
	if err := rotator.openCurrentFile(); err != nil {
		return nil, err
	}
	
	// 清理旧的备份文件
	if err := rotator.cleanupOldFiles(); err != nil {
		// 记录错误但不影响初始化
		fmt.Printf("Warning: failed to cleanup old files: %v\n", err)
	}
	
	return rotator, nil
}

// Write 写入数据到日志文件
func (fr *FileRotator) Write(data []byte) (int, error) {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	
	// 检查是否需要切割
	if fr.shouldRotate(len(data)) {
		if err := fr.rotate(); err != nil {
			return 0, fmt.Errorf("failed to rotate file: %w", err)
		}
	}
	
	// 写入数据
	n, err := fr.currentFile.Write(data)
	if err != nil {
		return n, err
	}
	
	fr.currentSize += int64(n)
	return n, nil
}

// Close 关闭文件切割器
func (fr *FileRotator) Close() error {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	
	if fr.currentFile != nil {
		err := fr.currentFile.Close()
		fr.currentFile = nil
		return err
	}
	return nil
}

// shouldRotate 检查是否应该切割文件
func (fr *FileRotator) shouldRotate(dataSize int) bool {
	if fr.currentFile == nil {
		return false
	}
	
	// 计算写入后的文件大小
	newSize := fr.currentSize + int64(dataSize)
	maxSizeBytes := int64(fr.maxSize) * 1024 * 1024 // 转换为字节
	
	return newSize >= maxSizeBytes
}

// rotate 执行文件切割
func (fr *FileRotator) rotate() error {
	// 关闭当前文件
	if fr.currentFile != nil {
		if err := fr.currentFile.Close(); err != nil {
			return err
		}
		fr.currentFile = nil
	}
	
	// 重命名当前文件为备份文件
	if err := fr.createBackup(); err != nil {
		return err
	}
	
	// 创建新的日志文件
	if err := fr.openCurrentFile(); err != nil {
		return err
	}
	
	// 清理旧的备份文件
	return fr.cleanupOldFiles()
}

// createBackup 创建备份文件
func (fr *FileRotator) createBackup() error {
	// 检查当前文件是否存在且不为空
	if stat, err := os.Stat(fr.baseFileName); err != nil || stat.Size() == 0 {
		return nil // 文件不存在或为空，无需备份
	}
	
	// 获取现有备份文件并重新编号
	backupFiles, err := fr.getBackupFiles()
	if err != nil {
		return err
	}
	
	// 按索引降序排序，从最大索引开始重命名
	sort.Slice(backupFiles, func(i, j int) bool {
		return backupFiles[i].Index > backupFiles[j].Index
	})
	
	// 删除超出限制的备份文件
	if fr.maxBackups > 0 && len(backupFiles) >= fr.maxBackups {
		filesToDelete := backupFiles[fr.maxBackups-1:]
		for _, file := range filesToDelete {
			if err := os.Remove(file.Path); err != nil {
				fmt.Printf("Warning: failed to remove old backup file %s: %v\n", file.Path, err)
			}
		}
		backupFiles = backupFiles[:fr.maxBackups-1]
	}
	
	// 重命名现有备份文件（索引+1）
	for _, file := range backupFiles {
		newIndex := file.Index + 1
		newName := fr.generateBackupName(newIndex)
		if err := os.Rename(file.Path, newName); err != nil {
			return fmt.Errorf("failed to rename backup file %s to %s: %w", file.Path, newName, err)
		}
	}
	
	// 重命名当前文件为第一个备份
	backupName := fr.generateBackupName(1)
	if err := os.Rename(fr.baseFileName, backupName); err != nil {
		return fmt.Errorf("failed to rename current file to backup: %w", err)
	}
	
	return nil
}

// openCurrentFile 打开当前日志文件
func (fr *FileRotator) openCurrentFile() error {
	file, err := os.OpenFile(fr.baseFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	
	fr.currentFile = file
	
	// 获取当前文件大小
	if stat, err := file.Stat(); err == nil {
		fr.currentSize = stat.Size()
	} else {
		fr.currentSize = 0
	}
	
	return nil
}

// generateBackupName 生成备份文件名
func (fr *FileRotator) generateBackupName(index int) string {
	return fmt.Sprintf("%s.%d", fr.baseFileName, index)
}

// getBackupFiles 获取所有备份文件
func (fr *FileRotator) getBackupFiles() ([]*BackupFile, error) {
	dir := filepath.Dir(fr.baseFileName)
	baseName := filepath.Base(fr.baseFileName)
	
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	
	var backupFiles []*BackupFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		name := entry.Name()
		if !strings.HasPrefix(name, baseName+".") {
			continue
		}
		
		// 提取索引
		suffix := strings.TrimPrefix(name, baseName+".")
		index, err := strconv.Atoi(suffix)
		if err != nil {
			continue // 不是数字后缀，跳过
		}
		
		fullPath := filepath.Join(dir, name)
		info, err := entry.Info()
		if err != nil {
			continue
		}
		
		backupFiles = append(backupFiles, &BackupFile{
			Name:    name,
			Path:    fullPath,
			Size:    info.Size(),
			ModTime: info.ModTime(),
			Index:   index,
		})
	}
	
	return backupFiles, nil
}

// cleanupOldFiles 清理超出数量限制的旧备份文件
func (fr *FileRotator) cleanupOldFiles() error {
	if fr.maxBackups <= 0 {
		return nil // 没有限制
	}
	
	backupFiles, err := fr.getBackupFiles()
	if err != nil {
		return err
	}
	
	if len(backupFiles) <= fr.maxBackups {
		return nil // 没有超出限制
	}
	
	// 按索引排序，保留索引小的文件（较新的）
	sort.Slice(backupFiles, func(i, j int) bool {
		return backupFiles[i].Index < backupFiles[j].Index
	})
	
	// 删除超出限制的文件
	filesToDelete := backupFiles[fr.maxBackups:]
	for _, file := range filesToDelete {
		if err := os.Remove(file.Path); err != nil {
			fmt.Printf("Warning: failed to remove old backup file %s: %v\n", file.Path, err)
		}
	}
	
	return nil
}

// GetCurrentSize 获取当前文件大小
func (fr *FileRotator) GetCurrentSize() int64 {
	fr.mu.RLock()
	defer fr.mu.RUnlock()
	return fr.currentSize
}

// GetMaxSize 获取最大文件大小
func (fr *FileRotator) GetMaxSize() SizeMB {
	return fr.maxSize
}

// GetMaxBackups 获取最大备份数
func (fr *FileRotator) GetMaxBackups() int {
	return fr.maxBackups
}

// GetCurrentFileName 获取当前文件名
func (fr *FileRotator) GetCurrentFileName() string {
	return fr.baseFileName
}

// GetBackupFileNames 获取所有备份文件名
func (fr *FileRotator) GetBackupFileNames() ([]string, error) {
	backupFiles, err := fr.getBackupFiles()
	if err != nil {
		return nil, err
	}
	
	var names []string
	for _, file := range backupFiles {
		names = append(names, file.Name)
	}
	
	sort.Strings(names)
	return names, nil
}

// SetMaxSize 设置最大文件大小
func (fr *FileRotator) SetMaxSize(maxSize SizeMB) {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	
	if maxSize > 0 {
		fr.maxSize = maxSize
		fr.policy.SizeLimit = maxSize
	}
}

// SetMaxBackups 设置最大备份数
func (fr *FileRotator) SetMaxBackups(maxBackups int) {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	
	if maxBackups >= 0 {
		fr.maxBackups = maxBackups
		fr.policy.BackupLimit = maxBackups
	}
}

// ForceRotate 强制执行文件切割
func (fr *FileRotator) ForceRotate() error {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	
	return fr.rotate()
}

// Sync 同步文件内容到磁盘
func (fr *FileRotator) Sync() error {
	fr.mu.RLock()
	defer fr.mu.RUnlock()
	
	if fr.currentFile != nil {
		return fr.currentFile.Sync()
	}
	return nil
}

// GetStats 获取文件切割器统计信息
func (fr *FileRotator) GetStats() (*RotatorStats, error) {
	fr.mu.RLock()
	defer fr.mu.RUnlock()
	
	backupFiles, err := fr.getBackupFiles()
	if err != nil {
		return nil, err
	}
	
	var totalBackupSize int64
	for _, file := range backupFiles {
		totalBackupSize += file.Size
	}
	
	return &RotatorStats{
		CurrentFileSize: fr.currentSize,
		BackupCount:     len(backupFiles),
		TotalBackupSize: totalBackupSize,
		MaxSize:         fr.maxSize,
		MaxBackups:      fr.maxBackups,
	}, nil
}

// RotatorStats 文件切割器统计信息
type RotatorStats struct {
	CurrentFileSize int64
	BackupCount     int
	TotalBackupSize int64
	MaxSize         SizeMB
	MaxBackups      int
}

// String 返回统计信息的字符串表示
func (rs *RotatorStats) String() string {
	return fmt.Sprintf("Current: %d bytes, Backups: %d files (%d bytes total), Limits: %dMB/%d files",
		rs.CurrentFileSize, rs.BackupCount, rs.TotalBackupSize, rs.MaxSize, rs.MaxBackups)
}

// ValidateConfiguration 验证配置参数
func ValidateRotatorConfiguration(baseFileName string, maxSize SizeMB, maxBackups int) error {
	if baseFileName == "" {
		return fmt.Errorf("base file name cannot be empty")
	}
	
	if maxSize <= 0 {
		return fmt.Errorf("max size must be positive, got: %d", maxSize)
	}
	
	if maxBackups < 0 {
		return fmt.Errorf("max backups cannot be negative, got: %d", maxBackups)
	}
	
	// 检查目录是否可写
	dir := filepath.Dir(baseFileName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("cannot create log directory %s: %w", dir, err)
	}
	
	// 测试文件写入权限
	testFile := filepath.Join(dir, ".logex_write_test")
	if file, err := os.Create(testFile); err != nil {
		return fmt.Errorf("cannot write to log directory %s: %w", dir, err)
	} else {
		file.Close()
		os.Remove(testFile)
	}
	
	return nil
}

// CompressBackupFile 压缩备份文件（可选功能）
func CompressBackupFile(filePath string) error {
	// 这里可以实现文件压缩逻辑
	// 为了简化，暂时不实现压缩功能
	return nil
}

// RotatorOption 文件切割器选项函数类型
type RotatorOption func(*FileRotator) error

// WithCompressionEnabled 启用备份文件压缩
func WithCompressionEnabled(enabled bool) RotatorOption {
	return func(fr *FileRotator) error {
		fr.policy.CompressFiles = enabled
		return nil
	}
}

// WithTimeBasedRotation 启用基于时间的切割
func WithTimeBasedRotation(duration time.Duration) RotatorOption {
	return func(fr *FileRotator) error {
		fr.policy.TimeLimit = duration
		return nil
	}
}

// ApplyOptions 应用选项到文件切割器
func (fr *FileRotator) ApplyOptions(opts ...RotatorOption) error {
	for _, opt := range opts {
		if err := opt(fr); err != nil {
			return err
		}
	}
	return nil
}