package model

// Opts is the options for the program
var Opts Options

// IOOptions is the options for input and output
type IOOptions struct {
	InputFile  string `short:"i" long:"input-file" description:"The input file" required:"true" default:"input.txt"`
	OutputFile string `short:"o" long:"output-file" description:"The output file" required:"true" default:"output.jsonl"`
}

// Options is the options for the program
type Options struct {
	IOOptions
	Timeout    int  `short:"t" long:"timeout" description:"Timeout of each HTTP request (in seconds)" default:"16"`
	NumWorkers int  `short:"n" long:"num-workers" description:"Number of workers" default:"16"`
	Version    bool `short:"v" long:"version" description:"Version"`
}
