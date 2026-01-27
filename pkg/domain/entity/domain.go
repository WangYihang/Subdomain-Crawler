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
	Domain        string    `json:"domain"`
	IPs           []string  `json:"ips"`
	Subdomains    []string  `json:"subdomains"`
	Status        string    `json:"status"`
	StatusCode    int       `json:"status_code"`
	Title         string    `json:"title"`
	ContentLength int       `json:"content_length"`
	Error         string    `json:"error,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
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
