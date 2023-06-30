package util

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/WangYihang/Subdomain-Crawler/internal/common"
	"github.com/WangYihang/Subdomain-Crawler/internal/model"
	"github.com/jpillora/go-tld"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

func Crawl(domain string) ([]string, error) {
	// Create HTTP Request
	uri := fmt.Sprintf("https://%s/", domain)
	resp, err := common.RestyClient.R().SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/114.0").Get(uri)
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
			decor.Name(sld, decor.WCSyncWidth),
		),
		mpb.AppendDecorators(
			decor.CountersNoUnit("[%d / %d]", decor.WCSyncWidth),
			decor.Percentage(decor.WCSyncSpace),
			decor.OnComplete(
				decor.EwmaETA(decor.ET_STYLE_GO, 30, decor.WCSyncSpace), "done",
			),
		),
	)

	numDone, numAll := taskMap.GetState()
	bar.SetCurrent(int64(numDone))
	bar.SetTotal(int64(numAll), false)

	for i := 0; i < model.Opts.NumGoroutinesPerWorker; i++ {
		go func() {
			for {
				if taskMap.CheckDone() {
					break
				}

				task, err := taskMap.GetTask()
				start := time.Now()

				if err != nil {
					time.Sleep(2 * time.Second)
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
					atomic.AddInt64(&common.NumFoundSubdomains, 1)
				}

				numDone, numAll := taskMap.GetState()
				bar.EwmaSetCurrent(int64(numDone), time.Since(start))
				bar.SetTotal(int64(numAll), false)

				stateString := fmt.Sprintf("%s [%d / %d]", task.String(), common.NumDoneSlds, common.NumAllSlds)
				fmt.Printf("%s%s\r", strings.Repeat(" ", max(common.TerminalWidth-len(stateString), 0)), stateString)
			}
		}()
	}

	taskMap.Wait()

	bar.SetTotal(-1, true)

	wg.Done()
	atomic.AddInt64(&common.NumDoneSlds, 1)

	return nil
}
