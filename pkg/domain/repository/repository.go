package repository

import "github.com/WangYihang/Subdomain-Crawler/pkg/domain/entity"

// DomainFilter provides deduplication capabilities
type DomainFilter interface {
	// Contains checks if a domain has been seen before
	Contains(domain string) bool
	// Add adds a domain to the filter
	Add(domain string)
	// Save persists the filter state
	Save(filename string) error
	// Load restores the filter state
	Load(filename string) error
}

// ResultWriter writes crawl results
type ResultWriter interface {
	// Write writes a single result
	Write(result *entity.CrawlResult) error
	// Flush ensures all buffered data is written
	Flush() error
	// Close closes the writer
	Close() error
}

// LogWriter writes structured logs
type LogWriter interface {
	// WriteHTTPLog writes an HTTP request/response log
	WriteHTTPLog(data any) error
	// WriteDNSLog writes a DNS query/response log
	WriteDNSLog(data any) error
	// Close closes all log writers
	Close() error
}

// TaskQueue manages crawling tasks
type TaskQueue interface {
	// Enqueue adds a task to the queue
	Enqueue(task *entity.Task) bool
	// Dequeue removes and returns a task from the queue
	Dequeue() (*entity.Task, bool)
	// Len returns the current queue length
	Len() int
	// Close closes the queue
	Close()
}

// ResultQueue manages crawling results
type ResultQueue interface {
	// Send sends a result to the queue
	Send(result *entity.CrawlResult)
	// Receive receives a result from the queue
	Receive() (*entity.CrawlResult, bool)
	// Close closes the queue
	Close()
}
