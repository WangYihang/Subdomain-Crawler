package common

import "golang.org/x/crypto/ssh/terminal"

var TerminalWidth int

func init() {
	width, _, _ := terminal.GetSize(0)
	TerminalWidth = width
}
