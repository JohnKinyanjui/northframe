package jobs

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestMemoryRetriesAndSchedulerEnqueues(t *testing.T) {
	failed := make(chan struct{}, 1)
	queue := NewMemory(Options{RetryDelay: time.Millisecond, MaxAttempts: 2, OnError: func(Job, error) { failed <- struct{}{} }})
	queue.Handle("retry", func(context.Context, Job) error { return errors.New("no") })
	if err := queue.Enqueue(context.Background(), Job{Name: "retry"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-failed:
	case <-time.After(time.Second):
		t.Fatal("job was not retried")
	}
	var calls atomic.Int32
	queue.Handle("tick", func(context.Context, Job) error { calls.Add(1); return nil })
	scheduler := NewScheduler(queue)
	if err := scheduler.Every(5*time.Millisecond, Job{Name: "tick"}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = scheduler.Shutdown(ctx)
	_ = queue.Shutdown(ctx)
	if calls.Load() == 0 {
		t.Fatal("scheduled job did not run")
	}
}
