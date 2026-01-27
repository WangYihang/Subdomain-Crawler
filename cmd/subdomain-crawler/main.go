package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/WangYihang/Subdomain-Crawler/pkg/interface/cli"
	"github.com/WangYihang/Subdomain-Crawler/pkg/interface/presenter"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Parse command line flags
	config, err := cli.ParseFlags()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create assembler
	assembler := cli.NewAssembler(config)

	// Assemble use case with all dependencies
	useCase, err := assembler.AssembleUseCase()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Fprintln(os.Stderr, "\nReceived interrupt signal, shutting down gracefully...")
		cancel()
	}()

	// Setup dashboard if enabled
	if config.ShowDashboard {
		dashboard := presenter.NewDashboard()
		useCase.RegisterMetricsObserver(dashboard)

		// Run dashboard in TUI mode
		p := tea.NewProgram(dashboard, tea.WithAltScreen())

		// Run use case in background
		errChan := make(chan error, 1)
		go func() {
			if err := useCase.Execute(ctx); err != nil && err != context.Canceled {
				errChan <- err
			}
			// Signal dashboard to quit when done
			p.Send(tea.Quit())
		}()

		// Start TUI (blocks until quit)
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
			cancel() // Cancel the context to stop crawling
			os.Exit(1)
		}

		// Check for crawling errors
		select {
		case err := <-errChan:
			fmt.Fprintf(os.Stderr, "Crawling error: %v\n", err)
			os.Exit(1)
		default:
			// No errors
		}
	} else {
		// Non-dashboard mode: simple console output
		fmt.Fprintln(os.Stderr, "Starting subdomain crawler...")
		if err := useCase.Execute(ctx); err != nil && err != context.Canceled {
			fmt.Fprintf(os.Stderr, "Crawling error: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "Crawling completed successfully")
	}
}
