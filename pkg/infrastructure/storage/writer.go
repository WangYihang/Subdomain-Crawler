package storage

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/WangYihang/Subdomain-Crawler/pkg/domain/entity"
	"github.com/WangYihang/Subdomain-Crawler/pkg/domain/repository"
)

// ResultWriter implements repository.ResultWriter
type ResultWriter struct {
	file    *os.File
	encoder *json.Encoder
	mu      sync.Mutex
}

// NewResultWriter creates a new result writer
func NewResultWriter(filename string) (repository.ResultWriter, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	return &ResultWriter{
		file:    file,
		encoder: json.NewEncoder(file),
	}, nil
}

// Write writes a single result
func (w *ResultWriter) Write(result *entity.CrawlResult) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.encoder.Encode(result)
}

// Flush ensures all buffered data is written
func (w *ResultWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.file.Sync()
}

// Close closes the writer
func (w *ResultWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.file.Close()
}

// LogWriter implements repository.LogWriter
type LogWriter struct {
	httpFile *os.File
	dnsFile  *os.File
	httpEnc  *json.Encoder
	dnsEnc   *json.Encoder
	mu       sync.Mutex
}

// NewLogWriter creates a new log writer
func NewLogWriter(httpLogFile, dnsLogFile string) (repository.LogWriter, error) {
	httpFile, err := os.Create(httpLogFile)
	if err != nil {
		return nil, err
	}

	dnsFile, err := os.Create(dnsLogFile)
	if err != nil {
		httpFile.Close()
		return nil, err
	}

	return &LogWriter{
		httpFile: httpFile,
		dnsFile:  dnsFile,
		httpEnc:  json.NewEncoder(httpFile),
		dnsEnc:   json.NewEncoder(dnsFile),
	}, nil
}

// WriteHTTPLog writes an HTTP request/response log
func (w *LogWriter) WriteHTTPLog(data any) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.httpEnc.Encode(data)
}

// WriteDNSLog writes a DNS query/response log
func (w *LogWriter) WriteDNSLog(data any) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.dnsEnc.Encode(data)
}

// Close closes all log writers
func (w *LogWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	err1 := w.httpFile.Close()
	err2 := w.dnsFile.Close()

	if err1 != nil {
		return err1
	}
	return err2
}
