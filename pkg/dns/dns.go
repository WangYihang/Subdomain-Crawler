package dns

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// Resolver resolves domains
type Resolver struct {
	timeout     time.Duration
	dnsServers  []string
	client      *dns.Client
	fallbackNet bool // fallback to stdlib net.LookupIP if true
}

// NewResolver creates resolver
func NewResolver(timeout time.Duration) *Resolver {
	// Use public DNS servers
	dnsServers := []string{
		"8.8.8.8:53",        // Google
		"8.8.4.4:53",        // Google
		"1.1.1.1:53",        // Cloudflare
		"1.0.0.1:53",        // Cloudflare
		"208.67.222.222:53", // OpenDNS
		"208.67.220.220:53", // OpenDNS
	}

	// Try to get system DNS servers
	config, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err == nil && len(config.Servers) > 0 {
		var systemServers []string
		for _, s := range config.Servers {
			if config.Port == "" {
				systemServers = append(systemServers, net.JoinHostPort(s, "53"))
			} else {
				systemServers = append(systemServers, net.JoinHostPort(s, config.Port))
			}
		}
		// Prefer system DNS servers
		dnsServers = append(systemServers, dnsServers...)
	}

	return &Resolver{
		timeout:    timeout,
		dnsServers: dnsServers,
		client: &dns.Client{
			Timeout: timeout,
		},
		fallbackNet: true,
	}
}

// DNSResponse represents a single DNS response
type DNSResponse struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	TTL   uint32 `json:"ttl,omitempty"`
	Class string `json:"class,omitempty"`
}

// DNSRequest represents a DNS request for logging
type DNSRequest struct {
	Domain    string   `json:"domain"`
	Types     []string `json:"types"`
	RequestAt int64    `json:"request_at"`
	Timestamp string   `json:"timestamp"`
}

// DNSResponseLog represents a DNS response for logging
type DNSResponseLog struct {
	Answers    []DNSResponse `json:"answers"`
	IPs        []string      `json:"ips"`
	Error      string        `json:"error,omitempty"`
	RTTMs      int64         `json:"rtt_ms"`
	ResponseAt int64         `json:"response_at"`
	Timestamp  string        `json:"timestamp"`
	DNSServer  string        `json:"dns_server,omitempty"`
}

// DNSLog represents a complete DNS query log entry
type DNSLog struct {
	Request     DNSRequest     `json:"request"`
	Response    DNSResponseLog `json:"response"`
	RawRequest  string         `json:"raw_request,omitempty"`
	RawResponse string         `json:"raw_response,omitempty"`
}

// DNSResult contains detailed DNS query result (for internal use)
type DNSResult struct {
	Domain       string
	RequestTypes []string
	RequestAt    time.Time
	Responses    []DNSResponse
	IPs          []string
	Error        string
	RTTMs        int64
	ResponseAt   time.Time
	UsedServer   string
	RawRequest   string // DNS request in wire format (hex)
	RawResponse  string // DNS response in wire format (hex)
}

// ResolveDetailed performs DNS resolution with detailed logging
func (r *Resolver) ResolveDetailed(domain string) *DNSResult {
	startTime := time.Now()
	result := &DNSResult{
		Domain:       domain,
		RequestTypes: []string{"A", "AAAA"},
		RequestAt:    startTime,
	}

	var ips []string
	var responses []DNSResponse
	var lastErr error
	var usedServer string

	// Query A records
	for _, server := range r.dnsServers {
		msg := new(dns.Msg)
		msg.SetQuestion(dns.Fqdn(domain), dns.TypeA)
		msg.RecursionDesired = true

		resp, rtt, err := r.client.Exchange(msg, server)
		if err == nil && resp != nil && resp.Rcode == dns.RcodeSuccess {
			usedServer = server
			result.RTTMs = rtt.Milliseconds()

			// Store raw request/response
			if reqWire, err := msg.Pack(); err == nil {
				result.RawRequest = fmt.Sprintf("%x", reqWire)
			}
			if respWire, err := resp.Pack(); err == nil {
				result.RawResponse = fmt.Sprintf("%x", respWire)
			}

			for _, ans := range resp.Answer {
				if a, ok := ans.(*dns.A); ok {
					ip := a.A.String()
					ips = append(ips, ip)
					responses = append(responses, DNSResponse{
						Type:  "A",
						Value: ip,
						TTL:   a.Hdr.Ttl,
						Class: dns.ClassToString[a.Hdr.Class],
					})
				}
			}
			break
		}
		lastErr = err
	}

	// Query AAAA records
	for _, server := range r.dnsServers {
		msg := new(dns.Msg)
		msg.SetQuestion(dns.Fqdn(domain), dns.TypeAAAA)
		msg.RecursionDesired = true

		resp, rtt, err := r.client.Exchange(msg, server)
		if err == nil && resp != nil && resp.Rcode == dns.RcodeSuccess {
			if usedServer == "" {
				usedServer = server
			}
			if result.RTTMs == 0 {
				result.RTTMs = rtt.Milliseconds()
			}

			for _, ans := range resp.Answer {
				if aaaa, ok := ans.(*dns.AAAA); ok {
					ip := aaaa.AAAA.String()
					ips = append(ips, ip)
					responses = append(responses, DNSResponse{
						Type:  "AAAA",
						Value: ip,
						TTL:   aaaa.Hdr.Ttl,
						Class: dns.ClassToString[aaaa.Hdr.Class],
					})
				}
			}
			break
		}
		if lastErr == nil {
			lastErr = err
		}
	}

	result.ResponseAt = time.Now()
	result.IPs = ips
	result.Responses = responses
	result.UsedServer = usedServer

	if len(ips) == 0 && lastErr != nil {
		result.Error = lastErr.Error()
		// Fallback to stdlib if enabled
		if r.fallbackNet {
			if fallbackIPs, err := r.resolveFallback(domain); err == nil && len(fallbackIPs) > 0 {
				result.IPs = fallbackIPs
				result.Error = ""
				for _, ip := range fallbackIPs {
					ipObj := net.ParseIP(ip)
					typ := "AAAA"
					if ipObj != nil && ipObj.To4() != nil {
						typ = "A"
					}
					result.Responses = append(result.Responses, DNSResponse{
						Type:  typ,
						Value: ip,
					})
				}
			}
		}
	}

	result.RTTMs = time.Since(startTime).Milliseconds()
	return result
}

// resolveFallback uses stdlib net.LookupIP as fallback
func (r *Resolver) resolveFallback(domain string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	resolver := &net.Resolver{}
	ips, err := resolver.LookupIP(ctx, "ip", domain)
	if err != nil {
		return nil, err
	}

	result := make([]string, len(ips))
	for i, ip := range ips {
		result[i] = ip.String()
	}
	return result, nil
}

// Resolve resolves domain to IPs (simple interface for backward compatibility)
func (r *Resolver) Resolve(domain string) ([]string, error) {
	result := r.ResolveDetailed(domain)
	if result.Error != "" {
		return result.IPs, fmt.Errorf("%s", result.Error)
	}
	return result.IPs, nil
}

// WildcardDetector detects wildcards
type WildcardDetector struct {
	resolver    *Resolver
	wildcardIPs map[string]map[string]bool
	mu          sync.RWMutex
}

// NewWildcardDetector creates detector
func NewWildcardDetector(resolver *Resolver) *WildcardDetector {
	return &WildcardDetector{
		resolver:    resolver,
		wildcardIPs: make(map[string]map[string]bool),
	}
}

// DetectWildcard detects wildcard
func (wd *WildcardDetector) DetectWildcard(rootDomain string) ([]string, error) {
	randomSubdomain := "notexistcrawlerprobe.invalid." + rootDomain

	ips, err := wd.resolver.Resolve(randomSubdomain)
	if err != nil {
		return []string{}, nil
	}

	if len(ips) > 0 {
		wd.registerWildcardIPs(rootDomain, ips)
	}

	return ips, nil
}

// registerWildcardIPs registers wildcard IPs
func (wd *WildcardDetector) registerWildcardIPs(rootDomain string, ips []string) {
	wd.mu.Lock()
	defer wd.mu.Unlock()

	if _, exists := wd.wildcardIPs[rootDomain]; !exists {
		wd.wildcardIPs[rootDomain] = make(map[string]bool)
	}

	for _, ip := range ips {
		wd.wildcardIPs[rootDomain][ip] = true
	}
}

// IsWildcardIP checks if IP is wildcard
func (wd *WildcardDetector) IsWildcardIP(rootDomain string, ip string) bool {
	wd.mu.RLock()
	defer wd.mu.RUnlock()

	knownIPs, exists := wd.wildcardIPs[rootDomain]
	if !exists {
		return false
	}

	return knownIPs[ip]
}
