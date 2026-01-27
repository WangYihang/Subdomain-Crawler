package extract

import "testing"

func TestDomainExtractorFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"single", "Visit example.com", 1},
		{"multiple", "example.com and test.org", 2},
		{"none", "hello world", 0},
	}

	e := NewDomainExtractor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.FromString(tt.input)
			if len(result) != tt.expected {
				t.Errorf("FromString(%q) = %d, want %d", tt.input, len(result), tt.expected)
			}
		})
	}
}

func TestFilterFilter(t *testing.T) {
	f := NewFilter(".com")
	domains := []string{"example.com", "test.org", "another.com"}

	result := f.Filter(domains)
	if len(result) != 2 {
		t.Errorf("Filter returned %d, want 2", len(result))
	}
}

func TestDeduplicatorDeduplicate(t *testing.T) {
	d := NewDeduplicator()

	tests := []struct {
		name     string
		domains  []string
		expected int
	}{
		{"duplicates", []string{"example.com", "example.com", "test.org"}, 2},
		{"case insensitive", []string{"EXAMPLE.COM", "example.com"}, 1},
		{"nodups", []string{"example.com", "test.org"}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.Deduplicate(tt.domains)
			if len(result) != tt.expected {
				t.Errorf("Deduplicate = %d, want %d", len(result), tt.expected)
			}
		})
	}
}

func TestSanitizerSanitize(t *testing.T) {
	s := NewSanitizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal", "example.com", "example.com"},
		{"uppercase", "EXAMPLE.COM", "example.com"},
		{"spaces", "  example.com  ", "example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := s.Sanitize(tt.input); result != tt.expected {
				t.Errorf("Sanitize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
