package common

import (
	"strings"
	"sync"
)

// ScopeManager maintains the root domains whitelist and provides scope validation
type ScopeManager struct {
	rootDomains map[string]bool
	mu          sync.RWMutex
}

var (
	// GlobalScopeManager is the global scope manager instance
	GlobalScopeManager *ScopeManager
	onceScopeManager   sync.Once
)

// InitScopeManager initializes the global scope manager with root domains
func InitScopeManager(rootDomains []string) {
	onceScopeManager.Do(func() {
		GlobalScopeManager = &ScopeManager{
			rootDomains: make(map[string]bool),
		}
		for _, domain := range rootDomains {
			domain = strings.ToLower(strings.TrimSpace(domain))
			if domain != "" {
				GlobalScopeManager.rootDomains[domain] = true
			}
		}
	})
}

// IsAllowed checks if the given domain is allowed (belongs to a root domain in whitelist)
func (sm *ScopeManager) IsAllowed(domain string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	domain = strings.ToLower(domain)

	// Check if it's an exact match to a root domain
	if _, exists := sm.rootDomains[domain]; exists {
		return true
	}

	// Check if it's a subdomain of any root domain
	for root := range sm.rootDomains {
		if strings.HasSuffix(domain, "."+root) {
			return true
		}
	}

	return false
}

// GetDepth returns the depth level of the domain relative to its root domain
// Examples:
// - "a.com" has depth 0
// - "b.a.com" has depth 1
// - "c.b.a.com" has depth 2
// Returns -1 if domain is not in scope
func (sm *ScopeManager) GetDepth(domain string) int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	domain = strings.ToLower(domain)

	// Check if it's an exact match to a root domain
	if _, exists := sm.rootDomains[domain]; exists {
		return 0
	}

	// Check if it's a subdomain of any root domain
	for root := range sm.rootDomains {
		if strings.HasSuffix(domain, "."+root) {
			// Count the dots from the beginning to determine depth
			prefix := domain[:len(domain)-len(root)-1] // Remove the root domain and the dot
			depth := strings.Count(prefix, ".") + 1
			return depth
		}
	}

	return -1
}

// GetRootDomain returns the root domain of a given domain
// Returns empty string if domain is not in scope
func (sm *ScopeManager) GetRootDomain(domain string) string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	domain = strings.ToLower(domain)

	// Check if it's an exact match to a root domain
	if _, exists := sm.rootDomains[domain]; exists {
		return domain
	}

	// Find the matching root domain
	for root := range sm.rootDomains {
		if strings.HasSuffix(domain, "."+root) {
			return root
		}
	}

	return ""
}
