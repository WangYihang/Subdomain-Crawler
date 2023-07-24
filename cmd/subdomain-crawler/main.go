package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"math"
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
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
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
	timeout := model.Opts.Timeout
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
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	common.HTTPClient = &http.Client{
		Transport: &transport,
		Timeout:   time.Duration(timeout) * time.Second,
	}
	log.SetOutput(io.Discard)

	// Count all tasks
	common.NumAllTasks = util.CountNumLines(model.Opts.InputFile)

	// Init progress bar
	common.Progress = mpb.New(
		mpb.WithWaitGroup(nil),
		mpb.WithRefreshRate(500*time.Millisecond),
	)
	common.Bar = common.Progress.AddBar(
		int64(common.NumAllTasks),
		mpb.BarOptional(mpb.BarRemoveOnComplete(), true),
		mpb.PrependDecorators(
			decor.Name("total", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(
			decor.CountersNoUnit("[%d / %d]", decor.WCSyncWidth),
			decor.Percentage(decor.WCSyncSpace),
			decor.OnComplete(
				decor.EwmaETA(decor.ET_STYLE_GO, 30, decor.WCSyncSpace), "done",
			),
		),
	)
	common.Progress.UpdateBarPriority(common.Bar, math.MaxInt)

	if model.Opts.Debug {
		go util.PrometheusExporter()
		go func() {
			log.Println(http.ListenAndServe("localhost:36060", nil))
		}()
	}
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

func prod() {
	tasks := Loader(model.Opts.InputFile)
	stop := make(chan bool)

	for i := 0; i < model.Opts.NumWorkers; i++ {
		go func() {
			var startTime time.Time
			for task := range tasks {
				startTime = time.Now()
				util.CrawlAllSubdomains(task)
				common.Bar.EwmaIncrInt64(1, time.Since(startTime))
			}
			stop <- true
		}()
	}

	for i := 0; i < model.Opts.NumWorkers; i++ {
		<-stop
	}

	// Set total number of tasks to trigger progress bar completion
	common.Bar.SetTotal(common.NumAllTasks, true)

	// Wait for progress bar to finish
	common.Progress.Wait()
}

func dev() {
	fmt.Println(model.Opts.Domain)
	util.CrawlAllSubdomains(model.Opts.Domain)
}

func main() {
	prod()
}
