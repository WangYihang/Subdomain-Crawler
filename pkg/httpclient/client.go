package httpclient

import (
	"crypto/tls"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"
)

// Config holds HTTP config
type Config struct {
	Timeout time.Duration
}

// Client wraps http.Client
type Client struct {
	client    *http.Client
	userAgent UserAgent
}

// UserAgent provides random agents
type UserAgent struct {
	agents []string
	mu     sync.RWMutex
}

// NewUserAgent creates agent provider
func NewUserAgent() *UserAgent {
	return &UserAgent{
		agents: []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
		},
	}
}

// Random returns random agent
func (ua *UserAgent) Random() string {
	ua.mu.RLock()
	defer ua.mu.RUnlock()
	if len(ua.agents) == 0 {
		return ""
	}
	return ua.agents[rand.Intn(len(ua.agents))]
}

// NewClient creates client
func NewClient(config *Config) *Client {
	transport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   config.Timeout,
			KeepAlive: 0,
		}).Dial,
		TLSHandshakeTimeout:   config.Timeout,
		IdleConnTimeout:       config.Timeout,
		ResponseHeaderTimeout: config.Timeout,
		DisableKeepAlives:     true,
		MaxIdleConns:          0,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	return &Client{
		client:    &http.Client{Transport: transport, Timeout: config.Timeout},
		userAgent: *NewUserAgent(),
	}
}

// Do performs request
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", c.userAgent.Random())
	return c.client.Do(req)
}

// Get performs GET
func (c *Client) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}
