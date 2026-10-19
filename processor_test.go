package main

import (
	"os"
	"testing"
)

// The test executable also hosts the real worker entry point. HTTP integration
// tests therefore exercise the same process boundary as the production binary.
func TestMain(m *testing.M) {
	if len(os.Args) == 2 && os.Args[1] == "--pdf-worker" {
		os.Exit(pdfWorkerMain())
	}
	os.Exit(m.Run())
}
