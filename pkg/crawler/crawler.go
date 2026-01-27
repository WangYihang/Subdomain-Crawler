package crawler

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/WangYihang/Subdomain-Crawler/pkg/config"
	"github.com/WangYihang/Subdomain-Crawler/pkg/dedup"
	"github.com/WangYihang/Subdomain-Crawler/pkg/domain"
	"github.com/WangYihang/Subdomain-Crawler/pkg/extract"
	"github.com/WangYihang/Subdomain-Crawler/pkg/fetcher"
	"github.com/WangYihang/Subdomain-Crawler/pkg/httpclient"
	"github.com/WangYihang/Subdomain-Crawler/pkg/input"
	"github.com/WangYihang/Subdomain-Crawler/pkg/output"
	"github.com/WangYihang/Subdomain-Crawler/pkg/queue"
	"github.com/WangYihang/Subdomain-Crawler/pkg/worker"
	"golang.org/x/net/publicsuffix"
)

// Crawler coordinates crawling
type Crawler struct {
	cfg         *config.Config
	rootDomains []string
	extractor   *domain.Extractor
	calculator  *domain.Calculator
	scope       *domain.Scope
	dedupFilter *dedup.Filter
	jobQueue    *queue.JobQueue
	resultQueue *queue.ResultQueue
	httpclient  *httpclient.Client
	fetcher     *fetcher.Fetcher
	workers     []*worker.Worker
	writer      *output.Writer
	flusher     *output.Flusher
	stopChan    chan struct{}
	wg          sync.WaitGroup
	stats       *Stats
}

// Stats holds statistics
type Stats struct {
	TasksProcessed  int64
	SubdomainsFound int64
	TotalDiscovered int64
	mu              sync.RWMutex
}

// NewCrawler creates crawler
func NewCrawler(cfg *config.Config) (*Crawler, error) {
	loader := input.NewLoader()
	rootDomains, err := loader.Load(cfg.Input.File)
	if err != nil {
		return nil, fmt.Errorf("failed to load root domains: %w", err)
	}

	if len(rootDomains) == 0 {
		return nil, fmt.Errorf("no root domains loaded from %s", cfg.Input.File)
	}

	// Normalize roots to eTLD+1 (registrable domain) so that *.tsinghua.edu.cn
	// is in scope when input is "www.tsinghua.edu.cn".
	rootsForScope := make(map[string]bool)
	for _, d := range rootDomains {
		base, err := publicsuffix.EffectiveTLDPlusOne(strings.TrimSpace(d))
		if err == nil && base != "" {
			rootsForScope[strings.ToLower(base)] = true
		} else {
			rootsForScope[strings.ToLower(strings.TrimSpace(d))] = true
		}
	}
	var rootsList []string
	for r := range rootsForScope {
		rootsList = append(rootsList, r)
	}
	extractor := domain.NewExtractor(rootsList)
	calculator := domain.NewCalculator(extractor)
	scope := domain.NewScope(extractor)

	dedupFilter := dedup.NewFilter(cfg.Dedup.BloomFilterSize, cfg.Dedup.BloomFilterFalsePositive)
	_ = dedupFilter.LoadFromFile(cfg.Dedup.BloomFilterFile)

	httpClient := httpclient.NewClient(&httpclient.Config{
		Timeout: cfg.HTTP.Timeout,
	})

	filter := extract.NewFilter("")
	fetcherConfig := &fetcher.Config{
		Client:          httpClient,
		Filter:          filter,
		MaxResponseSize: 10 * 1024 * 1024,
	}
	fetcherInstance := fetcher.NewFetcher(fetcherConfig)

	jobQueue := queue.NewJobQueue(cfg.Concurrency.QueueSize)
	resultQueue := queue.NewResultQueue(cfg.Concurrency.QueueSize)

	writer, err := output.NewWriter(cfg.Output.ResultsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create output writer: %w", err)
	}

	flusher := output.NewFlusher(writer, resultQueue)

	return &Crawler{
		cfg:         cfg,
		rootDomains: rootDomains,
		extractor:   extractor,
		calculator:  calculator,
		scope:       scope,
		dedupFilter: dedupFilter,
		jobQueue:    jobQueue,
		resultQueue: resultQueue,
		httpclient:  httpClient,
		fetcher:     fetcherInstance,
		writer:      writer,
		flusher:     flusher,
		stopChan:    make(chan struct{}),
		stats:       &Stats{},
	}, nil
}

// Start starts crawling
func (c *Crawler) Start() error {
	log.Printf("Starting crawler with %d root domains", len(c.rootDomains))

	pm := dedup.NewPersistenceManager(c.dedupFilter, 1*time.Minute)
	pm.StartPeriodicSave(c.cfg.Dedup.BloomFilterFile)

	c.flusher.Start()

	c.workers = make([]*worker.Worker, c.cfg.Concurrency.NumWorkers)
	for i := 0; i < c.cfg.Concurrency.NumWorkers; i++ {
		workerConfig := &worker.Config{
			ID:         i,
			Jobs:       c.jobQueue,
			Results:    c.resultQueue,
			Fetcher:    c.fetcher,
			Scope:      c.scope,
			Calculator: c.calculator,
			Dedup:      c.dedupFilter,
			StopChan:   c.stopChan,
		}
		c.workers[i] = worker.NewWorker(workerConfig)
		c.workers[i].Start(&c.wg)
	}

	for _, d := range c.rootDomains {
		d = strings.TrimSpace(d)
		root, err := publicsuffix.EffectiveTLDPlusOne(d)
		if err != nil || root == "" {
			root = d
		} else {
			root = strings.ToLower(root)
		}
		c.jobQueue.Enqueue(queue.Task{
			Domain: d,
			Depth:  0,
			Root:   root,
		})
	}

	go func() {
		c.wg.Wait()
		c.resultQueue.Close()
	}()

	c.flusher.Stop()

	return c.Close()
}

// Stop stops crawler
func (c *Crawler) Stop() {
	close(c.stopChan)
	c.jobQueue.Close()
}

// Close closes resources
func (c *Crawler) Close() error {
	if c.writer != nil {
		_ = c.writer.Close()
	}
	if c.dedupFilter != nil {
		_ = c.dedupFilter.SaveToFile(c.cfg.Dedup.BloomFilterFile)
	}
	return nil
}

// GetStats returns statistics
func (c *Crawler) GetStats() map[string]interface{} {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()

	return map[string]interface{}{
		"tasks_processed":  c.stats.TasksProcessed,
		"subdomains_found": c.stats.SubdomainsFound,
		"total_discovered": c.stats.TotalDiscovered,
	}
}
