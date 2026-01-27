package model

// Opts is the options for the program
var Opts Options

// IOOptions is the options for input and output
type IOOptions struct {
	Input    string `short:"i" long:"input" description:"The input file path" required:"true" default:"input.txt"`
	Output   string `short:"o" long:"output" description:"The output file path" required:"true" default:"output.jsonl"`
	Status   string `short:"s" long:"status" description:"The status file path" default:"status.jsonl"`
	Metadata string `short:"m" long:"metadata" description:"The metadata file path" default:"metadata.jsonl"`
}

// Options is the options for the program
type Options struct {
	IOOptions
	Timeout    int  `short:"t" long:"timeout" description:"Timeout of each HTTP request (in seconds)" default:"16"`
	NumWorkers int  `short:"n" long:"num-workers" description:"Number of workers" default:"16"`
	Version    bool `short:"v" long:"version" description:"Version"`
}
