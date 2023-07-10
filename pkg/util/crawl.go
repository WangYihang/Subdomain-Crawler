package util

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/WangYihang/Subdomain-Crawler/pkg/common"
	"github.com/WangYihang/Subdomain-Crawler/pkg/model"
	"github.com/jpillora/go-tld"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

// Crawl crawls one single domain, returns all subdomains matched in the response
// and write the task log to logs channel.
func Crawl(domain string, logs chan []byte) (chan string, error) {
	uri := fmt.Sprintf("https://%s/", domain)
	request, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	response, err := common.HTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	endTime := time.Now()

	// Save task log
	jsonObject, err := json.Marshal(map[string]interface{}{
		"start_time": startTime.UTC().Unix(),
		"end_time":   endTime.UTC().Unix(),
		"domain":     domain,
		"response": map[string]interface{}{
			"status":  response.Status,
			"proto":   response.Proto,
			"headers": response.Header,
		},
	})
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	logs <- jsonObject

	u, err := tld.Parse(uri)
	if err != nil {
		return nil, err
	}

	root := strings.Join([]string{u.Domain, u.TLD}, ".")

	queue := make(chan string)

	go func() {
		defer close(queue)

		// Extract subdomains from body content
		for _, subdomainsFromContent := range FilterDomain(MatchDomains(body), root) {
			queue <- subdomainsFromContent
		}

		// Extract subdomains from header
		for _, headerValues := range response.Header {
			for _, headerValue := range headerValues {
				for _, subdomainFromHeaderValue := range FilterDomain(MatchDomains([]byte(headerValue)), root) {
					queue <- subdomainFromHeaderValue
				}
			}
		}
	}()

	return queue, nil
}

func worker(tasks chan string, logs chan []byte, wg *sync.WaitGroup, scheduled *sync.Map, bar *mpb.Bar, numAll, numDone *int64) {
	for task := range tasks {
		start := time.Now()

		subdomains, err := Crawl(task, logs)

		bar.EwmaSetCurrent(*numDone, time.Since(start))
		bar.SetTotal(*numAll, false)

		if err != nil {
			atomic.AddInt64(numDone, 1)
			wg.Done()
		} else {
			go func() {
				for subdomain := range subdomains {
					if _, ok := scheduled.LoadOrStore(subdomain, true); !ok {
						atomic.AddInt64(numAll, 1)
						wg.Add(1)
						tasks <- subdomain
					}
				}
				atomic.AddInt64(numDone, 1)
				wg.Done()
			}()
		}
	}
}

func logsaver(domain string, logs chan []byte) {
	outputFilepath := filepath.Join(model.Opts.OutputFolder, fmt.Sprintf("%s.json", domain))
	os.MkdirAll(filepath.Dir(outputFilepath), 0755)

	f, err := os.OpenFile(outputFilepath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	for log := range logs {
		f.Write(log)
		f.Write([]byte("\n"))
	}
}

// CrawlAllSubdomains crawls all subdomains of domain
func CrawlAllSubdomains(domain string) {
	start := time.Now()
	wg := &sync.WaitGroup{}
	scheduled := &sync.Map{}
	tasks := make(chan string)
	logs := make(chan []byte)
	var numAll int64 = 0
	var numDone int64 = 0

	// Add progress bar
	mpb.BarStyle()
	bar := common.Progress.AddBar(0,
		mpb.BarOptional(mpb.BarRemoveOnComplete(), true),
		mpb.PrependDecorators(
			decor.Name(domain, decor.WCSyncWidth),
		),
		mpb.AppendDecorators(
			decor.CountersNoUnit("[%d / %d]", decor.WCSyncWidth),
			decor.Percentage(decor.WCSyncSpace),
			decor.OnComplete(
				decor.EwmaETA(decor.ET_STYLE_GO, 30, decor.WCSyncSpace), "done",
			),
		),
	)

	bar.SetCurrent(int64(numDone))
	bar.SetTotal(int64(numAll), false)

	// Start workers
	for i := 0; i < model.Opts.NumGoroutinesPerWorker; i++ {
		go worker(tasks, logs, wg, scheduled, bar, &numAll, &numDone)
	}

	// Start log saver
	go logsaver(domain, logs)

	// Add tasks
	for subdomain := range ExpandSubdomains(domain) {
		scheduled.Store(subdomain, true)
		atomic.AddInt64(&numAll, 1)
		wg.Add(1)
		tasks <- subdomain
	}

	// Wait for all tasks to be done
	wg.Wait()

	// Close progress bar
	bar.SetTotal(-1, true)

	// Close tasks channel
	close(tasks)

	// Close logs channel
	close(logs)

	// Increment number of done tasks
	common.Bar.EwmaIncrBy(1, time.Since(start))
}
