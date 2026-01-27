package common

import (
	"sync"

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

// InitGlobalBloomFilter initializes the global bloom filter
func InitGlobalBloomFilter(n uint, fp float64) {
	onceBloom.Do(func() {
		BloomFilter = &GlobalBloomFilter{
			filter: bloom.NewWithEstimates(n, fp),
		}
	})
}

// TestAndAdd checks if the data is in the bloom filter, and adds it if it's not.
// Returns true if the data was likely already in the set.
func (bf *GlobalBloomFilter) TestAndAdd(data []byte) bool {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	return bf.filter.TestAndAdd(data)
}
