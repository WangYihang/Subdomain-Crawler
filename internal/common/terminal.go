package common

import (
	"github.com/olekukonko/ts"
)

var TerminalWidth int

func init() {
	size, _ := ts.GetSize()
	TerminalWidth = size.Col()
}
