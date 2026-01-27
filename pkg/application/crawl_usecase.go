package application

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/WangYihang/Subdomain-Crawler/pkg/domain/entity"
	"github.com/WangYihang/Subdomain-Crawler/pkg/domain/repository"
	"github.com/WangYihang/Subdomain-Crawler/pkg/domain/service"
)

// CrawlUseCase orchestrates the subdomain crawling process
type CrawlUseCase struct {
	config Config

	// Services
	validator  service.DomainValidator
	calculator service.DomainCalculator
	extractor  service.DomainExtractor
	fetcher    service.HTTPFetcher
	resolver   service.DNSResolver

	// Repositories
	filter       repository.DomainFilter
	taskQueue    repository.TaskQueue
	resultQueue  repository.ResultQueue
	resultWriter repository.ResultWriter
	logWriter    repository.LogWriter

	// State
	metrics          *entity.Metrics
	metricsLock      sync.RWMutex
	workers          []*Worker
	stopChan         chan struct{}
	wg               sync.WaitGroup
	metricsObservers []MetricsObserver
}

// Config holds the use case configuration
type Config struct {
	NumWorkers  int
	MaxDepth    int
	Protocols   []string
	RootDomains []string
}

// MetricsObserver observes metrics changes
type MetricsObserver interface {
	OnMetricsUpdate(metrics *entity.Metrics)
	AddSubdomain(subdomain string) // Notify when a new subdomain is discovered
}

// NewCrawlUseCase creates a new crawl use case
func NewCrawlUseCase(
	config Config,
	validator service.DomainValidator,
	calculator service.DomainCalculator,
	extractor service.DomainExtractor,
	fetcher service.HTTPFetcher,
	resolver service.DNSResolver,
	filter repository.DomainFilter,
	taskQueue repository.TaskQueue,
	resultQueue repository.ResultQueue,
	resultWriter repository.ResultWriter,
	logWriter repository.LogWriter,
) *CrawlUseCase {
	return &CrawlUseCase{
		config:           config,
		validator:        validator,
		calculator:       calculator,
		extractor:        extractor,
		fetcher:          fetcher,
		resolver:         resolver,
		filter:           filter,
		taskQueue:        taskQueue,
		resultQueue:      resultQueue,
		resultWriter:     resultWriter,
		logWriter:        logWriter,
		metrics:          &entity.Metrics{TotalWorkers: config.NumWorkers},
		stopChan:         make(chan struct{}),
		metricsObservers: make([]MetricsObserver, 0),
	}
}

// RegisterMetricsObserver registers a metrics observer
func (uc *CrawlUseCase) RegisterMetricsObserver(observer MetricsObserver) {
	uc.metricsObservers = append(uc.metricsObservers, observer)
}

// notifyMetricsObservers notifies all registered observers
func (uc *CrawlUseCase) notifyMetricsObservers() {
	uc.metricsLock.RLock()
	metrics := *uc.metrics
	uc.metricsLock.RUnlock()

	for _, observer := range uc.metricsObservers {
		observer.OnMetricsUpdate(&metrics)
	}
}

// Execute executes the crawl use case
func (uc *CrawlUseCase) Execute(ctx context.Context) error {
	// Update start time
	uc.metricsLock.Lock()
	uc.metrics.StartTime = time.Now()
	uc.metricsLock.Unlock()

	// Start periodic metrics updates
	go uc.updateMetricsPeriodically(ctx)

	// Start result flusher
	go uc.flushResults(ctx)

	// Start workers
	uc.startWorkers()

	// Enqueue initial tasks
	if err := uc.enqueueRootDomains(); err != nil {
		return fmt.Errorf("failed to enqueue root domains: %w", err)
	}

	// Wait for context cancellation or completion
	select {
	case <-ctx.Done():
		uc.Stop()
		return ctx.Err()
	case <-uc.waitForCompletion():
		return nil
	}
}

// updateMetricsPeriodically periodically updates and notifies observers
func (uc *CrawlUseCase) updateMetricsPeriodically(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			uc.metricsLock.Lock()
			uc.metrics.QueueLength = uc.taskQueue.Len()
			uc.metrics.LastUpdateTime = time.Now()

			// Count active workers and collect their current domains
			activeWorkers := 0
			var activeDomains []string
			for _, worker := range uc.workers {
				if worker != nil && worker.IsActive() {
					activeWorkers++
					if domain := worker.GetCurrentDomain(); domain != "" {
						activeDomains = append(activeDomains, domain)
					}
				}
			}
			uc.metrics.ActiveWorkers = activeWorkers
			uc.metrics.ActiveDomains = activeDomains
			uc.metricsLock.Unlock()

			uc.notifyMetricsObservers()
		}
	}
}

// startWorkers starts all worker goroutines
func (uc *CrawlUseCase) startWorkers() {
	uc.workers = make([]*Worker, uc.config.NumWorkers)
	for i := 0; i < uc.config.NumWorkers; i++ {
		worker := &Worker{
			id:          i,
			useCase:     uc,
			taskQueue:   uc.taskQueue,
			resultQueue: uc.resultQueue,
			fetcher:     uc.fetcher,
			resolver:    uc.resolver,
			validator:   uc.validator,
			calculator:  uc.calculator,
			extractor:   uc.extractor,
			filter:      uc.filter,
			logWriter:   uc.logWriter,
			stopChan:    uc.stopChan,
			maxDepth:    uc.config.MaxDepth,
			protocols:   uc.config.Protocols,
		}
		uc.workers[i] = worker
		uc.wg.Add(1)
		go worker.Run(&uc.wg)
	}
}

// enqueueRootDomains enqueues the initial root domains
func (uc *CrawlUseCase) enqueueRootDomains() error {
	for _, domain := range uc.config.RootDomains {
		root, err := uc.calculator.GetRoot(domain)
		if err != nil {
			root = domain
		}

		task := &entity.Task{
			Domain: entity.Domain{
				Name:  domain,
				Root:  root,
				Depth: 0,
			},
			Protocols: uc.config.Protocols,
		}

		if !uc.taskQueue.Enqueue(task) {
			return fmt.Errorf("failed to enqueue domain: %s", domain)
		}

		// Track enqueued tasks
		atomic.AddInt64(&uc.metrics.TasksEnqueued, 1)
	}
	return nil
}

// flushResults continuously flushes results to the writer
func (uc *CrawlUseCase) flushResults(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			result, ok := uc.resultQueue.Receive()
			if !ok {
				return
			}
			if err := uc.resultWriter.Write(result); err != nil {
				// Log error but continue
				continue
			}
		}
	}
}

// waitForCompletion waits for all workers to complete
func (uc *CrawlUseCase) waitForCompletion() <-chan struct{} {
	done := make(chan struct{})
	go func() {
		uc.wg.Wait()
		uc.resultQueue.Close()
		close(done)
	}()
	return done
}

// Stop stops the crawl use case
func (uc *CrawlUseCase) Stop() {
	close(uc.stopChan)
	uc.taskQueue.Close()
	uc.wg.Wait()
	uc.resultWriter.Flush()
	uc.resultWriter.Close()
	uc.logWriter.Close()
}

// GetMetrics returns the current metrics
func (uc *CrawlUseCase) GetMetrics() *entity.Metrics {
	uc.metricsLock.RLock()
	defer uc.metricsLock.RUnlock()

	metrics := *uc.metrics
	metrics.QueueLength = uc.taskQueue.Len()

	// Count active workers
	activeWorkers := 0
	for _, worker := range uc.workers {
		if worker.IsActive() {
			activeWorkers++
		}
	}
	metrics.ActiveWorkers = activeWorkers

	return &metrics
}

// incrementHTTPRequests increments the HTTP request counter
func (uc *CrawlUseCase) incrementHTTPRequests(success bool) {
	atomic.AddInt64(&uc.metrics.HTTPRequests, 1)
	if success {
		atomic.AddInt64(&uc.metrics.SuccessCount, 1)
	}
	uc.notifyMetricsObservers()
}

// incrementDNSRequests increments the DNS request counter
func (uc *CrawlUseCase) incrementDNSRequests() {
	atomic.AddInt64(&uc.metrics.DNSRequests, 1)
	uc.notifyMetricsObservers()
}

// incrementUniqueSubdomains increments the unique subdomains counter
func (uc *CrawlUseCase) incrementUniqueSubdomains(count int64) {
	atomic.AddInt64(&uc.metrics.UniqueSubdomains, count)
	uc.notifyMetricsObservers()
}

// incrementTasksProcessed increments the tasks processed counter
func (uc *CrawlUseCase) incrementTasksProcessed() {
	atomic.AddInt64(&uc.metrics.TasksProcessed, 1)
	uc.notifyMetricsObservers()
}

// incrementErrorCount increments the error counter
func (uc *CrawlUseCase) incrementErrorCount() {
	atomic.AddInt64(&uc.metrics.ErrorCount, 1)
	uc.notifyMetricsObservers()
}
