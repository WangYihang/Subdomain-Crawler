package queue

import "sync"

// Task represents work item
type Task struct {
	Domain string
	Depth  int
	Root   string
}

// Result represents work result
type Result struct {
	Domain        string
	Root          string
	Subdomains    []string
	IPs           []string // DNS resolution result
	Title         string   // from HTTP <title>
	ContentLength int64    // from HTTP Content-Length, -1 when unknown
	Error         string
}

// JobQueue manages tasks
type JobQueue struct {
	ch chan Task
}

// NewJobQueue creates queue
func NewJobQueue(capacity int) *JobQueue {
	return &JobQueue{ch: make(chan Task, capacity)}
}

// Enqueue adds task
func (q *JobQueue) Enqueue(task Task) bool {
	select {
	case q.ch <- task:
		return true
	default:
		return false
	}
}

// Dequeue gets task
func (q *JobQueue) Dequeue() (Task, bool) {
	task, ok := <-q.ch
	return task, ok
}

// Close closes queue
func (q *JobQueue) Close() {
	close(q.ch)
}

// ResultQueue manages results
type ResultQueue struct {
	ch chan Result
	mu sync.Mutex
}

// NewResultQueue creates result queue
func NewResultQueue(capacity int) *ResultQueue {
	return &ResultQueue{ch: make(chan Result, capacity)}
}

// Send sends result
func (q *ResultQueue) Send(result Result) bool {
	select {
	case q.ch <- result:
		return true
	default:
		return false
	}
}

// Receive gets result
func (q *ResultQueue) Receive() (Result, bool) {
	result, ok := <-q.ch
	return result, ok
}

// Close closes queue
func (q *ResultQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	close(q.ch)
}
