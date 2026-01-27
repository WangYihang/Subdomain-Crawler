package domain

import "strings"

// Validator validates domains
type Validator struct{}

// NewValidator creates validator
func NewValidator() *Validator {
	return &Validator{}
}

// IsValid checks domain validity
func (v *Validator) IsValid(domain string) bool {
	domain = strings.TrimSpace(domain)
	if len(domain) == 0 || len(domain) > 253 {
		return false
	}
	return !strings.Contains(domain, " ")
}

// Normalizer normalizes domains
type Normalizer struct{}

// NewNormalizer creates normalizer
func NewNormalizer() *Normalizer {
	return &Normalizer{}
}

// Normalize converts domain to lowercase
func (n *Normalizer) Normalize(domain string) string {
	return strings.ToLower(strings.TrimSpace(domain))
}

// Extractor extracts root domains
type Extractor struct {
	rootDomains map[string]bool
}

// NewExtractor creates extractor
func NewExtractor(rootDomains []string) *Extractor {
	normalizer := NewNormalizer()
	roots := make(map[string]bool)
	for _, root := range rootDomains {
		normalized := normalizer.Normalize(root)
		if normalized != "" {
			roots[normalized] = true
		}
	}
	return &Extractor{rootDomains: roots}
}

// ExtractRoot returns root domain
func (e *Extractor) ExtractRoot(domain string) string {
	normalizer := NewNormalizer()
	domain = normalizer.Normalize(domain)

	if e.rootDomains[domain] {
		return domain
	}

	for root := range e.rootDomains {
		if strings.HasSuffix(domain, "."+root) {
			return root
		}
	}
	return ""
}

// Calculator calculates depth
type Calculator struct {
	extractor *Extractor
}

// NewCalculator creates calculator
func NewCalculator(extractor *Extractor) *Calculator {
	return &Calculator{extractor: extractor}
}

// GetDepth returns domain depth
func (c *Calculator) GetDepth(domain string) int {
	normalizer := NewNormalizer()
	domain = normalizer.Normalize(domain)

	root := c.extractor.ExtractRoot(domain)
	if root == "" {
		return -1
	}

	if domain == root {
		return 0
	}

	prefix := domain[:len(domain)-len(root)-1]
	return strings.Count(prefix, ".") + 1
}

// Scope checks domain scope
type Scope struct {
	extractor *Extractor
}

// NewScope creates scope
func NewScope(extractor *Extractor) *Scope {
	return &Scope{extractor: extractor}
}

// IsAllowed checks if domain is allowed
func (s *Scope) IsAllowed(domain string) bool {
	return s.extractor.ExtractRoot(domain) != ""
}

// IsRootDomain checks if domain is root
func (s *Scope) IsRootDomain(domain string) bool {
	normalizer := NewNormalizer()
	domain = normalizer.Normalize(domain)
	return s.extractor.rootDomains[domain]
}
