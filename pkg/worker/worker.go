package worker

import (
	"sync"
	"sync/atomic"

	"github.com/WangYihang/Subdomain-Crawler/pkg/dedup"
	"github.com/WangYihang/Subdomain-Crawler/pkg/domain"
	"github.com/WangYihang/Subdomain-Crawler/pkg/fetcher"
	"github.com/WangYihang/Subdomain-Crawler/pkg/queue"
)

// Protocols to try
var Protocols = []string{"http", "https"}

// Worker processes tasks
type Worker struct {
	id              int
	jobs            *queue.JobQueue
	results         *queue.ResultQueue
	fetcher         *fetcher.Fetcher
	scope           *domain.Scope
	calculator      *domain.Calculator
	dedup           *dedup.Filter
	stopChan        <-chan struct{}
	wg              *sync.WaitGroup
	tasksProcessed  int64
	subdomainsFound int64
}

// Config holds worker config
type Config struct {
	ID         int
	Jobs       *queue.JobQueue
	Results    *queue.ResultQueue
	Fetcher    *fetcher.Fetcher
	Scope      *domain.Scope
	Calculator *domain.Calculator
	Dedup      *dedup.Filter
	StopChan   <-chan struct{}
}

// NewWorker creates worker
func NewWorker(config *Config) *Worker {
	return &Worker{
		id:         config.ID,
		jobs:       config.Jobs,
		results:    config.Results,
		fetcher:    config.Fetcher,
		scope:      config.Scope,
		calculator: config.Calculator,
		dedup:      config.Dedup,
		stopChan:   config.StopChan,
	}
}

// Start starts worker
func (w *Worker) Start(wg *sync.WaitGroup) {
	w.wg = wg
	wg.Add(1)

	go func() {
		defer wg.Done()
		w.process()
	}()
}

// process processes tasks
func (w *Worker) process() {
	for {
		select {
		case <-w.stopChan:
			return
		default:
		}

		task, ok := w.jobs.Dequeue()
		if !ok {
			return
		}

		w.processTask(task)
	}
}

// processTask processes single task
func (w *Worker) processTask(task queue.Task) {
	if w.calculator.GetDepth(task.Domain) > 2 {
		return
	}

	result := w.fetcher.Fetch(task.Domain, task.Root, Protocols)

	var uniqueSubdomains []string
	for _, subdomain := range result.Subdomains {
		if !w.dedup.TestAndAdd([]byte(subdomain)) {
			uniqueSubdomains = append(uniqueSubdomains, subdomain)
		}
	}

	result.Subdomains = uniqueSubdomains

	queueResult := queue.Result{
		Domain:     result.Domain,
		Root:       result.Root,
		Subdomains: result.Subdomains,
		Error:      result.Error,
	}

	w.results.Send(queueResult)

	w.enqueueSubdomains(task, uniqueSubdomains)

	atomic.AddInt64(&w.tasksProcessed, 1)
	atomic.AddInt64(&w.subdomainsFound, int64(len(uniqueSubdomains)))
}

// enqueueSubdomains enqueues subdomains
func (w *Worker) enqueueSubdomains(parentTask queue.Task, subdomains []string) {
	for _, subdomain := range subdomains {
		if !w.scope.IsAllowed(subdomain) {
			continue
		}

		newDepth := w.calculator.GetDepth(subdomain)
		if newDepth > 3 {
			continue
		}

		newTask := queue.Task{
			Domain: subdomain,
			Depth:  newDepth,
			Root:   parentTask.Root,
		}

		if !w.jobs.Enqueue(newTask) {
			continue
		}
	}
}

// GetStats returns stats
func (w *Worker) GetStats() (processed, found int64) {
	return atomic.LoadInt64(&w.tasksProcessed), atomic.LoadInt64(&w.subdomainsFound)
}
