package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/WangYihang/Subdomain-Crawler/internal/common"
	"github.com/WangYihang/Subdomain-Crawler/internal/model"
	"github.com/WangYihang/Subdomain-Crawler/internal/util"
	"github.com/go-resty/resty/v2"
	"github.com/jessevdk/go-flags"
	"github.com/vbauerster/mpb/v8"
)

var (
	wg    *sync.WaitGroup
	queue chan string
	p     *mpb.Progress
)

func init() {
	_, err := flags.Parse(&model.Opts)
	if err != nil {
		os.Exit(1)
	}

	common.RestyClient = resty.New()
	common.RestyClient.SetTimeout(time.Duration(model.Opts.Timeout) * time.Second)
	common.RestyClient.SetRedirectPolicy(resty.FlexibleRedirectPolicy(8))
	log.SetOutput(io.Discard)

	if model.Opts.Version {
		fmt.Println(common.PV.String())
		os.Exit(0)
	}

	wg = &sync.WaitGroup{}
	queue = make(chan string, model.Opts.NumWorkers)
	p = mpb.New(
		mpb.WithWaitGroup(nil),
		mpb.WithRefreshRate(500*time.Millisecond),
	)
	common.NumAllSlds = util.CountNumLines(model.Opts.InputFile)
}

func loader(filepath string) {
	readFile, err := os.Open(filepath)
	if err != nil {
		panic(err)
	}
	defer readFile.Close()

	fileScanner := bufio.NewScanner(readFile)
	for fileScanner.Scan() {
		queue <- fileScanner.Text()
		wg.Add(1)
		atomic.AddInt64(&common.NumScheduledSlds, 1)
	}
}

func main() {
	if model.Opts.Debug {
		go util.PrometheusExporter()
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}
	for i := 0; i < model.Opts.NumWorkers; i++ {
		go func() {
			for domain := range queue {
				err := util.CrawlAllSubdomains(domain, wg, p)
				if err != nil {
					fmt.Println(err)
				}
			}
			wg.Done()
		}()
	}
	loader(model.Opts.InputFile)
	wg.Wait()
	p.Wait()
}
