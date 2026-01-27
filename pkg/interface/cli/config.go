package cli

import (
	"flag"
	"fmt"
	"os"
	"time"
)

// Config holds all application configuration
type Config struct {
	// Input/Output
	InputFile   string
	OutputFile  string
	HTTPLogFile string
	DNSLogFile  string

	// Crawling
	MaxDepth   int
	Protocols  []string
	NumWorkers int
	QueueSize  int
	ExpandSLD  bool // Expand SLD to common subdomains

	// HTTP
	HTTPTimeout     time.Duration
	MaxResponseSize int64
	UserAgent       string

	// DNS
	DNSTimeout time.Duration
	DNSServers []string

	// Dedup
	BloomFilterSize uint
	BloomFilterFP   float64
	BloomFilterFile string

	// UI
	ShowDashboard bool
}

// ParseFlags parses command line flags
func ParseFlags() (*Config, error) {
	cfg := &Config{}

	// Input/Output
	flag.StringVar(&cfg.InputFile, "i", "input.txt", "Input file containing root domains (one per line)")
	flag.StringVar(&cfg.OutputFile, "o", "result.jsonl", "Output file for crawl results")
	flag.StringVar(&cfg.HTTPLogFile, "http-log", "http.jsonl", "HTTP request/response log file")
	flag.StringVar(&cfg.DNSLogFile, "dns-log", "dns.jsonl", "DNS query/response log file")

	// Crawling
	flag.IntVar(&cfg.MaxDepth, "max-depth", 3, "Maximum subdomain depth to crawl")
	flag.IntVar(&cfg.NumWorkers, "workers", 32, "Number of concurrent workers")
	flag.IntVar(&cfg.QueueSize, "queue-size", 10000, "Size of task queue")
	flag.BoolVar(&cfg.ExpandSLD, "expand-sld", true, "Automatically expand SLD with common subdomains (www, api, mail, etc.)")

	// HTTP
	var httpTimeout int
	flag.IntVar(&httpTimeout, "http-timeout", 10, "HTTP request timeout in seconds")
	flag.Int64Var(&cfg.MaxResponseSize, "max-response-size", 10*1024*1024, "Maximum HTTP response size in bytes")
	flag.StringVar(&cfg.UserAgent, "user-agent", "SubdomainCrawler/2.0", "HTTP User-Agent header")

	// DNS
	var dnsTimeout int
	flag.IntVar(&dnsTimeout, "dns-timeout", 5, "DNS query timeout in seconds")

	// Dedup
	var bloomSize uint64
	flag.Uint64Var(&bloomSize, "bloom-size", 1000000, "Bloom filter size (number of expected elements)")
	flag.Float64Var(&cfg.BloomFilterFP, "bloom-fp", 0.01, "Bloom filter false positive rate")
	flag.StringVar(&cfg.BloomFilterFile, "bloom-file", "bloom.filter", "Bloom filter persistence file")

	// UI
	flag.BoolVar(&cfg.ShowDashboard, "dashboard", true, "Show interactive TUI dashboard")

	// Help
	var showHelp bool
	flag.BoolVar(&showHelp, "h", false, "Show help message")
	flag.BoolVar(&showHelp, "help", false, "Show help message")

	flag.Parse()

	if showHelp {
		printHelp()
		os.Exit(0)
	}

	// Convert timeouts
	cfg.HTTPTimeout = time.Duration(httpTimeout) * time.Second
	cfg.DNSTimeout = time.Duration(dnsTimeout) * time.Second

	// Set bloom filter size
	cfg.BloomFilterSize = uint(bloomSize)

	// Default protocols
	cfg.Protocols = []string{"https", "http"}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.NumWorkers <= 0 {
		return fmt.Errorf("number of workers must be > 0, got %d", c.NumWorkers)
	}

	if c.MaxDepth < 0 {
		return fmt.Errorf("max depth must be >= 0, got %d", c.MaxDepth)
	}

	if c.QueueSize <= 0 {
		return fmt.Errorf("queue size must be > 0, got %d", c.QueueSize)
	}

	if c.HTTPTimeout <= 0 {
		return fmt.Errorf("HTTP timeout must be > 0, got %s", c.HTTPTimeout)
	}

	if c.DNSTimeout <= 0 {
		return fmt.Errorf("DNS timeout must be > 0, got %s", c.DNSTimeout)
	}

	if c.MaxResponseSize <= 0 {
		return fmt.Errorf("max response size must be > 0, got %d", c.MaxResponseSize)
	}

	if c.BloomFilterFP <= 0 || c.BloomFilterFP >= 1 {
		return fmt.Errorf("bloom filter false positive rate must be between 0 and 1, got %f", c.BloomFilterFP)
	}

	return nil
}

func printHelp() {
	fmt.Print(`Subdomain Crawler v2.0 - High-performance subdomain discovery tool

USAGE:
    subdomain-crawler [OPTIONS]

INPUT/OUTPUT OPTIONS:
    -i, -input <file>          Input file with root domains (one per line, default: input.txt)
    -o, -output <file>         Output file for results (default: result.jsonl)
    -http-log <file>           HTTP request/response log (default: http.jsonl)
    -dns-log <file>            DNS query/response log (default: dns.jsonl)

CRAWLING OPTIONS:
    -max-depth <n>             Maximum subdomain depth (default: 3)
    -workers <n>               Number of concurrent workers (default: 32)
    -queue-size <n>            Task queue size (default: 10000)
    -expand-sld                Auto-expand SLD with common subdomains (default: true)

HTTP OPTIONS:
    -http-timeout <seconds>    HTTP request timeout (default: 10)
    -max-response-size <bytes> Maximum response size (default: 10485760)
    -user-agent <string>       HTTP User-Agent header (default: SubdomainCrawler/2.0)

DNS OPTIONS:
    -dns-timeout <seconds>     DNS query timeout (default: 5)

DEDUPLICATION OPTIONS:
    -bloom-size <n>            Expected number of unique domains (default: 1000000)
    -bloom-fp <rate>           False positive rate (default: 0.01)
    -bloom-file <file>         Bloom filter persistence file (default: bloom.filter)

UI OPTIONS:
    -dashboard                 Show interactive TUI dashboard (default: true)

OTHER OPTIONS:
    -h, -help                  Show this help message

EXAMPLES:
    # Basic usage (reads domains from input.txt)
    subdomain-crawler

    # Use custom input file and workers
    subdomain-crawler -i domains.txt -workers 64

    # Save results to custom location
    subdomain-crawler -i domains.txt -o results.jsonl

    # Adjust crawling depth
    subdomain-crawler -i domains.txt -max-depth 5

    # Disable dashboard for logging/automation
    subdomain-crawler -dashboard=false

For more information, visit: https://github.com/WangYihang/Subdomain-Crawler
`)
}
