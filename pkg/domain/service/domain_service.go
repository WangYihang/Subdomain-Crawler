package service

// DomainValidator validates domain names
type DomainValidator interface {
	// IsValid checks if a domain name is valid
	IsValid(domain string) bool
	// IsAllowed checks if a domain is allowed to be crawled
	IsAllowed(domain string) bool
	// IsInScope checks if a domain is within the scope
	IsInScope(domain, root string) bool
}

// DomainCalculator calculates domain properties
type DomainCalculator interface {
	// GetDepth calculates the subdomain depth
	GetDepth(domain string) int
	// GetRoot extracts the root domain (eTLD+1)
	GetRoot(domain string) (string, error)
	// GetDistance calculates distance between two domains
	GetDistance(domain, root string) int
}

// DomainExtractor extracts domains from content
type DomainExtractor interface {
	// ExtractFromText extracts domains from text content
	ExtractFromText(text string) []string
	// ExtractFromHTML extracts domains from HTML content
	ExtractFromHTML(html string) []string
	// FilterByRoot filters domains by root domain
	FilterByRoot(domains []string, root string) []string
}

// HTTPFetcher fetches web content
type HTTPFetcher interface {
	// Fetch fetches a URL and returns the response
	Fetch(url string) (*HTTPResponse, error)
}

// HTTPResponse represents an HTTP response
type HTTPResponse struct {
	URL           string
	StatusCode    int
	Headers       map[string]string
	Body          string
	ContentLength int
	Error         string
}

// DNSResolver resolves domain names
type DNSResolver interface {
	// Resolve resolves a domain to IP addresses
	Resolve(domain string) ([]string, error)
	// ResolveWithDetails resolves a domain and returns detailed records
	ResolveWithDetails(domain string) (*DNSResolution, error)
}

// DNSResolution represents detailed DNS resolution result
type DNSResolution struct {
	Domain      string
	IPs         []string
	Records     []DNSRecord
	Server      string
	RTTMs       int64
	Error       string
	RequestAt   int64
	ResponseAt  int64
	RawRequest  string
	RawResponse string
}

// DNSRecord represents a DNS record
type DNSRecord struct {
	Type  string
	Value string
	TTL   uint32
	Class string
}
