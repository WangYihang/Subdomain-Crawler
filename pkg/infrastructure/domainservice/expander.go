package domainservice

import (
	"fmt"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// CommonSubdomains is a predefined list of common subdomain prefixes
var CommonSubdomains = []string{
	// Web servers
	"www", "web", "www1", "www2", "www3",

	// Services
	"mail", "smtp", "pop", "imap", "webmail",
	"ftp", "sftp", "files",
	"vpn", "remote",
	"api", "apis", "rest", "graphql",
	"cdn", "static", "assets", "img", "images",
	"blog", "forum", "wiki", "docs", "help", "support",

	// Development & Testing
	"dev", "development", "test", "testing", "qa",
	"stage", "staging", "uat", "preprod", "demo",
	"beta", "alpha", "preview",

	// Admin & Management
	"admin", "administrator", "manage", "management",
	"cpanel", "whm", "plesk",
	"portal", "dashboard", "console",

	// Databases
	"db", "database", "mysql", "postgres", "mongo", "redis",
	"sql", "mssql", "oracle",

	// Cloud & Infrastructure
	"cloud", "aws", "azure", "gcp",
	"ns", "ns1", "ns2", "ns3", "ns4",
	"dns", "dns1", "dns2",
	"mx", "mx1", "mx2",

	// Mobile & Apps
	"m", "mobile", "app", "apps", "wap",
	"ios", "android",

	// Regional/Geographic
	"us", "eu", "asia", "cn", "jp", "uk",
	"east", "west", "north", "south",

	// Business Functions
	"shop", "store", "ecommerce", "cart",
	"payment", "pay", "billing",
	"crm", "erp", "hr",

	// Media & Content
	"video", "videos", "media", "stream",
	"news", "press",

	// Monitoring & Analytics
	"monitor", "monitoring", "status",
	"stats", "analytics", "metrics",
	"log", "logs", "logging",

	// Security
	"secure", "ssl", "auth", "oauth", "sso",
	"proxy", "gateway",

	// Communication
	"chat", "im", "slack", "teams",
	"conference", "meet", "zoom",
}

// Expander expands SLDs to common subdomains
type Expander struct {
	subdomains []string
}

// NewExpander creates a new domain expander
func NewExpander(customSubdomains []string) *Expander {
	// Merge custom subdomains with common ones
	subdomains := make([]string, 0, len(CommonSubdomains)+len(customSubdomains))
	subdomains = append(subdomains, CommonSubdomains...)
	subdomains = append(subdomains, customSubdomains...)

	// Deduplicate
	seen := make(map[string]bool)
	unique := make([]string, 0, len(subdomains))
	for _, sub := range subdomains {
		sub = strings.ToLower(strings.TrimSpace(sub))
		if sub != "" && !seen[sub] {
			seen[sub] = true
			unique = append(unique, sub)
		}
	}

	return &Expander{subdomains: unique}
}

// ExpandDomain expands a domain with common subdomains
// If the input is already a subdomain (depth > 0), it returns the input as-is
// If the input is an SLD (e.g., "example.com"), it generates common subdomains
func (e *Expander) ExpandDomain(domain string) []string {
	domain = strings.ToLower(strings.TrimSpace(domain))

	// Check if it's a second-level domain (eTLD+1)
	etld1, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		// Can't determine, return as-is
		return []string{domain}
	}

	// If the input is already a subdomain (not just the SLD), don't expand
	if domain != etld1 {
		return []string{domain}
	}

	// It's an SLD, expand it with common subdomains
	expanded := make([]string, 0, len(e.subdomains)+1)

	// Add the domain itself
	expanded = append(expanded, domain)

	// Add common subdomains
	for _, prefix := range e.subdomains {
		subdomain := fmt.Sprintf("%s.%s", prefix, domain)
		expanded = append(expanded, subdomain)
	}

	return expanded
}

// IsSLD checks if a domain is a second-level domain (no subdomain parts)
func (e *Expander) IsSLD(domain string) bool {
	domain = strings.ToLower(strings.TrimSpace(domain))
	etld1, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		return false
	}
	return domain == etld1
}
