// Package jobs provides adapter-friendly background jobs plus a bounded
// in-memory worker and interval scheduler for single-binary applications.
package jobs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Job struct {
	Name        string
	Payload     []byte
	Attempts    int
	AvailableAt time.Time
}
type Handler func(context.Context, Job) error
type Queue interface {
	Enqueue(context.Context, Job) error
}

type Options struct {
	Workers, Capacity, MaxAttempts int
	RetryDelay                     time.Duration
	OnError                        func(Job, error)
}
type Memory struct {
	jobs     chan Job
	handlers map[string]Handler
	options  Options
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.RWMutex
}

func NewMemory(options Options) *Memory {
	if options.Workers <= 0 {
		options.Workers = 1
	}
	if options.Capacity <= 0 {
		options.Capacity = 128
	}
	if options.MaxAttempts <= 0 {
		options.MaxAttempts = 3
	}
	if options.RetryDelay <= 0 {
		options.RetryDelay = 100 * time.Millisecond
	}
	ctx, cancel := context.WithCancel(context.Background())
	queue := &Memory{jobs: make(chan Job, options.Capacity), handlers: map[string]Handler{}, options: options, ctx: ctx, cancel: cancel}
	for range options.Workers {
		queue.wg.Add(1)
		go queue.work()
	}
	return queue
}
func (queue *Memory) Handle(name string, handler Handler) {
	queue.mu.Lock()
	queue.handlers[name] = handler
	queue.mu.Unlock()
}
func (queue *Memory) Enqueue(ctx context.Context, job Job) error {
	if job.Name == "" {
		return errors.New("job name is required")
	}
	job.Payload = append([]byte(nil), job.Payload...)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-queue.ctx.Done():
		return errors.New("queue is closed")
	case queue.jobs <- job:
		return nil
	}
}
func (queue *Memory) work() {
	defer queue.wg.Done()
	for {
		select {
		case <-queue.ctx.Done():
			return
		case job := <-queue.jobs:
			queue.run(job)
		}
	}
}
func (queue *Memory) run(job Job) {
	if delay := time.Until(job.AvailableAt); delay > 0 {
		timer := time.NewTimer(delay)
		select {
		case <-queue.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
	queue.mu.RLock()
	handler := queue.handlers[job.Name]
	queue.mu.RUnlock()
	var err error
	if handler == nil {
		err = fmt.Errorf("no handler registered for job %q", job.Name)
	} else {
		err = handler(queue.ctx, job)
	}
	if err == nil {
		return
	}
	job.Attempts++
	if job.Attempts < queue.options.MaxAttempts {
		job.AvailableAt = time.Now().Add(queue.options.RetryDelay)
		_ = queue.Enqueue(queue.ctx, job)
		return
	}
	if queue.options.OnError != nil {
		queue.options.OnError(job, err)
	}
}
func (queue *Memory) Shutdown(ctx context.Context) error {
	queue.cancel()
	done := make(chan struct{})
	go func() { queue.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type Scheduler struct {
	queue  Queue
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewScheduler(queue Queue) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{queue: queue, ctx: ctx, cancel: cancel}
}
func (scheduler *Scheduler) Every(interval time.Duration, job Job) error {
	if interval <= 0 {
		return errors.New("schedule interval must be positive")
	}
	if job.Name == "" {
		return errors.New("job name is required")
	}
	scheduler.wg.Add(1)
	go func() {
		defer scheduler.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-scheduler.ctx.Done():
				return
			case <-ticker.C:
				_ = scheduler.queue.Enqueue(scheduler.ctx, job)
			}
		}
	}()
	return nil
}
func (scheduler *Scheduler) Shutdown(ctx context.Context) error {
	scheduler.cancel()
	done := make(chan struct{})
	go func() { scheduler.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
