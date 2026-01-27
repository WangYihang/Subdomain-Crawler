package common

import (
	"net"
	"sync"
	"time"
)

// WildcardDetector detects if a domain is pointing to a wildcard DNS resolution
type WildcardDetector struct {
	// wildcardIPs stores the wildcard resolution IPs for root domains
	// Key: root domain, Value: set of IPs that are wildcard resolutions
	wildcardIPs map[string]map[string]bool
	mu          sync.RWMutex
	// Timeout for DNS queries
	timeout time.Duration
}

var (
	// GlobalWildcardDetector is the global wildcard detector instance
	GlobalWildcardDetector *WildcardDetector
	onceWildcardDetector   sync.Once
)

// InitWildcardDetector initializes the global wildcard detector
func InitWildcardDetector(timeout time.Duration) {
	onceWildcardDetector.Do(func() {
		GlobalWildcardDetector = &WildcardDetector{
			wildcardIPs: make(map[string]map[string]bool),
			timeout:     timeout,
		}
	})
}

// RegisterWildcardIPs registers known wildcard resolution IPs for a root domain
// This is typically called during initialization with the root domain
func (wd *WildcardDetector) RegisterWildcardIPs(rootDomain string, ips []string) {
	wd.mu.Lock()
	defer wd.mu.Unlock()

	if _, exists := wd.wildcardIPs[rootDomain]; !exists {
		wd.wildcardIPs[rootDomain] = make(map[string]bool)
	}

	for _, ip := range ips {
		wd.wildcardIPs[rootDomain][ip] = true
	}
}

// IsWildcard checks if the given domain is a wildcard DNS resolution
// It resolves the domain and compares the IP against known wildcard IPs for its root domain
func (wd *WildcardDetector) IsWildcard(domain string, rootDomain string) bool {
	wd.mu.RLock()
	knownWildcardIPs, exists := wd.wildcardIPs[rootDomain]
	wd.mu.RUnlock()

	// If no wildcard IPs registered for this root domain, we can't detect
	// In this case, we assume it's not a wildcard (allow it)
	if !exists || len(knownWildcardIPs) == 0 {
		return false
	}

	// Simple DNS A record lookup
	// In a real implementation, you might want to handle IPv6 and other edge cases
	ips, err := net.LookupIP(domain)
	if err != nil {
		// If resolution fails, it's not a wildcard
		return false
	}

	// Check if any resolved IP matches the known wildcard IPs
	for _, ip := range ips {
		if knownWildcardIPs[ip.String()] {
			return true
		}
	}

	return false
}

// DetectWildcardIPs attempts to detect wildcard DNS resolution for a root domain
// by testing a random subdomain that is unlikely to exist
func (wd *WildcardDetector) DetectWildcardIPs(rootDomain string) ([]string, error) {
	// Use a pseudo-random subdomain that is very unlikely to exist
	randomSubdomain := "notexistcrawlerprobe.invalid." + rootDomain

	ips, err := net.LookupIP(randomSubdomain)
	if err != nil {
		// If resolution fails, there's likely no wildcard
		return []string{}, nil
	}

	var detectedIPs []string
	for _, ip := range ips {
		detectedIPs = append(detectedIPs, ip.String())
	}

	// Register the detected wildcard IPs
	if len(detectedIPs) > 0 {
		wd.RegisterWildcardIPs(rootDomain, detectedIPs)
	}

	return detectedIPs, nil
}
