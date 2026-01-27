package domain

import "testing"

func TestValidatorIsValid(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		expected bool
	}{
		{"valid", "example.com", true},
		{"subdomain", "sub.example.com", true},
		{"empty", "", false},
		{"spaces", "ex ample.com", false},
	}

	v := NewValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := v.IsValid(tt.domain); result != tt.expected {
				t.Errorf("IsValid(%q) = %v, want %v", tt.domain, result, tt.expected)
			}
		})
	}
}

func TestNormalizerNormalize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"uppercase", "EXAMPLE.COM", "example.com"},
		{"spaces", "  example.com  ", "example.com"},
		{"mixed", "  EXAMPLE.COM  ", "example.com"},
	}

	n := NewNormalizer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := n.Normalize(tt.input); result != tt.expected {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractorExtractRoot(t *testing.T) {
	e := NewExtractor([]string{"example.com", "test.org"})

	tests := []struct {
		name     string
		domain   string
		expected string
	}{
		{"root", "example.com", "example.com"},
		{"subdomain", "sub.example.com", "example.com"},
		{"deep", "a.b.example.com", "example.com"},
		{"unknown", "unknown.net", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := e.ExtractRoot(tt.domain); result != tt.expected {
				t.Errorf("ExtractRoot(%q) = %q, want %q", tt.domain, result, tt.expected)
			}
		})
	}
}

func TestCalculatorGetDepth(t *testing.T) {
	e := NewExtractor([]string{"example.com"})
	c := NewCalculator(e)

	tests := []struct {
		name     string
		domain   string
		expected int
	}{
		{"root", "example.com", 0},
		{"1level", "www.example.com", 1},
		{"2level", "api.v1.example.com", 2},
		{"3level", "a.b.c.example.com", 3},
		{"unknown", "unknown.net", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := c.GetDepth(tt.domain); result != tt.expected {
				t.Errorf("GetDepth(%q) = %d, want %d", tt.domain, result, tt.expected)
			}
		})
	}
}

func TestScopeIsAllowed(t *testing.T) {
	e := NewExtractor([]string{"example.com", "test.org"})
	s := NewScope(e)

	tests := []struct {
		name     string
		domain   string
		expected bool
	}{
		{"root", "example.com", true},
		{"subdomain", "www.example.com", true},
		{"other", "other.org", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := s.IsAllowed(tt.domain); result != tt.expected {
				t.Errorf("IsAllowed(%q) = %v, want %v", tt.domain, result, tt.expected)
			}
		})
	}
}
