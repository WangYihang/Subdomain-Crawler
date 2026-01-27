package common

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
)

// Job represents a task to be processed by workers
type Job struct {
	Domain string // The domain to crawl
	Depth  int    // The current depth level
	Root   string // The root domain of this job
}

// Scheduler is the core control center of the system
type Scheduler struct {
	// Configuration
	NumWorkers      int
	MaxDepth        int
	JobQueueSize    int
	Timeout         time.Duration
	OutputFile      string
	FindingsFile    string
	SessionFile     string
	BloomFilterFile string

	// Core components
	ScopeManager     *ScopeManager
	WildcardDetector *WildcardDetector
	BloomFilter      *GlobalBloomFilter

	// Job management
	JobQueue chan Job
	Results  chan JobResult

	// Coordination
	wg sync.WaitGroup
	mu sync.Mutex

	// Statistics
	TotalProcessed  int64
	TotalQueued     int64
	TotalDiscovered int64

	// Progress tracking
	ProgressBar progress.Model

	// Graceful shutdown
	stopChan     chan struct{}
	queueClosed  sync.Once
	resultClosed sync.Once
}

// JobResult represents the result of processing a job
type JobResult struct {
	Domain     string
	Root       string
	Subdomains []string
	Error      string
	StartTime  int64
	EndTime    int64
}

// FindingResult represents a simplified result with DNS and HTTP title info
type FindingResult struct {
	Domain string   `json:"domain"`
	IPs    []string `json:"ips"`
	Title  string   `json:"title"`
}

// NewScheduler creates a new scheduler with the given configuration
func NewScheduler(
	numWorkers int,
	maxDepth int,
	jobQueueSize int,
	timeout time.Duration,
	outputFile string,
) *Scheduler {
	return &Scheduler{
		NumWorkers:      numWorkers,
		MaxDepth:        maxDepth,
		JobQueueSize:    jobQueueSize,
		Timeout:         timeout,
		OutputFile:      outputFile,
		FindingsFile:    "findings.jsonl",
		SessionFile:     "session.bloom",
		BloomFilterFile: "bloom.filter",
		JobQueue:        make(chan Job, jobQueueSize),
		Results:         make(chan JobResult, jobQueueSize),
		stopChan:        make(chan struct{}),
		ProgressBar:     progress.New(progress.WithDefaultGradient()),
	}
}

// Initialize initializes the scheduler with root domains and other components
func (s *Scheduler) Initialize(rootDomains []string) error {
	// Initialize scope manager
	InitScopeManager(rootDomains)
	s.ScopeManager = GlobalScopeManager

	// Initialize wildcard detector
	InitWildcardDetector(s.Timeout)
	s.WildcardDetector = GlobalWildcardDetector

	// Initialize bloom filter
	InitGlobalBloomFilter(1048576, 0.01)
	s.BloomFilter = BloomFilter

	// Start periodic bloom filter save
	s.BloomFilter.StartPeriodicSave(s.BloomFilterFile, time.Minute)

	// Detect wildcard IPs for each root domain
	for _, rootDomain := range rootDomains {
		rootDomain = strings.ToLower(strings.TrimSpace(rootDomain))
		if rootDomain != "" {
			// Try to detect wildcard IPs
			_, _ = s.WildcardDetector.DetectWildcardIPs(rootDomain)
		}
	}

	return nil
}

// EnqueueJob adds a job to the queue if it passes all checks
// Returns true if the job was enqueued, false otherwise
func (s *Scheduler) EnqueueJob(domain, root string, depth int) bool {
	// Scope check
	if !s.ScopeManager.IsAllowed(domain) {
		return false
	}

	// Depth check
	if depth > s.MaxDepth {
		return false
	}

	// Wildcard check
	if s.WildcardDetector.IsWildcard(domain, root) {
		return false
	}

	// Deduplication check - create a key for the domain
	key := []byte(domain)
	if s.BloomFilter.TestAndAdd(key) {
		// Already processed or in queue
		return false
	}

	// All checks passed, enqueue the job
	select {
	case s.JobQueue <- Job{Domain: domain, Depth: depth, Root: root}:
		s.mu.Lock()
		s.TotalQueued++
		s.mu.Unlock()
		return true
	case <-s.stopChan:
		// Shutdown signal received, don't enqueue
		return false
	default:
		// Queue is full, try to enqueue with timeout
		timer := time.NewTimer(time.Second)
		select {
		case s.JobQueue <- Job{Domain: domain, Depth: depth, Root: root}:
			timer.Stop()
			s.mu.Lock()
			s.TotalQueued++
			s.mu.Unlock()
			return true
		case <-timer.C:
			return false
		case <-s.stopChan:
			// Shutdown signal received, don't enqueue
			timer.Stop()
			return false
		}
	}
}

// Start starts the scheduler and its worker pool
// It blocks until the scheduler is shut down
func (s *Scheduler) Start(initialDomains []string) error {
	// Initialize with root domains
	if err := s.Initialize(initialDomains); err != nil {
		return err
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Enqueue initial jobs
	for _, domain := range initialDomains {
		domain = strings.ToLower(strings.TrimSpace(domain))
		if domain != "" {
			s.EnqueueJob(domain, domain, 0)
		}
	}

	// Start workers
	for i := 0; i < s.NumWorkers; i++ {
		s.wg.Add(1)
		go s.workerLoop(i)
	}

	// Start result writers
	s.wg.Add(1)
	go s.resultWriter()

	s.wg.Add(1)
	go s.findingsWriter()

	// Start progress updater
	s.wg.Add(1)
	go s.progressUpdater()

	// Handle graceful shutdown in a separate goroutine
	go func() {
		sig := <-sigChan
		fmt.Fprintf(os.Stderr, "\n\nReceived signal: %v\n", sig)
		fmt.Fprintf(os.Stderr, "Gracefully shutting down...\n")
		close(s.stopChan)
		// Close the job queue to wake up any blocked senders
		s.queueClosed.Do(func() {
			close(s.JobQueue)
		})
	}()

	// Wait for all work to complete
	s.wg.Wait()

	// Clean up
	s.queueClosed.Do(func() {
		close(s.JobQueue)
	})
	s.resultClosed.Do(func() {
		close(s.Results)
	})

	// Print final statistics
	s.printFinalStats()

	// Save bloom filter
	_ = s.BloomFilter.SaveToFile(s.BloomFilterFile)

	return nil
}

// workerLoop is the main loop for a worker
func (s *Scheduler) workerLoop(workerID int) {
	defer s.wg.Done()

	for {
		select {
		case job, ok := <-s.JobQueue:
			if !ok {
				// Job queue closed, exit worker
				return
			}

			s.mu.Lock()
			s.TotalProcessed++
			s.mu.Unlock()

			// Process the job
			result := s.processJob(job)

			// Send result to result channel
			select {
			case s.Results <- result:
			case <-s.stopChan:
				// Shutdown signal received, exit worker
				return
			}

			// Enqueue discovered subdomains
			for _, subdomain := range result.Subdomains {
				// Check if we should stop
				select {
				case <-s.stopChan:
					return
				default:
				}

				s.EnqueueJob(subdomain, job.Root, job.Depth+1)
			}

		case <-s.stopChan:
			// Graceful shutdown signal
			return
		}
	}
}

// processJob processes a single job
// This performs HTTP crawling and subdomain extraction
func (s *Scheduler) processJob(job Job) JobResult {
	result := JobResult{
		Domain:     job.Domain,
		Root:       job.Root,
		Subdomains: []string{},
		StartTime:  time.Now().Unix(),
	}

	// Try both HTTPS and HTTP
	protocols := []string{"https", "http"}
	var response *http.Response
	var lastErr error

	for _, protocol := range protocols {
		url := protocol + "://" + job.Domain + "/"

		// Create HTTP request with context for timeout
		ctx, cancel := context.WithTimeout(context.Background(), s.Timeout)
		request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		defer cancel()

		if err != nil {
			lastErr = err
			continue
		}

		// Add random User-Agent to avoid being blocked
		request.Header.Set("User-Agent", GetRandomUserAgent())
		request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		request.Header.Set("Accept-Language", "en-US,en;q=0.5")

		// Send the HTTP request
		response, err = GetHTTPClient().Do(request)
		if err != nil {
			lastErr = err
			continue
		}

		// Check response status
		if response.StatusCode >= 500 {
			response.Body.Close()
			lastErr = fmt.Errorf("server error: %d", response.StatusCode)
			continue
		}

		// Successfully got a response
		defer response.Body.Close()

		// Extract subdomains from response
		result.Subdomains = extractSubdomainsFromResponse(response, job.Root)
		result.EndTime = time.Now().Unix()
		return result
	}

	// All protocols failed
	if lastErr != nil {
		result.Error = lastErr.Error()
	} else {
		result.Error = "Failed to get response from domain"
	}
	result.EndTime = time.Now().Unix()
	return result
}

// extractSubdomainsFromResponse extracts subdomains from HTTP response
// It searches both the response body and headers for domain names
func extractSubdomainsFromResponse(response *http.Response, rootDomain string) []string {
	subdomainMap := make(map[string]bool)

	// Extract from response body
	if response.Body != nil {
		bodyDomains := extractDomainsFromBody(response.Body)
		for _, domain := range bodyDomains {
			domain = normalizeSubdomain(domain, rootDomain)
			if domain != "" {
				subdomainMap[domain] = true
			}
		}
	}

	// Extract from response headers
	headerDomains := extractDomainsFromHeaders(response.Header)
	for _, domain := range headerDomains {
		domain = normalizeSubdomain(domain, rootDomain)
		if domain != "" {
			subdomainMap[domain] = true
		}
	}

	// Convert map to slice
	var result []string
	for domain := range subdomainMap {
		result = append(result, domain)
	}
	return result
}

// extractDomainsFromBody extracts domain names from response body using regex
func extractDomainsFromBody(body io.ReadCloser) []string {
	var result []string
	if body == nil {
		return result
	}
	defer body.Close()

	// Read response body with size limit (10MB)
	limitedBody := io.LimitReader(body, 10*1024*1024)
	data, err := io.ReadAll(limitedBody)
	if err != nil {
		return result
	}

	// Convert to string
	bodyStr := string(data)

	// Regex patterns for domain extraction
	// Match URLs: http://domain, https://domain, //domain
	urlPattern := regexp.MustCompile(`(?:https?:)?//([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}`)

	// Match bare domains
	domainPattern := regexp.MustCompile(`\b([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}\b`)

	// Extract from URLs
	for _, match := range urlPattern.FindAllString(bodyStr, -1) {
		// Remove protocol prefix
		domain := strings.TrimPrefix(match, "https://")
		domain = strings.TrimPrefix(domain, "http://")
		domain = strings.TrimPrefix(domain, "//")
		domain = strings.Split(domain, "/")[0] // Remove path
		domain = strings.Split(domain, ":")[0] // Remove port
		domain = strings.ToLower(strings.TrimSpace(domain))
		if domain != "" {
			result = append(result, domain)
		}
	}

	// Extract bare domains
	for _, match := range domainPattern.FindAllString(bodyStr, -1) {
		domain := strings.ToLower(strings.TrimSpace(match))
		if domain != "" && !isDomainInList(domain, result) {
			result = append(result, domain)
		}
	}

	return result
}

// extractDomainsFromHeaders extracts domain names from HTTP headers
func extractDomainsFromHeaders(headers http.Header) []string {
	var result []string

	// Check specific headers that might contain domains
	headerNames := []string{
		"Content-Security-Policy",
		"Content-Security-Policy-Report-Only",
		"Link",
		"Location",
		"Set-Cookie",
	}

	for _, headerName := range headerNames {
		values := headers.Values(headerName)
		for _, value := range values {
			// Extract domains from header value
			domains := extractDomainsFromString(value)
			result = append(result, domains...)
		}
	}

	return result
}

// extractDomainsFromString extracts domains from a string using regex
func extractDomainsFromString(str string) []string {
	var result []string
	domainPattern := regexp.MustCompile(`([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}`)

	for _, match := range domainPattern.FindAllString(str, -1) {
		domain := strings.ToLower(strings.TrimSpace(match))
		if domain != "" {
			result = append(result, domain)
		}
	}

	return result
}

// normalizeSubdomain normalizes a domain and checks if it's a valid subdomain of rootDomain
func normalizeSubdomain(domain, rootDomain string) string {
	// Convert to lowercase
	domain = strings.ToLower(domain)
	rootDomain = strings.ToLower(rootDomain)

	// Remove port if present
	domain = strings.Split(domain, ":")[0]

	// Trim whitespace
	domain = strings.TrimSpace(domain)

	// Basic validation: must be a valid domain
	if domain == "" || !isValidDomain(domain) {
		return ""
	}

	// Check if it's the root domain or a subdomain
	if domain == rootDomain {
		return domain
	}

	if strings.HasSuffix(domain, "."+rootDomain) {
		return domain
	}

	// Not part of this root domain
	return ""
}

// isValidDomain checks if a string is a valid domain name
func isValidDomain(domain string) bool {
	if len(domain) == 0 || len(domain) > 253 {
		return false
	}

	// Must have at least one dot (except for localhost-like tests)
	if !strings.Contains(domain, ".") {
		return false
	}

	// Split into labels
	labels := strings.Split(domain, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}

		// Labels must start and end with alphanumeric
		if !isAlphanumeric(rune(label[0])) || !isAlphanumeric(rune(label[len(label)-1])) {
			return false
		}

		// Labels can only contain alphanumeric and hyphens
		for _, ch := range label {
			if !isAlphanumeric(ch) && ch != '-' {
				return false
			}
		}
	}

	return true
}

// isAlphanumeric checks if a rune is alphanumeric
func isAlphanumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

// isDomainInList checks if a domain is already in a list
func isDomainInList(domain string, list []string) bool {
	for _, d := range list {
		if d == domain {
			return true
		}
	}
	return false
}

// resultWriter writes results to the output file in JSON format
func (s *Scheduler) resultWriter() {
	defer s.wg.Done()

	f, err := os.OpenFile(s.OutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open output file: %v\n", err)
		return
	}
	defer f.Close()

	for result := range s.Results {
		// Write result to file in JSON format
		line := fmt.Sprintf(`{"domain":"%s","root":"%s","subdomains":%d,"error":"%s","start_time":%d,"end_time":%d}`,
			result.Domain, result.Root, len(result.Subdomains), result.Error, result.StartTime, result.EndTime)

		// Add subdomains list if any
		if len(result.Subdomains) > 0 {
			// Reconstruct JSON with subdomains
			subdomainsJSON := "["
			for i, sub := range result.Subdomains {
				if i > 0 {
					subdomainsJSON += ","
				}
				subdomainsJSON += fmt.Sprintf(`"%s"`, sub)
			}
			subdomainsJSON += "]"

			line = fmt.Sprintf(`{"domain":"%s","root":"%s","subdomains":%s,"error":"%s","start_time":%d,"end_time":%d}`,
				result.Domain, result.Root, subdomainsJSON, result.Error, result.StartTime, result.EndTime)
		}

		f.WriteString(line + "\n")
	}
}

// queryDNS queries DNS to get IP addresses for a domain
func (s *Scheduler) queryDNS(domain string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), s.Timeout)
	defer cancel()

	resolver := &net.Resolver{}
	ips, err := resolver.LookupIPAddr(ctx, domain)
	if err != nil {
		return []string{}
	}

	var ipAddrs []string
	for _, ip := range ips {
		ipAddrs = append(ipAddrs, ip.IP.String())
	}
	return ipAddrs
}

// extractTitle extracts the title from HTTP response body
func (s *Scheduler) extractTitle(domain string) string {
	ctx, cancel := context.WithTimeout(context.Background(), s.Timeout)
	defer cancel()

	// Try HTTPS first, then HTTP
	for _, protocol := range []string{"https", "http"} {
		url := fmt.Sprintf("%s://%s/", protocol, domain)
		request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue
		}
		request.Header.Set("User-Agent", GetRandomUserAgent())

		resp, err := GetHTTPClient().Do(request)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		// Read response body with size limit
		limitedReader := io.LimitReader(resp.Body, 10*1024*1024) // 10MB limit
		body, err := io.ReadAll(limitedReader)
		if err != nil {
			continue
		}

		// Extract title from HTML
		titleRegex := regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
		matches := titleRegex.FindStringSubmatch(string(body))
		if len(matches) > 1 {
			title := strings.TrimSpace(matches[1])
			// Escape quotes and newlines for JSON
			title = strings.ReplaceAll(title, "\"", "\\\"")
			title = strings.ReplaceAll(title, "\n", " ")
			return title
		}

		// If we got a response, don't try HTTP after HTTPS
		if protocol == "https" {
			return ""
		}
	}

	return ""
}

// findingsWriter writes findings to a separate file with domain, IPs, and title
func (s *Scheduler) findingsWriter() {
	defer s.wg.Done()

	f, err := os.OpenFile(s.FindingsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open findings file: %v\n", err)
		return
	}
	defer f.Close()

	processed := make(map[string]bool)

	for result := range s.Results {
		// Skip if error or no subdomains
		if result.Error != "" || len(result.Subdomains) == 0 {
			continue
		}

		// Process each found subdomain
		for _, subdomain := range result.Subdomains {
			if _, exists := processed[subdomain]; !exists {
				processed[subdomain] = true
				atomic.AddInt64(&s.TotalDiscovered, 1)

				// Query DNS
				ips := s.queryDNS(subdomain)

				// Extract title
				title := s.extractTitle(subdomain)

				// Escape IPs for JSON
				var ipsJSON string
				if len(ips) > 0 {
					ipsJSON = "["
					for i, ip := range ips {
						if i > 0 {
							ipsJSON += ","
						}
						ipsJSON += fmt.Sprintf(`"%s"`, ip)
					}
					ipsJSON += "]"
				} else {
					ipsJSON = "[]"
				}

				// Write to file
				line := fmt.Sprintf(`{"domain":"%s","ips":%s,"title":"%s"}`,
					subdomain, ipsJSON, title)
				f.WriteString(line + "\n")
			}
		}
	}
}

// LoadRootDomainsFromFile loads root domains from a file
func LoadRootDomainsFromFile(filepath string) ([]string, error) {
	var domains []string

	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		domain := strings.TrimSpace(scanner.Text())
		if domain != "" && !strings.HasPrefix(domain, "#") {
			domains = append(domains, domain)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return domains, nil
}

// progressUpdater periodically updates and displays the progress bar
func (s *Scheduler) progressUpdater() {
	defer s.wg.Done()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	maxEstimate := int64(1000)

	for {
		select {
		case <-ticker.C:
			processed := atomic.LoadInt64(&s.TotalProcessed)
			percentage := float64(processed) / float64(maxEstimate)
			if percentage > 1.0 {
				percentage = 0.99
			}
			s.ProgressBar.SetPercent(percentage)

			// Render the progress bar
			fmt.Printf("\r%s", s.ProgressBar.View())

		case <-s.stopChan:
			return
		}
	}
}

// printFinalStats prints final statistics using lipgloss
func (s *Scheduler) printFinalStats() {
	s.mu.Lock()
	processed := s.TotalProcessed
	queued := s.TotalQueued
	s.mu.Unlock()
	discovered := atomic.LoadInt64(&s.TotalDiscovered)

	// Define styles
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true).
		Padding(1, 2)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("11")).
		Bold(true)

	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render(strings.Repeat("─", 70))

	// Build output
	title := titleStyle.Render("✨ Crawling Complete")

	stats := fmt.Sprintf(
		"%s\n📊 Statistics:\n  %s Domains Processed       %s\n  %s Domains Queued         %s\n  %s Subdomains Discovered  %s\n",
		divider,
		keyStyle.Render("✓"),
		valueStyle.Render(fmt.Sprintf("%d", processed)),
		keyStyle.Render("✓"),
		valueStyle.Render(fmt.Sprintf("%d", queued)),
		keyStyle.Render("✓"),
		valueStyle.Render(fmt.Sprintf("%d", discovered)),
	)

	files := fmt.Sprintf(
		"\n📁 Output Files:\n  %s Detailed Log            %s\n  %s Findings Results       %s\n%s",
		keyStyle.Render("✓"),
		valueStyle.Render(s.OutputFile),
		keyStyle.Render("✓"),
		valueStyle.Render(s.FindingsFile),
		divider,
	)

	success := lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")).
		Bold(true).
		Render("✅ Crawl finished successfully!\n")

	// Print everything
	fmt.Println("\n" + title)
	fmt.Println(stats + files)
	fmt.Println(success)
}
