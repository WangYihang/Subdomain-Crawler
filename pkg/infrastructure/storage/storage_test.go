package storage

import (
	"os"
	"testing"

	"github.com/WangYihang/Subdomain-Crawler/pkg/domain/entity"
)

func TestBloomFilter_Basic(t *testing.T) {
	filter := NewBloomFilter(Config{
		Size:              1000,
		FalsePositiveRate: 0.01,
	})

	// Test Add and Contains
	testDomain := "example.com"

	if filter.Contains(testDomain) {
		t.Errorf("Filter should not contain %s initially", testDomain)
	}

	filter.Add(testDomain)

	if !filter.Contains(testDomain) {
		t.Errorf("Filter should contain %s after Add", testDomain)
	}
}

func TestBloomFilter_SaveLoad(t *testing.T) {
	tmpFile := "/tmp/test_bloom.filter"
	defer os.Remove(tmpFile)

	// Create and populate filter
	filter1 := NewBloomFilter(Config{
		Size:              1000,
		FalsePositiveRate: 0.01,
	})

	testDomains := []string{"example.com", "test.com", "demo.com"}
	for _, d := range testDomains {
		filter1.Add(d)
	}

	// Save to file
	if err := filter1.Save(tmpFile); err != nil {
		t.Fatalf("Failed to save filter: %v", err)
	}

	// Load into new filter
	filter2 := NewBloomFilter(Config{
		Size:              1000,
		FalsePositiveRate: 0.01,
	})

	if err := filter2.Load(tmpFile); err != nil {
		t.Fatalf("Failed to load filter: %v", err)
	}

	// Verify all domains are present
	for _, d := range testDomains {
		if !filter2.Contains(d) {
			t.Errorf("Loaded filter should contain %s", d)
		}
	}
}

func TestTaskQueue_Basic(t *testing.T) {
	queue := NewTaskQueue(10)

	// Check initial state
	if queue.Len() != 0 {
		t.Errorf("New queue should be empty, got length %d", queue.Len())
	}

	// Create a dummy task
	task := &entity.Task{
		Domain: entity.Domain{
			Name:  "example.com",
			Root:  "example.com",
			Depth: 0,
		},
	}

	// Enqueue
	if !queue.Enqueue(task) {
		t.Error("Enqueue should succeed")
	}

	if queue.Len() != 1 {
		t.Errorf("Queue length should be 1, got %d", queue.Len())
	}

	// Dequeue
	dequeued, ok := queue.Dequeue()
	if !ok {
		t.Error("Dequeue should succeed")
	}

	if dequeued == nil {
		t.Error("Dequeued task should not be nil")
	}

	if queue.Len() != 0 {
		t.Errorf("Queue should be empty after dequeue, got length %d", queue.Len())
	}
}

func TestTaskQueue_Close(t *testing.T) {
	queue := NewTaskQueue(10)

	task := &entity.Task{
		Domain: entity.Domain{
			Name:  "example.com",
			Root:  "example.com",
			Depth: 0,
		},
	}
	queue.Enqueue(task)

	queue.Close()

	// Should not be able to enqueue after close
	if queue.Enqueue(task) {
		t.Error("Enqueue should fail after close")
	}

	// Should still be able to dequeue existing items
	_, ok := queue.Dequeue()
	if !ok {
		t.Error("Should be able to dequeue existing items after close")
	}
}

func TestResultQueue_Basic(t *testing.T) {
	queue := NewResultQueue(10)

	result := &entity.CrawlResult{
		Domain: "example.com",
	}

	// Send
	queue.Send(result)

	// Receive
	received, ok := queue.Receive()
	if !ok {
		t.Error("Receive should succeed")
	}

	if received == nil {
		t.Error("Received result should not be nil")
	}
}

func TestResultQueue_Close(t *testing.T) {
	queue := NewResultQueue(10)

	result := &entity.CrawlResult{
		Domain: "example.com",
	}
	queue.Send(result)

	queue.Close()

	// Should still be able to receive existing items
	_, ok := queue.Receive()
	if !ok {
		t.Error("Should be able to receive existing items after close")
	}

	// Receiving from closed empty queue should return false
	_, ok = queue.Receive()
	if ok {
		t.Error("Receiving from closed empty queue should return false")
	}
}
