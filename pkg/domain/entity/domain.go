package entity

import "time"

// Domain represents a domain name entity
type Domain struct {
	Name  string
	Root  string
	Depth int
}

// Task represents a crawling task
type Task struct {
	Domain    Domain
	Protocols []string
	CreatedAt time.Time
}

// CrawlResult represents the result of crawling a domain
type CrawlResult struct {
	Domain        string
	Root          string
	Protocol      string
	StatusCode    int
	Title         string
	ContentLength int
	Subdomains    []string
	IPs           []string
	Error         string
	CrawledAt     time.Time
}

// DNSRecord represents a DNS resolution record
type DNSRecord struct {
	Domain     string
	RecordType string
	Value      string
	TTL        uint32
	ResolvedAt time.Time
}

// Metrics represents crawling metrics
type Metrics struct {
	QueueLength      int
	ActiveWorkers    int
	TotalWorkers     int
	HTTPRequests     int64
	DNSRequests      int64
	UniqueSubdomains int64
	TasksProcessed   int64
	TasksEnqueued    int64
	ErrorCount       int64
	SuccessCount     int64
	StartTime        time.Time
	LastUpdateTime   time.Time
	ActiveDomains    []string
}
