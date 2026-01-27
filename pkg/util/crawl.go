package util

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/WangYihang/Subdomain-Crawler/pkg/common"
)

type Task struct {
	URL    string `json:"url"`
	Domain string `json:"domain"`
	Result Result `json:"result"`
}

func NewTask(url, domain string) Task {
	return Task{
		URL:    url,
		Domain: domain,
	}
}

func (t Task) Do() error {
	for subdomain := range ExpandSubdomains(t.Domain) {
		for url := range LoadUrlFromDomain(subdomain) {
			fmt.Println(url)
			// t.Result = Processer(t, subdomain)
		}
	}
	return nil
}

// Crawl crawls one single domain, returns all subdomains matched in the response
// and write the task log to logs channel.
type Result struct {
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

func NewResult() Result {
	return Result{
		StartTime:       time.Now().UnixMilli(),
		EndTime:         -1,
		Subdomains:      []string{},
		RequestHeaders:  http.Header{},
		ResponseHeaders: http.Header{},
		Error:           "",
	}
}

func (r Result) ToJSON() []byte {
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
func Processer(task Task, suffix string) (result Result) {
	// Create result object
	result = NewResult()
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
	response, err := common.GetHTTPClient().Do(request)
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

func LoadUrlFromDomain(domain string) chan string {
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

// func Worker(tasks chan Task, numScheduled *int64, numMaxSubdomains int64, scheduled *sync.Map, wg *sync.WaitGroup, suffix string) chan Result {
// 	results := make(chan Result)
// 	go func() {
// 		defer close(results)
// 		for task := range tasks {
// 			r := Processer(task, suffix)
// 			results <- r
// 			go func(subdomains []string) {
// 				for _, subdomain := range subdomains {
// 					if _, exists := scheduled.LoadOrStore(subdomain, true); !exists {
// 						for url := range LoadUrlFromDomain(subdomain) {
// 							if atomic.LoadInt64(numScheduled) < int64(numMaxSubdomains) {
// 								wg.Add(1)
// 								tasks <- NewTask(url, subdomain)
// 								atomic.AddInt64(numScheduled, 1)
// 							}
// 						}
// 					}
// 				}
// 				wg.Done()
// 			}(r.Subdomains)
// 		}
// 	}()
// 	return results
// }

// func Loader(domain string, numScheduled *int64, numMaxSubdomains int64, wg *sync.WaitGroup, scheduled *sync.Map) chan Task {
// 	tasks := make(chan Task, numMaxSubdomains)
// 	for subdomain := range ExpandSubdomains(domain) {
// 		if _, exists := scheduled.LoadOrStore(subdomain, true); !exists {
// 			for url := range LoadUrlFromDomain(subdomain) {
// 				if atomic.LoadInt64(numScheduled) < int64(numMaxSubdomains) {
// 					wg.Add(1)
// 					tasks <- NewTask(url, subdomain)
// 					atomic.AddInt64(numScheduled, 1)
// 				}
// 			}
// 		}
// 	}
// 	return tasks
// }

// // CrawlAllSubdomains crawls all subdomains of domain
// func CrawlAllSubdomains(task Task) {
// 	// var numScheduled, numMaxSubdomains int64 = 0, 1024
// 	// domain := task.Domain
// 	// rank := task.Rank
// 	// results := []chan Result{}
// 	// scheduled := &sync.Map{}
// 	// wg := &sync.WaitGroup{}
// 	// tasks := Loader(rank, domain, &numScheduled, numMaxSubdomains, wg, scheduled)
// 	// go func() {
// 	// 	wg.Wait()
// 	// 	close(tasks)
// 	// }()
// 	// for i := 0; i < model.Opts.NumGoroutinesPerWorker; i++ {
// 	// 	results = append(results, Worker(tasks, &numScheduled, numMaxSubdomains, scheduled, wg, domain))
// 	// }
// 	// hash := Sha1Hash(domain)
// 	// // path := filepath.Join(model.Opts.OutputFolder, fmt.Sprintf("%s/%s/%s.json", hash[0:2], hash[2:4], domain))
// 	// count := Printer(path, Merger(results...))
// 	// suffix := fmt.Sprintf("(%d) %s has %d subdomains", rank, domain, count)
// 	// fmt.Println(suffix)
// }
