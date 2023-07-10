package common

import (
	"github.com/olekukonko/ts"
	"github.com/vbauerster/mpb/v8"
)

// TerminalWidth is the width of the terminal
var TerminalWidth int

// Progress is the progress bar
var Progress *mpb.Progress

// Bar is the bar that show the total progress instead of the progress of a single task
var Bar *mpb.Bar

func init() {
	size, _ := ts.GetSize()
	TerminalWidth = size.Col()
}
