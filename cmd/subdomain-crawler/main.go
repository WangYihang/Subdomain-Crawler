package main

import (
	"bufio"
	"fmt"
	"os"
	"sync"

	_ "net/http/pprof"

	"github.com/WangYihang/Subdomain-Crawler/internal/common"
	"github.com/WangYihang/Subdomain-Crawler/internal/model"
	"github.com/WangYihang/Subdomain-Crawler/internal/util"
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
		fmt.Println(err)
		os.Exit(1)
	}

	if model.Opts.Version {
		fmt.Println(common.PV.String())
		os.Exit(0)
	}

	wg = &sync.WaitGroup{}
	queue = make(chan string, model.Opts.NumWorkers)
	p = mpb.New(mpb.WithWaitGroup(nil))
}

func loader(filepath string) {
	readFile, err := os.Open(model.Opts.InputFile)
	if err != nil {
		panic(err)
	}
	defer readFile.Close()

	fileScanner := bufio.NewScanner(readFile)
	for fileScanner.Scan() {
		queue <- fileScanner.Text()
		wg.Add(1)
	}
}

func main() {
	if model.Opts.Debug {
		go util.PrometheusExporter()
	}

	loader(model.Opts.InputFile)

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

	wg.Wait()
	p.Wait()
}
