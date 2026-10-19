package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func pdfRequest(t *testing.T, operation string, pdf []byte, fields map[string]string) *http.Request {
	t.Helper()
	body, contentType := createMultipartBody(t, "pdf", "document.pdf", pdf, fields)
	r := httptest.NewRequest(http.MethodPost, "/process/"+operation, body)
	r.Header.Set("Content-Type", contentType)
	return r
}

func TestRotatePageSelection(t *testing.T) {
	for _, selection := range []string{"1,3", "1-2,!2,3"} {
		t.Run(selection, func(t *testing.T) {
			r := pdfRequest(t, "pdf-rotate", generatedPDF(t, 3), map[string]string{"degrees": "90", "pages": selection})
			w := httptest.NewRecorder()
			handlePdfRotate(w, r)
			if w.Code != http.StatusOK {
				t.Fatalf("status=%d: %s", w.Code, w.Body.String())
			}
			ctx, err := api.ReadAndValidate(bytes.NewReader(w.Body.Bytes()), nil)
			if err != nil {
				t.Fatal(err)
			}
			for page, rotation := range map[int]int{1: 90, 2: 0, 3: 90} {
				_, _, attrs, err := ctx.PageDict(page, false)
				if err != nil || attrs.Rotate != rotation {
					t.Fatalf("page=%d rotation=%d want=%d err=%v", page, attrs.Rotate, rotation, err)
				}
			}
		})
	}
}

func TestExtractedImageNamesAreSafeAndUnique(t *testing.T) {
	ctx, err := api.ReadAndValidate(bytes.NewReader(generatedPDF(t, 3)), nil)
	if err != nil {
		t.Fatal(err)
	}
	page, _, _, err := ctx.PageDict(1, false)
	if err != nil {
		t.Fatal(err)
	}
	resources, err := ctx.DereferenceDict(page["Resources"])
	if err != nil {
		t.Fatal(err)
	}
	objects, err := ctx.DereferenceDict(resources["XObject"])
	if err != nil {
		t.Fatal(err)
	}
	for name, value := range objects {
		delete(objects, name)
		objects["../outside"] = value
		objects["second-reference-to-same-object"] = value
		break
	}
	var input bytes.Buffer
	if err := api.WriteContext(ctx, &input); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	handlePdfToImg(w, pdfRequest(t, "pdf-to-img", input.Bytes(), nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d: %s", w.Code, w.Body.String())
	}
	archive, err := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if len(archive.File) < 3 {
		t.Fatalf("entries=%d", len(archive.File))
	}
	seen := map[string]bool{}
	for _, entry := range archive.File {
		if seen[entry.Name] || strings.ContainsAny(entry.Name, "/\\") || strings.Contains(entry.Name, "..") {
			t.Fatalf("unsafe or duplicate ZIP name: %q", entry.Name)
		}
		seen[entry.Name] = true
	}
}

func TestAllPDFOperationsRejectTooManyPages(t *testing.T) {
	input := generatedPDF(t, maxPDFPages+1)
	for operation, handler := range pdfHandlers() {
		if operation == "img-to-pdf" || operation == "pdf-merge" {
			continue
		}
		t.Run(operation, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler(w, pdfRequest(t, operation, input, map[string]string{"degrees": "90", "password": "test-password", "text": "Sample"}))
			if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "50 páginas") {
				t.Fatalf("status=%d: %s", w.Code, w.Body.String())
			}
		})
	}
	files := map[string][]byte{"a.pdf": generatedPDF(t, 26), "b.pdf": generatedPDF(t, 25)}
	body, contentType := createFakePDFBody(t, "pdfs", files, nil)
	r := httptest.NewRequest(http.MethodPost, "/process/pdf-merge", body)
	r.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	handlePdfMerge(w, r)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "no máximo 50 páginas no total") {
		t.Fatalf("aggregate page limit: status=%d: %s", w.Code, w.Body.String())
	}
}

func TestPDFWorkerHTTPResults(t *testing.T) {
	handler := newHandler()
	for _, operation := range []string{"pdf-merge", "img-to-pdf"} {
		t.Run(operation, func(t *testing.T) {
			var body *bytes.Buffer
			var contentType string
			if operation == "pdf-merge" {
				body, contentType = createFakePDFBody(t, "pdfs", map[string][]byte{
					"first.pdf": generatedPDF(t, 1), "second.pdf": generatedPDF(t, 1),
				}, nil)
			} else {
				var raw bytes.Buffer
				writer := multipart.NewWriter(&raw)
				for i := 0; i < 2; i++ {
					part, err := writer.CreateFormFile("images", fmt.Sprintf("image-%d.png", i))
					if err != nil {
						t.Fatal(err)
					}
					if _, err := part.Write(createTestImage(t, 2, 2, "png").Bytes()); err != nil {
						t.Fatal(err)
					}
				}
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
				body, contentType = &raw, writer.FormDataContentType()
			}
			r := httptest.NewRequest(http.MethodPost, "/process/"+operation, body)
			r.Header.Set("Content-Type", contentType)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != http.StatusOK {
				t.Fatalf("status=%d: %s", w.Code, w.Body.String())
			}
			count, err := api.PageCount(bytes.NewReader(w.Body.Bytes()), nil)
			if err != nil || count != 2 {
				t.Fatalf("pages=%d err=%v", count, err)
			}
		})
	}
	for _, operation := range []string{"pdf-rotate", "pdf-watermark", "pdf-optimize", "pdf-protect", "pdf-split", "pdf-to-img"} {
		t.Run(operation, func(t *testing.T) {
			r := pdfRequest(t, operation, generatedPDF(t, 2), map[string]string{"degrees": "90", "pages": "1,2", "password": "test-password", "text": "Sample"})
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != http.StatusOK {
				t.Fatalf("status=%d: %s", w.Code, w.Body.String())
			}
			if w.Header().Get("X-Content-Type-Options") != "nosniff" || w.Header().Get("Content-Disposition") == "" {
				t.Fatal("missing download/security headers")
			}
			if strings.HasPrefix(w.Header().Get("Content-Type"), "application/zip") {
				if _, err := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len())); err != nil {
					t.Fatal(err)
				}
				return
			}
			conf := model.NewDefaultConfiguration()
			if operation == "pdf-protect" {
				conf.UserPW = "test-password"
			}
			count, err := api.PageCount(bytes.NewReader(w.Body.Bytes()), conf)
			if err != nil || count != 2 {
				t.Fatalf("pages=%d err=%v", count, err)
			}
		})
	}
}

func TestPDFWorkerCancellationAndCleanup(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("TMPDIR", directory)
	for _, cancelled := range []bool{true, false} {
		r := pdfRequest(t, "pdf-rotate", generatedPDF(t, 1), map[string]string{"degrees": "90"})
		if cancelled {
			ctx, cancel := context.WithCancel(r.Context())
			cancel()
			r = r.WithContext(ctx)
		}
		w := httptest.NewRecorder()
		isolatePDF("pdf-rotate", time.Nanosecond).ServeHTTP(w, r)
		want := http.StatusGatewayTimeout
		if cancelled {
			want = http.StatusRequestTimeout
		}
		if w.Code != want {
			t.Fatalf("cancelled=%v status=%d want=%d: %s", cancelled, w.Code, want, w.Body.String())
		}
		entries, err := os.ReadDir(directory)
		if err != nil || len(entries) != 0 {
			t.Fatalf("temporary files retained: %v err=%v", entries, err)
		}
	}
}

func TestPDFWriterDetectsIgnoredOutputErrors(t *testing.T) {
	var output bytes.Buffer
	err := writeLimitedPDF(&output, func(w io.Writer) error {
		bounded := w.(*limitedWriter)
		bounded.remaining = 2
		_, _ = w.Write([]byte("oversized")) // Simulate a library ignoring a final flush error.
		return nil
	})
	if err == nil || output.Len() != 0 {
		t.Fatalf("ignored bounded output failure: %v", err)
	}
}

func TestWorkerOutputLimitCancelsProcess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bounded := &limitedWriter{writer: io.Discard, remaining: 2}
	writer := &cancelOnErrorWriter{writer: bounded, cancel: cancel}
	if _, err := writer.Write([]byte("oversized")); err == nil || ctx.Err() == nil {
		t.Fatal("output limit failed to cancel the worker")
	}
}

func TestRealRenderers(t *testing.T) {
	for _, program := range []string{"pdftoppm", "rsvg-convert"} {
		if _, err := exec.LookPath(program); err != nil {
			t.Skipf("%s unavailable; CI and the Docker image run this integration test", program)
		}
	}
	t.Run("PDF", func(t *testing.T) {
		w := httptest.NewRecorder()
		newHandler().ServeHTTP(w, pdfRequest(t, "pdf-rasterize", generatedPDF(t, 2), nil))
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d: %s", w.Code, w.Body.String())
		}
		archive, err := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len()))
		if err != nil || len(archive.File) != 2 {
			t.Fatalf("invalid rasterization archive: %v", err)
		}
	})
	t.Run("SVG", func(t *testing.T) {
		body, contentType := createMultipartBody(t, "svg", "sample.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="20" height="10"><rect width="20" height="10" fill="red"/></svg>`), nil)
		r := httptest.NewRequest(http.MethodPost, "/process/svg-to-img", body)
		r.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()
		newHandler().ServeHTTP(w, r)
		if w.Code != http.StatusOK || !bytes.HasPrefix(w.Body.Bytes(), []byte("\x89PNG\r\n\x1a\n")) {
			t.Fatalf("invalid SVG conversion: status=%d", w.Code)
		}
	})
}
