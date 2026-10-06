package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func TestSVGDimensions(t *testing.T) {
	tests := []struct {
		name, source, requested string
		width, height           int
		invalid                 bool
	}{
		{"viewBox", `<svg viewBox="0 0 200 100"/>`, "", 200, 100, false},
		{"scaled", `<svg width="200" height="100"/>`, "100", 100, 50, false},
		{"units", `<svg width="1in" height="72pt"/>`, "", 96, 96, false},
		{"natural huge", `<svg width="99999999" height="100"/>`, "", 0, 0, true},
		{"scaled huge height", `<svg viewBox="0 0 1 99999999"/>`, "100", 0, 0, true},
		{"pixel budget", `<svg width="5000" height="5000"/>`, "", 0, 0, true},
		{"NaN", `<svg width="NaN"/>`, "", 0, 0, true},
		{"negative viewBox", `<svg viewBox="0 0 -1 5"/>`, "", 0, 0, true},
		{"invalid request", `<svg/>`, "-1", 0, 0, true},
		{"not SVG", `<html/>`, "", 0, 0, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w, h, err := svgDimensions(strings.NewReader(test.source), test.requested)
			if (err != nil) != test.invalid {
				t.Fatalf("error=%v", err)
			}
			if !test.invalid && (w != test.width || h != test.height) {
				t.Fatalf("got %dx%d", w, h)
			}
		})
	}
}

func TestLimitedWriter(t *testing.T) {
	var output bytes.Buffer
	writer := &limitedWriter{writer: &output, remaining: 4}
	if _, err := writer.Write([]byte("abcd")); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("e")); err == nil {
		t.Fatal("accepted excess output")
	}
	if output.String() != "abcd" {
		t.Fatal("wrote excess output")
	}
	sentinel := errors.New("disk full")
	writer = &limitedWriter{writer: failingWriter{sentinel}, remaining: 10}
	if _, err := writer.Write([]byte("a")); !errors.Is(err, sentinel) {
		t.Fatalf("lost write error: %v", err)
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write(p []byte) (int, error) { return 0, w.err }

func generatedPDF(t *testing.T, pages int) []byte {
	t.Helper()
	readers := make([]io.Reader, pages)
	img := createTestImage(t, 1, 1, "png").Bytes()
	for i := range readers {
		readers[i] = bytes.NewReader(img)
	}
	var pdf bytes.Buffer
	if err := api.ImportImages(nil, &pdf, readers, nil, model.NewDefaultConfiguration()); err != nil {
		t.Fatal(err)
	}
	return pdf.Bytes()
}
func TestPDFPageLimit(t *testing.T) {
	for _, pages := range []int{1, maxPDFPages, maxPDFPages + 1} {
		reader := bytes.NewReader(generatedPDF(t, pages))
		count, err := validatePDFPages(reader)
		if pages > maxPDFPages {
			if err == nil {
				t.Fatal("accepted too many pages")
			}
			continue
		}
		if err != nil || count != pages {
			t.Fatalf("count=%d err=%v", count, err)
		}
		offset, _ := reader.Seek(0, io.SeekCurrent)
		if offset != 0 {
			t.Fatal("reader not rewound")
		}
	}
}

func TestBatchStripDuplicateNames(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	img := createTestImage(t, 10, 10, "png").Bytes()
	for i := 0; i < 2; i++ {
		part, err := writer.CreateFormFile("images", "same.png")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(img); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("POST", "/process/img-exif-strip", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	handleImgExifStrip(response, request)
	if response.Code != 200 {
		t.Fatalf("status=%d: %s", response.Code, response.Body.String())
	}
	archive, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if len(archive.File) != 2 || archive.File[0].Name == archive.File[1].Name {
		t.Fatal("duplicate names")
	}
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil || !bytes.Equal(data, img) {
			t.Fatalf("corrupt ZIP entry: %v", err)
		}
	}
}

func TestHealthcheck(t *testing.T) {
	bin := t.TempDir()
	for _, name := range []string{"pdftoppm", "rsvg-convert"} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin)
	server := httptest.NewServer(newHandler())
	defer server.Close()
	parsed, _ := url.Parse(server.URL)
	if err := checkHealth(parsed.Port()); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing"))
	if err := checkHealth(parsed.Port()); err == nil {
		t.Fatal("unwritable temp accepted")
	}
	t.Setenv("PATH", t.TempDir())
	if err := checkHealth(parsed.Port()); err == nil {
		t.Fatal("missing renderer accepted")
	}
}

func TestSVGRejectsHugeDimensionsBeforeRendering(t *testing.T) {
	body, contentType := createMultipartBody(t, "svg", "huge.svg", []byte(`<svg width="8000" height="8000"/>`), nil)
	request := httptest.NewRequest(http.MethodPost, "/process/svg-to-img", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()
	handleSvgToImg(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", response.Code)
	}
}

func TestSplitProducesOnePagePerEntry(t *testing.T) {
	body, contentType := createMultipartBody(t, "pdf", "three.pdf", generatedPDF(t, 3), nil)
	request := httptest.NewRequest(http.MethodPost, "/process/pdf-split", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()
	handlePdfSplit(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d: %s", response.Code, response.Body.String())
	}
	archive, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if len(archive.File) != 3 {
		t.Fatalf("entries=%d", len(archive.File))
	}
	for _, entry := range archive.File {
		reader, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		count, err := validatePDFPages(bytes.NewReader(data))
		if err != nil || count != 1 {
			t.Fatalf("count=%d err=%v", count, err)
		}
	}
}
