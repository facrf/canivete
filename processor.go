package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"golang.org/x/image/bmp"
)

const maxUploadSize = 20 << 20

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
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	r.ParseMultipartForm(maxUploadSize)

	tolStr := r.FormValue("tolerance")
	tolerance, _ := strconv.ParseFloat(tolStr, 64)
	if tolerance <= 0 {
		tolerance = 10 // Padrão
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

	img, _, err := image.Decode(file)
	if err != nil {
		http.Error(w, "Erro ao decodificar imagem", http.StatusBadRequest)
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

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", "attachment; filename=\"no-bg.png\"")
	png.Encode(w, newImg)
}

func handleResize(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	r.ParseMultipartForm(maxUploadSize)

	width, _ := strconv.Atoi(r.FormValue("width"))
	height, _ := strconv.Atoi(r.FormValue("height"))

	if width <= 0 || height <= 0 {
		http.Error(w, "Dimensões inválidas", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Erro ao ler arquivo", http.StatusBadRequest)
		return
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		http.Error(w, "Erro ao decodificar imagem", http.StatusBadRequest)
		return
	}

	resized := imaging.Resize(img, width, height, imaging.Lanczos)

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", "attachment; filename=\"resized.png\"")
	png.Encode(w, resized)
}

func handleCrop(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	r.ParseMultipartForm(maxUploadSize)

	xStr, yStr := r.FormValue("x"), r.FormValue("y")
	wStr, hStr := r.FormValue("width"), r.FormValue("height")
	finalSizeStr := r.FormValue("size")

	x, _ := strconv.ParseFloat(xStr, 64)
	y, _ := strconv.ParseFloat(yStr, 64)
	cW, _ := strconv.ParseFloat(wStr, 64)
	cH, _ := strconv.ParseFloat(hStr, 64)
	finalSize, _ := strconv.Atoi(finalSizeStr)

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Erro ao ler arquivo", http.StatusBadRequest)
		return
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		http.Error(w, "Erro ao decodificar imagem", http.StatusBadRequest)
		return
	}

	// Crop
	rect := image.Rect(int(x), int(y), int(x+cW), int(y+cH))
	cropped := imaging.Crop(img, rect)

	if finalSize > 0 {
		cropped = imaging.Resize(cropped, finalSize, finalSize, imaging.Lanczos)
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", "attachment; filename=\"cropped.png\"")
	png.Encode(w, cropped)
}

func handleConvert(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	r.ParseMultipartForm(maxUploadSize)

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

	img, _, err := image.Decode(file)
	if err != nil {
		http.Error(w, "Erro ao decodificar imagem", http.StatusBadRequest)
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

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"converted.%s\"", ext))
	w.Write(buf.Bytes())
}

func handleImgToPdf(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize*10)
	r.ParseMultipartForm(maxUploadSize * 10)

	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		http.Error(w, "Nenhuma imagem enviada", http.StatusBadRequest)
		return
	}

	var imgReaders []io.Reader

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, "Erro", http.StatusBadRequest)
			return
		}
		defer file.Close()

		buf, _ := io.ReadAll(file)
		imgReaders = append(imgReaders, bytes.NewReader(buf))
	}

	var pdfBuf bytes.Buffer
	conf := model.NewDefaultConfiguration()
	imp, _ := api.Import("form:A4, pos:c, sc:1.0", types.POINTS)

	err := api.ImportImages(nil, &pdfBuf, imgReaders, imp, conf)
	if err != nil {
		http.Error(w, "Erro ao gerar PDF: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\"output.pdf\"")
	w.Write(pdfBuf.Bytes())
}

func handlePdfToImg(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize*5)
	r.ParseMultipartForm(maxUploadSize * 5)

	file, _, err := r.FormFile("pdf")
	if err != nil {
		http.Error(w, "Erro ao ler arquivo", http.StatusBadRequest)
		return
	}
	defer file.Close()

	buf, _ := io.ReadAll(file)
	rs := bytes.NewReader(buf)

	var zipBuf bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuf)

	err = api.ExtractImages(rs, nil, func(img model.Image, singleImgPerPage bool, maxPage int) error {
		fw, err := zipWriter.Create(fmt.Sprintf("%s.%s", img.Name, img.FileType))
		if err != nil {
			return err
		}
		_, err = io.Copy(fw, img)
		return err
	}, model.NewDefaultConfiguration())

	if err != nil {
		http.Error(w, "Erro ao extrair imagens: "+err.Error(), http.StatusInternalServerError)
		return
	}
	zipWriter.Close()

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"imagens_extraidas.zip\"")
	w.Write(zipBuf.Bytes())
}

func handlePdfRasterize(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize*5)
	r.ParseMultipartForm(maxUploadSize * 5)

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
	
	io.Copy(tmpPdf, file)
	tmpPdf.Close()
	
	tmpDir, err := os.MkdirTemp("", "pdf-rasterize")
	if err != nil {
		http.Error(w, "Erro ao criar pasta temp", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	// pdftoppm -jpeg -r 150 <pdf> <prefix>
	cmd := exec.Command("pdftoppm", "-jpeg", "-r", "150", tmpPdf.Name(), filepath.Join(tmpDir, "page"))
	if err := cmd.Run(); err != nil {
		http.Error(w, "Erro ao rasterizar PDF. Verifique se poppler-utils está instalado.", http.StatusInternalServerError)
		return
	}

	// Zipar o resultado
	var zipBuf bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuf)

	entries, _ := os.ReadDir(tmpDir)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fPath := filepath.Join(tmpDir, entry.Name())
		b, err := os.ReadFile(fPath)
		if err == nil {
			fw, err := zipWriter.Create(entry.Name())
			if err == nil {
				fw.Write(b)
			}
		}
	}
	zipWriter.Close()

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"paginas_pdf.zip\"")
	w.Write(zipBuf.Bytes())
}

func handleSvgToImg(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	r.ParseMultipartForm(maxUploadSize)

	width := r.FormValue("width") // opcional, ex: 1024

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
	
	io.Copy(tmpSvg, file)
	tmpSvg.Close()

	outName := tmpSvg.Name() + ".png"
	defer os.Remove(outName)

	args := []string{"-f", "png", "-o", outName}
	if width != "" && width != "0" {
		args = append(args, "-w", width)
	}
	args = append(args, tmpSvg.Name())

	cmd := exec.Command("rsvg-convert", args...)
	if err := cmd.Run(); err != nil {
		http.Error(w, "Erro ao renderizar SVG. Verifique se librsvg está instalado.", http.StatusInternalServerError)
		return
	}

	buf, err := os.ReadFile(outName)
	if err != nil {
		http.Error(w, "Erro ao ler imagem renderizada", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", "attachment; filename=\"vetor-renderizado.png\"")
	w.Write(buf)
}
