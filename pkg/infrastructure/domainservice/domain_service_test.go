package domainservice

import (
	"testing"
)

func TestValidator_IsValid(t *testing.T) {
	validator := NewValidator([]string{"example.com"})

	tests := []struct {
		name     string
		domain   string
		expected bool
	}{
		{"valid domain", "www.example.com", true},
		{"valid subdomain", "api.example.com", true},
		{"invalid empty", "", false},
		{"invalid format", "not a domain", false},
		{"valid deep subdomain", "deep.sub.example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.IsValid(tt.domain)
			if result != tt.expected {
				t.Errorf("IsValid(%s) = %v, want %v", tt.domain, result, tt.expected)
			}
		})
	}
}

func TestValidator_IsInScope(t *testing.T) {
	validator := NewValidator([]string{"example.com", "test.com"})

	tests := []struct {
		name     string
		domain   string
		root     string
		expected bool
	}{
		{"in scope - exact match", "example.com", "example.com", true},
		{"in scope - subdomain", "www.example.com", "example.com", true},
		{"in scope - deep subdomain", "api.v1.example.com", "example.com", true},
		{"out of scope", "attacker.com", "example.com", false},
		{"in scope - test.com", "api.test.com", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.IsInScope(tt.domain, tt.root)
			if result != tt.expected {
				t.Errorf("IsInScope(%s, %s) = %v, want %v", tt.domain, tt.root, result, tt.expected)
			}
		})
	}
}

func TestCalculator_GetDepth(t *testing.T) {
	calc := NewCalculator()

	tests := []struct {
		name     string
		domain   string
		expected int
	}{
		{"root domain", "example.com", 0},
		{"one level", "www.example.com", 1},
		{"two levels", "api.www.example.com", 2},
		{"three levels", "v1.api.www.example.com", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calc.GetDepth(tt.domain)
			if result != tt.expected {
				t.Errorf("GetDepth(%s) = %d, want %d", tt.domain, result, tt.expected)
			}
		})
	}
}

func TestCalculator_GetRoot(t *testing.T) {
	calc := NewCalculator()

	tests := []struct {
		name        string
		domain      string
		expected    string
		expectError bool
	}{
		{"simple domain", "example.com", "example.com", false},
		{"subdomain", "www.example.com", "example.com", false},
		{"deep subdomain", "api.v1.example.com", "example.com", false},
		{"edu domain", "cs.tsinghua.edu.cn", "tsinghua.edu.cn", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := calc.GetRoot(tt.domain)
			if (err != nil) != tt.expectError {
				t.Errorf("GetRoot(%s) error = %v, expectError %v", tt.domain, err, tt.expectError)
				return
			}
			if result != tt.expected {
				t.Errorf("GetRoot(%s) = %s, want %s", tt.domain, result, tt.expected)
			}
		})
	}
}

func TestExtractor_ExtractFromText(t *testing.T) {
	extractor := NewExtractor()

	tests := []struct {
		name     string
		text     string
		minCount int // Minimum expected domains
	}{
		{
			"simple text",
			"Visit www.example.com and api.example.com",
			2,
		},
		{
			"HTML content",
			`<a href="http://www.example.com">Link</a> Contact: admin@example.com`,
			2,
		},
		{
			"no domains",
			"No domains here!",
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.ExtractFromText(tt.text)
			if len(result) < tt.minCount {
				t.Errorf("ExtractFromText() found %d domains, want at least %d", len(result), tt.minCount)
			}
		})
	}
}

func TestExtractor_FilterByRoot(t *testing.T) {
	extractor := NewExtractor()

	domains := []string{
		"www.example.com",
		"api.example.com",
		"www.attacker.com",
		"blog.example.com",
	}

	result := extractor.FilterByRoot(domains, "example.com")

	if len(result) != 3 {
		t.Errorf("FilterByRoot() returned %d domains, want 3", len(result))
	}

	// Check that attacker.com is filtered out
	for _, d := range result {
		if d == "www.attacker.com" {
			t.Errorf("FilterByRoot() should not include www.attacker.com")
		}
	}
}

func TestExpander_IsSLD(t *testing.T) {
	expander := NewExpander(nil)

	tests := []struct {
		name     string
		domain   string
		expected bool
	}{
		{"is SLD", "example.com", true},
		{"is SLD - edu", "tsinghua.edu.cn", true},
		{"not SLD - has subdomain", "www.example.com", false},
		{"not SLD - deep", "api.v1.example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expander.IsSLD(tt.domain)
			if result != tt.expected {
				t.Errorf("IsSLD(%s) = %v, want %v", tt.domain, result, tt.expected)
			}
		})
	}
}

func TestExpander_ExpandDomain(t *testing.T) {
	expander := NewExpander(nil)

	t.Run("expand SLD", func(t *testing.T) {
		result := expander.ExpandDomain("example.com")
		// Should include original + common subdomains
		if len(result) < 100 {
			t.Errorf("ExpandDomain(example.com) returned %d domains, expected at least 100", len(result))
		}

		// Check that original domain is included
		found := false
		for _, d := range result {
			if d == "example.com" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ExpandDomain should include original domain")
		}
	})

	t.Run("don't expand subdomain", func(t *testing.T) {
		result := expander.ExpandDomain("www.example.com")
		if len(result) != 1 {
			t.Errorf("ExpandDomain(www.example.com) should return 1 domain, got %d", len(result))
		}
		if result[0] != "www.example.com" {
			t.Errorf("ExpandDomain(www.example.com) = %s, want www.example.com", result[0])
		}
	})
}

func TestExpander_CustomSubdomains(t *testing.T) {
	custom := []string{"custom1", "custom2"}
	expander := NewExpander(custom)

	result := expander.ExpandDomain("example.com")

	// Check that custom subdomains are included
	foundCustom1 := false
	foundCustom2 := false
	for _, d := range result {
		if d == "custom1.example.com" {
			foundCustom1 = true
		}
		if d == "custom2.example.com" {
			foundCustom2 = true
		}
	}

	if !foundCustom1 || !foundCustom2 {
		t.Errorf("ExpandDomain should include custom subdomains")
	}
}
