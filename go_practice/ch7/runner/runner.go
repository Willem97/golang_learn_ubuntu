// Package runner 管理处理任务的运行和生命周期
package runner

import (
	"errors"
	"os"
	"os/signal"
	"time"
)

// Runner 在给定的超时时间内执行一组任务
// 并且在操作系统发送中断信号时结束这个任务
type Runner struct {
	// interrupt 通道报告从操作系统发送的信号
	interrupt chan os.Signal

	// complete通道报告处理任务已经完成
	complete chan error

	// timeout报告处理任务已经超时
	timeout <-chan time.Time

	// tasks持有一组以索引顺序依次执行的函数
	tasks []func(int)
}

// ErrTimeout 会在任务执行超时时返回
var ErrTimeout = errors.New("received timeout")

// ErrInterrupt 会在接收到操作系统的事件时返回
var ErrInterrupt = errors.New("received interrupt")

// New 返回一个新的准备使用的 Runner
func New(d time.Duration) *Runner {
	return &Runner{
		interrupt: make(chan os.Signal, 1),
		complete:  make(chan error),
		timeout:   time.After(d),
	}
}

// Add 将一个任务附加到Runner上，这个任务是一个接收一个int类型的ID作为参数的函数
func (r *Runner) Add(tasks ...func(int)) {
	r.tasks = append(r.tasks, tasks...)
}

// Start 执行所有任务，并监视通道事件
func (r *Runner) Start() error {
	signal.Notify(r.interrupt, os.Interrupt)

	// 用不同的goroutine执行不同的任务
	go func() {
		r.complete <- r.run()
	}()

	select {
	// 当任务处理完成时发出信号
	case err := <-r.complete:
		return err
	case <-r.timeout:
		return ErrTimeout
	}
}

// run执行每一个已注册的任务
func (r *Runner) run() error {
	for id, task := range r.tasks {
		if r.gotInterrupt() {
			return ErrInterrupt
		}

		task(id)
	}

	return nil
}

func (r *Runner) gotInterrupt() bool {
	select {
	// 当中断时间被触发时发出的信号
	case <-r.interrupt:
		// 停止接收后续的任何信号
		signal.Stop(r.interrupt)
		return true
	// 继续正常运行
	default:
		return false
	}
}
