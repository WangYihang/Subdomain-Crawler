# Subdomain Crawler

A high-performance subdomain discovery tool built with Go.

## Installation

```bash
go install github.com/WangYihang/Subdomain-Crawler/cmd/subdomain-crawler@main
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

# Output files

> result.jsonl

```jsonl
{"domain":"mail.tsinghua.edu.cn","ips":["166.111.204.8"],"subdomains":["mail.tsinghua.edu.cn","sslvpn.tsinghua.edu.cn","mail-d.tsinghua.edu.cn"],"status":"200 ","status_code":200,"title":"清华大学电子邮件系统","content_length":21865,"timestamp":"2026-01-28T17:33:40.043992276+08:00"}
{"domain":"news.tsinghua.edu.cn","ips":["101.6.15.66"],"subdomains":[],"status":"200 OK","status_code":200,"title":"新闻网跳转","content_length":1055,"timestamp":"2026-01-28T17:33:40.080993411+08:00"}
...
```
