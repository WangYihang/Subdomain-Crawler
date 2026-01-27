package dns

import (
	"net"
	"sync"
	"time"
)

// Resolver resolves domains
type Resolver struct {
	timeout time.Duration
}

// NewResolver creates resolver
func NewResolver(timeout time.Duration) *Resolver {
	return &Resolver{timeout: timeout}
}

// Resolve resolves domain to IPs
func (r *Resolver) Resolve(domain string) ([]string, error) {
	ips, err := net.LookupIP(domain)
	if err != nil {
		return nil, err
	}

	result := make([]string, len(ips))
	for i, ip := range ips {
		result[i] = ip.String()
	}
	return result, nil
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
