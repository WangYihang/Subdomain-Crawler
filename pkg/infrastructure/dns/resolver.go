package dns

import (
	"context"
	"fmt"
	"time"

	"github.com/WangYihang/Subdomain-Crawler/pkg/domain/service"
	"github.com/miekg/dns"
)

// Resolver implements service.DNSResolver
type Resolver struct {
	servers []string
	timeout time.Duration
	client  *dns.Client
}

// Config holds DNS resolver configuration
type Config struct {
	Servers []string
	Timeout time.Duration
}

// NewResolver creates a new DNS resolver
func NewResolver(config Config) *Resolver {
	if len(config.Servers) == 0 {
		config.Servers = []string{
			"8.8.8.8:53",
			"8.8.4.4:53",
			"1.1.1.1:53",
			"1.0.0.1:53",
		}
	}

	return &Resolver{
		servers: config.Servers,
		timeout: config.Timeout,
		client: &dns.Client{
			Timeout: config.Timeout,
		},
	}
}

// Resolve implements service.DNSResolver
func (r *Resolver) Resolve(domain string) ([]string, error) {
	resolution, err := r.ResolveWithDetails(domain)
	if err != nil {
		return nil, err
	}
	return resolution.IPs, nil
}

// ResolveWithDetails implements service.DNSResolver
func (r *Resolver) ResolveWithDetails(domain string) (*service.DNSResolution, error) {
	requestAt := time.Now()

	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	msg.RecursionDesired = true

	var lastErr error
	var response *dns.Msg
	var usedServer string
	var rtt time.Duration

	// Try each DNS server
	for _, server := range r.servers {
		ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
		resp, rttMs, err := r.client.ExchangeContext(ctx, msg, server)
		cancel()

		if err == nil && resp != nil {
			response = resp
			usedServer = server
			rtt = rttMs
			break
		}
		lastErr = err
	}

	responseAt := time.Now()

	if response == nil {
		errMsg := "no response from any DNS server"
		if lastErr != nil {
			errMsg = lastErr.Error()
		}
		return &service.DNSResolution{
			Domain:     domain,
			Error:      errMsg,
			RequestAt:  requestAt.UnixMilli(),
			ResponseAt: responseAt.UnixMilli(),
			RTTMs:      responseAt.Sub(requestAt).Milliseconds(),
		}, fmt.Errorf("%s", errMsg)
	}

	// Extract IPs and records
	var ips []string
	var records []service.DNSRecord

	for _, answer := range response.Answer {
		if aRecord, ok := answer.(*dns.A); ok {
			ip := aRecord.A.String()
			ips = append(ips, ip)
			records = append(records, service.DNSRecord{
				Type:  "A",
				Value: ip,
				TTL:   aRecord.Hdr.Ttl,
				Class: dns.ClassToString[aRecord.Hdr.Class],
			})
		}
	}

	return &service.DNSResolution{
		Domain:      domain,
		IPs:         ips,
		Records:     records,
		Server:      usedServer,
		RTTMs:       rtt.Milliseconds(),
		RequestAt:   requestAt.UnixMilli(),
		ResponseAt:  responseAt.UnixMilli(),
		RawRequest:  msg.String(),
		RawResponse: response.String(),
	}, nil
}
