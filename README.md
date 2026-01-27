# Subdomain Crawler

A high-performance, well-architected subdomain discovery tool built with Go, featuring a beautiful TUI dashboard and automatic subdomain expansion.

## ✨ Features

- 🎨 **Beautiful TUI Dashboard** - Real-time monitoring with Bubble Tea
- 🔄 **SLD Auto-Expansion** - Automatically expands 134 common subdomains
- ⚡ **High Performance** - Concurrent crawling with configurable workers
- 🏗️ **Clean Architecture** - SOLID principles, high cohesion, low coupling
- 🧪 **Testable** - Interface-based design for easy unit testing
- ⚙️ **Highly Configurable** - 15+ command-line options
- 📊 **Comprehensive Logging** - Separate logs for HTTP and DNS requests
- 🎯 **Smart Deduplication** - Bloom filter for efficient deduplication

## 🚀 Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/WangYihang/Subdomain-Crawler.git
cd Subdomain-Crawler

# Build
go build -o subdomain-crawler ./cmd/subdomain-crawler/

# Or install directly
go install ./cmd/subdomain-crawler/
```

### Basic Usage

```bash
# From file
./subdomain-crawler -i domains.txt

# From stdin
echo "example.com" | ./subdomain-crawler

# With custom settings
./subdomain-crawler -i domains.txt -workers 64 -max-depth 5

# Run demo
./run-demo.sh
```

## 📖 Usage

### Command-Line Options

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

### Examples

```bash
# Basic scan with default settings
echo "example.com" | ./subdomain-crawler

# Large-scale scan
./subdomain-crawler -i domains.txt -workers 128 -max-depth 5 -bloom-size 10000000

# Quick test without expansion
./subdomain-crawler -i test.txt -workers 4 -max-depth 1 -expand-sld=false

# Automation mode (no dashboard)
./subdomain-crawler -i domains.txt -dashboard=false > output.log 2>&1
```

## 🎨 TUI Dashboard

The interactive dashboard provides real-time insights:

```
┌────────────────────────────────────────┐
│ 🔍 Subdomain Crawler - Running for 10s │
└────────────────────────────────────────┘

╭────────────────────────────────────╮
│ 📊 Statistics                       │
│                                     │
│ Queue Length:      1234             │
│ Active Workers:    8 / 8            │
│ Tasks Processed:   5678             │
│ Unique Subdomains: 892              │
│ HTTP Requests:     5682             │
│ DNS Requests:      5680             │
│ Errors:            4                │
│                                     │
│ HTTP Rate:         568.2 req/s      │
│ DNS Rate:          568.0 req/s      │
╰────────────────────────────────────╯
```

Press `q` or `Ctrl+C` to quit.

## 🔄 SLD Auto-Expansion

When you input a second-level domain (SLD), the crawler automatically generates **134 common subdomain variations**:

```bash
echo "example.com" | ./subdomain-crawler
# Automatically generates:
# - www.example.com
# - api.example.com
# - mail.example.com
# - ftp.example.com
# - vpn.example.com
# - admin.example.com
# - dev.example.com
# ... and 127 more
```

**Predefined subdomains include:**
- Web servers: www, www1, www2, web, m, mobile
- Mail services: mail, smtp, imap, pop, webmail
- APIs: api, apis, rest, graphql
- Development: dev, test, staging, beta, alpha
- Admin: admin, cpanel, dashboard, portal
- Databases: db, mysql, postgres, redis
- Cloud: cloud, aws, azure, cdn, static
- Security: vpn, ssl, auth, gateway
- Media: cdn, static, img, video, media
- And many more...

Disable with `-expand-sld=false` if not needed.

## 🏗️ Architecture

This project follows **Clean Architecture** principles:

```
pkg/
├── domain/              # Domain Layer (Core Business Rules)
│   ├── entity/         # Business entities
│   ├── repository/     # Repository interfaces
│   └── service/        # Domain service interfaces
│
├── application/         # Application Layer (Use Cases)
│   ├── crawl_usecase.go # Crawling orchestration
│   └── worker.go       # Worker implementation
│
├── infrastructure/      # Infrastructure Layer (Implementations)
│   ├── http/           # HTTP fetcher
│   ├── dns/            # DNS resolver
│   ├── storage/        # Bloom filter, queues, writers
│   └── domainservice/  # Domain services
│
└── interface/          # Interface Layer (External Interfaces)
    ├── cli/            # Command-line interface
    └── presenter/      # TUI dashboard
```

### Key Design Principles

- ✅ **Single Responsibility** - Each component has one clear purpose
- ✅ **Dependency Inversion** - Depend on abstractions, not concretions
- ✅ **Interface Segregation** - 15+ fine-grained interfaces
- ✅ **Open/Closed** - Easy to extend without modification
- ✅ **High Cohesion, Low Coupling** - Minimal dependencies

## 📊 Performance

- **Startup Time**: ~80ms
- **Memory Usage**: ~45MB
- **Concurrency**: 1-1024 workers supported
- **Binary Size**: ~11MB (single binary, no dependencies)

## 🧪 Testing

The architecture makes testing easy with interface-based design:

```go
// Example: Mock DNS resolver for testing
type MockResolver struct {
    ResolveFunc func(domain string) ([]string, error)
}

func (m *MockResolver) Resolve(domain string) ([]string, error) {
    return m.ResolveFunc(domain)
}

// Use in tests
func TestWorker(t *testing.T) {
    mock := &MockResolver{
        ResolveFunc: func(d string) ([]string, error) {
            return []string{"1.2.3.4"}, nil
        },
    }
    // Test with mock...
}
```

## 🤝 Contributing

Contributions are welcome! The clean architecture makes it easy to extend:

1. **Add new storage backend**: Implement `repository.ResultWriter`
2. **Add new DNS resolver**: Implement `service.DNSResolver`
3. **Add new filter**: Implement `repository.DomainFilter`
4. **Add new UI**: Implement `MetricsObserver`

## 📚 Documentation

- [Architecture Guide](docs/ARCHITECTURE.md) - Detailed architecture explanation

## 📈 Roadmap

- [ ] Unit test coverage
- [ ] Integration tests
- [ ] YAML configuration file support
- [ ] Request retry mechanism
- [ ] Rate limiting
- [ ] Proxy support
- [ ] Additional DNS record types

## 📄 License

See [LICENSE](LICENSE) file.

## 🙏 Acknowledgments

Built with:
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [miekg/dns](https://github.com/miekg/dns) - DNS library
- [bloom](https://github.com/bits-and-blooms/bloom) - Bloom filter

---

⭐ If you find this project useful, please consider giving it a star!
