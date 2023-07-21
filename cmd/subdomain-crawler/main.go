package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync/atomic"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/WangYihang/Subdomain-Crawler/pkg/common"
	"github.com/WangYihang/Subdomain-Crawler/pkg/model"
	"github.com/WangYihang/Subdomain-Crawler/pkg/util"
	"github.com/jessevdk/go-flags"
)

func init() {
	_, err := flags.Parse(&model.Opts)
	if err != nil {
		os.Exit(1)
	}

	if model.Opts.Version {
		fmt.Println(common.PV.String())
		os.Exit(0)
	}

	// Init HTTP client
	timeout := 1
	transport := http.Transport{
		Dial: (&net.Dialer{
			// Modify the time to wait for a connection to establish
			Timeout:   time.Duration(timeout) * time.Second,
			KeepAlive: time.Duration(timeout) * time.Second,
		}).Dial,
		TLSHandshakeTimeout:   time.Duration(timeout) * time.Second,
		IdleConnTimeout:       time.Duration(timeout) * time.Second,
		ResponseHeaderTimeout: time.Duration(timeout) * time.Second,
		ExpectContinueTimeout: time.Duration(timeout) * time.Second,
		DisableKeepAlives:     true,
	}
	common.HTTPClient = &http.Client{
		Transport: &transport,
		Timeout:   time.Duration(timeout) * time.Second,
	}
	log.SetOutput(io.Discard)

	// Count all tasks
	common.NumAllTasks = util.CountNumLines(model.Opts.InputFile)

	// // Init progress bar
	// common.Progress = mpb.New(
	// 	mpb.WithWaitGroup(nil),
	// 	mpb.WithRefreshRate(500*time.Millisecond),
	// )
	// common.Bar = common.Progress.AddBar(
	// 	int64(common.NumAllTasks),
	// 	mpb.BarOptional(mpb.BarRemoveOnComplete(), true),
	// 	mpb.PrependDecorators(
	// 		decor.Name("total", decor.WCSyncWidth),
	// 	),
	// 	mpb.AppendDecorators(
	// 		decor.CountersNoUnit("[%d / %d]", decor.WCSyncWidth),
	// 		decor.Percentage(decor.WCSyncSpace),
	// 		decor.OnComplete(
	// 			decor.EwmaETA(decor.ET_STYLE_GO, 30, decor.WCSyncSpace), "done",
	// 		),
	// 	),
	// )
	// common.Progress.UpdateBarPriority(common.Bar, math.MaxInt)
}

func Loader(filepath string) chan string {
	taskQueue := make(chan string)

	go func() {
		defer close(taskQueue)

		readFile, err := os.Open(filepath)
		if err != nil {
			panic(err)
		}
		defer readFile.Close()

		fileScanner := bufio.NewScanner(readFile)
		for fileScanner.Scan() {
			taskQueue <- fileScanner.Text()
			atomic.AddInt64(&common.NumScheduledTasks, 1)
		}
	}()

	return taskQueue
}

func mainA() {
	if model.Opts.Debug {
		go util.PrometheusExporter()
		go func() {
			log.Println(http.ListenAndServe("localhost:36060", nil))
		}()
	}

	tasks := Loader(model.Opts.InputFile)
	stop := make(chan bool)

	for i := 0; i < model.Opts.NumWorkers; i++ {
		go func() {
			for task := range tasks {
				util.CrawlAllSubdomains(task)
			}
			stop <- true
		}()
	}

	for i := 0; i < model.Opts.NumWorkers; i++ {
		<-stop
	}

	// // Set total number of tasks to trigger progress bar completion
	// common.Bar.SetTotal(common.NumAllTasks, true)

	// // Wait for progress bar to finish
	// common.Progress.Wait()
}

func mainB() {
	// util.CrawlAllSubdomains("tsinghua.edu.cn")
	// util.CrawlAllSubdomains("sjtu.edu.cn")
}

func main() {
	mainA()
}
