package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestApplicationRoutesAndSecurityHeaders(t *testing.T) {
	handler := newHandler()

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{name: "index", method: http.MethodGet, path: "/", wantStatus: http.StatusOK},
		{name: "health", method: http.MethodGet, path: "/healthz", wantStatus: http.StatusNoContent},
		{name: "not found", method: http.MethodGet, path: "/missing", wantStatus: http.StatusNotFound},
		{name: "processing endpoint rejects GET", method: http.MethodGet, path: "/process/img-to-pdf", wantStatus: http.StatusMethodNotAllowed},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(test.method, test.path, nil)
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)

			if res.Code != test.wantStatus {
				t.Fatalf("status = %d; esperado %d", res.Code, test.wantStatus)
			}
			if got := res.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Errorf("X-Content-Type-Options = %q", got)
			}
			if got := res.Header().Get("Content-Security-Policy"); got == "" {
				t.Error("Content-Security-Policy ausente")
			}
		})
	}
}

func TestMalformedMultipartDoesNotPanic(t *testing.T) {
	handler := newHandler()
	paths := []string{"/process/img-to-pdf", "/process/pdf-merge"}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, path, strings.NewReader("invalid multipart body"))
			req.Header.Set("Content-Type", "multipart/form-data; boundary=missing")
			res := httptest.NewRecorder()

			handler.ServeHTTP(res, req)
			if res.Code != http.StatusBadRequest {
				t.Fatalf("status = %d; esperado %d", res.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestMultipartLimit(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "large.bin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(bytes.Repeat([]byte("a"), 1024)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res := httptest.NewRecorder()
	if parseMultipartForm(res, req, 128) {
		t.Fatal("formulário acima do limite foi aceito")
	}
	if res.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d; esperado %d", res.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestTemplateUsesSafeDOMAndContainsSignature(t *testing.T) {
	source, err := templatesFS.ReadFile("templates/index.html")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	if strings.Contains(text, "custom.innerHTML") {
		t.Error("template ainda contém atribuição insegura a innerHTML")
	}
	if !strings.Contains(text, "<!-- Developed with care by FACRF - https://github.com/facrf -->") {
		t.Error("assinatura obrigatória ausente")
	}
}

func TestLocaleKeysAreConsistent(t *testing.T) {
	files, err := localesFS.ReadDir("locales")
	if err != nil {
		t.Fatal(err)
	}

	locales := make(map[string]map[string]string)
	for _, file := range files {
		data, err := localesFS.ReadFile("locales/" + file.Name())
		if err != nil {
			t.Fatal(err)
		}
		var values map[string]string
		if err := json.Unmarshal(data, &values); err != nil {
			t.Fatalf("locale %s inválido: %v", file.Name(), err)
		}
		locales[file.Name()] = values
	}

	reference := locales["pt.json"]
	for name, values := range locales {
		for key := range reference {
			if values[key] == "" {
				t.Errorf("locale %s sem a chave %q", name, key)
			}
		}
		for key := range values {
			if _, ok := reference[key]; !ok {
				t.Errorf("locale %s contém chave extra %q", name, key)
			}
		}
	}
}

func TestImageDimensionLimits(t *testing.T) {
	tests := []struct {
		name        string
		width       int
		height      int
		shouldError bool
	}{
		{name: "valid", width: 4000, height: 4000},
		{name: "zero", width: 0, height: 100, shouldError: true},
		{name: "dimension", width: maxImageDimension + 1, height: 1, shouldError: true},
		{name: "pixels", width: 5000, height: 5000, shouldError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateImageDimensions(test.width, test.height)
			if (err != nil) != test.shouldError {
				t.Fatalf("erro = %v; shouldError = %v", err, test.shouldError)
			}
		})
	}
}

func TestConcurrentJobLimit(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	firstDone := make(chan struct{})

	handler := limitConcurrentJobs(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusNoContent)
	}), 1)

	go func() {
		defer close(firstDone)
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/process/test", nil))
	}()
	<-started

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/process/test", nil))
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d; esperado %d", res.Code, http.StatusServiceUnavailable)
	}
	if res.Header().Get("Retry-After") == "" {
		t.Error("Retry-After ausente")
	}

	close(release)
	<-firstDone
}
