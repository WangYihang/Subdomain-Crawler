package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/jessevdk/go-flags"
)

// Config holds all application configuration
type Config struct {
	// Input/Output
	InputFile   string `short:"i" long:"input" description:"Input file with root domains (one per line)" default:"-"`
	OutputFile  string `short:"o" long:"output" description:"Output file for results" default:"result.jsonl"`
	HTTPLogFile string `long:"http-log" description:"HTTP request/response log file" default:"http.jsonl"`
	DNSLogFile  string `long:"dns-log" description:"DNS query/response log file" default:"dns.jsonl"`

	// Crawling
	MaxDepth   int  `long:"max-depth" description:"Maximum subdomain depth to crawl" default:"3"`
	NumWorkers int  `long:"workers" description:"Number of concurrent workers" default:"32"`
	QueueSize  int  `long:"queue-size" description:"Size of task queue" default:"10000"`
	ExpandSLD  bool `long:"expand-sld" description:"Automatically expand SLD with common subdomains (www, api, mail, etc.)"`

	Protocols []string

	// HTTP
	HTTPTimeout     int    `long:"http-timeout" description:"HTTP request timeout in seconds" default:"10"`
	MaxResponseSize int64  `long:"max-response-size" description:"Maximum HTTP response size in bytes" default:"10485760"`
	UserAgent       string `long:"user-agent" description:"HTTP User-Agent header" default:"SubdomainCrawler/2.0"`

	// Real HTTP timeout duration (not parsed from flags directly)
	HTTPTimeoutDuration time.Duration

	// DNS
	DNSTimeout int `long:"dns-timeout" description:"DNS query timeout in seconds" default:"5"`

	// Real DNS timeout duration
	DNSTimeoutDuration time.Duration
	DNSServers         []string

	// Dedup
	BloomFilterSize uint64  `long:"bloom-size" description:"Bloom filter size (number of expected elements)" default:"1000000"`
	BloomFilterFP   float64 `long:"bloom-fp" description:"Bloom filter false positive rate" default:"0.01"`
	BloomFilterFile string  `long:"bloom-file" description:"Bloom filter persistence file" default:"bloom.filter"`

	// Real bloom filter size (uint)
	RealBloomFilterSize uint

	// UI
	ShowDashboard bool `long:"dashboard" description:"Show interactive TUI dashboard"`
}

// ParseFlags parses command line flags
func ParseFlags() (*Config, error) {
	cfg := &Config{
		ExpandSLD:     true, // Default value that cannot be easily set via struct tag for boolean if we want it true by default
		ShowDashboard: true,
	}

	parser := flags.NewParser(cfg, flags.Default)
	parser.Usage = "[OPTIONS]"

	if _, err := parser.Parse(); err != nil {
		if flags.WroteHelp(err) {
			// Help has been printed by the library, exit cleanly
			os.Exit(0)
		}
		return nil, err
	}

	// Convert timeouts
	cfg.HTTPTimeoutDuration = time.Duration(cfg.HTTPTimeout) * time.Second
	cfg.DNSTimeoutDuration = time.Duration(cfg.DNSTimeout) * time.Second

	// Set bloom filter size
	cfg.RealBloomFilterSize = uint(cfg.BloomFilterSize)

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

	if c.HTTPTimeoutDuration <= 0 {
		return fmt.Errorf("HTTP timeout must be > 0, got %s", c.HTTPTimeoutDuration)
	}

	if c.DNSTimeoutDuration <= 0 {
		return fmt.Errorf("DNS timeout must be > 0, got %s", c.DNSTimeoutDuration)
	}

	if c.MaxResponseSize <= 0 {
		return fmt.Errorf("max response size must be > 0, got %d", c.MaxResponseSize)
	}

	if c.BloomFilterFP <= 0 || c.BloomFilterFP >= 1 {
		return fmt.Errorf("bloom filter false positive rate must be between 0 and 1, got %f", c.BloomFilterFP)
	}

	return nil
}
