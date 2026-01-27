# Subdomain Crawler

A high-performance subdomain discovery tool built with Go.

## Installation

```bash
go install github.com/WangYihang/Subdomain-Crawler/cmd/subdomain-crawler@latest
```

## Usage

```bash
subdomain-crawler --help
```

## Examples

```bash
# Basic usage (defaults to stdin)
echo "example.com" | subdomain-crawler

# With custom settings
subdomain-crawler --input domains.txt --workers 64 --max-depth 5 --output results.jsonl

# Automation mode (no dashboard)
subdomain-crawler --input domains.txt --no-dashboard
```
