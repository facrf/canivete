package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

const (
	maxPDFPages                = 50
	maxBatchImagePixels        = 40_000_000
	maxRenderedDimension       = 4000
	maxGeneratedBytes    int64 = 64 << 20
)

var errGeneratedLimit = errors.New("saída excede o limite de 64 MiB")

// limitedWriter prevents generated files from filling temporary storage.
type limitedWriter struct {
	writer    io.Writer
	remaining int64
	err       error
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	if int64(len(p)) > w.remaining {
		w.err = errGeneratedLimit
		return 0, w.err
	}
	n, err := w.writer.Write(p)
	w.remaining -= int64(n)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	w.err = err
	return n, err
}

// Some PDF APIs flush buffered output in a defer and do not return that flush
// error. Always inspect the underlying writer before offering a download.
func writeLimitedPDF(output io.Writer, generate func(io.Writer) error) error {
	writer := &limitedWriter{writer: output, remaining: maxGeneratedBytes}
	if err := generate(writer); err != nil {
		return err
	}
	return writer.err
}

func validatePDFPages(file io.ReadSeeker) (int, error) {
	count, err := api.PageCount(file, model.NewDefaultConfiguration())
	if err != nil {
		return 0, err
	}
	if count < 1 || count > maxPDFPages {
		return 0, fmt.Errorf("PDF deve ter entre 1 e %d páginas", maxPDFPages)
	}
	_, err = file.Seek(0, io.SeekStart)
	return count, err
}

// SVG lengths are converted to CSS pixels (96 dpi), matching rsvg-convert.
func svgLength(value string) (float64, error) {
	value = strings.TrimSpace(value)
	factor := 1.0
	for _, unit := range []struct {
		suffix string
		factor float64
	}{
		{"px", 1}, {"pt", 96.0 / 72}, {"pc", 16}, {"mm", 96.0 / 25.4}, {"cm", 96.0 / 2.54}, {"in", 96},
	} {
		if strings.HasSuffix(value, unit.suffix) {
			value = strings.TrimSpace(strings.TrimSuffix(value, unit.suffix))
			factor = unit.factor
			break
		}
	}
	number, err := strconv.ParseFloat(value, 64)
	number *= factor
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) || number <= 0 {
		return 0, errors.New("dimensão SVG inválida")
	}
	return number, nil
}

func svgDimensions(reader io.Reader, requestedWidth string) (int, int, error) {
	decoder := xml.NewDecoder(reader)
	var root xml.StartElement
	for {
		token, err := decoder.Token()
		if err != nil {
			return 0, 0, err
		}
		if element, ok := token.(xml.StartElement); ok {
			root = element
			break
		}
	}
	if root.Name.Local != "svg" || (root.Name.Space != "" && root.Name.Space != "http://www.w3.org/2000/svg") {
		return 0, 0, errors.New("raiz SVG inválida")
	}
	attrs := map[string]string{}
	for _, attr := range root.Attr {
		if attr.Name.Space == "" {
			attrs[attr.Name.Local] = attr.Value
		}
	}
	width, height := 300.0, 150.0
	if box := attrs["viewBox"]; box != "" {
		parts := strings.Fields(strings.ReplaceAll(box, ",", " "))
		if len(parts) != 4 {
			return 0, 0, errors.New("viewBox inválido")
		}
		for _, part := range parts {
			n, err := strconv.ParseFloat(part, 64)
			if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
				return 0, 0, errors.New("viewBox inválido")
			}
		}
		width, _ = strconv.ParseFloat(parts[2], 64)
		height, _ = strconv.ParseFloat(parts[3], 64)
		if width <= 0 || height <= 0 {
			return 0, 0, errors.New("viewBox inválido")
		}
	}
	var err error
	if value := attrs["width"]; value != "" {
		width, err = svgLength(value)
		if err != nil {
			return 0, 0, err
		}
	}
	if value := attrs["height"]; value != "" {
		height, err = svgLength(value)
		if err != nil {
			return 0, 0, err
		}
	}
	if requestedWidth != "" && requestedWidth != "0" {
		requested, err := strconv.Atoi(requestedWidth)
		if err != nil || requested < 1 || requested > maxImageDimension {
			return 0, 0, errors.New("largura SVG inválida")
		}
		height *= float64(requested) / width
		width = float64(requested)
	}
	// Check floats before converting to int to avoid overflow.
	if width > maxImageDimension || height > maxImageDimension || math.IsInf(height, 0) || math.IsNaN(height) {
		return 0, 0, errors.New("dimensões SVG acima do limite")
	}
	w, h := int(math.Ceil(width)), int(math.Ceil(height))
	if err := validateImageDimensions(w, h); err != nil {
		return 0, 0, err
	}
	return w, h, nil
}

// Image resource names are user-controlled PDF data. ZIP names use only numeric
// page/object identifiers and a fixed extension allowlist.
func safeImageExtension(extension string) string {
	switch strings.ToLower(extension) {
	case "png", "jpg", "jpeg", "tif", "tiff", "bmp", "jp2", "jpx", "webp":
		return strings.ToLower(extension)
	default:
		return "bin"
	}
}
