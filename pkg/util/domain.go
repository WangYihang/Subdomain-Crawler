package util

import (
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/anaskhan96/soup"
	mapset "github.com/deckarep/golang-set/v2"
)

var re *regexp.Regexp = regexp.MustCompile(`(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]`)

// MatchDomains returns all matched domains in body
func MatchDomains(body []byte) []string {
	return re.FindAllString(string(body), -1)
}

// MatchDomainsBytes returns all matched domains in body in bytes type to avoid []byte to string conversion
func MatchDomainsBytes(body []byte) [][]byte {
	return re.FindAll(body, -1)
}

// ExpandSubdomains returns all subdomains of domain
func ExpandSubdomains(domain string) chan string {
	queue := make(chan string)
	go func() {
		defer close(queue)

		queue <- domain
		subDomainPrefixes := []string{
			"www", "mail", "forum", "m", "blog", "shop", "forums", "wiki",
			"community", "news", "api", "cdn", "admin", "cloud", "email",
			"web", "bbs", "portal", "test", "ftp", "vpn", "secure", "webmail",
			"remote", "dev", "support",
		}
		for _, subDomainPrefix := range subDomainPrefixes {
			subDomain := fmt.Sprintf("%s.%s", subDomainPrefix, domain)
			queue <- subDomain
		}
	}()
	return queue
}

// ParseDomains returns all domains in doc
func ParseDomains(doc soup.Root) chan string {
	queue := make(chan string)
	go func() {
		defer close(queue)

		tags := map[string]string{
			"a":      "href",
			"link":   "href",
			"script": "src",
			"img":    "src",
		}
		for tagName, attrName := range tags {
			tags := doc.FindAll(tagName)
			for _, tag := range tags {
				link := tag.Attrs()[attrName]
				u, err := url.Parse(link)
				if err != nil {
					continue
				}
				if u.Hostname() != "" {
					queue <- u.Hostname()
				}
			}
		}
	}()
	return queue
}

// FilterDomain returns all domains that match root
func FilterDomain(domains []string, root string) []string {
	filteredDomains := mapset.NewSet[string]()
	invalidPrefixes := []string{"u002f", "2f"}
	for _, domain := range domains {
		if strings.HasSuffix(domain, root) {
			matched := false
			for _, invalidPrefix := range invalidPrefixes {
				if strings.HasPrefix(strings.ToLower(domain), invalidPrefix) {
					filteredDomains.Add(domain[len(invalidPrefix):])
					matched = true
				}
			}
			if !matched {
				filteredDomains.Add(domain)
			}
		}
	}
	return filteredDomains.ToSlice()
}

// FilterDomainBytes returns all domains that match root in bytes type to avoid []byte to string conversion
func FilterDomainBytes(domains [][]byte, root []byte) chan string {
	queue := make(chan string)

	go func() {
		defer close(queue)
		invalidPrefixes := [][]byte{[]byte("u002f"), []byte("2f")}
		for _, domain := range domains {
			if bytes.HasSuffix(domain, root) {
				matched := false
				for _, invalidPrefix := range invalidPrefixes {
					if bytes.HasPrefix(bytes.ToLower(domain), invalidPrefix) {
						queue <- string(domain[len(invalidPrefix):])
						matched = true
					}
				}
				if !matched {
					queue <- string(domain)
				}
			}
		}

	}()

	return queue
}
