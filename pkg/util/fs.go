package util

import (
	"bufio"
	"os"
	"strings"
)

// CountLines counts the number of lines in a file
func CountLines(filepath string) (int64, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return 0, err
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
	return lines, nil
}
