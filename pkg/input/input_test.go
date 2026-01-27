package input

import (
	"os"
	"testing"
)

func TestLoaderLoad(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	testData := `
example.com
# comment
test.org
  spaces.com  
`
	if _, err := tmpFile.WriteString(testData); err != nil {
		t.Fatalf("Failed to write: %v", err)
	}
	tmpFile.Close()

	l := NewLoader()
	domains, err := l.Load(tmpFile.Name())

	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	expected := []string{"example.com", "test.org", "spaces.com"}
	if len(domains) != len(expected) {
		t.Errorf("Load = %d, want %d", len(domains), len(expected))
	}
}
