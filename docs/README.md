# 递归式子域名挖掘工具设计文档 (Design Specification)

## 1. 项目概述 (Overview)

本工具旨在开发一个基于 GoLang 的高性能、并发式子域名挖掘器。它采用**主动递归 (Active Recursion)** 策略，以给定的域名列表为种子，通过爬取页面内容发现新的子域名，并将新发现的合法子域名作为新任务继续爬取，从而构建一个自动化的资产发现闭环。

## 2. 系统核心原则 (Core Principles)

- **并发优先**：利用 Goroutines 和 Channels 实现高并发爬取。
- **严格边界**：严防爬虫越界，仅允许爬取目标根域名的子域名。
- **状态持久化**：支持断点续传，防止长时间任务因崩溃导致进度丢失。
- **自我反馈**：新发现的子域名将作为输入源，驱动爬虫深入挖掘。

## 3. 系统架构设计 (Architecture)

系统采用 **"生产者-消费者 + 反馈环"** 架构模式。

### 3.1 核心组件

- **Scheduler (调度器)**：系统的总控中心，负责初始化组件、启动 Worker 池、监控程序状态以及处理优雅退出。

- **Scope Manager (作用域管理器)**：
  - 维护允许爬取的"根域名白名单"。
  - 提供 `IsTarget(url)` 方法，判断 URL 是否属于目标范围。
  - 负责层级深度检查，防止"蜘蛛陷阱"。

- **Deduplicator (去重管理器)**：
  - 基于 Bloom Filter 实现 URL 级别的去重。
  - 负责定期将过滤器状态 Dump 到磁盘，并在启动时 Load。

- **Wildcard Detector (泛解析检测器)**：
  - 在任务进入队列前，检测域名是否指向泛解析 IP。
  - 防止无效的随机子域名污染任务队列。

- **Job Queue (任务队列)**：
  - 基于 Go 的带缓冲 Channel。
  - 利用 Channel 的阻塞特性实现背压 (Backpressure)，防止内存溢出。

- **Worker Pool (工作池)**：
  - 执行 HTTP 请求。
  - 解析响应内容（正则匹配）。
  - 将结果分发到"结果处理器"和"任务队列"。

## 4. 详细功能需求 (Functional Requirements)

### 4.1 输入与配置 (Input & Configuration)

工具必须支持通过**命令行参数 (CLI Flags)** 配置以下行为：

- **输入源 (-f)**：包含种子域名的文本文件路径（一行一个）。
- **并发数 (-c)**：并发 Worker 的数量（默认：50）。
- **最大深度 (-d)**：允许递归的子域名层级深度（用于防止蜘蛛陷阱，默认：3）。
  - 定义：`a.com` 为 0 级，`b.a.com` 为 1 级。如果设为 2，则 `c.b.a.com` 不会被加入递归队列。
- **超时时间 (-t)**：HTTP 请求超时设置。
- **持久化间隔**：自动保存布隆过滤器的时间间隔。

### 4.2 爬虫逻辑 (Crawling Logic)

- **协议处理**：自动处理 HTTP/HTTPS。如果输入没有协议头，默认为 HTTPS，失败降级为 HTTP（或反之，需在配置中确定）。
- **请求头**：必须使用随机或伪造的 User-Agent，防止被简单 WAF 拦截。
- **HTTP Client**：禁用 Keep-Alive，禁用默认 Cookie 存储（除非有特殊需求），忽略 TLS 证书错误。

### 4.3 内容解析与提取 (Parsing)

- **提取方式**：不依赖 DOM 解析，直接对 Response Body 进行**正则表达式 (Regex)** 全文匹配。
- **匹配目标**：提取所有符合 URL/Hostname 特征的字符串。
- **清洗**：
  - 转换为小写。
  - 去除端口号（如 `:8080`）。
  - 去除路径和参数。

### 4.4 递归与反馈 (Recursion & Feedback)

当 Worker 发现一个潜在子域名 `sub.target.com` 时：

1. **Scope Check**：检查 `sub.target.com` 是否属于输入的根域名列表。
2. **Depth Check**：检查域名层级是否超过 `-d` 设定的阈值。
3. **Wildcard Check**：进行 DNS A 记录查询，对比泛解析 IP 黑名单。
4. **Deduplication**：查询布隆过滤器，检查该 URL 是否已爬取过。
5. **Enqueue**：若以上检查全部通过，将 `sub.target.com` 构造为完整 URL，推入 Job Queue。

### 4.5 持久化 (Persistence)

- **Bloom Filter Dump**：每隔 N 分钟（或接收到退出信号时），将内存中的 BitSet 写入磁盘文件（如 `session.bloom`）。
- **Recovery**：程序启动时，检测是否存在 `session.bloom`。如果存在，加载到内存，避免对旧目标的重复爬取。

## 5. 异常处理与防御机制 (Safety Mechanisms)

| 风险点 | 应对策略 |
|--------|----------|
| **蜘蛛陷阱 (无限子域)** | 最大深度限制 (MaxDepth)。如限制为 3 层，超过层级的子域名只记录结果，不加入爬取队列。 |
| **泛解析 (Wildcard DNS)** | DNS 验证。在入队前解析 IP，如果解析结果匹配该主域名的泛解析 IP 集合，则判定为无效，不入队。 |
| **内存溢出 (OOM)** | 有缓冲 Channel。当生产速度 > 消费速度时，Channel 满后阻塞 Worker，自然降低爬取速度。 |
| **网络阻塞** | 自定义 Transport。设置严格的 DialTimeout 和 ResponseHeaderTimeout，快速失败，不卡死 Worker。 |
| **程序崩溃** | 定期 Dump。保证重启后只丢失极少量进度（仅丢失 Channel 中未处理的任务）。 |

## 6. 数据流转图 (Data Flow)

```
[Disk/Input] --(Load)--> [Scope Manager]
                              |
[Startup] --(Load)--> [Bloom Filter]
                              |
                              v
                       [Job Queue (Buffered)]
                              |
                      (Pick Task) <-----------------------+
                              |                           |
                        [Worker Pool]                     |
                              |                           |
                    (HTTP Request & Regex)                |
                              |                           |
                  +-----------+-----------+               |
                  |                       |               |
             [New Subs]               [Results]           |
                  |                       |               |
           (Scope Check?)           (Write File/DB)       |
           (Depth Check?)                                 |
           (Wildcard Check?)                              |
                  |                                       |
           (Bloom Check?) --(No)--> (Mark Seen) --> [Enqueue]
                  |
                (Yes) -> [Discard]
```

## 7. 接口定义概览 (Interface Concept)

不需要具体代码，但需要定义模块间的交互契约。

### ScopeManager

```go
type ScopeManager interface {
    Init(rootDomains []string)
    IsAllowed(domain string) bool
    GetDepth(domain string) int
}
```

### BloomFilter

```go
type BloomFilter interface {
    ContainsOrAdd(key []byte) bool
    SaveToFile(path string) error
    LoadFromFile(path string) error
}
```

### WildcardDetector

```go
type WildcardDetector interface {
    IsWildcard(domain string) bool
}
```

## 8. 交付物清单 (Deliverables)

- **subcrawler** 二进制文件：编译后的可执行文件。
- **config.yaml** (可选)：如果不想用命令行参数，支持配置文件。
- **results.txt**：最终发现的子域名列表。
- **session.bloom**：运行时的状态快照文件。

---

**确认**：这份设计文档是否符合你的预期？如果确认无误，我们将以此为标准进入代码实现阶段。
