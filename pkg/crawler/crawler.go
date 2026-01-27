package crawler

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/WangYihang/Subdomain-Crawler/pkg/config"
	"github.com/WangYihang/Subdomain-Crawler/pkg/dedup"
	"github.com/WangYihang/Subdomain-Crawler/pkg/dns"
	"github.com/WangYihang/Subdomain-Crawler/pkg/domain"
	"github.com/WangYihang/Subdomain-Crawler/pkg/extract"
	"github.com/WangYihang/Subdomain-Crawler/pkg/fetcher"
	"github.com/WangYihang/Subdomain-Crawler/pkg/httpclient"
	"github.com/WangYihang/Subdomain-Crawler/pkg/input"
	"github.com/WangYihang/Subdomain-Crawler/pkg/output"
	"github.com/WangYihang/Subdomain-Crawler/pkg/queue"
	"github.com/WangYihang/Subdomain-Crawler/pkg/worker"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/net/publicsuffix"
)

// progressTracker records current domain per worker for progress display.
type progressTracker struct {
	mu      sync.RWMutex
	current map[int]string
}

func (p *progressTracker) Set(workerID int, domain string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.current == nil {
		p.current = make(map[int]string)
	}
	if domain == "" {
		delete(p.current, workerID)
	} else {
		p.current[workerID] = domain
	}
}

func (p *progressTracker) snapshot() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var out []string
	for _, d := range p.current {
		if d != "" {
			out = append(out, d)
		}
	}
	return out
}

// Crawler coordinates crawling
type Crawler struct {
	cfg         *config.Config
	rootDomains []string
	extractor   *domain.Extractor
	calculator  *domain.Calculator
	scope       *domain.Scope
	dedupFilter *dedup.Filter
	dnsResolver *dns.Resolver
	progress    *progressTracker
	jobQueue    *queue.JobQueue
	resultQueue *queue.ResultQueue
	httpclient  *httpclient.Client
	fetcher     *fetcher.Fetcher
	workers     []*worker.Worker
	writer      *output.Writer
	httpLog     *output.JsonlWriter
	dnsLog      *output.JsonlWriter
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

	httpLog, err := output.NewJsonlWriter(cfg.Output.HttpLogFile)
	if err != nil {
		return nil, fmt.Errorf("open http log: %w", err)
	}
	dnsLog, err := output.NewJsonlWriter(cfg.Output.DnsLogFile)
	if err != nil {
		_ = httpLog.Close()
		return nil, fmt.Errorf("open dns log: %w", err)
	}

	filter := extract.NewFilter("")
	fetcherConfig := &fetcher.Config{
		Client:          httpClient,
		Filter:          filter,
		MaxResponseSize: 10 * 1024 * 1024,
		HttpLog:         httpLog,
	}
	fetcherInstance := fetcher.NewFetcher(fetcherConfig)

	dnsResolver := dns.NewResolver(cfg.HTTP.Timeout)
	progress := &progressTracker{current: make(map[int]string)}

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
		dnsResolver: dnsResolver,
		progress:    progress,
		jobQueue:    jobQueue,
		resultQueue: resultQueue,
		httpclient:  httpClient,
		fetcher:     fetcherInstance,
		writer:      writer,
		httpLog:     httpLog,
		dnsLog:      dnsLog,
		flusher:     flusher,
		stopChan:    make(chan struct{}),
		stats:       &Stats{},
	}, nil
}

// runProgress updates progressbar description with queue length, request counts, and unique subdomains.
func (c *Crawler) runProgress(done <-chan struct{}, bar *progressbar.ProgressBar, interval time.Duration) {
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		select {
		case <-done:
			return
		case <-tick.C:
			queueLen := c.jobQueue.Len()
			var httpCount, dnsCount, uniqueCount int64
			for _, w := range c.workers {
				if w == nil {
					continue
				}
				processed, found := w.GetStats()
				httpCount += processed // 1 HTTP fetch per task
				dnsCount += processed  // 1 DNS resolve per task
				uniqueCount += found
			}

			// Count active workers (those currently processing tasks)
			activeWorkers := 0
			currentDomains := c.progress.snapshot()
			activeWorkers = len(currentDomains)

			// Build description with more detailed info
			desc := "Queue: " + strconv.Itoa(queueLen)
			if activeWorkers > 0 {
				desc += " | Active: " + strconv.Itoa(activeWorkers) + "/" + strconv.Itoa(len(c.workers))
			}
			desc += " | HTTP: " + strconv.FormatInt(httpCount, 10) +
				", DNS: " + strconv.FormatInt(dnsCount, 10) +
				" | Unique: " + strconv.FormatInt(uniqueCount, 10)

			// Show some active domains being processed
			if len(currentDomains) > 0 {
				desc += " | Processing: "
				maxShow := 3
				if len(currentDomains) > maxShow {
					for i := 0; i < maxShow; i++ {
						if i > 0 {
							desc += ", "
						}
						desc += currentDomains[i]
					}
					desc += "..."
				} else {
					for i, d := range currentDomains {
						if i > 0 {
							desc += ", "
						}
						desc += d
					}
				}
			}

			bar.Describe(desc)
		}
	}
}

// Start starts crawling
func (c *Crawler) Start() error {
	log.Printf("Starting crawler with %d root domains", len(c.rootDomains))

	pm := dedup.NewPersistenceManager(c.dedupFilter, 1*time.Minute)
	pm.StartPeriodicSave(c.cfg.Dedup.BloomFilterFile)

	c.flusher.Start()

	// -1 = indeterminate spinner; output to stderr so stdout can be used for results
	bar := progressbar.NewOptions64(-1,
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSetDescription("Queue: 0"),
		progressbar.OptionShowDescriptionAtLineEnd(),
	)
	progressDone := make(chan struct{})
	go c.runProgress(progressDone, bar, 400*time.Millisecond)

	c.workers = make([]*worker.Worker, c.cfg.Concurrency.NumWorkers)
	for i := 0; i < c.cfg.Concurrency.NumWorkers; i++ {
		workerConfig := &worker.Config{
			ID:         i,
			Jobs:       c.jobQueue,
			Results:    c.resultQueue,
			Fetcher:    c.fetcher,
			Resolver:   c.dnsResolver,
			Scope:      c.scope,
			Calculator: c.calculator,
			Dedup:      c.dedupFilter,
			Activity:   c.progress,
			DnsLog:     c.dnsLog,
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
		_ = bar.Finish()
		close(progressDone)
		c.resultQueue.Close()
	}()

	c.flusher.Stop()

	// Newline after progress bar so log line isn't glued to the bar
	fmt.Fprintln(os.Stderr)

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
	if c.httpLog != nil {
		_ = c.httpLog.Close()
	}
	if c.dnsLog != nil {
		_ = c.dnsLog.Close()
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
