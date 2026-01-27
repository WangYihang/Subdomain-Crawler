package domainservice

import (
	"regexp"
	"strings"

	"github.com/WangYihang/Subdomain-Crawler/pkg/domain/service"
	"golang.org/x/net/publicsuffix"
)

// Validator implements service.DomainValidator
type Validator struct {
	rootDomains map[string]bool
	domainRegex *regexp.Regexp
}

// NewValidator creates a new domain validator
func NewValidator(rootDomains []string) service.DomainValidator {
	roots := make(map[string]bool)
	for _, domain := range rootDomains {
		domain = strings.ToLower(strings.TrimSpace(domain))
		if root, err := publicsuffix.EffectiveTLDPlusOne(domain); err == nil {
			roots[root] = true
		} else {
			roots[domain] = true
		}
	}

	return &Validator{
		rootDomains: roots,
		domainRegex: regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`),
	}
}

// IsValid checks if a domain name is valid
func (v *Validator) IsValid(domain string) bool {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return false
	}
	return v.domainRegex.MatchString(domain)
}

// IsAllowed checks if a domain is allowed to be crawled
func (v *Validator) IsAllowed(domain string) bool {
	if !v.IsValid(domain) {
		return false
	}
	return v.IsInScope(domain, "")
}

// IsInScope checks if a domain is within the scope
func (v *Validator) IsInScope(domain, root string) bool {
	domain = strings.ToLower(strings.TrimSpace(domain))

	// If root is specified, check if domain is under that root
	if root != "" {
		return strings.HasSuffix(domain, "."+root) || domain == root
	}

	// Otherwise, check if domain is under any root
	domainRoot, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		return false
	}

	return v.rootDomains[domainRoot]
}

// Calculator implements service.DomainCalculator
type Calculator struct{}

// NewCalculator creates a new domain calculator
func NewCalculator() service.DomainCalculator {
	return &Calculator{}
}

// GetDepth calculates the subdomain depth
func (c *Calculator) GetDepth(domain string) int {
	domain = strings.ToLower(strings.TrimSpace(domain))
	root, err := c.GetRoot(domain)
	if err != nil {
		// Count all parts if we can't determine root
		return strings.Count(domain, ".") + 1
	}

	if domain == root {
		return 0
	}

	// Remove the root part and count remaining dots
	prefix := strings.TrimSuffix(domain, "."+root)
	if prefix == domain {
		return 0
	}

	return strings.Count(prefix, ".") + 1
}

// GetRoot extracts the root domain (eTLD+1)
func (c *Calculator) GetRoot(domain string) (string, error) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	return publicsuffix.EffectiveTLDPlusOne(domain)
}

// GetDistance calculates distance between two domains
func (c *Calculator) GetDistance(domain, root string) int {
	domain = strings.ToLower(strings.TrimSpace(domain))
	root = strings.ToLower(strings.TrimSpace(root))

	if !strings.HasSuffix(domain, root) {
		return -1 // Not related
	}

	if domain == root {
		return 0
	}

	prefix := strings.TrimSuffix(domain, "."+root)
	return strings.Count(prefix, ".") + 1
}

// Extractor implements service.DomainExtractor
type Extractor struct {
	domainRegex *regexp.Regexp
}

// NewExtractor creates a new domain extractor
func NewExtractor() service.DomainExtractor {
	return &Extractor{
		domainRegex: regexp.MustCompile(`(?i)(?:[a-zA-Z0-9](?:[a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}`),
	}
}

// ExtractFromText extracts domains from text content
func (e *Extractor) ExtractFromText(text string) []string {
	matches := e.domainRegex.FindAllString(text, -1)

	// Deduplicate
	seen := make(map[string]bool)
	var unique []string
	for _, match := range matches {
		match = strings.ToLower(strings.TrimSpace(match))
		if match != "" && !seen[match] {
			seen[match] = true
			unique = append(unique, match)
		}
	}

	return unique
}

// ExtractFromHTML extracts domains from HTML content
func (e *Extractor) ExtractFromHTML(html string) []string {
	// For now, treat HTML as text
	// In a more sophisticated version, we'd parse HTML and extract from specific tags
	return e.ExtractFromText(html)
}

// FilterByRoot filters domains by root domain
func (e *Extractor) FilterByRoot(domains []string, root string) []string {
	root = strings.ToLower(strings.TrimSpace(root))
	var filtered []string

	for _, domain := range domains {
		domain = strings.ToLower(strings.TrimSpace(domain))
		if strings.HasSuffix(domain, "."+root) || domain == root {
			filtered = append(filtered, domain)
		}
	}

	return filtered
}

// ExtractTitle extracts the title from HTML content
func (e *Extractor) ExtractTitle(html string) string {
	// Regex to find title with support for attributes and newlines
	re := regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		title := strings.TrimSpace(matches[1])
		// Replace newlines and tabs with spaces
		title = strings.Map(func(r rune) rune {
			if r == '\n' || r == '\r' || r == '\t' {
				return ' '
			}
			return r
		}, title)
		// Collapse multiple spaces
		spaceRe := regexp.MustCompile(`\s+`)
		title = spaceRe.ReplaceAllString(title, " ")
		return title
	}
	return ""
}
