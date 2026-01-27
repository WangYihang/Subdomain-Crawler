package dedup

import (
	"os"
	"testing"
	"time"
)

func TestFilterTestAndAdd(t *testing.T) {
	f := NewFilter(1000, 0.01)

	data1 := []byte("example.com")

	if f.TestAndAdd(data1) {
		t.Errorf("First TestAndAdd should return false")
	}

	if !f.TestAndAdd(data1) {
		t.Errorf("Second TestAndAdd should return true")
	}
}

func TestFilterTest(t *testing.T) {
	f := NewFilter(1000, 0.01)
	data := []byte("example.com")

	f.Add(data)

	if !f.Test(data) {
		t.Errorf("Test should return true for added data")
	}
}

func TestFilterPersistence(t *testing.T) {
	tmpFile := "test_bloom.tmp"
	defer os.Remove(tmpFile)

	f1 := NewFilter(1000, 0.01)
	data := []byte("example.com")
	f1.Add(data)

	if err := f1.SaveToFile(tmpFile); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	f2 := NewFilter(1000, 0.01)
	if err := f2.LoadFromFile(tmpFile); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if !f2.Test(data) {
		t.Errorf("Test should return true for loaded data")
	}
}

func TestPersistenceManager(t *testing.T) {
	tmpFile := "test_pm.tmp"
	defer os.Remove(tmpFile)

	f := NewFilter(1000, 0.01)
	pm := NewPersistenceManager(f, 100*time.Millisecond)

	f.Add([]byte("test.com"))
	pm.StartPeriodicSave(tmpFile)

	time.Sleep(200 * time.Millisecond)

	info, err := os.Stat(tmpFile)
	if err != nil || info.Size() == 0 {
		t.Errorf("Saved file should exist and have content")
	}

	pm.Stop()
}
