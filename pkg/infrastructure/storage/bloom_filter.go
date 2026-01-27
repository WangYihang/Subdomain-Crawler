package storage

import (
	"os"

	"github.com/WangYihang/Subdomain-Crawler/pkg/domain/repository"
	"github.com/bits-and-blooms/bloom/v3"
)

// BloomFilter implements repository.DomainFilter using Bloom filter
type BloomFilter struct {
	filter *bloom.BloomFilter
	size   uint
	fpRate float64
}

// Config holds Bloom filter configuration
type Config struct {
	Size              uint
	FalsePositiveRate float64
}

// NewBloomFilter creates a new Bloom filter
func NewBloomFilter(config Config) repository.DomainFilter {
	return &BloomFilter{
		filter: bloom.NewWithEstimates(config.Size, config.FalsePositiveRate),
		size:   config.Size,
		fpRate: config.FalsePositiveRate,
	}
}

// Contains checks if a domain has been seen before
func (bf *BloomFilter) Contains(domain string) bool {
	return bf.filter.Test([]byte(domain))
}

// Add adds a domain to the filter
func (bf *BloomFilter) Add(domain string) {
	bf.filter.Add([]byte(domain))
}

// Save persists the filter state
func (bf *BloomFilter) Save(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = bf.filter.WriteTo(file)
	return err
}

// Load restores the filter state
func (bf *BloomFilter) Load(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, that's OK
		}
		return err
	}
	defer file.Close()

	bf.filter = bloom.NewWithEstimates(bf.size, bf.fpRate)
	_, err = bf.filter.ReadFrom(file)
	return err
}
