package storage

import (
	"sync"

	"github.com/WangYihang/Subdomain-Crawler/pkg/domain/entity"
	"github.com/WangYihang/Subdomain-Crawler/pkg/domain/repository"
)

// TaskQueue implements repository.TaskQueue
type TaskQueue struct {
	ch     chan *entity.Task
	closed bool
	mu     sync.RWMutex
}

// NewTaskQueue creates a new task queue
func NewTaskQueue(size int) repository.TaskQueue {
	return &TaskQueue{
		ch: make(chan *entity.Task, size),
	}
}

// Enqueue adds a task to the queue
func (q *TaskQueue) Enqueue(task *entity.Task) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.closed {
		return false
	}

	select {
	case q.ch <- task:
		return true
	default:
		return false
	}
}

// Dequeue removes and returns a task from the queue
func (q *TaskQueue) Dequeue() (*entity.Task, bool) {
	task, ok := <-q.ch
	return task, ok
}

// Len returns the current queue length
func (q *TaskQueue) Len() int {
	return len(q.ch)
}

// Close closes the queue
func (q *TaskQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.closed {
		q.closed = true
		close(q.ch)
	}
}

// ResultQueue implements repository.ResultQueue
type ResultQueue struct {
	ch     chan *entity.CrawlResult
	closed bool
	mu     sync.RWMutex
}

// NewResultQueue creates a new result queue
func NewResultQueue(size int) repository.ResultQueue {
	return &ResultQueue{
		ch: make(chan *entity.CrawlResult, size),
	}
}

// Send sends a result to the queue
func (q *ResultQueue) Send(result *entity.CrawlResult) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if !q.closed {
		q.ch <- result
	}
}

// Receive receives a result from the queue
func (q *ResultQueue) Receive() (*entity.CrawlResult, bool) {
	result, ok := <-q.ch
	return result, ok
}

// Close closes the queue
func (q *ResultQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.closed {
		q.closed = true
		close(q.ch)
	}
}
