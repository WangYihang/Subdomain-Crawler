package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/WangYihang/Subdomain-Crawler/pkg/common"
	"github.com/WangYihang/Subdomain-Crawler/pkg/model"
	"github.com/jessevdk/go-flags"
)

func main() {
	// Parse command-line flags
	_, err := flags.Parse(&model.Opts)
	if err != nil {
		os.Exit(1)
	}

	// Show version if requested
	if model.Opts.Version {
		fmt.Println(common.PV.String())
		os.Exit(0)
	}

	// Load root domains from input file
	rootDomains, err := common.LoadRootDomainsFromFile(model.Opts.InputFile)
	if err != nil {
		log.Fatalf("Failed to load root domains: %v", err)
	}

	if len(rootDomains) == 0 {
		log.Fatalf("No root domains loaded from %s", model.Opts.InputFile)
	}

	log.Printf("Loaded %d root domains from %s", len(rootDomains), model.Opts.InputFile)
	if model.Opts.Verbose {
		for _, domain := range rootDomains {
			log.Printf("  - %s", domain)
		}
	}

	// Create scheduler with configuration
	scheduler := common.NewScheduler(
		model.Opts.Concurrency,
		model.Opts.MaxDepth,
		model.Opts.Concurrency*10, // Job queue size
		time.Duration(model.Opts.Timeout)*time.Second,
		model.Opts.Output,
	)

	// Start the crawling process
	log.Printf("Starting subdomain crawler with %d workers, max depth %d, timeout %ds",
		model.Opts.Concurrency, model.Opts.MaxDepth, model.Opts.Timeout)

	if err := scheduler.Start(rootDomains); err != nil {
		log.Fatalf("Scheduler failed: %v", err)
	}

	log.Printf("Crawling completed. Processed: %d, Queued: %d",
		scheduler.TotalProcessed, scheduler.TotalQueued)
}
