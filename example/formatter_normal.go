package main

import (
	"fmt"
	"time"

	log "github.com/civet148/log/v2"
)

func init() {
	log.SetLevel(log.LevelTrace)
	log.EnableShowStack()
}

func main() {
	fmt.Println("=== logex 使用示例 ===\n")

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

	// 2. 格式化日志输出
	fmt.Println("2. 格式化日志输出:")
	user := "Alice"
	count := 42
	log.Infof("用户 %s 已登录，当前在线用户数: %d", user, count)
	log.Debugf("调试信息: %v", map[string]int{"connections": count})
	fmt.Println()

	// 3. JSON日志输出
	fmt.Println("3. JSON日志输出:")

	// 简单JSON对象
	log.Json(map[string]interface{}{
		"event":      "user_login",
		"user_id":    123,
		"username":   "alice",
		"success":    true,
		"login_time": time.Now(),
	})

	// 复杂JSON对象
	userData := struct {
		ID       int       `json:"id"`
		Name     string    `json:"name"`
		Email    string    `json:"email"`
		Roles    []string  `json:"roles"`
		Active   bool      `json:"active"`
		LastSeen time.Time `json:"last_seen"`
	}{
		ID:       456,
		Name:     "Bob Smith",
		Email:    "bob@example.com",
		Roles:    []string{"admin", "user"},
		Active:   true,
		LastSeen: time.Now(),
	}
	log.Json(userData)

	// 多参数JSON
	log.Json("operation", "file_upload", map[string]interface{}{
		"file_name": "document.pdf",
		"file_size": 1048576,
		"mime_type": "application/pdf",
	}, true)

	// 数组JSON
	log.Json([]string{"task1", "task2", "task3"})
	fmt.Println()

	// 4. 配置示例
	fmt.Println("4. 自定义配置:")

	// 创建自定义Logger
	customLogger, err := log.NewLogger(
		log.WithLevel(log.LevelDebug),
		log.WithLogFile("example.log"),
		log.WithFileSize(10), // 10MB
		log.WithMaxBackups(3),
	)
	if err != nil {
		log.Errorf("创建自定义logger失败: %v", err)
		return
	}
	defer customLogger.Close()

	customLogger.Info("这是来自自定义logger的消息")
	customLogger.Json(map[string]interface{}{
		"logger":    "fields",
		"msg":       "自定义logger的JSON输出",
		"timestamp": time.Now(),
	})
	fmt.Println()

	// 5. 错误处理示例
	fmt.Println("5. 错误处理:")

	err = log.Error("这是一个错误示例")
	if err != nil {
		fmt.Printf("捕获到错误: %v\n", err)
	}

	err = customLogger.Errorf("格式化错误: %s", "网络连接失败")
	if err != nil {
		fmt.Printf("捕获到格式化错误: %v\n", err)
	}
	fmt.Println()

	// 6. 不同配置演示
	fmt.Println("6. 不同配置演示:")

	// 仅文件输出
	fileLogger, err := log.NewLogger(
		log.WithLevel(log.LevelInfo),
		log.WithLogFile("file_only.log"),
		log.WithDisableConsole(),
	)
	if err == nil {
		fileLogger.Info("这条消息只会写入文件")
		fileLogger.Close()
		fmt.Println("已写入仅文件日志 (file_only.log)")
	}

	// 无颜色输出
	noColorLogger, err := log.NewLogger(
		log.WithLevel(log.LevelInfo),
	)
	if err == nil {
		noColorLogger.Info("这是无颜色的日志输出")
		noColorLogger.Close()
	}
	fmt.Println()

	// 7. 级别过滤演示
	fmt.Println("7. 级别过滤演示:")

	// 设置为WARN级别，只显示WARN及以上级别
	warnLogger, err := log.NewLogger(
		log.WithLevel(log.LevelWarn),
	)
	if err == nil {
		fmt.Println("设置为WARN级别，以下只会显示WARN和ERROR:")
		warnLogger.Debug("这条DEBUG不会显示")
		warnLogger.Info("这条INFO不会显示")
		warnLogger.Warn("这条WARN会显示")
		warnLogger.Error("这条ERROR会显示")
		warnLogger.Close()
	}
	fmt.Println()

	// 8. 性能测试示例
	fmt.Println("8. 性能测试:")

	start := time.Now()
	perfLogger, err := log.NewLogger(
		log.WithLevel(log.LevelInfo),
		log.WithLogFile("perf_test.log"),
	)
	if err == nil {
		for i := 0; i < 10; i++ {
			perfLogger.Infof("性能测试消息 #%d", i)
			if i%100 == 0 {
				perfLogger.Json(map[string]interface{}{
					"batch":     i/100 + 1,
					"processed": i + 1,
					"timestamp": time.Now(),
				})
			}
		}
		perfLogger.Flush()
		perfLogger.Close()
	}
	duration := time.Since(start)
	fmt.Printf("写入10条日志耗时: %v\n", duration)
	fmt.Println()

	// 9. 新功能演示：WithJsonFormatter选项
	fmt.Println("9. 新功能演示：WithJsonFormatter选项:")

	// 创建使用WithJsonFormatter选项的logger
	jsonFormatterLogger, err := log.NewLogger(
		log.WithJsonFormatter(),
	)
	if err == nil {
		jsonFormatterLogger.Info("这条INFO消息将以JSON格式输出")
		jsonFormatterLogger.Warn("这条WARN消息也将以JSON格式输出")

		// Json方法将参数序列化为普通消息字符串后也采用JSON格式输出
		jsonFormatterLogger.Json(map[string]interface{}{
			"feature": "WithJsonFormatter",
			"status":  "demonstrated",
		})
		jsonFormatterLogger.Close()
	}
	fmt.Println()

	// 10. 清理和刷新
	fmt.Println("10. 清理和刷新:")
	log.Flush() // 刷新全局logger
	fmt.Println("所有日志已刷新到磁盘")

	fmt.Println("\n=== 示例完成 ===")
	fmt.Println("生成的日志文件:")
	fmt.Println("- example.log (自定义logger)")
	fmt.Println("- file_only.log (仅文件输出)")
	fmt.Println("- perf_test.log (性能测试)")
}
