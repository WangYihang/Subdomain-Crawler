package dedup

import (
	"os"
	"sync"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
)

// Filter implements bloom filter
type Filter struct {
	filter *bloom.BloomFilter
	mu     sync.Mutex
}

// NewFilter creates filter
func NewFilter(n uint, fp float64) *Filter {
	return &Filter{filter: bloom.NewWithEstimates(n, fp)}
}

// TestAndAdd tests and adds
func (f *Filter) TestAndAdd(data []byte) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.filter.TestAndAdd(data)
}

// Test tests membership
func (f *Filter) Test(data []byte) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.filter.Test(data)
}

// Add adds data
func (f *Filter) Add(data []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.filter.Add(data)
}

// SaveToFile saves filter
func (f *Filter) SaveToFile(path string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := f.filter.MarshalBinary()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadFromFile loads filter
func (f *Filter) LoadFromFile(path string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return f.filter.UnmarshalBinary(data)
}

// PersistenceManager manages periodic saves
type PersistenceManager struct {
	filter   *Filter
	ticker   *time.Ticker
	stopChan chan struct{}
}

// NewPersistenceManager creates manager
func NewPersistenceManager(filter *Filter, interval time.Duration) *PersistenceManager {
	return &PersistenceManager{
		filter:   filter,
		ticker:   time.NewTicker(interval),
		stopChan: make(chan struct{}),
	}
}

// StartPeriodicSave starts saving
func (pm *PersistenceManager) StartPeriodicSave(filePath string) {
	go func() {
		for {
			select {
			case <-pm.ticker.C:
				_ = pm.filter.SaveToFile(filePath)
			case <-pm.stopChan:
				return
			}
		}
	}()
}

// Stop stops saving
func (pm *PersistenceManager) Stop() {
	pm.ticker.Stop()
	close(pm.stopChan)
}
