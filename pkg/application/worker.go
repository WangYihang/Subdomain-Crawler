package application

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/WangYihang/Subdomain-Crawler/pkg/domain/entity"
	"github.com/WangYihang/Subdomain-Crawler/pkg/domain/repository"
	"github.com/WangYihang/Subdomain-Crawler/pkg/domain/service"
)

// Worker processes crawling tasks
type Worker struct {
	id          int
	useCase     *CrawlUseCase
	taskQueue   repository.TaskQueue
	resultQueue repository.ResultQueue
	fetcher     service.HTTPFetcher
	resolver    service.DNSResolver
	validator   service.DomainValidator
	calculator  service.DomainCalculator
	extractor   service.DomainExtractor
	filter      repository.DomainFilter
	logWriter   repository.LogWriter
	stopChan    <-chan struct{}
	maxDepth    int
	protocols   []string

	currentDomain atomic.Value // stores string
	isActive      atomic.Bool
}

// Run starts the worker processing loop
func (w *Worker) Run(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-w.stopChan:
			return
		default:
		}

		task, ok := w.taskQueue.Dequeue()
		if !ok {
			return
		}

		w.processTask(task)
	}
}

// IsActive returns whether the worker is currently processing a task
func (w *Worker) IsActive() bool {
	return w.isActive.Load()
}

// GetCurrentDomain returns the domain currently being processed
func (w *Worker) GetCurrentDomain() string {
	if v := w.currentDomain.Load(); v != nil {
		return v.(string)
	}
	return ""
}

// processTask processes a single crawling task
func (w *Worker) processTask(task *entity.Task) {
	w.isActive.Store(true)
	w.currentDomain.Store(task.Domain.Name)
	defer func() {
		w.isActive.Store(false)
		w.currentDomain.Store("")
		w.useCase.incrementTasksProcessed()
	}()

	// Check depth limit
	if task.Domain.Depth > w.maxDepth {
		return
	}

	// Fetch HTTP content
	var subdomains []string
	var crawlResult *entity.CrawlResult
	successfulFetch := false

	for _, protocol := range task.Protocols {
		url := fmt.Sprintf("%s://%s", protocol, task.Domain.Name)
		resp, err := w.fetcher.Fetch(url)

		success := err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300
		w.useCase.incrementHTTPRequests(success)

		// Log HTTP request
		w.logWriter.WriteHTTPLog(map[string]any{
			"url":         url,
			"status_code": resp.StatusCode,
			"error":       err,
		})

		if err != nil {
			w.useCase.incrementErrorCount()
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			successfulFetch = true
			// Extract subdomains from response
			domains := w.extractor.ExtractFromText(resp.Body)
			filtered := w.extractor.FilterByRoot(domains, task.Domain.Root)
			subdomains = append(subdomains, filtered...)

			// Extract title
			title := w.extractor.ExtractTitle(resp.Body)

			crawlResult = &entity.CrawlResult{
				Domain:        task.Domain.Name,
				Root:          task.Domain.Root,
				Protocol:      protocol,
				StatusCode:    resp.StatusCode,
				Title:         title,
				ContentLength: resp.ContentLength,
				Subdomains:    filtered,
			}
			break // Success, no need to try other protocols
		}
	}

	if !successfulFetch {
		w.useCase.incrementErrorCount()
	}

	// Deduplicate subdomains
	uniqueSubdomains := w.deduplicateSubdomains(subdomains)

	// Notify observers of new discoveries
	for _, subdomain := range uniqueSubdomains {
		for _, observer := range w.useCase.metricsObservers {
			observer.AddSubdomain(subdomain)
		}
	}

	// Resolve DNS
	ips, dnsErr := w.resolveDNS(task.Domain.Name)
	w.useCase.incrementDNSRequests()

	// Update crawl result
	if crawlResult != nil {
		crawlResult.Subdomains = uniqueSubdomains
		crawlResult.IPs = ips
		if dnsErr != nil {
			crawlResult.Error = dnsErr.Error()
		}
		w.resultQueue.Send(crawlResult)
	}

	// Enqueue unique subdomains for further crawling
	w.enqueueSubdomains(task, uniqueSubdomains)

	// Update metrics
	w.useCase.incrementUniqueSubdomains(int64(len(uniqueSubdomains)))
}

// deduplicateSubdomains removes duplicate subdomains
func (w *Worker) deduplicateSubdomains(subdomains []string) []string {
	unique := make([]string, 0)
	for _, subdomain := range subdomains {
		subdomain = strings.ToLower(strings.TrimSpace(subdomain))
		if subdomain == "" {
			continue
		}

		if !w.filter.Contains(subdomain) {
			w.filter.Add(subdomain)
			unique = append(unique, subdomain)
		}
	}
	return unique
}

// resolveDNS resolves the domain to IP addresses
func (w *Worker) resolveDNS(domain string) ([]string, error) {
	resolution, err := w.resolver.ResolveWithDetails(domain)
	if err != nil {
		return nil, err
	}

	// Log DNS query
	w.logWriter.WriteDNSLog(map[string]any{
		"domain": domain,
		"ips":    resolution.IPs,
		"server": resolution.Server,
		"rtt_ms": resolution.RTTMs,
		"error":  resolution.Error,
	})

	return resolution.IPs, nil
}

// enqueueSubdomains enqueues discovered subdomains for crawling
func (w *Worker) enqueueSubdomains(parentTask *entity.Task, subdomains []string) {
	for _, subdomain := range subdomains {
		// Validate scope
		if !w.validator.IsInScope(subdomain, parentTask.Domain.Root) {
			continue
		}

		// Calculate depth
		newDepth := w.calculator.GetDepth(subdomain)
		if newDepth > w.maxDepth {
			continue
		}

		// Create new task
		newTask := &entity.Task{
			Domain: entity.Domain{
				Name:  subdomain,
				Root:  parentTask.Domain.Root,
				Depth: newDepth,
			},
			Protocols: w.protocols,
		}

		if w.taskQueue.Enqueue(newTask) {
			// Track enqueued tasks
			atomic.AddInt64(&w.useCase.metrics.TasksEnqueued, 1)
		}
	}
}
