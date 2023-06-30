package util

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/WangYihang/Subdomain-Crawler/internal/common"
	"github.com/WangYihang/Subdomain-Crawler/internal/model"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/enriquebris/goconcurrentqueue"
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

func CrawlAllSubdomains(sld string, sldWaitGroup *sync.WaitGroup, p *mpb.Progress) error {
	queue := goconcurrentqueue.NewFIFO()
	wg := &sync.WaitGroup{}
	scheduledDomains := mapset.NewSet[string]()
	var numAll int64 = 0
	var numDone int64 = 0

	for _, subdomain := range ExpandSubdomains(sld) {
		queue.Enqueue(subdomain)
		scheduledDomains.Add(subdomain)
		wg.Add(1)
		atomic.AddInt64(&numAll, 1)
	}

	mpb.BarStyle()
	bar := p.AddBar(0,
		mpb.BarOptional(mpb.BarRemoveOnComplete(), false),
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

	bar.SetCurrent(int64(numDone))
	bar.SetTotal(int64(numAll), false)

	for i := 0; i < model.Opts.NumGoroutinesPerWorker; i++ {
		go func() {
			for {
				task, err := queue.DequeueOrWaitForNextElement()

				if err != nil {
					continue
				}

				if task == nil {
					break
				}

				start := time.Now()
				domain := task.(string)
				domains, err := Crawl(domain)

				if err == nil {
					for _, domain := range domains {
						if !scheduledDomains.Contains(domain) {
							queue.Enqueue(domain)
							scheduledDomains.Add(domain)
							wg.Add(1)
							atomic.AddInt64(&numAll, 1)
						}
					}
					atomic.AddInt64(&common.NumFoundSubdomains, 1)
				}

				atomic.AddInt64(&numDone, 1)
				bar.EwmaSetCurrent(int64(numDone), time.Since(start))
				bar.SetTotal(int64(numAll), false)

				stateString := fmt.Sprintf("%s [%d / %d]", domain, common.NumDoneSlds, common.NumAllSlds)
				fmt.Printf("%s%s\r", strings.Repeat(" ", max(common.TerminalWidth-len(stateString), 0)), stateString)

				wg.Done()
			}
		}()
	}

	wg.Wait()

	bar.SetTotal(-1, true)

	for i := 0; i < model.Opts.NumGoroutinesPerWorker; i++ {
		queue.Enqueue(nil)
	}
	atomic.AddInt64(&common.NumDoneSlds, 1)

	sldWaitGroup.Done()

	return nil
}
