package main

import (
	"errors"
	"fmt"
	"image"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxUploadSize      int64 = 20 << 20
	maxPDFUploadSize   int64 = 50 << 20
	maxBatchUploadSize int64 = 100 << 20
	maxFormMemory      int64 = 8 << 20

	maxImageDimension = 8000
	maxImagePixels    = 20_000_000
	maxBatchFiles     = 50
)

func parseMultipartForm(w http.ResponseWriter, r *http.Request, maxBytes int64) bool {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return false
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	// #nosec G120 -- MaxBytesReader impõe o limite total antes do parser multipart.
	if err := r.ParseMultipartForm(min(maxBytes, maxFormMemory)); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			http.Error(w, "Arquivo ou formulário excede o limite permitido", http.StatusRequestEntityTooLarge)
			return false
		}
		http.Error(w, "Formulário inválido", http.StatusBadRequest)
		return false
	}

	return true
}

func cleanupMultipartForm(r *http.Request) {
	if r.MultipartForm == nil {
		return
	}
	if err := r.MultipartForm.RemoveAll(); err != nil {
		log.Printf("Erro ao remover arquivos temporários do upload: %v", err)
	}
}

func decodeImage(file multipart.File) (image.Image, string, error) {
	format, err := validateImageFile(file)
	if err != nil {
		return nil, "", err
	}

	img, decodedFormat, err := image.Decode(file)
	if err != nil {
		return nil, "", fmt.Errorf("decodificar imagem: %w", err)
	}
	if decodedFormat != "" {
		format = decodedFormat
	}
	return img, format, nil
}

func validateImageFile(file multipart.File) (string, error) {
	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return "", fmt.Errorf("ler cabeçalho da imagem: %w", err)
	}
	if err := validateImageDimensions(config.Width, config.Height); err != nil {
		return "", err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("reposicionar arquivo de imagem: %w", err)
	}
	return format, nil
}

func validateImageDimensions(width, height int) error {
	if width <= 0 || height <= 0 {
		return errors.New("dimensões de imagem inválidas")
	}
	if width > maxImageDimension || height > maxImageDimension {
		return fmt.Errorf("dimensão máxima permitida é %dpx", maxImageDimension)
	}
	if int64(width)*int64(height) > maxImagePixels {
		return fmt.Errorf("imagem excede o limite de %d megapixels", maxImagePixels/1_000_000)
	}
	return nil
}

func setDownloadHeaders(w http.ResponseWriter, contentType, filename string) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{
		"filename": safeFilename(filename),
	}))
}

func safeFilename(filename string) string {
	filename = strings.ReplaceAll(filename, "\\", "/")
	filename = filepath.Base(filename)
	filename = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, filename)
	filename = strings.TrimSpace(filename)
	if filename == "" || filename == "." {
		return "download"
	}

	runes := []rune(filename)
	if len(runes) > 120 {
		filename = string(runes[:120])
	}
	return filename
}

func internalError(w http.ResponseWriter, publicMessage string, err error) {
	log.Printf("%s: %v", publicMessage, err)
	http.Error(w, publicMessage, http.StatusInternalServerError)
}

func serveDownloadFile(w http.ResponseWriter, file *os.File, contentType, filename string) error {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	setDownloadHeaders(w, contentType, filename)
	_, err := io.Copy(w, file)
	return err
}
