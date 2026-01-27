package main

import (
	"flag"
	"io"
	"log"
	"os"

	"github.com/WangYihang/Subdomain-Crawler/pkg/config"
	"github.com/WangYihang/Subdomain-Crawler/pkg/crawler"
)

func main() {
	inputFile := flag.String("i", "", "Input file (one domain per line, default: stdin)")
	outputFile := flag.String("o", "", "Output file for results (default: result.jsonl)")
	flag.Parse()

	if *inputFile == "" {
		*inputFile = "-"
	}
	if *outputFile == "" {
		*outputFile = "result.jsonl"
	}

	cfg := config.New(
		*inputFile,
		*outputFile,
		16, 32, 1048576, 0.01,
	)

	c, err := crawler.NewCrawler(cfg)
	if err != nil {
		log.Fatalf("Failed to create crawler: %v", err)
	}

	// Terminal only shows progress bar; suppress routine log lines
	log.SetOutput(io.Discard)
	defer func() { log.SetOutput(os.Stderr) }()

	if err := c.Start(); err != nil {
		log.SetOutput(os.Stderr)
		log.Fatalf("Crawler failed: %v", err)
	}
}
