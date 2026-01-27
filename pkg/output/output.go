package output

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/WangYihang/Subdomain-Crawler/pkg/queue"
)

// Writer writes results
type Writer struct {
	resultsFile string
	file        *os.File
	mu          sync.Mutex
	encoder     *json.Encoder
}

// NewWriter creates writer (use "-" for stdout)
func NewWriter(filePath string) (*Writer, error) {
	var file *os.File
	var err error

	// Use stdout if filePath is "-"
	if filePath == "-" {
		file = os.Stdout
	} else {
		file, err = os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, err
		}
	}

	return &Writer{
		resultsFile: filePath,
		file:        file,
		encoder:     json.NewEncoder(file),
	}, nil
}

// WriteResult writes result
func (w *Writer) WriteResult(result queue.Result) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	subdomains := result.Subdomains
	if subdomains == nil {
		subdomains = []string{}
	}

	output := map[string]interface{}{
		"domain":     result.Domain,
		"root":       result.Root,
		"subdomains": subdomains,
		"error":      result.Error,
	}

	return w.encoder.Encode(output)
}

// Close closes writer (does not close stdout)
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Don't close stdout or stdin
	if w.resultsFile == "-" {
		return nil
	}

	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// Flusher flushes results
type Flusher struct {
	writer *Writer
	queue  *queue.ResultQueue
	done   chan struct{}
	wg     sync.WaitGroup
}

// NewFlusher creates flusher
func NewFlusher(writer *Writer, queue *queue.ResultQueue) *Flusher {
	return &Flusher{
		writer: writer,
		queue:  queue,
		done:   make(chan struct{}),
	}
}

// Start starts flusher
func (f *Flusher) Start() {
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		f.flush()
	}()
}

// flush flushes results
func (f *Flusher) flush() {
	for {
		select {
		case <-f.done:
			for {
				result, ok := f.queue.Receive()
				if !ok {
					return
				}
				_ = f.writer.WriteResult(result)
			}
		default:
			result, ok := f.queue.Receive()
			if !ok {
				return
			}
			_ = f.writer.WriteResult(result)
		}
	}
}

// Stop stops flusher
func (f *Flusher) Stop() {
	close(f.done)
	f.wg.Wait()
}
