package main

import (
	"time"

	"github.com/civet148/log/v2"
)

func init() {
	log.SetLevel(log.LevelDebug)
	log.SetJsonFormatter()
	log.SetFileName("output-json.log") //同时输出到文件
}

func main() {
	// 示例1: 使用WithAllJson选项，所有级别都采用JSON格式输出
	loggerJsonExample()

	// 示例2: 使用默认的Json方法，将参数序列化为普通消息字符串
	defaultJsonExample()
}

// loggerJsonExample 演示WithAllJson选项的使用
func loggerJsonExample() {
	println("=== 示例1: 新建logger输出JSON ===")

	// 创建使用WithAllJson选项的logger
	logger, err := log.NewLogger(
		log.WithJsonFormatter(), // 所有级别都采用JSON格式输出
	)
	if err != nil {
		log.Errorf("创建logger失败: %v", err)
		return
	}
	defer logger.Close()

	// 这些日志都将采用JSON格式输出
	logger.Tracef("这是一条TRACE级别的日志")
	logger.Debugf("这是一条DEBUG级别的日志")
	logger.Info("这是一条INFO级别的日志")
	logger.Warn("这是一条WARN级别的日志")
	logger.Error("这是一条ERROR级别的日志")

	// Json方法也会将参数序列化为普通消息字符串后采用JSON格式输出
	logger.Json(map[string]interface{}{
		"user_id": 123,
		"action":  "login",
		"success": true,
	})

	println()
}

// defaultJsonExample 演示默认的Json方法行为
func defaultJsonExample() {
	println("=== 示例2: 默认Json方法行为 ===")

	// 普通日志级别
	log.Tracef("这是一条TRACE级别的日志")
	log.Debugf("这是一条DEBUG级别的日志")
	log.Info("这是一条INFO级别的日志")
	log.Warn("这是一条WARN级别的日志")
	log.Error("这是一条ERROR级别的日志")

	// Json方法将参数序列化为JSON并使用专用的JSON输出器
	log.Json(map[string]interface{}{
		"user_id":   456,
		"action":    "logout",
		"success":   true,
		"timestamp": time.Now(), // 包含时间戳
	})

	// 多参数的Json方法
	log.Json("user_operation", "file_upload", map[string]interface{}{
		"file_name": "document.pdf",
		"file_size": 1024,
	}, true)

	println()
}
