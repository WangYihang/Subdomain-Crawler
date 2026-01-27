package main

import (
	"flag"
	"log"
	"os"

	"github.com/WangYihang/Subdomain-Crawler/pkg/config"
	"github.com/WangYihang/Subdomain-Crawler/pkg/crawler"
)

const version = "2.0.0"

func main() {
	// Define command-line flags
	inputFile := flag.String("i", "input.txt", "Input file containing root domains (one per line)")
	outputFile := flag.String("o", "output.jsonl", "Output file for results")
	timeout := flag.Int("t", 16, "HTTP request timeout in seconds")
	numWorkers := flag.Int("n", 32, "Number of concurrent workers")
	showVersion := flag.Bool("v", false, "Show version")

	flag.Parse()

	// Show version if requested
	if *showVersion {
		log.Printf("Subdomain Crawler v%s", version)
		os.Exit(0)
	}

	// Create configuration
	cfg := config.New(
		*inputFile,
		*outputFile,
		*timeout,
		*numWorkers,
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
