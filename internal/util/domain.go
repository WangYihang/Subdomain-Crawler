package util

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/anaskhan96/soup"
	mapset "github.com/deckarep/golang-set/v2"
)

var re *regexp.Regexp

func init() {
	var err error
	re, err = regexp.Compile(`(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]`)
	if err != nil {
		panic(err)
	}
}

func MatchDomains(body []byte) []string {
	matches := []string{}
	for _, match := range re.FindAll(body, -1) {
		matches = append(matches, string(match))
	}
	return matches
}

func ExpandSubdomains(domain string) []string {
	subDomainPrefixes := []string{
		"www", "mail", "forum", "m", "blog", "shop", "forums", "wiki",
		"community", "news", "api", "cdn", "admin", "cloud", "email",
		"web", "bbs", "portal", "test", "ftp", "vpn", "secure", "webmail",
		"remote", "dev", "support",
	}

	domains := []string{}
	domains = append(domains, domain)
	for _, subDomainPrefix := range subDomainPrefixes {
		subDomain := fmt.Sprintf("%s.%s", subDomainPrefix, domain)
		if !strings.HasPrefix(subDomain, domain) {
			domains = append(domains, subDomain)
		}
	}
	return domains
}

func ParseDomains(doc soup.Root) []string {
	tags := map[string]string{
		"a":      "href",
		"link":   "href",
		"script": "src",
		"img":    "src",
	}
	domains := []string{}
	for tagName, attrName := range tags {
		tags := doc.FindAll(tagName)
		for _, tag := range tags {
			link := tag.Attrs()[attrName]
			u, err := url.Parse(link)
			if err != nil {
				continue
			}
			if u.Hostname() != "" {
				domains = append(domains, u.Hostname())
			}
		}
	}
	return domains
}

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
