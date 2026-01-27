package config

import "time"

// Config holds all configuration
type Config struct {
	Input       InputConfig
	Output      OutputConfig
	HTTP        HTTPConfig
	Concurrency ConcurrencyConfig
	Dedup       DedupConfig
}

type InputConfig struct {
	File string
}

type OutputConfig struct {
	ResultsFile  string
	HttpLogFile  string
	DnsLogFile   string
	FindingsFile string
	StatusFile   string
}

type HTTPConfig struct {
	Timeout time.Duration
}

type ConcurrencyConfig struct {
	NumWorkers int
	QueueSize  int
}

type DedupConfig struct {
	BloomFilterSize          uint
	BloomFilterFalsePositive float64
	BloomFilterFile          string
}

// New creates config
func New(inputFile, outputFile string, timeout, numWorkers int, bfSize uint, bfFP float64) *Config {
	return &Config{
		Input: InputConfig{File: inputFile},
		Output: OutputConfig{
			ResultsFile:  outputFile,
			HttpLogFile:  "http.jsonl",
			DnsLogFile:   "dns.jsonl",
			FindingsFile: "findings.jsonl",
			StatusFile:   "status.jsonl",
		},
		HTTP: HTTPConfig{Timeout: time.Duration(timeout) * time.Second},
		Concurrency: ConcurrencyConfig{
			NumWorkers: numWorkers,
			QueueSize:  numWorkers * 10,
		},
		Dedup: DedupConfig{
			BloomFilterSize:          bfSize,
			BloomFilterFalsePositive: bfFP,
			BloomFilterFile:          "bloom.filter",
		},
	}
}
