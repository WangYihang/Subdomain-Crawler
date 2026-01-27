package common

import (
	"crypto/tls"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/WangYihang/Subdomain-Crawler/pkg/model"
)

var (
	// HTTPClient is the http client
	client *http.Client
	once   sync.Once

	// Common User-Agent strings to randomize requests
	userAgents = []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Edge/91.0.864.59",
	}
)

// GetHTTPClient returns a configured HTTP client
func GetHTTPClient() *http.Client {
	once.Do(func() {
		timeout := model.Opts.Timeout
		transport := http.Transport{
			Dial: (&net.Dialer{
				Timeout:   time.Duration(timeout) * time.Second,
				KeepAlive: 0, // Disable keep-alive
			}).Dial,
			TLSHandshakeTimeout:   time.Duration(timeout) * time.Second,
			IdleConnTimeout:       time.Duration(timeout) * time.Second,
			ResponseHeaderTimeout: time.Duration(timeout) * time.Second,
			ExpectContinueTimeout: time.Duration(timeout) * time.Second,
			DisableKeepAlives:     true,
			DisableCompression:    false,
			MaxIdleConns:          0,
			MaxIdleConnsPerHost:   0,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
		client = &http.Client{
			Transport: &transport,
			Timeout:   time.Duration(timeout) * time.Second,
		}
		log.SetOutput(io.Discard)
	})
	return client
}

// GetRandomUserAgent returns a random User-Agent string
func GetRandomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}
