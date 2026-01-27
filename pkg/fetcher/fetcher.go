package fetcher

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/WangYihang/Subdomain-Crawler/pkg/extract"
	"github.com/WangYihang/Subdomain-Crawler/pkg/httpclient"
	"github.com/WangYihang/Subdomain-Crawler/pkg/output"
)

// HTTPRequest represents a complete HTTP request for logging
type HTTPRequest struct {
	Method    string              `json:"method"`
	URL       string              `json:"url"`
	Host      string              `json:"host"`
	Headers   map[string][]string `json:"headers"`
	Body      string              `json:"body,omitempty"`
	RequestAt int64               `json:"request_at"`
	Timestamp string              `json:"timestamp"`
}

// TLSInfo represents TLS connection information
type TLSInfo struct {
	Version     string `json:"version"`
	CipherSuite string `json:"cipher_suite"`
	ServerName  string `json:"server_name"`
	Negotiated  bool   `json:"negotiated"`
}

// HTTPResponse represents a complete HTTP response for logging
type HTTPResponse struct {
	Status         string              `json:"status"`
	StatusCode     int                 `json:"status_code"`
	Proto          string              `json:"proto"`
	Headers        map[string][]string `json:"headers"`
	ContentLength  int64               `json:"content_length"`
	BodySize       int                 `json:"body_size"`
	Body           string              `json:"body,omitempty"`
	BodyTruncated  bool                `json:"body_truncated"`
	Title          string              `json:"title,omitempty"`
	TLS            bool                `json:"tls"`
	TLSInfo        *TLSInfo            `json:"tls_info,omitempty"`
	Error          string              `json:"error,omitempty"`
	ResponseTimeMs int64               `json:"response_time_ms"`
	ResponseAt     int64               `json:"response_at"`
	Timestamp      string              `json:"timestamp"`
}

// HTTPLog represents a complete HTTP transaction log entry
type HTTPLog struct {
	Request  HTTPRequest  `json:"request"`
	Response HTTPResponse `json:"response"`
}

// Result represents fetch result
type Result struct {
	Domain             string
	Root               string
	Subdomains         []string
	Title              string
	ContentLength      int64 // from HTTP header, -1 when unknown
	ResponseStatusCode int
	ResponseTime       int64
	Error              string
	Timestamp          int64
}

// Fetcher handles fetching
type Fetcher struct {
	client          *httpclient.Client
	extractor       *extract.DomainExtractor
	filter          *extract.Filter
	deduplicator    *extract.Deduplicator
	sanitizer       *extract.Sanitizer
	maxResponseSize int64
	httpLog         *output.JsonlWriter
}

// Config holds fetcher config
type Config struct {
	Client          *httpclient.Client
	Filter          *extract.Filter
	MaxResponseSize int64
	HttpLog         *output.JsonlWriter
}

// NewFetcher creates fetcher
func NewFetcher(config *Config) *Fetcher {
	if config.MaxResponseSize == 0 {
		config.MaxResponseSize = 10 * 1024 * 1024
	}

	return &Fetcher{
		client:          config.Client,
		extractor:       extract.NewDomainExtractor(),
		filter:          config.Filter,
		deduplicator:    extract.NewDeduplicator(),
		sanitizer:       extract.NewSanitizer(),
		maxResponseSize: config.MaxResponseSize,
		httpLog:         config.HttpLog,
	}
}

// Fetch fetches domain
func (f *Fetcher) Fetch(domain, root string, protocols []string) *Result {
	result := &Result{
		Domain:    domain,
		Root:      root,
		Timestamp: time.Now().UnixMilli(),
	}

	for _, protocol := range protocols {
		url := fmt.Sprintf("%s://%s/", protocol, domain)
		if err := f.fetchURL(url, result); err == nil {
			return result
		}
	}

	if result.Error == "" {
		result.Error = "failed to fetch"
	}

	return result
}

// headersToMap converts http.Header to map[string][]string for JSON.
func headersToMap(h http.Header) map[string][]string {
	if h == nil {
		return nil
	}
	m := make(map[string][]string, len(h))
	for k, v := range h {
		m[k] = v
	}
	return m
}

// getTLSVersion returns TLS version as string.
func getTLSVersion(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown(0x%04x)", version)
	}
}

// getTLSCipherSuite returns cipher suite as string.
func getTLSCipherSuite(suite uint16) string {
	// Common cipher suites
	cipherSuites := map[uint16]string{
		tls.TLS_RSA_WITH_RC4_128_SHA:                      "TLS_RSA_WITH_RC4_128_SHA",
		tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA:                 "TLS_RSA_WITH_3DES_EDE_CBC_SHA",
		tls.TLS_RSA_WITH_AES_128_CBC_SHA:                  "TLS_RSA_WITH_AES_128_CBC_SHA",
		tls.TLS_RSA_WITH_AES_256_CBC_SHA:                  "TLS_RSA_WITH_AES_256_CBC_SHA",
		tls.TLS_RSA_WITH_AES_128_CBC_SHA256:               "TLS_RSA_WITH_AES_128_CBC_SHA256",
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256:               "TLS_RSA_WITH_AES_128_GCM_SHA256",
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384:               "TLS_RSA_WITH_AES_256_GCM_SHA384",
		tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA:              "TLS_ECDHE_ECDSA_WITH_RC4_128_SHA",
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA:          "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA",
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA:          "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
		tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA:                "TLS_ECDHE_RSA_WITH_RC4_128_SHA",
		tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA:           "TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA",
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA:            "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA:            "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256:       "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256",
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256:         "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256",
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256:         "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256:       "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384:         "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384:       "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256:   "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256",
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256: "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256",
		tls.TLS_AES_128_GCM_SHA256:                        "TLS_AES_128_GCM_SHA256",
		tls.TLS_AES_256_GCM_SHA384:                        "TLS_AES_256_GCM_SHA384",
		tls.TLS_CHACHA20_POLY1305_SHA256:                  "TLS_CHACHA20_POLY1305_SHA256",
	}
	if name, ok := cipherSuites[suite]; ok {
		return name
	}
	return fmt.Sprintf("Unknown(0x%04x)", suite)
}

func (f *Fetcher) writeHTTPLog(req *http.Request, url string, resp *http.Response, body []byte, result *Result, startTime time.Time) {
	if f.httpLog == nil {
		return
	}

	requestAt := startTime
	responseAt := time.Now()
	durationMs := result.ResponseTime

	// Build HTTP request log
	httpReq := HTTPRequest{
		Method:    "GET",
		URL:       url,
		RequestAt: requestAt.UnixMilli(),
		Timestamp: requestAt.Format(time.RFC3339Nano),
	}
	if req != nil {
		httpReq.Method = req.Method
		httpReq.URL = req.URL.String()
		httpReq.Host = req.Host
		httpReq.Headers = headersToMap(req.Header)
		// Add request body if present (for POST, PUT, etc.)
		if req.Body != nil && req.ContentLength > 0 {
			// Note: In current implementation we only do GET requests,
			// but this is here for future extensibility
			httpReq.Body = "[body not captured for GET requests]"
		}
	} else {
		httpReq.Headers = make(map[string][]string)
	}

	// Build HTTP response log
	httpResp := HTTPResponse{
		Error:          result.Error,
		ResponseTimeMs: durationMs,
		ResponseAt:     responseAt.UnixMilli(),
		Timestamp:      responseAt.Format(time.RFC3339Nano),
	}

	if resp != nil {
		httpResp.Status = resp.Status
		httpResp.StatusCode = resp.StatusCode
		httpResp.Headers = headersToMap(resp.Header)
		httpResp.ContentLength = resp.ContentLength
		httpResp.Title = result.Title
		httpResp.Proto = resp.Proto
		httpResp.TLS = resp.TLS != nil

		// Add TLS information if present
		if resp.TLS != nil {
			httpResp.TLSInfo = &TLSInfo{
				Version:     getTLSVersion(resp.TLS.Version),
				CipherSuite: getTLSCipherSuite(resp.TLS.CipherSuite),
				ServerName:  resp.TLS.ServerName,
				Negotiated:  resp.TLS.HandshakeComplete,
			}
		}
	}

	// Add body information
	if body != nil {
		httpResp.BodySize = len(body)
		httpResp.BodyTruncated = int64(len(body)) >= f.maxResponseSize

		if len(body) > 0 {
			// Store complete body without any size limits
			httpResp.Body = string(body)
		}
	}

	// Create complete log entry
	logEntry := HTTPLog{
		Request:  httpReq,
		Response: httpResp,
	}

	_ = f.httpLog.Log(logEntry)
}

// fetchURL fetches URL
func (f *Fetcher) fetchURL(url string, result *Result) error {
	startTime := time.Now()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		result.Error = err.Error()
		result.ResponseTime = time.Since(startTime).Milliseconds()
		f.writeHTTPLog(nil, url, nil, nil, result, startTime)
		return err
	}

	resp, err := f.client.Do(req)
	if err != nil {
		result.Error = err.Error()
		result.ResponseTime = time.Since(startTime).Milliseconds()
		f.writeHTTPLog(req, url, nil, nil, result, startTime)
		return err
	}
	defer resp.Body.Close()

	result.ResponseStatusCode = resp.StatusCode
	result.ResponseTime = time.Since(startTime).Milliseconds()
	result.ContentLength = resp.ContentLength

	body, err := io.ReadAll(io.LimitReader(resp.Body, f.maxResponseSize))
	if err != nil {
		result.Error = fmt.Sprintf("read body: %v", err)
		f.writeHTTPLog(req, url, resp, nil, result, startTime)
		return err
	}
	bodyStr := string(body)
	result.Title = extract.ExtractTitle(bodyStr)

	domains := f.extractor.FromString(bodyStr)
	filtered := extract.FilterBySuffix(domains, result.Root)
	sanitized := make([]string, len(filtered))
	for i, d := range filtered {
		sanitized[i] = f.sanitizer.Sanitize(d)
	}
	result.Subdomains = f.deduplicator.Deduplicate(sanitized)

	f.writeHTTPLog(req, url, resp, body, result, startTime)
	return nil
}
