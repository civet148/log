# 高性能Go日志库

一个功能丰富、高性能的Go日志库，支持多种输出格式、文件切割、并发安全等特性。

## 特性

- 🚀 **高性能**: 内置对象池、零拷贝字符串构建等优化
- 🎨 **多种格式**: 支持彩色终端输出、纯文本、JSON格式
- 📁 **文件切割**: 基于大小的自动文件切割和备份管理
- ⚡ **并发安全**: 线程安全的日志记录
- 🔧 **灵活配置**: 丰富的配置选项
- 📊 **JSON支持**: 专门的JSON日志输出，支持复杂数据结构
- 🎯 **级别控制**: 7个日志级别精确控制

## 快速开始

### 基本使用

```go
package main

import (
	"fmt"
	log "github.com/civet148/log/v2"
)

func init() {
	log.SetLevel(log.LevelTrace)
}

func main() {
	var err error
	// 1. 基本日志输出
	fmt.Println("1. 基本日志输出:")
	log.Trace("这是TRACE级别日志", "1")
	log.Debug("这是DEBUG级别日志", "2")
	log.Info("这是INFO级别日志", 3)
	log.Warn("这是WARN级别日志", 4)
	log.Error("这是ERROR级别日志", 5)
	err = log.Error(fmt.Errorf("打印一个error对象"))
	fmt.Printf("err: %s\n", err)
	err = log.Errorf("返回一个error对象")
	fmt.Printf("err: %s\n", err)
	fmt.Println()
}
```
