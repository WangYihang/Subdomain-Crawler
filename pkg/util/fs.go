package util

import (
	"bufio"
	"os"
	"strings"
)

// CountNumLines counts the number of lines in a file
func CountNumLines(filepath string) int64 {
	file, err := os.Open(filepath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	fileScanner := bufio.NewScanner(file)
	var lines int64 = 0
	for fileScanner.Scan() {
		if strings.TrimSpace(fileScanner.Text()) == "" {
			continue
		}
		lines++
	}
	return lines
}
