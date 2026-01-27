package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/WangYihang/Subdomain-Crawler/pkg/common"
	"github.com/WangYihang/Subdomain-Crawler/pkg/model"
	"github.com/WangYihang/Subdomain-Crawler/pkg/util"
	"github.com/WangYihang/gojob"
	"github.com/jessevdk/go-flags"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

func init() {
	_, err := flags.Parse(&model.Opts)
	if err != nil {
		os.Exit(1)
	}

	common.InitGlobalBloomFilter(model.Opts.BloomFilterSize, model.Opts.BloomFilterFalsePositive)

	if model.Opts.Version {
		fmt.Println(common.PV.String())
		os.Exit(0)
	}

	// Init progress bar
	common.Progress = mpb.New(
		mpb.WithWaitGroup(nil),
		mpb.WithRefreshRate(time.Second),
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
	common.Progress.UpdateBarPriority(common.Bar, math.MaxInt, false)

	EnableObservability()
}

func EnableObservability() {
	go util.PrometheusExporter()
	go func() {
		log.Println(http.ListenAndServe("localhost:36060", nil))
	}()
}

func ParseLine(line string) (int, string, error) {
	index := strings.Index(line, ",")
	if index == -1 {
		return -1, line, nil
	}
	rankString := line[:index]
	domain := line[index+1:]
	rank, err := strconv.Atoi(rankString)
	if err != nil {
		return -1, "", err
	}
	return rank, domain, nil
}

func LoadTasks(filepath string) chan util.Task {
	out := make(chan util.Task)
	go func() {
		defer close(out)
		fd, err := os.Open(filepath)
		if err != nil {
			return
		}
		defer fd.Close()
		scanner := bufio.NewScanner(fd)
		for scanner.Scan() {
			domain := strings.TrimSpace(scanner.Text())
			out <- util.NewTask("", domain)
		}
	}()
	return out
}

func Count(channel chan util.Task) int64 {
	count := int64(0)
	for range channel {
		count++
	}
	return count
}

func main() {
	var numTotalTasks int64
	var err error

	numTotalTasks, err = util.CountLines(model.Opts.Input)
	if err != nil {
		log.Fatal(err)
	}

	scheduler := gojob.New(
		gojob.WithNumWorkers(model.Opts.NumWorkers),
		gojob.WithMaxRetries(4),
		gojob.WithMaxRuntimePerTaskSeconds(model.Opts.Timeout),
		gojob.WithNumShards(4),
		gojob.WithShard(0),
		gojob.WithTotalTasks(numTotalTasks),
		gojob.WithStatusFilePath(model.Opts.Status),
		gojob.WithResultFilePath(model.Opts.Output),
		gojob.WithMetadataFilePath(model.Opts.Metadata),
	).
		Start()

	for task := range LoadTasks(model.Opts.Input) {
		scheduler.Submit(task)
	}
	scheduler.Wait()
}
