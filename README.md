# Subdomain Crawler

A high-performance, well-architected subdomain discovery tool built with Go.

## Installation

```bash
go install github.com/WangYihang/Subdomain-Crawler/cmd/subdomain-crawler@latest
```

## Usage

```
INPUT/OUTPUT OPTIONS:
    -i <file>              Input file with root domains (default: stdin)
    -o <file>              Output file for results (default: result.jsonl)
    -http-log <file>       HTTP request/response log (default: http.jsonl)
    -dns-log <file>        DNS query/response log (default: dns.jsonl)

CRAWLING OPTIONS:
    -max-depth <n>         Maximum subdomain depth (default: 3)
    -workers <n>           Number of concurrent workers (default: 32)
    -queue-size <n>        Task queue size (default: 10000)
    -expand-sld            Auto-expand SLD with common subdomains (default: true)

HTTP OPTIONS:
    -http-timeout <sec>    HTTP request timeout (default: 10)
    -max-response-size <b> Maximum response size (default: 10485760)
    -user-agent <string>   HTTP User-Agent header (default: SubdomainCrawler/2.0)

DNS OPTIONS:
    -dns-timeout <sec>     DNS query timeout (default: 5)

DEDUPLICATION OPTIONS:
    -bloom-size <n>        Expected number of unique domains (default: 1000000)
    -bloom-fp <rate>       False positive rate (default: 0.01)
    -bloom-file <file>     Bloom filter persistence file (default: bloom.filter)

UI OPTIONS:
    -dashboard             Show interactive TUI dashboard (default: true)
```

## Examples

```bash
# From file
subdomain-crawler -i domains.txt

# From stdin
echo "example.com" | subdomain-crawler

# With custom settings
subdomain-crawler -i domains.txt -workers 64 -max-depth 5

# Automation mode (no dashboard)
subdomain-crawler -i domains.txt -dashboard=false > output.log 2>&1
```
