package common

import (
	"os"
	"sync"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
)

type GlobalBloomFilter struct {
	filter *bloom.BloomFilter
	mu     sync.Mutex
}

var (
	// BloomFilter is the global bloom filter instance
	BloomFilter *GlobalBloomFilter
	onceBloom   sync.Once
)

const (
	// DefaultBloomFilterSessionFile is the default path to save/load bloom filter state
	DefaultBloomFilterSessionFile = "session.bloom"
)

// InitGlobalBloomFilter initializes the global bloom filter
func InitGlobalBloomFilter(n uint, fp float64) {
	onceBloom.Do(func() {
		BloomFilter = &GlobalBloomFilter{
			filter: bloom.NewWithEstimates(n, fp),
		}
		// Try to load from existing session file
		_ = BloomFilter.LoadFromFile(DefaultBloomFilterSessionFile)
	})
}

// TestAndAdd checks if the data is in the bloom filter, and adds it if it's not.
// Returns true if the data was likely already in the set.
func (bf *GlobalBloomFilter) TestAndAdd(data []byte) bool {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	return bf.filter.TestAndAdd(data)
}

// SaveToFile saves the bloom filter state to a file
func (bf *GlobalBloomFilter) SaveToFile(path string) error {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	data, err := bf.filter.MarshalBinary()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadFromFile loads the bloom filter state from a file
func (bf *GlobalBloomFilter) LoadFromFile(path string) error {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	bf.filter = bloom.NewWithEstimates(1000, 0.01) // Create a temporary filter
	return bf.filter.UnmarshalBinary(data)
}

// StartPeriodicSave starts a goroutine that saves the bloom filter every minute
func (bf *GlobalBloomFilter) StartPeriodicSave(path string, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			if err := bf.SaveToFile(path); err != nil {
				// Log error but don't stop the periodic save
				// In a real application, you might want proper logging here
			}
		}
	}()
}
