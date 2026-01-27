package extract

import (
	"io"
	"regexp"
	"strings"
)

// DomainExtractor extracts domains
type DomainExtractor struct {
	domainRegex *regexp.Regexp
}

// NewDomainExtractor creates extractor
func NewDomainExtractor() *DomainExtractor {
	domainRegex := regexp.MustCompile(`(?:[a-zA-Z0-9](?:[a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}`)
	return &DomainExtractor{domainRegex: domainRegex}
}

// FromBody extracts from body
func (de *DomainExtractor) FromBody(body io.ReadCloser) ([]string, error) {
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	return de.FromString(string(data)), nil
}

// FromString extracts from string
func (de *DomainExtractor) FromString(input string) []string {
	matches := de.domainRegex.FindAllString(input, -1)
	seen := make(map[string]bool)
	var result []string
	for _, match := range matches {
		if !seen[match] {
			seen[match] = true
			result = append(result, match)
		}
	}
	return result
}

// Filter filters domains
type Filter struct {
	suffix string
}

// NewFilter creates filter
func NewFilter(suffix string) *Filter {
	return &Filter{suffix: suffix}
}

// Filter filters by suffix
func (f *Filter) Filter(domains []string) []string {
	return FilterBySuffix(domains, f.suffix)
}

// FilterBySuffix keeps only domains that equal suffix or are subdomains of suffix
// (i.e. domain == suffix or domain ends with "."+suffix).
// When suffix is empty, returns empty list to avoid matching everything.
func FilterBySuffix(domains []string, suffix string) []string {
	if suffix == "" {
		return nil
	}
	suffix = strings.ToLower(strings.TrimSpace(suffix))
	var result []string
	for _, domain := range domains {
		d := strings.ToLower(strings.TrimSpace(domain))
		if d == suffix || strings.HasSuffix(d, "."+suffix) {
			result = append(result, domain)
		}
	}
	return result
}

// Deduplicator removes duplicates
type Deduplicator struct{}

// NewDeduplicator creates dedup
func NewDeduplicator() *Deduplicator {
	return &Deduplicator{}
}

// Deduplicate removes duplicates
func (d *Deduplicator) Deduplicate(domains []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, domain := range domains {
		normalized := strings.ToLower(strings.TrimSpace(domain))
		if !seen[normalized] {
			seen[normalized] = true
			result = append(result, normalized)
		}
	}
	return result
}

// Sanitizer sanitizes domains
type Sanitizer struct{}

// NewSanitizer creates sanitizer
func NewSanitizer() *Sanitizer {
	return &Sanitizer{}
}

// Sanitize cleans domain
func (s *Sanitizer) Sanitize(domain string) string {
	domain = strings.TrimSpace(domain)
	domain = strings.ToLower(domain)

	domain = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '-' {
			return r
		}
		return -1
	}, domain)

	domain = strings.Trim(domain, ".-")

	for strings.Contains(domain, "..") {
		domain = strings.ReplaceAll(domain, "..", ".")
	}

	return domain
}
