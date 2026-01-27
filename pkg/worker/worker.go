package worker

import (
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/WangYihang/Subdomain-Crawler/pkg/dedup"
	"github.com/WangYihang/Subdomain-Crawler/pkg/dns"
	"github.com/WangYihang/Subdomain-Crawler/pkg/domain"
	"github.com/WangYihang/Subdomain-Crawler/pkg/fetcher"
	"github.com/WangYihang/Subdomain-Crawler/pkg/output"
	"github.com/WangYihang/Subdomain-Crawler/pkg/queue"
)

// Protocols to try
var Protocols = []string{"http", "https"}

// ActivityTracker reports current domain per worker for progress display.
type ActivityTracker interface {
	Set(workerID int, domain string)
}

// Worker processes tasks
type Worker struct {
	id              int
	jobs            *queue.JobQueue
	results         *queue.ResultQueue
	fetcher         *fetcher.Fetcher
	resolver        *dns.Resolver
	scope           *domain.Scope
	calculator      *domain.Calculator
	dedup           *dedup.Filter
	activity        ActivityTracker
	dnsLog          *output.JsonlWriter
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
	Resolver   *dns.Resolver
	Scope      *domain.Scope
	Calculator *domain.Calculator
	Dedup      *dedup.Filter
	Activity   ActivityTracker
	DnsLog     *output.JsonlWriter
	StopChan   <-chan struct{}
}

// NewWorker creates worker
func NewWorker(config *Config) *Worker {
	return &Worker{
		id:         config.ID,
		jobs:       config.Jobs,
		results:    config.Results,
		fetcher:    config.Fetcher,
		resolver:   config.Resolver,
		scope:      config.Scope,
		calculator: config.Calculator,
		dedup:      config.Dedup,
		activity:   config.Activity,
		dnsLog:     config.DnsLog,
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
	if w.activity != nil {
		w.activity.Set(w.id, task.Domain)
		defer w.activity.Set(w.id, "")
	}

	result := w.fetcher.Fetch(task.Domain, task.Root, Protocols)

	var uniqueSubdomains []string
	for _, subdomain := range result.Subdomains {
		if !w.dedup.TestAndAdd([]byte(subdomain)) {
			uniqueSubdomains = append(uniqueSubdomains, subdomain)
		}
	}

	result.Subdomains = uniqueSubdomains

	var ips []string
	var dnsErr string
	if w.resolver != nil {
		reqAt := time.Now()
		resolved, err := w.resolver.Resolve(task.Domain)
		rttMs := time.Since(reqAt).Milliseconds()
		if err != nil {
			dnsErr = err.Error()
		} else {
			ips = resolved
		}
		if w.dnsLog != nil {
			answers := make([]map[string]string, 0, len(ips))
			for _, ipStr := range ips {
				typ := "AAAA"
				if ip := net.ParseIP(ipStr); ip != nil && ip.To4() != nil {
					typ = "A"
				}
				answers = append(answers, map[string]string{"ip": ipStr, "type": typ})
			}
			_ = w.dnsLog.Log(map[string]interface{}{
				"request": map[string]interface{}{
					"domain":     task.Domain,
					"types":      []string{"A", "AAAA"},
					"request_at": reqAt.UnixMilli(),
					"timestamp":  reqAt.Format(time.RFC3339Nano),
				},
				"response": map[string]interface{}{
					"answers":     answers,
					"ips":         ips,
					"error":       dnsErr,
					"rtt_ms":      rttMs,
					"response_at": time.Now().UnixMilli(),
					"timestamp":   time.Now().Format(time.RFC3339Nano),
				},
			})
		}
	}

	queueResult := queue.Result{
		Domain:        result.Domain,
		Root:          result.Root,
		Subdomains:    result.Subdomains,
		IPs:           ips,
		Title:         result.Title,
		ContentLength: result.ContentLength,
		Error:         result.Error,
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
