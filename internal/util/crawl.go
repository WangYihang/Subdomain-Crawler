package util

import (
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/WangYihang/Subdomain-Crawler/internal/model"
	"github.com/go-resty/resty/v2"
	"github.com/jpillora/go-tld"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

var client *resty.Client

func init() {
	client = resty.New()
	client.SetTimeout(time.Duration(model.Opts.Timeout) * time.Second)
	client.SetRedirectPolicy(resty.FlexibleRedirectPolicy(8))
	log.SetOutput(io.Discard)
}

func Crawl(domain string) ([]string, error) {
	// Create HTTP Request
	uri := fmt.Sprintf("https://%s/", domain)
	resp, err := client.R().SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/114.0").Get(uri)
	if err != nil {
		return []string{}, err
	}

	u, err := tld.Parse(uri)
	if err != nil {
		return []string{}, err
	}

	root := strings.Join([]string{u.Domain, u.TLD}, ".")
	return FilterDomain(MatchDomains(resp.Body()), root), nil
}

func CrawlAllSubdomains(sld string, wg *sync.WaitGroup, p *mpb.Progress) error {
	taskMap, err := model.CreateTaskMap(sld)
	if err != nil {
		return err
	}

	for _, subdomain := range ExpandSubdomains(sld) {
		taskMap.AddTask(subdomain, sld, true)
	}

	bar := p.AddBar(0,
		mpb.PrependDecorators(
			// simple name decorator
			decor.Name(fmt.Sprintf("%16s", sld)),
			// decor.DSyncWidth bit enables column width synchronization
			decor.Percentage(decor.WCSyncSpace),
		),
		mpb.AppendDecorators(
			// replace ETA decorator with "done" message, OnComplete event
			decor.OnComplete(
				// ETA decorator with ewma age of 30
				decor.EwmaETA(decor.ET_STYLE_GO, 30, decor.WCSyncWidth), "done",
			),
		),
	)

	numDone, numAll := taskMap.GetState()
	bar.SetCurrent(int64(numDone))
	bar.SetTotal(int64(numAll), false)

	numWorkers := 8
	for i := 0; i < numWorkers; i++ {
		go func() {
			for {
				task, err := taskMap.GetTask()
				start := time.Now()

				if err != nil {
					time.Sleep(64 * time.Millisecond)
					continue
				}

				domains, err := Crawl(task.Domain)

				if err != nil {
					taskMap.DoneWithFail(task.Domain)
				} else {
					for _, domain := range domains {
						taskMap.AddTask(domain, task.Sld, false)
					}
					taskMap.DoneWithSuccess(task.Domain)
				}

				numDone, numAll := taskMap.GetState()
				bar.EwmaSetCurrent(int64(numDone), time.Since(start))
				bar.SetTotal(int64(numAll), false)

				fmt.Printf("%d, %d %v\r", numDone, numAll, task)
			}
		}()
	}

	taskMap.Wait()

	bar.SetTotal(-1, true)

	wg.Done()

	return nil
}
