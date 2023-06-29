package model

import (
	"github.com/jessevdk/go-flags"
)

var Opts Options

func init() {
	_, err := flags.Parse(&Opts)
	if err != nil {
		panic(err)
	}
}

type IOOptions struct {
	InputFile    string `short:"i" long:"input-file" description:"The input file" required:"true" default:"input.txt"`
	OutputFolder string `short:"o" long:"output-folder" description:"The output folder" required:"true" default:"output"`
}

type Options struct {
	IOOptions
	Timeout    int  `short:"t" long:"timeout" description:"Timeout of each HTTP request (in seconds)" default:"4"`
	NumWorkers int  `short:"n" long:"num-workers" description:"Number of workers" default:"64"`
	Debug      bool `short:"d" long:"debug" description:"Enable debug mode"`
	Version    bool `short:"v" long:"version" description:"Version"`
}
