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
	ips := result.IPs
	if ips == nil {
		ips = []string{}
	}

	out := map[string]interface{}{
		"domain":         result.Domain,
		"root":           result.Root,
		"subdomains":     subdomains,
		"ips":            ips,
		"title":          result.Title,
		"content_length": result.ContentLength,
		"error":          result.Error,
	}

	return w.encoder.Encode(out)
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

// JsonlWriter writes one JSON object per line (for http/dns logs).
type JsonlWriter struct {
	file *os.File
	enc  *json.Encoder
	mu   sync.Mutex
}

// NewJsonlWriter creates a writer for the given path.
func NewJsonlWriter(path string) (*JsonlWriter, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &JsonlWriter{file: f, enc: json.NewEncoder(f)}, nil
}

// Log encodes v as JSON and writes one line.
func (w *JsonlWriter) Log(v interface{}) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.enc.Encode(v)
}

// Close closes the underlying file.
func (w *JsonlWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		err := w.file.Close()
		w.file = nil
		return err
	}
	return nil
}
