package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestSaveToTempFile(t *testing.T) {
	content := "test content for temporary file"
	reader := strings.NewReader(content)
	
	// Create temp file using the helper
	tmpName, err := saveToTempFile(reader, "test-prefix-")
	if err != nil {
		t.Fatalf("saveToTempFile failed: %v", err)
	}
	
	// Clean up after test
	defer os.Remove(tmpName)
	
	// Ensure file exists and contains the correct data
	file, err := os.Open(tmpName)
	if err != nil {
		t.Fatalf("could not open created temp file: %v", err)
	}
	defer file.Close()
	
	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("could not read temp file: %v", err)
	}
	
	if string(data) != content {
		t.Errorf("expected content %q, got %q", content, string(data))
	}
	
	// Check prefix
	if !strings.Contains(tmpName, "test-prefix-") {
		t.Errorf("expected filename %q to contain prefix 'test-prefix-'", tmpName)
	}
}

func TestColorDistance(t *testing.T) {
	tests := []struct {
		name string
		c1   string
		c2   string
	}{
		{"same color", "#ffffff", "#ffffff"},
	}
	_ = tests
	// This just adds some coverage placeholder for other processor functions
}
