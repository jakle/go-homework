package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Task 任务接口
type Task interface {
	Execute(ctx context.Context) error
	GetID() string
}

// SimpleTask 简单任务
type SimpleTask struct {
	ID      string
	Timeout time.Duration
}

func NewSimpleTask(id string, timeout time.Duration) *SimpleTask {
	return &SimpleTask{ID: id, Timeout: timeout}
}

func (t *SimpleTask) Execute(ctx context.Context) error {
	fmt.Printf("任务 %s 开始执行\n", t.ID)

	// 创建带超时的上下文
	taskCtx, cancel := context.WithTimeout(ctx, t.Timeout)
	defer cancel()

	// 模拟任务执行
	done := make(chan error, 1)
	go func() {
		// 模拟任务处理时间
		duration := time.Duration(rand.Intn(100)+300) * time.Millisecond
		time.Sleep(duration)

		select {
		case <-taskCtx.Done():
			done <- taskCtx.Err()
		default:
			fmt.Printf("任务 %s 执行成功，耗时 %v\n", t.ID, duration)
			done <- nil
		}
	}()

	select {
	case <-taskCtx.Done():
		return fmt.Errorf("任务 %s 超时: %v", t.ID, taskCtx.Err())
	case err := <-done:
		return err
	}
}

func (t *SimpleTask) GetID() string {
	return t.ID
}

// LongRunningTask 长时间运行任务
type LongRunningTask struct {
	ID      string
	Count   int
	Timeout time.Duration
}

func NewLongRunningTask(id string, count int, timeout time.Duration) *LongRunningTask {
	return &LongRunningTask{ID: id, Count: count, Timeout: timeout}
}

func (t *LongRunningTask) Execute(ctx context.Context) error {
	fmt.Printf("长任务 %s 开始，需要处理 %d 个项目\n", t.ID, t.Count)

	taskCtx, cancel := context.WithTimeout(ctx, t.Timeout)
	defer cancel()

	for i := 0; i < t.Count; i++ {
		select {
		case <-taskCtx.Done():
			return fmt.Errorf("任务 %s 在第 %d 步被取消: %v", t.ID, i, taskCtx.Err())
		default:
			// 模拟处理每个项目
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("任务 %s 进度: %d/%d\n", t.ID, i+1, t.Count)
		}
	}

	fmt.Printf("长任务 %s 完成\n", t.ID)
	return nil
}

func (t *LongRunningTask) GetID() string {
	return t.ID
}

// TaskScheduler 任务调度器
type TaskScheduler struct {
	tasks       []Task
	wg          sync.WaitGroup
	mu          sync.Mutex
	results     map[string]error
	timeout     time.Duration
	workerCount int
}

func NewTaskScheduler(workerCount int, timeout time.Duration) *TaskScheduler {
	return &TaskScheduler{
		tasks:       make([]Task, 0),
		results:     make(map[string]error),
		timeout:     timeout,
		workerCount: workerCount,
	}
}

func (s *TaskScheduler) AddTask(task Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks = append(s.tasks, task)
}

func (s *TaskScheduler) Run() map[string]error {
	taskChan := make(chan Task, len(s.tasks))

	// 添加任务到通道
	for _, task := range s.tasks {
		taskChan <- task
	}
	close(taskChan)

	// 创建主上下文
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	// 启动worker
	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go s.worker(ctx, i, taskChan)
	}

	s.wg.Wait()
	return s.results
}

func (s *TaskScheduler) worker(ctx context.Context, id int, taskChan <-chan Task) {
	defer s.wg.Done()

	for task := range taskChan {
		select {
		case <-ctx.Done():
			// 主上下文已取消，停止处理新任务
			fmt.Printf("Worker %d 停止，原因: %v\n", id, ctx.Err())
			return
		default:
			fmt.Printf("Worker %d 开始处理任务 %s\n", id, task.GetID())

			err := task.Execute(ctx)

			s.mu.Lock()
			s.results[task.GetID()] = err
			s.mu.Unlock()

			if err != nil {
				fmt.Printf("Worker %d 任务 %s 失败: %v\n", id, task.GetID(), err)
			} else {
				fmt.Printf("Worker %d 任务 %s 完成\n", id, task.GetID())
			}
		}
	}
}

func main() {
	fmt.Println("=== 任务调度器demo ===")

	// 初始化随机数种子，用于生成随机的任务处理时间
	//rand.Seed(time.Now().UnixNano())

	// 创建调度器（3个worker，总超时4秒）
	scheduler := NewTaskScheduler(3, 4*time.Second)
	scheduler.AddTask(NewSimpleTask("task-1", 2*time.Second))
	scheduler.AddTask(NewLongRunningTask("long-task-1", 50, 5*time.Second)) //超时
	scheduler.AddTask(NewLongRunningTask("long-task-1", 40, 4*time.Second))
	scheduler.AddTask(NewLongRunningTask("long-task-2", 20, 1*time.Second)) //超时
	scheduler.AddTask(NewSimpleTask("task-3", 500*time.Millisecond))

	start := time.Now()
	results := scheduler.Run()
	since := time.Since(start)

	fmt.Printf("\n=== 任务执行结果 (总耗时: %v) ===\n", since)
	for taskID, err := range results {
		if err != nil {
			fmt.Printf("任务 %s: 失败 - %v\n", taskID, err)
		} else {
			fmt.Printf("任务 %s: 成功\n", taskID)
		}
	}
}
