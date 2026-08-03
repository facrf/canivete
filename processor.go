package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"golang.org/x/image/bmp"
)

// Função auxiliar para distância de cor
func colorDistance(c1, c2 color.Color) float64 {
	r1, g1, b1, _ := c1.RGBA()
	r2, g2, b2, _ := c2.RGBA()

	// converter para 0-255
	r1, g1, b1 = r1>>8, g1>>8, b1>>8
	r2, g2, b2 = r2>>8, g2>>8, b2>>8

	dr := float64(r1) - float64(r2)
	dg := float64(g1) - float64(g2)
	db := float64(b1) - float64(b2)

	return math.Sqrt(dr*dr + dg*dg + db*db)
}

func handleRemoveBg(w http.ResponseWriter, r *http.Request) {
	if !parseMultipartForm(w, r, maxUploadSize) {
		return
	}
	defer cleanupMultipartForm(r)

	tolStr := r.FormValue("tolerance")
	tolerance, err := strconv.ParseFloat(tolStr, 64)
	if err != nil || tolerance <= 0 {
		tolerance = 10 // Padrão
	}
	if tolerance > 100 {
		tolerance = 100
	}

	// Max distance is roughly 441.67 for sqrt(255^2*3).
	// Tolerance in percentage: 100% = distance 442
	maxDist := 441.67
	allowedDist := maxDist * (tolerance / 100.0)

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Erro ao ler arquivo", http.StatusBadRequest)
		return
	}
	defer file.Close()

	img, _, err := decodeImage(file)
	if err != nil {
		http.Error(w, "Imagem inválida ou acima dos limites permitidos", http.StatusBadRequest)
		return
	}

	bounds := img.Bounds()
	newImg := image.NewRGBA(bounds)

	white := color.RGBA{255, 255, 255, 255}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)

			dist := colorDistance(c, white)

			if dist <= allowedDist {
				// Calcular alpha proporcional se quiser transição suave (opcional)
				// Aqui faremos totalmente transparente se estiver dentro da tolerância
				newImg.Set(x, y, color.Transparent)
			} else {
				newImg.Set(x, y, c)
			}
		}
	}

	setDownloadHeaders(w, "image/png", "no-bg.png")
	if err := png.Encode(w, newImg); err != nil {
		log.Printf("Erro ao codificar imagem sem fundo: %v", err)
	}
}

func handleResize(w http.ResponseWriter, r *http.Request) {
	if !parseMultipartForm(w, r, maxUploadSize) {
		return
	}
	defer cleanupMultipartForm(r)

	width, widthErr := strconv.Atoi(r.FormValue("width"))
	height, heightErr := strconv.Atoi(r.FormValue("height"))
	if widthErr != nil || heightErr != nil || validateImageDimensions(width, height) != nil {
		http.Error(w, "Dimensões inválidas ou acima dos limites permitidos", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Erro ao ler arquivo", http.StatusBadRequest)
		return
	}
	defer file.Close()

	img, _, err := decodeImage(file)
	if err != nil {
		http.Error(w, "Imagem inválida ou acima dos limites permitidos", http.StatusBadRequest)
		return
	}

	resized := imaging.Resize(img, width, height, imaging.Lanczos)

	setDownloadHeaders(w, "image/png", "resized.png")
	if err := png.Encode(w, resized); err != nil {
		log.Printf("Erro ao codificar imagem redimensionada: %v", err)
	}
}

func handleCrop(w http.ResponseWriter, r *http.Request) {
	if !parseMultipartForm(w, r, maxUploadSize) {
		return
	}
	defer cleanupMultipartForm(r)

	xStr, yStr := r.FormValue("x"), r.FormValue("y")
	wStr, hStr := r.FormValue("width"), r.FormValue("height")
	finalSizeStr := r.FormValue("size")

	x, xErr := strconv.ParseFloat(xStr, 64)
	y, yErr := strconv.ParseFloat(yStr, 64)
	cW, widthErr := strconv.ParseFloat(wStr, 64)
	cH, heightErr := strconv.ParseFloat(hStr, 64)
	finalSize, sizeErr := strconv.Atoi(finalSizeStr)
	if xErr != nil || yErr != nil || widthErr != nil || heightErr != nil || sizeErr != nil ||
		math.IsNaN(x) || math.IsNaN(y) || math.IsNaN(cW) || math.IsNaN(cH) ||
		math.IsInf(x, 0) || math.IsInf(y, 0) || math.IsInf(cW, 0) || math.IsInf(cH, 0) ||
		x < 0 || y < 0 || cW <= 0 || cH <= 0 || finalSize < 0 {
		http.Error(w, "Área de recorte inválida", http.StatusBadRequest)
		return
	}
	if finalSize > 0 && validateImageDimensions(finalSize, finalSize) != nil {
		http.Error(w, "Tamanho final acima dos limites permitidos", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Erro ao ler arquivo", http.StatusBadRequest)
		return
	}
	defer file.Close()

	img, _, err := decodeImage(file)
	if err != nil {
		http.Error(w, "Imagem inválida ou acima dos limites permitidos", http.StatusBadRequest)
		return
	}

	rect := image.Rect(int(x), int(y), int(math.Ceil(x+cW)), int(math.Ceil(y+cH)))
	if !rect.In(img.Bounds()) || rect.Empty() {
		http.Error(w, "Área de recorte fora dos limites da imagem", http.StatusBadRequest)
		return
	}
	cropped := imaging.Crop(img, rect)

	if finalSize > 0 {
		cropped = imaging.Resize(cropped, finalSize, finalSize, imaging.Lanczos)
	}

	setDownloadHeaders(w, "image/png", "cropped.png")
	if err := png.Encode(w, cropped); err != nil {
		log.Printf("Erro ao codificar recorte: %v", err)
	}
}

func handleConvert(w http.ResponseWriter, r *http.Request) {
	if !parseMultipartForm(w, r, maxUploadSize) {
		return
	}
	defer cleanupMultipartForm(r)

	format := r.FormValue("format")
	quality, _ := strconv.Atoi(r.FormValue("quality"))
	if quality < 1 || quality > 100 {
		quality = 80
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Erro ao ler arquivo", http.StatusBadRequest)
		return
	}
	defer file.Close()

	img, _, err := decodeImage(file)
	if err != nil {
		http.Error(w, "Imagem inválida ou acima dos limites permitidos", http.StatusBadRequest)
		return
	}

	if strings.ToLower(format) == "jpg" || strings.ToLower(format) == "jpeg" || strings.ToLower(format) == "bmp" {
		bounds := img.Bounds()
		opaqueImg := image.NewRGBA(bounds)
		draw.Draw(opaqueImg, bounds, &image.Uniform{color.White}, image.Point{}, draw.Src)
		draw.Draw(opaqueImg, bounds, img, bounds.Min, draw.Over)
		img = opaqueImg
	}

	var buf bytes.Buffer
	var contentType, ext string

	switch strings.ToLower(format) {
	case "png":
		err = png.Encode(&buf, img)
		contentType, ext = "image/png", "png"
	case "jpg", "jpeg":
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
		contentType, ext = "image/jpeg", "jpg"
	case "bmp":
		err = bmp.Encode(&buf, img)
		contentType, ext = "image/bmp", "bmp"
	default:
		http.Error(w, "Formato não suportado", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, "Erro ao codificar imagem", http.StatusInternalServerError)
		return
	}

	setDownloadHeaders(w, contentType, fmt.Sprintf("converted.%s", ext))
	_, _ = w.Write(buf.Bytes())
}

func handleImgToPdf(w http.ResponseWriter, r *http.Request) {
	if !parseMultipartForm(w, r, maxBatchUploadSize) {
		return
	}
	defer cleanupMultipartForm(r)

	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		http.Error(w, "Nenhuma imagem enviada", http.StatusBadRequest)
		return
	}
	if len(files) > maxBatchFiles {
		http.Error(w, "Quantidade máxima de imagens excedida", http.StatusBadRequest)
		return
	}

	imgReaders := make([]io.Reader, 0, len(files))
	openedFiles := make([]io.Closer, 0, len(files))
	defer func() {
		for _, openedFile := range openedFiles {
			if err := openedFile.Close(); err != nil {
				log.Printf("Erro ao fechar imagem do lote: %v", err)
			}
		}
	}()

	for _, fileHeader := range files {
		if fileHeader.Size > maxUploadSize {
			http.Error(w, "Uma das imagens excede o limite individual", http.StatusRequestEntityTooLarge)
			return
		}
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, "Erro ao abrir imagem", http.StatusBadRequest)
			return
		}
		if _, err := validateImageFile(file); err != nil {
			_ = file.Close()
			http.Error(w, "Imagem inválida ou acima dos limites permitidos", http.StatusBadRequest)
			return
		}
		openedFiles = append(openedFiles, file)
		imgReaders = append(imgReaders, file)
	}

	conf := model.NewDefaultConfiguration()
	imp, err := api.Import("form:A4, pos:c, sc:1.0", types.POINTS)
	if err != nil {
		internalError(w, "Erro ao configurar geração do PDF", err)
		return
	}

	pdfFile, err := os.CreateTemp("", "canivete-images-*.pdf")
	if err != nil {
		internalError(w, "Erro ao preparar PDF de saída", err)
		return
	}
	defer func() {
		_ = pdfFile.Close()
		_ = os.Remove(pdfFile.Name())
	}()

	err = api.ImportImages(nil, pdfFile, imgReaders, imp, conf)
	if err != nil {
		internalError(w, "Erro ao gerar PDF", err)
		return
	}

	if err := serveDownloadFile(w, pdfFile, "application/pdf", "output.pdf"); err != nil {
		log.Printf("Erro ao enviar PDF gerado: %v", err)
	}
}

func handlePdfToImg(w http.ResponseWriter, r *http.Request) {
	if !parseMultipartForm(w, r, maxPDFUploadSize) {
		return
	}
	defer cleanupMultipartForm(r)

	file, _, err := r.FormFile("pdf")
	if err != nil {
		http.Error(w, "Erro ao ler arquivo", http.StatusBadRequest)
		return
	}
	defer file.Close()

	zipFile, err := os.CreateTemp("", "canivete-images-*.zip")
	if err != nil {
		internalError(w, "Erro ao preparar arquivo ZIP", err)
		return
	}
	defer os.Remove(zipFile.Name())
	defer zipFile.Close()
	zipWriter := zip.NewWriter(zipFile)

	err = api.ExtractImages(file, nil, func(img model.Image, singleImgPerPage bool, maxPage int) error {
		fw, err := zipWriter.Create(fmt.Sprintf("%s.%s", img.Name, img.FileType))
		if err != nil {
			return err
		}
		_, err = io.Copy(fw, img)
		return err
	}, model.NewDefaultConfiguration())

	if err != nil {
		_ = zipWriter.Close()
		http.Error(w, "PDF inválido ou não suportado", http.StatusBadRequest)
		return
	}
	if err := zipWriter.Close(); err != nil {
		internalError(w, "Erro ao finalizar arquivo ZIP", err)
		return
	}
	if err := serveDownloadFile(w, zipFile, "application/zip", "imagens_extraidas.zip"); err != nil {
		log.Printf("Erro ao enviar imagens extraídas: %v", err)
	}
}

func handlePdfRasterize(w http.ResponseWriter, r *http.Request) {
	if !parseMultipartForm(w, r, maxPDFUploadSize) {
		return
	}
	defer cleanupMultipartForm(r)

	file, _, err := r.FormFile("pdf")
	if err != nil {
		http.Error(w, "Erro ao ler arquivo", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Usar pdftoppm (do pacote poppler-utils) para rasterizar o PDF
	tmpPdf, err := os.CreateTemp("", "upload-*.pdf")
	if err != nil {
		http.Error(w, "Erro ao criar temp", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpPdf.Name())

	if _, err := io.Copy(tmpPdf, file); err != nil {
		_ = tmpPdf.Close()
		internalError(w, "Erro ao salvar PDF temporário", err)
		return
	}
	if err := tmpPdf.Close(); err != nil {
		internalError(w, "Erro ao salvar PDF temporário", err)
		return
	}

	tmpDir, err := os.MkdirTemp("", "pdf-rasterize")
	if err != nil {
		http.Error(w, "Erro ao criar pasta temp", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	// pdftoppm -jpeg -r 150 <pdf> <prefix>
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	// #nosec G204 -- executável e opções são fixos; os caminhos vêm de os.CreateTemp/MkdirTemp.
	cmd := exec.CommandContext(ctx, "pdftoppm", "-jpeg", "-r", "150", tmpPdf.Name(), filepath.Join(tmpDir, "page"))
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("Erro ao rasterizar PDF: %v: %s", err, strings.TrimSpace(string(output)))
		http.Error(w, "PDF inválido, não suportado ou muito complexo", http.StatusBadRequest)
		return
	}

	zipFile, err := os.CreateTemp("", "canivete-pages-*.zip")
	if err != nil {
		internalError(w, "Erro ao preparar arquivo ZIP", err)
		return
	}
	defer os.Remove(zipFile.Name())
	defer zipFile.Close()
	zipWriter := zip.NewWriter(zipFile)
	if err := addDirectoryToZip(zipWriter, tmpDir); err != nil {
		_ = zipWriter.Close()
		internalError(w, "Erro ao compactar páginas do PDF", err)
		return
	}
	if err := zipWriter.Close(); err != nil {
		internalError(w, "Erro ao finalizar arquivo ZIP", err)
		return
	}
	if err := serveDownloadFile(w, zipFile, "application/zip", "paginas_pdf.zip"); err != nil {
		log.Printf("Erro ao enviar páginas rasterizadas: %v", err)
	}
}

func handleSvgToImg(w http.ResponseWriter, r *http.Request) {
	if !parseMultipartForm(w, r, maxUploadSize) {
		return
	}
	defer cleanupMultipartForm(r)

	widthStr := r.FormValue("width") // opcional, ex: 1024

	file, _, err := r.FormFile("svg")
	if err != nil {
		http.Error(w, "Erro ao ler arquivo SVG", http.StatusBadRequest)
		return
	}
	defer file.Close()

	tmpSvg, err := os.CreateTemp("", "upload-*.svg")
	if err != nil {
		http.Error(w, "Erro interno ao criar temporário", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpSvg.Name())

	if _, err := io.Copy(tmpSvg, file); err != nil {
		_ = tmpSvg.Close()
		internalError(w, "Erro ao salvar SVG temporário", err)
		return
	}
	if err := tmpSvg.Close(); err != nil {
		internalError(w, "Erro ao salvar SVG temporário", err)
		return
	}

	outName := tmpSvg.Name() + ".png"
	defer os.Remove(outName)

	args := []string{"-f", "png", "-o", outName}
	// Segurança: validar width como inteiro positivo dentro de limite razoável
	// antes de passar como argumento ao processo externo rsvg-convert
	if widthStr != "" && widthStr != "0" {
		wParsed, err := strconv.Atoi(widthStr)
		if err != nil || wParsed <= 0 || wParsed > 8000 {
			http.Error(w, "Largura inválida: deve ser um número entre 1 e 8000", http.StatusBadRequest)
			return
		}
		args = append(args, "-w", strconv.Itoa(wParsed))
	}
	args = append(args, tmpSvg.Name())

	ctx, cancel := context.WithTimeout(r.Context(), time.Minute)
	defer cancel()
	// #nosec G204,G702 -- não há shell; largura validada e caminhos gerados por os.CreateTemp.
	cmd := exec.CommandContext(ctx, "rsvg-convert", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("Erro ao renderizar SVG: %v: %s", err, strings.TrimSpace(string(output)))
		http.Error(w, "SVG inválido, não suportado ou muito complexo", http.StatusBadRequest)
		return
	}

	// #nosec G304 -- outName deriva exclusivamente do caminho retornado por os.CreateTemp.
	outputFile, err := os.Open(outName)
	if err != nil {
		http.Error(w, "Erro ao ler imagem renderizada", http.StatusInternalServerError)
		return
	}
	defer outputFile.Close()

	if err := serveDownloadFile(w, outputFile, "image/png", "vetor-renderizado.png"); err != nil {
		log.Printf("Erro ao enviar SVG renderizado: %v", err)
	}
}
