package main

import (
	"flag"
	"log"

	"github.com/WangYihang/Subdomain-Crawler/pkg/config"
	"github.com/WangYihang/Subdomain-Crawler/pkg/crawler"
)

func main() {
	// Define command-line flags (both optional for Unix pipe style)
	inputFile := flag.String("i", "", "Input file containing root domains (one per line, default: stdin)")
	outputFile := flag.String("o", "", "Output file for results (default: stdout)")

	flag.Parse()

	// Use stdin if no input file specified
	if *inputFile == "" {
		*inputFile = "-"
	}

	// Use stdout if no output file specified
	if *outputFile == "" {
		*outputFile = "-"
	}

	// Create configuration with defaults
	cfg := config.New(
		*inputFile,
		*outputFile,
		16,      // HTTP timeout in seconds
		32,      // Number of concurrent workers
		1048576, // Bloom filter size
		0.01,    // False positive rate
	)

	// Create crawler
	c, err := crawler.NewCrawler(cfg)
	if err != nil {
		log.Fatalf("Failed to create crawler: %v", err)
	}

	log.Printf("Starting subdomain crawler with %d workers", cfg.Concurrency.NumWorkers)

	// Start crawling
	if err := c.Start(); err != nil {
		log.Fatalf("Crawler failed: %v", err)
	}

	log.Printf("Crawling completed")
}
