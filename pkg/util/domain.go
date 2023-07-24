package util

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"unsafe"
)

// ExpandSubdomains returns all subdomains of domain
func ExpandSubdomains(domain string) chan string {
	subDomainPrefixes := []string{
		"www", "mail", "forum", "m", "blog", "shop", "forums", "wiki",
		"community", "news", "api", "cdn", "admin", "cloud", "email",
		"web", "bbs", "portal", "test", "ftp", "vpn", "secure", "webmail",
		"remote", "dev", "support",
	}
	queue := make(chan string, len(subDomainPrefixes)+1)
	go func() {
		defer close(queue)
		queue <- domain
		for _, subDomainPrefix := range subDomainPrefixes {
			subDomain := fmt.Sprintf("%s.%s", subDomainPrefix, domain)
			queue <- subDomain
		}
	}()
	return queue
}

// SubdomainFilter returns a channel in which the domain has the given suffix
func SubdomainFilter(in chan string, suffix string) chan string {
	out := make(chan string)
	go func() {
		defer close(out)
		for domain := range in {
			if strings.HasSuffix(domain, suffix) {
				out <- domain
			}
		}
	}()
	return out
}

// DomainExtracter extracts domains from *http.Response
func DomainExtracter(response *http.Response) chan string {
	out := make(chan string)
	go func() {
		defer close(out)
		// Extract domains from response body
		for domain := range BodyDomainExtracter(response.Body) {
			out <- domain
		}

		// Extract domains from response header (e.g., Content-Security-Policy)
		for domain := range HeadersDomainExtracter(response.Header) {
			out <- domain
		}
	}()
	return out
}

// BodyDomainExtracter extracts domains from response body
func BodyDomainExtracter(body io.ReadCloser) chan string {
	out := make(chan string)
	go func() {
		defer close(out)
		for domain := range ExtractDomains(body) {
			out <- string(domain)
		}
	}()
	return out
}

// HeadersDomainExtracter extracts domains from response header (e.g., Content-Security-Policy)
func HeadersDomainExtracter(header http.Header) chan string {
	out := make(chan string)
	go func() {
		defer close(out)
		for _, values := range header {
			for _, value := range values {
				reader := io.NopCloser(strings.NewReader(value))
				for domain := range ExtractDomains(reader) {
					out <- domain
				}
			}
		}
	}()
	return out
}

type DomainBuilder struct {
	domain [253]byte
	index  int
}

func (db *DomainBuilder) Append(ch byte) {
	if db.index >= 253 {
		return
	}
	db.domain[db.index] = ch
	db.index++
}

func (db *DomainBuilder) String() string {
	return db.StringUnsafe()
}

func (db *DomainBuilder) StringBuilder() string {
	builder := strings.Builder{}
	builder.Grow(db.index)
	builder.Write(db.domain[:db.index])
	return builder.String()
}

func (db *DomainBuilder) StringUnsafe() string {
	domain := make([]byte, db.index)
	copy(domain, db.domain[:db.index])
	return unsafe.String(unsafe.SliceData(domain), db.index)
}

func (db *DomainBuilder) StringSlow() string {
	domain := make([]byte, db.index)
	copy(domain, db.domain[:db.index])
	return string(domain)
}

func (db *DomainBuilder) Reset() {
	db.index = 0
}

func (db *DomainBuilder) Len() int {
	return db.index
}

func ExtractDomains(body io.ReadCloser) chan string {
	out := make(chan string)
	validHexCharChecker := func(ch byte) bool {
		if ch >= 'a' && ch <= 'f' {
			return true
		}
		if ch >= 'A' && ch <= 'F' {
			return true
		}
		if ch >= '0' && ch <= '9' {
			return true
		}
		return false
	}
	validDomainPartCharChecker := func(ch byte) bool {
		if ch >= 'a' && ch <= 'z' {
			return true
		}
		if ch >= 'A' && ch <= 'Z' {
			return true
		}
		if ch >= '0' && ch <= '9' {
			return true
		}
		if ch == '-' || ch == '.' {
			return true
		}
		return false
	}
	validDomainChecker := func(domain string) bool {
		// Check if domain has at least one dot
		if !strings.Contains(domain, ".") {
			return false
		}
		// Check length of every part of domain
		parts := strings.Split(domain, ".")
		for _, part := range parts {
			if len(part) > 63 || len(part) == 0 {
				return false
			}
		}
		// Check length of domain
		if len(domain) > 253 {
			return false
		}
		return true
	}
	go func() {
		defer close(out)
		builder := DomainBuilder{}
		buf := make([]byte, 1)
		var ppch, pch, ch byte
		var domain string
		for {
			n, err := body.Read(buf)
			if err != nil {
				body.Close()
				break
			}
			if n == 0 {
				continue
			}
			ch = buf[0]

			if ppch == '%' && validHexCharChecker(pch) && validHexCharChecker(ch) {
				builder.Reset()
				continue
			}

			if validDomainPartCharChecker(ch) {
				builder.Append(ch)
			} else {
				domain = builder.String()
				if builder.Len() > 0 && validDomainChecker(domain) {
					out <- domain
				}
				builder.Reset()
			}

			ppch = pch
			pch = ch
		}
		domain = builder.String()
		if builder.Len() > 0 && validDomainChecker(domain) {
			out <- domain
		}
		builder.Reset()
	}()
	return out
}
