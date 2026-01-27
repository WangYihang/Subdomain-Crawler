package model

// Opts is the options for the program
var Opts Options

// IOOptions is the options for input and output
type IOOptions struct {
	InputFile string `short:"f" long:"file" description:"Input file path containing root domains (one per line)" required:"true" default:"input.txt"`
	Output    string `short:"o" long:"output" description:"Output file path for results" default:"output.jsonl"`
	Status    string `short:"s" long:"status" description:"Status file path" default:"status.jsonl"`
	Metadata  string `short:"m" long:"metadata" description:"Metadata file path" default:"metadata.jsonl"`
}

// Options is the options for the program
type Options struct {
	IOOptions
	Concurrency              int     `short:"c" long:"concurrency" description:"Number of concurrent workers" default:"50"`
	MaxDepth                 int     `short:"d" long:"max-depth" description:"Maximum subdomain depth level (0 = root only, 1 = a.com, 2 = b.a.com, etc.)" default:"3"`
	Timeout                  int     `short:"t" long:"timeout" description:"HTTP request timeout in seconds" default:"16"`
	BloomFilterSize          uint    `short:"b" long:"bloom-filter-size" description:"Size of the bloom filter" default:"1048576"`
	BloomFilterFalsePositive float64 `long:"bloom-filter-fp" description:"False positive rate of the bloom filter" default:"0.01"`
	Version                  bool    `short:"v" long:"version" description:"Show version"`
	Verbose                  bool    `long:"verbose" description:"Enable verbose output"`
}
