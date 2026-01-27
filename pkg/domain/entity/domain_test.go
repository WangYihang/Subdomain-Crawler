package entity

import (
	"testing"
	"time"
)

func TestDomain_Creation(t *testing.T) {
	domain := Domain{
		Name:  "example.com",
		Root:  "example.com",
		Depth: 0,
	}

	if domain.Name != "example.com" {
		t.Errorf("Domain.Name = %s, want example.com", domain.Name)
	}

	if domain.Root != "example.com" {
		t.Errorf("Domain.Root = %s, want example.com", domain.Root)
	}

	if domain.Depth != 0 {
		t.Errorf("Domain.Depth = %d, want 0", domain.Depth)
	}
}

func TestTask_Creation(t *testing.T) {
	now := time.Now()

	task := Task{
		Domain: Domain{
			Name:  "www.example.com",
			Root:  "example.com",
			Depth: 1,
		},
		Protocols: []string{"http", "https"},
		CreatedAt: now,
	}

	if task.Domain.Name != "www.example.com" {
		t.Errorf("Task.Domain.Name = %s, want www.example.com", task.Domain.Name)
	}

	if len(task.Protocols) != 2 {
		t.Errorf("len(Task.Protocols) = %d, want 2", len(task.Protocols))
	}

	if task.CreatedAt != now {
		t.Errorf("Task.CreatedAt mismatch")
	}
}

func TestCrawlResult_Creation(t *testing.T) {
	result := CrawlResult{
		Domain:        "www.example.com",
		Status:        "200 OK",
		StatusCode:    200,
		Title:         "Example Domain",
		ContentLength: 1234,
		Subdomains:    []string{"api.example.com", "blog.example.com"},
		IPs:           []string{"93.184.216.34"},
		Error:         "",
		Timestamp:     time.Now(),
	}

	if result.StatusCode != 200 {
		t.Errorf("CrawlResult.StatusCode = %d, want 200", result.StatusCode)
	}

	if len(result.Subdomains) != 2 {
		t.Errorf("len(CrawlResult.Subdomains) = %d, want 2", len(result.Subdomains))
	}

	if len(result.IPs) != 1 {
		t.Errorf("len(CrawlResult.IPs) = %d, want 1", len(result.IPs))
	}
}

func TestMetrics_Creation(t *testing.T) {
	metrics := Metrics{
		QueueLength:      100,
		ActiveWorkers:    32,
		TotalWorkers:     32,
		HTTPRequests:     1000,
		DNSRequests:      950,
		UniqueSubdomains: 500,
		TasksProcessed:   950,
		TasksEnqueued:    1000,
		ErrorCount:       50,
		SuccessCount:     950,
		StartTime:        time.Now(),
		LastUpdateTime:   time.Now(),
		ActiveDomains:    []string{"www.example.com", "api.example.com"},
	}

	if metrics.QueueLength != 100 {
		t.Errorf("Metrics.QueueLength = %d, want 100", metrics.QueueLength)
	}

	if metrics.TotalWorkers != 32 {
		t.Errorf("Metrics.TotalWorkers = %d, want 32", metrics.TotalWorkers)
	}

	if len(metrics.ActiveDomains) != 2 {
		t.Errorf("len(Metrics.ActiveDomains) = %d, want 2", len(metrics.ActiveDomains))
	}

	// Test metrics calculations
	if metrics.HTTPRequests != metrics.SuccessCount+metrics.ErrorCount {
		t.Error("HTTPRequests should equal SuccessCount + ErrorCount")
	}
}

func TestDNSRecord_Creation(t *testing.T) {
	record := DNSRecord{
		Domain:     "example.com",
		RecordType: "A",
		Value:      "93.184.216.34",
		TTL:        3600,
		ResolvedAt: time.Now(),
	}

	if record.RecordType != "A" {
		t.Errorf("DNSRecord.RecordType = %s, want A", record.RecordType)
	}

	if record.TTL != 3600 {
		t.Errorf("DNSRecord.TTL = %d, want 3600", record.TTL)
	}
}
