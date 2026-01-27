package common

import (
	"strconv"
	"sync"
	"testing"
)

func TestGlobalBloomFilter_TestAndAdd(t *testing.T) {
	InitGlobalBloomFilter(1000, 0.01)

	// Test single addition
	item := []byte("test")
	if BloomFilter.TestAndAdd(item) {
		t.Errorf("Expected first add to return false (not present), got true")
	}
	if !BloomFilter.TestAndAdd(item) {
		t.Errorf("Expected second add to return true (present), got false")
	}

	// Test concurrency
	var wg sync.WaitGroup
	count := 100
	wg.Add(count)
	for i := 0; i < count; i++ {
		go func(val int) {
			defer wg.Done()
			BloomFilter.TestAndAdd([]byte(strconv.Itoa(val)))
		}(i)
	}
	wg.Wait()
}
