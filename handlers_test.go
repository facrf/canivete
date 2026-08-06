package main

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func createTestImage(t *testing.T, width, height int, format string) *bytes.Buffer {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	buf := new(bytes.Buffer)
	var err error
	if format == "jpeg" || format == "jpg" {
		err = jpeg.Encode(buf, img, nil)
	} else if format == "png" {
		err = png.Encode(buf, img)
	} else {
		t.Fatalf("unsupported test image format: %s", format)
	}
	if err != nil {
		t.Fatalf("failed to encode test image: %v", err)
	}
	return buf
}

func createMultipartBody(t *testing.T, fieldName, filename string, content []byte, extraFields map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	if content != nil {
		part, err := writer.CreateFormFile(fieldName, filename)
		if err != nil {
			t.Fatalf("failed to create form file: %v", err)
		}
		if _, err := part.Write(content); err != nil {
			t.Fatalf("failed to write content: %v", err)
		}
	}

	for k, v := range extraFields {
		if err := writer.WriteField(k, v); err != nil {
			t.Fatalf("failed to write field %s: %v", k, err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	return body, writer.FormDataContentType()
}

// PDF Helper para testar falhas de PDF enviando bytes falsos
func createFakePDFBody(t *testing.T, fieldName string, files map[string][]byte, extraFields map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	for filename, content := range files {
		part, err := writer.CreateFormFile(fieldName, filename)
		if err != nil {
			t.Fatalf("failed to create form file: %v", err)
		}
		if _, err := part.Write(content); err != nil {
			t.Fatalf("failed to write content: %v", err)
		}
	}

	for k, v := range extraFields {
		if err := writer.WriteField(k, v); err != nil {
			t.Fatalf("failed to write field %s: %v", k, err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	return body, writer.FormDataContentType()
}

func TestHandleResize(t *testing.T) {
	handler := http.HandlerFunc(handleResize)
	validImg := createTestImage(t, 10, 10, "png").Bytes()

	tests := []struct {
		name        string
		method      string
		fileContent []byte
		fileName    string
		fields      map[string]string
		wantStatus  int
		wantType    string
		wantDisp    string
	}{
		{
			name:        "Valid image with valid dimensions",
			method:      http.MethodPost,
			fileContent: validImg,
			fileName:    "test.png",
			fields:      map[string]string{"width": "5", "height": "5"},
			wantStatus:  http.StatusOK,
			wantType:    "image/png",
			wantDisp:    `attachment; filename=resized.png`,
		},
		{
			name:        "Missing dimensions",
			method:      http.MethodPost,
			fileContent: validImg,
			fileName:    "test.png",
			fields:      nil,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "Invalid dimensions (zero)",
			method:      http.MethodPost,
			fileContent: validImg,
			fileName:    "test.png",
			fields:      map[string]string{"width": "0", "height": "0"},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "Missing image file",
			method:      http.MethodPost,
			fileContent: nil,
			fields:      map[string]string{"width": "5", "height": "5"},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "Invalid image data",
			method:      http.MethodPost,
			fileContent: []byte("not an image"),
			fileName:    "test.txt",
			fields:      map[string]string{"width": "5", "height": "5"},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "Non-POST method",
			method:      http.MethodGet,
			fileContent: validImg,
			fileName:    "test.png",
			fields:      map[string]string{"width": "5", "height": "5"},
			wantStatus:  http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, contentType := createMultipartBody(t, "image", tt.fileName, tt.fileContent, tt.fields)
			req := httptest.NewRequest(tt.method, "/resize", body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if tt.wantStatus == http.StatusOK {
				if ct := rec.Header().Get("Content-Type"); ct != tt.wantType {
					t.Errorf("expected Content-Type %q, got %q", tt.wantType, ct)
				}
				if cd := rec.Header().Get("Content-Disposition"); cd != tt.wantDisp {
					t.Errorf("expected Content-Disposition %q, got %q", tt.wantDisp, cd)
				}
			}
		})
	}
}

func TestHandleCrop(t *testing.T) {
	handler := http.HandlerFunc(handleCrop)
	validImg := createTestImage(t, 20, 20, "png").Bytes()

	tests := []struct {
		name        string
		fileContent []byte
		fields      map[string]string
		wantStatus  int
	}{
		{
			name:        "Valid crop within bounds",
			fileContent: validImg,
			fields:      map[string]string{"x": "5", "y": "5", "width": "10", "height": "10", "size": "0"},
			wantStatus:  http.StatusOK,
		},
		{
			name:        "Crop area outside image bounds",
			fileContent: validImg,
			fields:      map[string]string{"x": "15", "y": "15", "width": "10", "height": "10", "size": "0"},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "Invalid/missing parameters",
			fileContent: validImg,
			fields:      map[string]string{"x": "5"}, // missing y, width, height
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "NaN/Inf values",
			fileContent: validImg,
			fields:      map[string]string{"x": "NaN", "y": "5", "width": "10", "height": "10", "size": "0"},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "Negative values",
			fileContent: validImg,
			fields:      map[string]string{"x": "-5", "y": "5", "width": "10", "height": "10", "size": "0"},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "With finalSize resize",
			fileContent: validImg,
			fields:      map[string]string{"x": "5", "y": "5", "width": "5", "height": "5", "size": "50"},
			wantStatus:  http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, contentType := createMultipartBody(t, "image", "test.png", tt.fileContent, tt.fields)
			req := httptest.NewRequest(http.MethodPost, "/crop", body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if tt.wantStatus == http.StatusOK {
				if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
					t.Errorf("expected Content-Type image/png, got %q", ct)
				}
				if cd := rec.Header().Get("Content-Disposition"); cd != `attachment; filename=cropped.png` {
					t.Errorf("expected Content-Disposition, got %q", cd)
				}
			}
		})
	}
}

func TestHandleConvert(t *testing.T) {
	handler := http.HandlerFunc(handleConvert)
	validImg := createTestImage(t, 10, 10, "png").Bytes()

	tests := []struct {
		name        string
		fileContent []byte
		fields      map[string]string
		wantStatus  int
		wantType    string
		wantDisp    string
	}{
		{
			name:        "Convert to PNG",
			fileContent: validImg,
			fields:      map[string]string{"format": "png", "quality": "80"},
			wantStatus:  http.StatusOK,
			wantType:    "image/png",
			wantDisp:    `attachment; filename=converted.png`,
		},
		{
			name:        "Convert to JPG",
			fileContent: validImg,
			fields:      map[string]string{"format": "jpg", "quality": "80"},
			wantStatus:  http.StatusOK,
			wantType:    "image/jpeg",
			wantDisp:    `attachment; filename=converted.jpg`,
		},
		{
			name:        "Convert to BMP",
			fileContent: validImg,
			fields:      map[string]string{"format": "bmp"},
			wantStatus:  http.StatusOK,
			wantType:    "image/bmp",
			wantDisp:    `attachment; filename=converted.bmp`,
		},
		{
			name:        "Unsupported format",
			fileContent: validImg,
			fields:      map[string]string{"format": "webp"},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "Missing format",
			fileContent: validImg,
			fields:      nil,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "Quality parameter clamping",
			fileContent: validImg,
			fields:      map[string]string{"format": "jpg", "quality": "150"}, // should clamp to 80 or valid range
			wantStatus:  http.StatusOK,
			wantType:    "image/jpeg",
			wantDisp:    `attachment; filename=converted.jpg`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, contentType := createMultipartBody(t, "image", "test.png", tt.fileContent, tt.fields)
			req := httptest.NewRequest(http.MethodPost, "/convert", body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if tt.wantStatus == http.StatusOK {
				if ct := rec.Header().Get("Content-Type"); ct != tt.wantType {
					t.Errorf("expected Content-Type %q, got %q", tt.wantType, ct)
				}
				if cd := rec.Header().Get("Content-Disposition"); cd != tt.wantDisp {
					t.Errorf("expected Content-Disposition %q, got %q", tt.wantDisp, cd)
				}
			}
		})
	}
}

func TestHandleRemoveBg(t *testing.T) {
	handler := http.HandlerFunc(handleRemoveBg)
	validImg := createTestImage(t, 10, 10, "png").Bytes()

	tests := []struct {
		name        string
		fileContent []byte
		fields      map[string]string
		wantStatus  int
	}{
		{
			name:        "Valid image",
			fileContent: validImg,
			fields:      map[string]string{"tolerance": "10"},
			wantStatus:  http.StatusOK,
		},
		{
			name:        "Custom tolerance",
			fileContent: validImg,
			fields:      map[string]string{"tolerance": "50"},
			wantStatus:  http.StatusOK,
		},
		{
			name:        "Tolerance out of range (>100)",
			fileContent: validImg,
			fields:      map[string]string{"tolerance": "150"},
			wantStatus:  http.StatusOK,
		},
		{
			name:        "Tolerance out of range (<=0)",
			fileContent: validImg,
			fields:      map[string]string{"tolerance": "-10"},
			wantStatus:  http.StatusOK,
		},
		{
			name:        "Missing image",
			fileContent: nil,
			fields:      map[string]string{"tolerance": "10"},
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, contentType := createMultipartBody(t, "image", "test.png", tt.fileContent, tt.fields)
			req := httptest.NewRequest(http.MethodPost, "/removebg", body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestHandleImgCompress(t *testing.T) {
	handler := http.HandlerFunc(handleImgCompress)
	validImg := createTestImage(t, 10, 10, "png").Bytes()

	tests := []struct {
		name        string
		fileContent []byte
		fields      map[string]string
		wantStatus  int
	}{
		{
			name:        "Valid image with quality",
			fileContent: validImg,
			fields:      map[string]string{"quality": "80"},
			wantStatus:  http.StatusOK,
		},
		{
			name:        "Quality clamping (0)",
			fileContent: validImg,
			fields:      map[string]string{"quality": "0"},
			wantStatus:  http.StatusOK,
		},
		{
			name:        "Quality clamping (200)",
			fileContent: validImg,
			fields:      map[string]string{"quality": "200"},
			wantStatus:  http.StatusOK,
		},
		{
			name:        "Missing image",
			fileContent: nil,
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, contentType := createMultipartBody(t, "image", "test.png", tt.fileContent, tt.fields)
			req := httptest.NewRequest(http.MethodPost, "/compress", body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if tt.wantStatus == http.StatusOK {
				if ct := rec.Header().Get("Content-Type"); ct != "image/jpeg" {
					t.Errorf("expected Content-Type image/jpeg, got %q", ct)
				}
			}
		})
	}
}

func TestHandleImgExifStrip(t *testing.T) {
	handler := http.HandlerFunc(handleImgExifStrip)
	validJpeg := createTestImage(t, 10, 10, "jpeg").Bytes()
	validPng := createTestImage(t, 10, 10, "png").Bytes()

	tests := []struct {
		name        string
		fileContent []byte
		fileName    string
		wantStatus  int
		wantType    string
	}{
		{
			name:        "JPEG image",
			fileContent: validJpeg,
			fileName:    "test.jpg",
			wantStatus:  http.StatusOK,
			wantType:    "image/jpeg",
		},
		{
			name:        "PNG image",
			fileContent: validPng,
			fileName:    "test.png",
			wantStatus:  http.StatusOK,
			wantType:    "image/png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, contentType := createMultipartBody(t, "image", tt.fileName, tt.fileContent, nil)
			req := httptest.NewRequest(http.MethodPost, "/exifstrip", body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if tt.wantStatus == http.StatusOK {
				if ct := rec.Header().Get("Content-Type"); ct != tt.wantType {
					t.Errorf("expected Content-Type %q, got %q", tt.wantType, ct)
				}
			}
		})
	}
}

func TestHandleImgRotate(t *testing.T) {
	handler := http.HandlerFunc(handleImgRotate)
	validImg := createTestImage(t, 10, 10, "png").Bytes()

	tests := []struct {
		name        string
		fileContent []byte
		fields      map[string]string
		wantStatus  int
	}{
		{"rot90", validImg, map[string]string{"action": "rot90"}, http.StatusOK},
		{"rot180", validImg, map[string]string{"action": "rot180"}, http.StatusOK},
		{"rot270", validImg, map[string]string{"action": "rot270"}, http.StatusOK},
		{"fliph", validImg, map[string]string{"action": "fliph"}, http.StatusOK},
		{"flipv", validImg, map[string]string{"action": "flipv"}, http.StatusOK},
		{"invalid action", validImg, map[string]string{"action": "invalid"}, http.StatusBadRequest},
		{"missing action", validImg, nil, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, contentType := createMultipartBody(t, "image", "test.png", tt.fileContent, tt.fields)
			req := httptest.NewRequest(http.MethodPost, "/rotate", body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestHandleImgToIco(t *testing.T) {
	handler := http.HandlerFunc(handleImgToIco)
	validImg := createTestImage(t, 10, 10, "png").Bytes()

	tests := []struct {
		name        string
		fileContent []byte
		wantStatus  int
	}{
		{"Valid image", validImg, http.StatusOK},
		{"Missing image", nil, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, contentType := createMultipartBody(t, "image", "test.png", tt.fileContent, nil)
			req := httptest.NewRequest(http.MethodPost, "/toico", body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
			if tt.wantStatus == http.StatusOK {
				if ct := rec.Header().Get("Content-Type"); ct != "image/x-icon" {
					t.Errorf("expected Content-Type image/x-icon, got %q", ct)
				}
			}
		})
	}
}

func TestHandleQrGenerate(t *testing.T) {
	handler := http.HandlerFunc(handleQrGenerate)

	tests := []struct {
		name       string
		fields     map[string]string
		wantStatus int
	}{
		{"Valid text", map[string]string{"text": "https://example.com"}, http.StatusOK},
		{"Empty text", map[string]string{"text": ""}, http.StatusBadRequest},
		{"Very long text", map[string]string{"text": string(make([]rune, 4097))}, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, contentType := createMultipartBody(t, "", "", nil, tt.fields)
			req := httptest.NewRequest(http.MethodPost, "/qrgen", body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestHandleQrRead(t *testing.T) {
	handler := http.HandlerFunc(handleQrRead)

	t.Run("Missing image", func(t *testing.T) {
		body, contentType := createMultipartBody(t, "image", "", nil, nil)
		req := httptest.NewRequest(http.MethodPost, "/qrread", body)
		req.Header.Set("Content-Type", contentType)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})
}

func TestHandleBase64(t *testing.T) {
	handler := http.HandlerFunc(handleBase64)

	tests := []struct {
		name        string
		fields      map[string]string
		fileContent []byte
		wantStatus  int
	}{
		{
			name:        "Encode mode with file",
			fields:      map[string]string{"mode": "encode"},
			fileContent: []byte("hello world"),
			wantStatus:  http.StatusOK,
		},
		{
			name:        "Decode mode with valid base64",
			fields:      map[string]string{"mode": "decode", "text": "aGVsbG8gd29ybGQ="},
			fileContent: nil,
			wantStatus:  http.StatusOK,
		},
		{
			name:        "Decode mode with invalid base64",
			fields:      map[string]string{"mode": "decode", "text": "not-base64!"},
			fileContent: nil,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "Decode mode with empty text",
			fields:      map[string]string{"mode": "decode", "text": ""},
			fileContent: nil,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "Invalid mode",
			fields:      map[string]string{"mode": "invalid"},
			fileContent: nil,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "Missing mode",
			fields:      nil,
			fileContent: nil,
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, contentType := createMultipartBody(t, "file", "test.txt", tt.fileContent, tt.fields)
			req := httptest.NewRequest(http.MethodPost, "/base64", body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestHandleMinify(t *testing.T) {
	handler := http.HandlerFunc(handleMinify)

	tests := []struct {
		name        string
		fileName    string
		fileContent []byte
		fields      map[string]string
		wantStatus  int
	}{
		{"CSS file", "style.css", []byte("body { color: red; }"), nil, http.StatusOK},
		{"HTML file", "index.html", []byte("<html> <body> </body> </html>"), nil, http.StatusOK},
		{"JS file", "script.js", []byte("function test() { return 1; }"), nil, http.StatusOK},
		{"JSON file", "data.json", []byte(`{ "a": 1 }`), nil, http.StatusOK},
		{"Unsupported file type", "data.txt", []byte("hello"), nil, http.StatusBadRequest},
		{"With explicit lang parameter", "unknown.ext", []byte("body { color: red; }"), map[string]string{"lang": "text/css"}, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, contentType := createMultipartBody(t, "file", tt.fileName, tt.fileContent, tt.fields)
			req := httptest.NewRequest(http.MethodPost, "/minify", body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestHandlePdfMerge(t *testing.T) {
	handler := http.HandlerFunc(handlePdfMerge)

	tests := []struct {
		name       string
		files      map[string][]byte
		wantStatus int
	}{
		{
			name:       "Less than 2 files",
			files:      map[string][]byte{"1.pdf": []byte("fake")},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "Too many files",
			files: func() map[string][]byte {
				m := make(map[string][]byte)
				for i := 0; i < 51; i++ {
					m[string(rune(i))+".pdf"] = []byte("fake")
				}
				return m
			}(),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Invalid PDF data",
			files:      map[string][]byte{"1.pdf": []byte("fake"), "2.pdf": []byte("fake")},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, contentType := createFakePDFBody(t, "pdfs", tt.files, nil)
			req := httptest.NewRequest(http.MethodPost, "/pdfmerge", body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestHandlePdfSplit(t *testing.T) {
	handler := http.HandlerFunc(handlePdfSplit)

	t.Run("Missing file", func(t *testing.T) {
		body, contentType := createMultipartBody(t, "pdf", "", nil, nil)
		req := httptest.NewRequest(http.MethodPost, "/pdfsplit", body)
		req.Header.Set("Content-Type", contentType)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})
}

func TestHandlePdfProtect(t *testing.T) {
	handler := http.HandlerFunc(handlePdfProtect)

	tests := []struct {
		name       string
		fields     map[string]string
		wantStatus int
	}{
		{"Missing password", nil, http.StatusBadRequest},
		{"Password too short", map[string]string{"password": "short"}, http.StatusBadRequest},
		{"Password too long", map[string]string{"password": string(make([]rune, 129))}, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, contentType := createMultipartBody(t, "pdf", "test.pdf", []byte("fake pdf"), tt.fields)
			req := httptest.NewRequest(http.MethodPost, "/pdfprotect", body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestHandlePdfRotate(t *testing.T) {
	handler := http.HandlerFunc(handlePdfRotate)

	tests := []struct {
		name       string
		fields     map[string]string
		wantStatus int
	}{
		{"Invalid degrees", map[string]string{"degrees": "45"}, http.StatusBadRequest},
		{"Pages string too long", map[string]string{"degrees": "90", "pages": string(make([]rune, 201))}, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, contentType := createMultipartBody(t, "pdf", "test.pdf", []byte("fake pdf"), tt.fields)
			req := httptest.NewRequest(http.MethodPost, "/pdfrotate", body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestHandlePdfWatermark(t *testing.T) {
	handler := http.HandlerFunc(handlePdfWatermark)

	tests := []struct {
		name        string
		fileContent []byte
		fields      map[string]string
		wantStatus  int
	}{
		{"Missing PDF", nil, nil, http.StatusBadRequest},
		{"Text too long", []byte("fake pdf"), map[string]string{"text": string(make([]rune, 201))}, http.StatusBadRequest},
		// "Default watermark text when empty" requires parsing pdfcpu output or mocking.
		// Testing error path for invalid PDF data when empty text is provided.
		{"Default watermark text when empty", []byte("invalid pdf"), map[string]string{"text": ""}, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, contentType := createMultipartBody(t, "pdf", "test.pdf", tt.fileContent, tt.fields)
			req := httptest.NewRequest(http.MethodPost, "/pdfwatermark", body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}
