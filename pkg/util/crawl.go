package util

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/WangYihang/Subdomain-Crawler/pkg/common"
	"github.com/WangYihang/Subdomain-Crawler/pkg/model"
)

type Task struct {
	URL    string `json:"url"`
	Domain string `json:"domain"`
}

func NewTask(url, domain string) *Task {
	return &Task{
		URL:    url,
		Domain: domain,
	}
}

// Crawl crawls one single domain, returns all subdomains matched in the response
// and write the task log to logs channel.
type Result struct {
	Task
	RequestHeaders     http.Header `json:"request_headers"`
	CNAME              string      `json:"cname"`
	ResponseStatus     string      `json:"response_status"`
	ResponseStatusCode int         `json:"response_status_code"`
	ResponseProto      string      `json:"response_proto"`
	ResponseProtoMajor int         `json:"response_proto_major"`
	ResponseProtoMinor int         `json:"response_proto_minor"`
	ResponseHeaders    http.Header `json:"response_headers"`
	StartTime          int64       `json:"start_time"`
	EndTime            int64       `json:"end_time"`
	Subdomains         []string    `json:"subdomains"`
	Error              string      `json:"error"`
}

func NewResult(task *Task) *Result {
	return &Result{
		Task:            *task,
		StartTime:       time.Now().UnixMilli(),
		EndTime:         -1,
		Subdomains:      []string{},
		RequestHeaders:  http.Header{},
		ResponseHeaders: http.Header{},
		Error:           "",
	}
}

func (r *Result) ToJSON() []byte {
	jsonBytes, err := json.Marshal(r)
	if err != nil {
		return []byte{}
	}
	return jsonBytes
}

func Deduplicater(in chan string) chan string {
	out := make(chan string)
	go func() {
		defer close(out)
		dedup := sync.Map{}
		for s := range in {
			if _, exists := dedup.LoadOrStore(s, true); !exists {
				out <- s
			}
		}
	}()
	return out
}

func QueryCNAME(domain string) string {
	cname, err := net.LookupCNAME(domain)
	if err != nil {
		return ""
	}
	fqdn := strings.TrimRight(cname, ".")
	if fqdn == domain {
		return ""
	}
	return fqdn
}

func Processer(task *Task, suffix string) (result *Result) {
	// Create result object
	result = NewResult(task)
	defer func() {
		result.EndTime = time.Now().UnixMilli()
	}()

	// Create HTTP request
	request, err := http.NewRequest("GET", task.URL, nil)
	if err != nil {
		result.Error = err.Error()
		return
	}

	// Send the HTTP request
	response, err := common.HTTPClient.Do(request)
	if err != nil {
		result.Error = err.Error()
		return
	}
	result.RequestHeaders = response.Request.Header
	result.ResponseStatus = response.Status
	result.ResponseStatusCode = response.StatusCode
	result.ResponseProto = response.Proto
	result.ResponseProtoMajor = response.ProtoMajor
	result.ResponseProtoMinor = response.ProtoMinor
	result.ResponseHeaders = response.Header.Clone()

	// Extract subdomains from response
	for subdomain := range Deduplicater(SubdomainFilter(DomainExtracter(response), suffix)) {
		result.Subdomains = append(result.Subdomains, subdomain)
	}
	return result
}

func DomainToURLConverter(domain string) chan string {
	protocols := []string{"http", "https"}
	out := make(chan string)
	go func() {
		defer close(out)
		for _, protocol := range protocols {
			out <- fmt.Sprintf("%s://%s/", protocol, domain)
		}
	}()
	return out
}

func StringSliceToChan(s []string) chan string {
	out := make(chan string)
	go func() {
		defer close(out)
		for _, v := range s {
			out <- v
		}
	}()
	return out
}

func Worker(tasks chan *Task, scheduled *sync.Map, wg *sync.WaitGroup, suffix string) chan *Result {
	results := make(chan *Result)
	go func() {
		defer close(results)
		for task := range tasks {
			r := Processer(task, suffix)
			results <- r
			go func() {
				defer wg.Done()
				for _, subdomain := range r.Subdomains {
					if _, exists := scheduled.LoadOrStore(subdomain, true); !exists {
						for url := range DomainToURLConverter(subdomain) {
							wg.Add(1)
							tasks <- NewTask(url, subdomain)
						}
					}
				}
			}()
		}
	}()
	return results
}

func Loader(domain string, wg *sync.WaitGroup, scheduled *sync.Map) chan *Task {
	tasksSlice := []*Task{}
	for domain := range ExpandSubdomains(domain) {
		for url := range DomainToURLConverter(domain) {
			tasksSlice = append(tasksSlice, NewTask(url, domain))
		}
	}
	tasks := make(chan *Task, len(tasksSlice))
	for _, task := range tasksSlice {
		wg.Add(1)
		tasks <- task
		scheduled.Store(task.Domain, true)
	}
	return tasks
}

func Printer(path string, results chan *Result) {
	os.MkdirAll(filepath.Dir(path), 0755)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	for result := range results {
		f.Write(result.ToJSON())
		f.WriteString("\n")
	}
}

func Merger(cs ...chan *Result) chan *Result {
	var wg sync.WaitGroup
	out := make(chan *Result)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan *Result) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

// CrawlAllSubdomains crawls all subdomains of domain
func CrawlAllSubdomains(domain string) {
	results := []chan *Result{}
	scheduled := &sync.Map{}
	wg := &sync.WaitGroup{}
	tasks := Loader(domain, wg, scheduled)
	go func() {
		wg.Wait()
		close(tasks)
	}()
	for i := 0; i < model.Opts.NumGoroutinesPerWorker; i++ {
		results = append(results, Worker(tasks, scheduled, wg, domain))
	}
	path := filepath.Join(model.Opts.OutputFolder, fmt.Sprintf("%s.json", domain))
	Printer(path, Merger(results...))
	numSpaces := common.TerminalWidth - len(domain)
	fmt.Printf("%s%s\r", strings.Repeat(" ", numSpaces), domain)
}
