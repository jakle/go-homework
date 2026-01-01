package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// LogLevel 日志级别
type LogLevel int

// 定义日志级别常量，使用iota从0开始自动递增
const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

func (l LogLevel) String() string {
	return []string{"DEBUG", "INFO", "WARN", "ERROR"}[l]
}

// LogEntry 日志属性
type LogEntry struct {
	Level   LogLevel
	Message string
	Time    time.Time
}

// Logger 并发安全的日志系统
type Logger struct {
	entries    chan LogEntry  // 日志条目通道，用于异步处理日志
	wg         sync.WaitGroup // 用于等待写入goroutine完成
	file       *os.File       // 日志输出文件
	consoleOut bool           // 是否同时输出到控制台
	mu         sync.RWMutex   // 保护文件写入的读写锁
	running    bool           // 记录日志系统是否正在运行
}

// NewLogger 创建新的日志系统
func NewLogger(filename string, consoleOutput bool) (*Logger, error) {
	var file *os.File
	var err error

	if filename != "" {
		file, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
	}

	logger := Logger{
		entries:    make(chan LogEntry, 1000), // 缓冲通道
		file:       file,
		consoleOut: consoleOutput,
		running:    true,
	}

	// 启动日志写入goroutine
	logger.wg.Add(1)
	go logger.writeLoop()

	return &logger, nil
}

// writeLoop 日志写入循环
func (l *Logger) writeLoop() {
	defer l.wg.Done()

	for entry := range l.entries {
		logMsg := fmt.Sprintf("[%s] %s: %s\n",
			entry.Time.Format("2025-12-31 15:04:05"),
			entry.Level,
			entry.Message)

		// 写入文件
		if l.file != nil {
			l.mu.Lock()
			l.file.WriteString(logMsg)
			l.mu.Unlock()
		}

		// 控制台输出
		if l.consoleOut {
			fmt.Print(logMsg)
		}
	}
}

// Log 记录日志
func (l *Logger) Log(level LogLevel, format string, args ...interface{}) {
	if !l.running {
		return
	}

	entry := LogEntry{
		Level:   level,
		Message: fmt.Sprintf(format, args...),
		Time:    time.Now(),
	}

	select {
	case l.entries <- entry:
	default:
		// 队列已满，丢弃日志
		fmt.Printf("日志队列已满，丢弃日志: %s\n", entry.Message)
	}
}

// 便捷方法
func (l *Logger) Debug(format string, args ...interface{}) {
	l.Log(DEBUG, format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.Log(INFO, format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.Log(WARN, format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.Log(ERROR, format, args...)
}

// Close 关闭日志系统
func (l *Logger) Close() {
	if !l.running {
		return
	}

	l.running = false
	close(l.entries)
	l.wg.Wait() // 等待写入goroutine完成

	if l.file != nil {
		l.file.Close()
	}
}

// 使用示例
func main() {
	fmt.Println("=== 并发安全日志系统demo ===")

	// 创建日志系统（同时输出到文件和控制台）
	logger, err := NewLogger("app.log", true)
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	var wg sync.WaitGroup

	// 启动多个goroutine并发写日志
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				logger.Info("Goroutine %d - 日志 %d", id, j)
				time.Sleep(time.Millisecond * 10)
			}
		}(i)
	}

	// 模拟错误日志
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 3; i++ {
			logger.Error("发生错误: 连接超时 %d", i)
			time.Sleep(time.Millisecond * 50)
		}
	}()

	// 模拟警告日志
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 3; i++ {
			logger.Warn("警告: 服务器负载过高 %d", i)
			time.Sleep(time.Millisecond * 30)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 3; i++ {
			logger.Debug("调试信息: %d", i)
			time.Sleep(time.Millisecond * 20)
		}
	}()

	wg.Wait()
	logger.Info("所有日志写入完成")
}
