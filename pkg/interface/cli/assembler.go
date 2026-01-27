package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/WangYihang/Subdomain-Crawler/pkg/application"
	"github.com/WangYihang/Subdomain-Crawler/pkg/infrastructure/dns"
	"github.com/WangYihang/Subdomain-Crawler/pkg/infrastructure/domainservice"
	"github.com/WangYihang/Subdomain-Crawler/pkg/infrastructure/http"
	"github.com/WangYihang/Subdomain-Crawler/pkg/infrastructure/storage"
)

// Assembler assembles all components for the application
type Assembler struct {
	config *Config
}

// NewAssembler creates a new assembler
func NewAssembler(config *Config) *Assembler {
	return &Assembler{config: config}
}

// AssembleUseCase assembles the crawl use case with all dependencies
func (a *Assembler) AssembleUseCase() (*application.CrawlUseCase, error) {
	// Load root domains
	rootDomains, err := a.loadRootDomains()
	if err != nil {
		return nil, fmt.Errorf("failed to load root domains: %w", err)
	}

	if len(rootDomains) == 0 {
		return nil, fmt.Errorf("no root domains provided")
	}

	// Create domain services
	validator := domainservice.NewValidator(rootDomains)
	calculator := domainservice.NewCalculator()
	extractor := domainservice.NewExtractor()

	// Create HTTP fetcher
	fetcher := http.NewFetcher(http.Config{
		Timeout:         a.config.HTTPTimeoutDuration,
		MaxResponseSize: a.config.MaxResponseSize,
		UserAgent:       a.config.UserAgent,
	})

	// Create DNS resolver
	resolver := dns.NewResolver(dns.Config{
		Servers: a.config.DNSServers,
		Timeout: a.config.DNSTimeoutDuration,
	})

	// Create repositories
	filter := storage.NewBloomFilter(storage.Config{
		Size:              a.config.RealBloomFilterSize,
		FalsePositiveRate: a.config.BloomFilterFP,
	})

	// Load existing bloom filter if exists
	if err := filter.Load(a.config.BloomFilterFile); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load bloom filter: %v\n", err)
	}

	taskQueue := storage.NewTaskQueue(a.config.QueueSize)
	resultQueue := storage.NewResultQueue(a.config.QueueSize)

	resultWriter, err := storage.NewResultWriter(a.config.OutputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create result writer: %w", err)
	}

	logWriter, err := storage.NewLogWriter(a.config.HTTPLogFile, a.config.DNSLogFile)
	if err != nil {
		resultWriter.Close()
		return nil, fmt.Errorf("failed to create log writer: %w", err)
	}

	// Create use case
	useCase := application.NewCrawlUseCase(
		application.Config{
			NumWorkers:      a.config.NumWorkers,
			MaxDepth:        a.config.MaxDepth,
			Protocols:       a.config.Protocols,
			RootDomains:     rootDomains,
			BloomFilterFile: a.config.BloomFilterFile,
		},
		validator,
		calculator,
		extractor,
		fetcher,
		resolver,
		filter,
		taskQueue,
		resultQueue,
		resultWriter,
		logWriter,
	)

	return useCase, nil
}

// loadRootDomains loads root domains from input
func (a *Assembler) loadRootDomains() ([]string, error) {
	var scanner *bufio.Scanner

	if a.config.InputFile == "-" {
		scanner = bufio.NewScanner(os.Stdin)
	} else {
		file, err := os.Open(a.config.InputFile)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		scanner = bufio.NewScanner(file)
	}

	var domains []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		domains = append(domains, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Expand SLDs if enabled
	if a.config.ExpandSLD {
		domains = a.expandSLDs(domains)
	}

	return domains, nil
}

// expandSLDs expands second-level domains with common subdomains
func (a *Assembler) expandSLDs(domains []string) []string {
	expander := domainservice.NewExpander(nil)

	var expanded []string
	for _, domain := range domains {
		if expander.IsSLD(domain) {
			// It's an SLD, expand it
			subdomains := expander.ExpandDomain(domain)
			expanded = append(expanded, subdomains...)
			fmt.Fprintf(os.Stderr, "Expanded %s to %d subdomains\n", domain, len(subdomains))
		} else {
			// It's already a subdomain, keep as-is
			expanded = append(expanded, domain)
		}
	}

	return expanded
}
