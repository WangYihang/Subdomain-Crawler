package util

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

var re *regexp.Regexp = regexp.MustCompile(`(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]`)

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
		// Extract domains from response header (e.g., Content-Security-Policy)
		for domain := range HeadersDomainExtracter(response.Header) {
			out <- domain
		}
		// Read Response Body
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return
		}
		defer response.Body.Close()
		// Extract domains from response body
		for domain := range BodyDomainExtracter(body) {
			out <- domain
		}
	}()
	return out
}

// BodyDomainExtracter extracts domains from response body
func BodyDomainExtracter(body []byte) chan string {
	out := make(chan string)
	go func() {
		defer close(out)
		for _, domain := range re.FindAll(body, -1) {
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
				for _, domain := range re.FindAllString(value, -1) {
					out <- domain
				}
			}
		}
	}()
	return out
}
