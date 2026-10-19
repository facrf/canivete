package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/EdlinOrg/prominentcolor"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	goqr "github.com/skip2/go-qrcode"
)

// PDF Merge
func handlePdfMerge(w http.ResponseWriter, r *http.Request) {
	if !parseMultipartForm(w, r, maxBatchUploadSize) {
		return
	}
	defer cleanupMultipartForm(r)
	files := r.MultipartForm.File["pdfs"]
	if len(files) < 2 || len(files) > maxBatchFiles {
		http.Error(w, "Selecione entre 2 e 50 PDFs", http.StatusBadRequest)
		return
	}
	inputs := make([]io.ReadSeeker, 0, len(files))
	var opened []io.Closer
	defer func() {
		for _, file := range opened {
			_ = file.Close()
		}
	}()
	totalPages := 0
	for _, header := range files {
		if header.Size > maxPDFUploadSize {
			http.Error(w, "Um dos PDFs excede o limite individual", http.StatusRequestEntityTooLarge)
			return
		}
		file, err := header.Open()
		if err != nil {
			internalError(w, "Erro ao abrir PDF", err)
			return
		}
		opened = append(opened, file)
		input := contextReadSeeker{ctx: r.Context(), ReadSeeker: file}
		count, err := validatePDFPages(input)
		if err != nil {
			http.Error(w, "PDF inválido ou excede o limite de 50 páginas", http.StatusBadRequest)
			return
		}
		totalPages += count
		if totalPages > maxPDFPages {
			http.Error(w, "A junção deve ter no máximo 50 páginas no total", http.StatusBadRequest)
			return
		}
		inputs = append(inputs, input)
	}
	output, err := os.CreateTemp("", "canivete-merged-*.pdf")
	if err != nil {
		internalError(w, "Erro ao preparar PDF de saída", err)
		return
	}
	defer os.Remove(output.Name())
	defer output.Close()
	if err := writeLimitedPDF(output, func(writer io.Writer) error {
		return api.MergeRaw(inputs, writer, false, model.NewDefaultConfiguration())
	}); err != nil {
		log.Printf("Erro ao juntar PDFs: %v", err)
		http.Error(w, "PDF inválido ou saída excede 64 MiB", http.StatusBadRequest)
		return
	}
	if err := serveDownloadFile(w, output, "application/pdf", "merged.pdf"); err != nil {
		log.Printf("Erro ao enviar PDF unido: %v", err)
	}
}

// PDF Split
func handlePdfSplit(w http.ResponseWriter, r *http.Request) {
	if !parseMultipartForm(w, r, maxPDFUploadSize) {
		return
	}
	defer cleanupMultipartForm(r)
	file, _, err := r.FormFile("pdf")
	if err != nil {
		http.Error(w, "Arquivo PDF ausente ou inválido na requisição", http.StatusBadRequest)
		return
	}
	defer file.Close()
	pageCount, err := validatePDFPages(file)
	if err != nil {
		http.Error(w, "PDF inválido ou excede o limite de 50 páginas", http.StatusBadRequest)
		return
	}
	zipFile, err := os.CreateTemp("", "canivete-split-*.zip")
	if err != nil {
		internalError(w, "Erro ao preparar arquivo ZIP", err)
		return
	}
	defer os.Remove(zipFile.Name())
	defer zipFile.Close()
	zipWriter := zip.NewWriter(&limitedWriter{writer: zipFile, remaining: maxGeneratedBytes})
	remainingOutput := &limitedWriter{remaining: maxGeneratedBytes}
	for page := 1; page <= pageCount; page++ {
		if err := r.Context().Err(); err != nil {
			_ = zipWriter.Close()
			return
		}
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			_ = zipWriter.Close()
			internalError(w, "Erro ao ler PDF", err)
			return
		}
		entry, err := zipWriter.Create(fmt.Sprintf("page-%03d.pdf", page))
		if err != nil {
			_ = zipWriter.Close()
			internalError(w, "Erro ao compactar páginas", err)
			return
		}
		remainingOutput.writer = entry
		if err := api.Trim(file, remainingOutput, []string{strconv.Itoa(page)}, model.NewDefaultConfiguration()); err != nil {
			_ = zipWriter.Close()
			log.Printf("Erro ao dividir PDF: %v", err)
			http.Error(w, "PDF inválido ou saída excede 64 MiB", http.StatusBadRequest)
			return
		}
		if remainingOutput.err != nil {
			_ = zipWriter.Close()
			http.Error(w, "Saída excede 64 MiB", http.StatusBadRequest)
			return
		}
	}
	if err := zipWriter.Close(); err != nil {
		internalError(w, "Erro ao finalizar arquivo ZIP", err)
		return
	}
	if err := serveDownloadFile(w, zipFile, "application/zip", "split_pdfs.zip"); err != nil {
		log.Printf("Erro ao enviar páginas divididas: %v", err)
	}
}

// PDF Protect
func handlePdfProtect(w http.ResponseWriter, r *http.Request) {
	if !parseMultipartForm(w, r, maxPDFUploadSize) {
		return
	}
	defer cleanupMultipartForm(r)

	password := r.FormValue("password")
	if password == "" {
		http.Error(w, "Senha é obrigatória", http.StatusBadRequest)
		return
	}
	// Segurança: exigir senha com comprimento mínimo razoável
	if len([]rune(password)) < 8 {
		http.Error(w, "A senha deve ter no mínimo 8 caracteres", http.StatusBadRequest)
		return
	}
	if len([]rune(password)) > 128 {
		http.Error(w, "A senha deve ter no máximo 128 caracteres", http.StatusBadRequest)
		return
	}

	conf := model.NewDefaultConfiguration()
	conf.UserPW, conf.OwnerPW = password, password
	transformPDF(w, r, "protegido.pdf", func(input io.ReadSeeker, output io.Writer) error {
		return api.Encrypt(input, output, conf)
	})
}

// PDF Optimize
func handlePdfOptimize(w http.ResponseWriter, r *http.Request) {
	if !parseMultipartForm(w, r, maxPDFUploadSize) {
		return
	}
	defer cleanupMultipartForm(r)
	transformPDF(w, r, "otimizado.pdf", func(input io.ReadSeeker, output io.Writer) error {
		return api.Optimize(input, output, model.NewDefaultConfiguration())
	})
}

// Image Compress
func handleImgCompress(w http.ResponseWriter, r *http.Request) {
	if !parseMultipartForm(w, r, maxUploadSize) {
		return
	}
	defer cleanupMultipartForm(r)

	qualityStr := r.FormValue("quality")
	quality := 80
	if q, err := strconv.Atoi(qualityStr); err == nil {
		quality = q
	}
	// Segurança: clampar qualidade dentro do intervalo válido do codec JPEG
	if quality < 1 {
		quality = 1
	}
	if quality > 100 {
		quality = 100
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Arquivo de imagem ausente ou inválido na requisição", http.StatusBadRequest)
		return
	}
	defer file.Close()

	img, _, err := decodeImage(file)
	if err != nil {
		http.Error(w, "Imagem inválida ou acima dos limites permitidos", http.StatusBadRequest)
		return
	}

	bounds := img.Bounds()
	opaqueImg := image.NewRGBA(bounds)
	draw.Draw(opaqueImg, bounds, &image.Uniform{color.White}, image.Point{}, draw.Src)
	draw.Draw(opaqueImg, bounds, img, bounds.Min, draw.Over)
	img = opaqueImg

	setDownloadHeaders(w, "image/jpeg", "comprimida.jpg")
	if err := jpeg.Encode(w, img, &jpeg.Options{Quality: quality}); err != nil {
		log.Printf("Erro ao codificar imagem comprimida: %v", err)
	}
}

// QR Generate
func handleQrGenerate(w http.ResponseWriter, r *http.Request) {
	if !parseMultipartForm(w, r, 1<<20) {
		return
	}
	defer cleanupMultipartForm(r)

	text := r.FormValue("text")
	if text == "" {
		http.Error(w, "Texto vazio", http.StatusBadRequest)
		return
	}
	if len([]rune(text)) > 4096 {
		http.Error(w, "Texto muito longo para QR Code", http.StatusBadRequest)
		return
	}

	png, err := goqr.Encode(text, goqr.Medium, 256)
	if err != nil {
		http.Error(w, "Erro ao gerar QR Code", http.StatusInternalServerError)
		return
	}

	setDownloadHeaders(w, "image/png", "qrcode.png")
	// #nosec G705 -- bytes são um PNG gerado pela biblioteca, com nosniff e tipo fixo.
	_, _ = w.Write(png)
}

// QR Read
func handleQrRead(w http.ResponseWriter, r *http.Request) {
	if !parseMultipartForm(w, r, maxUploadSize) {
		return
	}
	defer cleanupMultipartForm(r)

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Arquivo de imagem ausente ou inválido na requisição", http.StatusBadRequest)
		return
	}
	defer file.Close()

	img, _, err := decodeImage(file)
	if err != nil {
		http.Error(w, "Imagem inválida ou acima dos limites permitidos", http.StatusBadRequest)
		return
	}

	bmp, _ := gozxing.NewBinaryBitmapFromImage(img)
	qrReader := qrcode.NewQRCodeReader()
	result, err := qrReader.Decode(bmp, nil)
	if err != nil {
		http.Error(w, "Nenhum QR Code encontrado ou legível", http.StatusBadRequest)
		return
	}

	setDownloadHeaders(w, "text/plain; charset=utf-8", "qrcode.txt")
	// #nosec G705 -- resposta é texto puro com nosniff; a UI usa textarea.value.
	_, _ = w.Write([]byte(result.GetText()))
}

// Image Palette
func handleImgPalette(w http.ResponseWriter, r *http.Request) {
	if !parseMultipartForm(w, r, maxUploadSize) {
		return
	}
	defer cleanupMultipartForm(r)

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Arquivo de imagem ausente ou inválido na requisição", http.StatusBadRequest)
		return
	}
	defer file.Close()

	img, _, err := decodeImage(file)
	if err != nil {
		http.Error(w, "Imagem inválida ou acima dos limites permitidos", http.StatusBadRequest)
		return
	}

	cols, err := prominentcolor.Kmeans(img)
	if err != nil {
		http.Error(w, "Erro ao extrair cores", http.StatusInternalServerError)
		return
	}

	var hexColors []string
	for _, c := range cols {
		hexColors = append(hexColors, "#"+c.AsString())
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"colors": hexColors}); err != nil {
		log.Printf("Erro ao enviar paleta de cores: %v", err)
	}
}
