package input

import (
	"bufio"
	"os"
	"strings"

	"github.com/WangYihang/Subdomain-Crawler/pkg/domain"
)

// Loader loads domains
type Loader struct {
	validator  *domain.Validator
	normalizer *domain.Normalizer
}

// NewLoader creates loader
func NewLoader() *Loader {
	return &Loader{
		validator:  domain.NewValidator(),
		normalizer: domain.NewNormalizer(),
	}
}

// Load loads domains from file
func (l *Loader) Load(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var domains []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if l.validator.IsValid(line) {
			normalized := l.normalizer.Normalize(line)
			domains = append(domains, normalized)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return domains, nil
}
