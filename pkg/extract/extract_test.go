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
	// Filter uses FilterBySuffix: keeps domain == root or domain ends with "."+root
	f := NewFilter("example.com")
	domains := []string{"example.com", "test.org", "www.example.com", "a.b.example.com", "other.com"}

	result := f.Filter(domains)
	want := []string{"example.com", "www.example.com", "a.b.example.com"}
	if len(result) != len(want) {
		t.Errorf("Filter returned %d, want %d: got %v", len(result), len(want), result)
	}
	seen := make(map[string]bool)
	for _, d := range result {
		seen[d] = true
	}
	for _, w := range want {
		if !seen[w] {
			t.Errorf("Filter missing expected %q", w)
		}
	}
}

func TestFilterBySuffix(t *testing.T) {
	root := "tsinghua.edu.cn"
	domains := []string{
		"index.css", "jquery.min.js", "news.htm", "document.getelementbyid",
		"www.tsinghua.edu.cn", "jobs.tsinghua.edu.cn", "info.tsinghua.edu.cn",
		"mails.tsinghua.edu.cn", "search.tsinghua.edu.cn", "tsinghua.edu.cn",
		"evil-tsinghua.edu.cn", "other.com",
	}

	got := FilterBySuffix(domains, root)
	want := []string{
		"www.tsinghua.edu.cn", "jobs.tsinghua.edu.cn", "info.tsinghua.edu.cn",
		"mails.tsinghua.edu.cn", "search.tsinghua.edu.cn", "tsinghua.edu.cn",
	}
	if len(got) != len(want) {
		t.Errorf("FilterBySuffix(%q) returned %d items, want %d: got %v", root, len(got), len(want), got)
	}
	seen := make(map[string]bool)
	for _, d := range got {
		seen[d] = true
	}
	for _, w := range want {
		if !seen[w] {
			t.Errorf("FilterBySuffix(%q) missing expected %q", root, w)
		}
	}
	// empty suffix returns nothing
	empty := FilterBySuffix(domains, "")
	if len(empty) != 0 {
		t.Errorf("FilterBySuffix(domains, \"\") = %d, want 0", len(empty))
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
