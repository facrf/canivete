package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestSafeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Normal filename", "photo.jpg", "photo.jpg"},
		{"Path traversal attempts", "../../../etc/passwd", "passwd"},
		{"Backslash paths", "folder\\file.txt", "file.txt"},
		{"Control characters", "file\x00name.txt", "filename.txt"},
		{"Empty string", "", "download"},
		{"Just a dot", ".", "download"},
		{"Whitespace only", "   ", "download"},
		{"Tab and newline chars", "file\t\nname.txt", "filename.txt"},
		{"Windows path", "C:\\Users\\file.txt", "file.txt"},
		{"Unicode filename", "imagem_ação.png", "imagem_ação.png"},
		{"Very long filename", strings.Repeat("a", 150) + ".txt", strings.Repeat("a", 120)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := safeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("safeFilename(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateImageDimensions(t *testing.T) {
	tests := []struct {
		name    string
		width   int
		height  int
		wantErr bool
	}{
		{"Valid small", 100, 100, false},
		{"Valid max", 8000, 2500, false},
		{"Zero width", 0, 100, true},
		{"Zero height", 100, 0, true},
		{"Negative", -1, 100, true},
		{"Over max dimension", 8001, 1, true},
		{"Over max pixels", 5000, 5000, true},
		{"Exactly at max dimension", 8000, 2500, false},
		{"Exactly at max pixels", 5000, 4000, false},
		{"Both zero", 0, 0, true},
		{"Huge numbers", 2147483647, 1, true},
		{"1x1", 1, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateImageDimensions(tt.width, tt.height)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateImageDimensions(%d, %d) error = %v, wantErr %v", tt.width, tt.height, err, tt.wantErr)
			}
		})
	}
}

type errReader struct{}

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

func TestSaveToTempFile(t *testing.T) {
	t.Run("Normal content", func(t *testing.T) {
		content := "test content"
		prefix := "test_normal_"

		filepath, err := saveToTempFile(strings.NewReader(content), prefix)
		if err != nil {
			t.Fatalf("saveToTempFile failed: %v", err)
		}

		t.Cleanup(func() { os.Remove(filepath) })

		if !strings.Contains(filepath, prefix) {
			t.Errorf("filename %q does not contain prefix %q", filepath, prefix)
		}

		savedContent, err := os.ReadFile(filepath)
		if err != nil {
			t.Fatalf("failed to read saved file: %v", err)
		}
		if string(savedContent) != content {
			t.Errorf("saved content = %q, want %q", string(savedContent), content)
		}
	})

	t.Run("Empty reader", func(t *testing.T) {
		filepath, err := saveToTempFile(strings.NewReader(""), "test_empty_")
		if err != nil {
			t.Fatalf("saveToTempFile failed: %v", err)
		}
		t.Cleanup(func() { os.Remove(filepath) })

		info, err := os.Stat(filepath)
		if err != nil {
			t.Fatalf("failed to stat file: %v", err)
		}
		if info.Size() != 0 {
			t.Errorf("expected empty file, got size %d", info.Size())
		}
	})

	t.Run("Error reader", func(t *testing.T) {
		filepath, err := saveToTempFile(errReader{}, "test_err_")
		if err == nil {
			t.Error("expected error with errReader, got nil")
		}
		if filepath != "" {
			t.Errorf("expected empty filepath, got %q", filepath)
			t.Cleanup(func() { os.Remove(filepath) })
		}
	})
}

func TestParseMultipartForm(t *testing.T) {
	t.Run("GET request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		result := parseMultipartForm(w, req, 1024)
		if result {
			t.Error("parseMultipartForm returned true for GET request")
		}
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("Valid POST with multipart", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("test", "value")
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		result := parseMultipartForm(w, req, 1024)
		if !result {
			t.Error("parseMultipartForm returned false for valid POST")
		}
	})

	t.Run("POST exceeding maxBytes", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("test", strings.Repeat("a", 2048))
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		result := parseMultipartForm(w, req, 1024)
		if result {
			t.Error("parseMultipartForm returned true for oversized POST")
		}
		if w.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("expected status %d, got %d", http.StatusRequestEntityTooLarge, w.Code)
		}
	})

	t.Run("POST with invalid multipart", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not multipart data"))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=invalid")
		w := httptest.NewRecorder()

		result := parseMultipartForm(w, req, 1024)
		if result {
			t.Error("parseMultipartForm returned true for invalid multipart data")
		}
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("POST with wrong content-type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("test=value"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		result := parseMultipartForm(w, req, 1024)
		if result {
			t.Error("parseMultipartForm returned true for wrong content type")
		}
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestSetDownloadHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	setDownloadHeaders(w, "image/jpeg", "../evil/path/image.jpg")

	if ct := w.Header().Get("Content-Type"); ct != "image/jpeg" {
		t.Errorf("Content-Type = %q, want %q", ct, "image/jpeg")
	}

	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "attachment") {
		t.Errorf("Content-Disposition %q does not contain 'attachment'", cd)
	}
	if !strings.Contains(cd, "filename=") || !strings.Contains(cd, "image.jpg") {
		t.Errorf("Content-Disposition %q does not contain correct filename", cd)
	}
}

func TestColorDistance(t *testing.T) {
	tests := []struct {
		name string
		c1   color.Color
		c2   color.Color
		want float64
	}{
		{"Same color", color.RGBA{255, 0, 0, 255}, color.RGBA{255, 0, 0, 255}, 0.0},
		{"Black vs White", color.RGBA{0, 0, 0, 255}, color.RGBA{255, 255, 255, 255}, 441.6729559300637},
		{"Red vs Green", color.RGBA{255, 0, 0, 255}, color.RGBA{0, 255, 0, 255}, 360.62445840513925},
		// colorDistance ignores alpha — it only compares RGB channels.
		// Two transparent colors with different RGB values still have RGB distance.
		{"Transparent same RGB", color.RGBA{255, 0, 0, 0}, color.RGBA{255, 0, 0, 128}, 0.0},
		{"Transparent different RGB", color.RGBA{255, 0, 0, 0}, color.RGBA{0, 255, 0, 0}, 360.62445840513925},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := colorDistance(tt.c1, tt.c2)
			// Allow small float differences
			if diff := got - tt.want; diff < -0.01 || diff > 0.01 {
				t.Errorf("colorDistance() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServerPort(t *testing.T) {
	t.Run("Empty PORT", func(t *testing.T) {
		t.Setenv("PORT", "")
		if got := serverPort(); got != "7001" {
			t.Errorf("serverPort() = %v, want 7001", got)
		}
	})

	t.Run("Valid PORT", func(t *testing.T) {
		t.Setenv("PORT", "8080")
		if got := serverPort(); got != "8080" {
			t.Errorf("serverPort() = %v, want 8080", got)
		}
	})
}

func createTestPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, img); err != nil {
		t.Fatalf("failed to create test PNG: %v", err)
	}
	return buf.Bytes()
}

func TestWriteICO(t *testing.T) {
	t.Run("Valid single 16x16 PNG", func(t *testing.T) {
		img := createTestPNG(t, 16, 16)
		buf := new(bytes.Buffer)

		err := writeICO(buf, [][]byte{img})
		if err != nil {
			t.Fatalf("writeICO failed: %v", err)
		}

		out := buf.Bytes()
		if len(out) < 6 {
			t.Fatalf("output too small for ICO header")
		}

		// Check ICO header (0, 1, count)
		if out[0] != 0 || out[1] != 0 {
			t.Errorf("invalid ICO reserved bytes: %v %v", out[0], out[1])
		}
		if out[2] != 1 || out[3] != 0 {
			t.Errorf("invalid ICO type bytes: %v %v", out[2], out[3])
		}
		count := binary.LittleEndian.Uint16(out[4:6])
		if count != 1 {
			t.Errorf("ICO count = %d, want 1", count)
		}
	})

	t.Run("Valid multiple sizes", func(t *testing.T) {
		img1 := createTestPNG(t, 16, 16)
		img2 := createTestPNG(t, 32, 32)
		buf := new(bytes.Buffer)

		err := writeICO(buf, [][]byte{img1, img2})
		if err != nil {
			t.Fatalf("writeICO failed: %v", err)
		}

		out := buf.Bytes()
		count := binary.LittleEndian.Uint16(out[4:6])
		if count != 2 {
			t.Errorf("ICO count = %d, want 2", count)
		}
	})

	t.Run("Empty images slice", func(t *testing.T) {
		buf := new(bytes.Buffer)
		err := writeICO(buf, [][]byte{})
		if err == nil {
			t.Error("writeICO expected error for empty images slice, got nil")
		}
	})

	t.Run("Invalid PNG data", func(t *testing.T) {
		buf := new(bytes.Buffer)
		err := writeICO(buf, [][]byte{[]byte("invalid data")})
		if err == nil {
			t.Error("writeICO expected error for invalid PNG data, got nil")
		}
	})
}

func TestRecoverMiddleware(t *testing.T) {
	t.Run("Normal handler", func(t *testing.T) {
		handler := recoverMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("Panicking handler", func(t *testing.T) {
		handler := recoverMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})
}

func TestSecurityHeaders(t *testing.T) {
	handler := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	tests := []struct {
		header string
		want   string
	}{
		{"Content-Security-Policy", "default-src 'self'; img-src 'self' blob: data:; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'"},
		{"Permissions-Policy", "camera=(), microphone=(), geolocation=()"},
		{"Referrer-Policy", "no-referrer"},
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			if got := w.Header().Get(tt.header); got != tt.want {
				t.Errorf("%s = %q, want %q", tt.header, got, tt.want)
			}
		})
	}
}

func TestLimitConcurrentJobs(t *testing.T) {
	t.Run("Limit zero", func(t *testing.T) {
		handler := limitConcurrentJobs(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}), 0)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})

	t.Run("Limit two", func(t *testing.T) {
		ch1 := make(chan struct{})
		ch2 := make(chan struct{})

		w1 := httptest.NewRecorder()
		w2 := httptest.NewRecorder()
		w3 := httptest.NewRecorder()

		// Send 3 requests. 2 should succeed, 1 should fail.
		blockingHandler := limitConcurrentJobs(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/1" || r.URL.Path == "/2" {
				ch1 <- struct{}{}
				<-ch2
			}
			w.WriteHeader(http.StatusOK)
		}), 2)

		go blockingHandler.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/1", nil))
		go blockingHandler.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/2", nil))

		// Wait for both to enter
		<-ch1
		<-ch1

		// 3rd should fail immediately
		blockingHandler.ServeHTTP(w3, httptest.NewRequest(http.MethodGet, "/3", nil))

		if w3.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 3rd request to fail with 503, got %d", w3.Code)
		}

		// Release blocked requests
		ch2 <- struct{}{}
		ch2 <- struct{}{}
	})
}
