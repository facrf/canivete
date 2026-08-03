package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/json"
)

func handleImgExifStrip(w http.ResponseWriter, r *http.Request) {
	if !parseMultipartForm(w, r, maxUploadSize) {
		return
	}
	defer cleanupMultipartForm(r)

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Imagem não enviada", http.StatusBadRequest)
		return
	}
	defer file.Close()

	img, format, err := decodeImage(file)
	if err != nil {
		http.Error(w, "Imagem inválida ou acima dos limites permitidos", http.StatusBadRequest)
		return
	}

	switch format {
	case "jpeg":
		setDownloadHeaders(w, "image/jpeg", "sem-metadados.jpg")
		if err := jpeg.Encode(w, img, &jpeg.Options{Quality: 95}); err != nil {
			log.Printf("Erro ao codificar JPEG sem metadados: %v", err)
		}
	case "png":
		setDownloadHeaders(w, "image/png", "sem-metadados.png")
		if err := png.Encode(w, img); err != nil {
			log.Printf("Erro ao codificar PNG sem metadados: %v", err)
		}
	default:
		http.Error(w, "A remoção de metadados aceita apenas JPG e PNG", http.StatusBadRequest)
	}
}

func handleImgRotate(w http.ResponseWriter, r *http.Request) {
	if !parseMultipartForm(w, r, maxUploadSize) {
		return
	}
	defer cleanupMultipartForm(r)

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Imagem não enviada", http.StatusBadRequest)
		return
	}
	defer file.Close()

	img, format, err := decodeImage(file)
	if err != nil {
		http.Error(w, "Imagem inválida ou acima dos limites permitidos", http.StatusBadRequest)
		return
	}

	var result image.Image
	switch r.FormValue("action") {
	case "rot90":
		result = imaging.Rotate90(img)
	case "rot180":
		result = imaging.Rotate180(img)
	case "rot270":
		result = imaging.Rotate270(img)
	case "fliph":
		result = imaging.FlipH(img)
	case "flipv":
		result = imaging.FlipV(img)
	default:
		http.Error(w, "Ação de rotação inválida", http.StatusBadRequest)
		return
	}

	if format == "jpeg" {
		setDownloadHeaders(w, "image/jpeg", "imagem-rotacionada.jpg")
		if err := jpeg.Encode(w, result, &jpeg.Options{Quality: 95}); err != nil {
			log.Printf("Erro ao codificar JPEG rotacionado: %v", err)
		}
		return
	}

	setDownloadHeaders(w, "image/png", "imagem-rotacionada.png")
	if err := png.Encode(w, result); err != nil {
		log.Printf("Erro ao codificar PNG rotacionado: %v", err)
	}
}

func handleBase64(w http.ResponseWriter, r *http.Request) {
	if !parseMultipartForm(w, r, maxUploadSize) {
		return
	}
	defer cleanupMultipartForm(r)

	switch r.FormValue("mode") {
	case "encode":
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Arquivo não enviado", http.StatusBadRequest)
			return
		}
		defer file.Close()

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		encoder := base64.NewEncoder(base64.StdEncoding, w)
		if _, err := io.Copy(encoder, file); err != nil {
			log.Printf("Erro ao codificar Base64: %v", err)
			_ = encoder.Close()
			return
		}
		if err := encoder.Close(); err != nil {
			log.Printf("Erro ao finalizar Base64: %v", err)
		}
	case "decode":
		encoded := strings.TrimSpace(r.FormValue("text"))
		if encoded == "" {
			http.Error(w, "Texto Base64 vazio", http.StatusBadRequest)
			return
		}
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			http.Error(w, "Texto Base64 inválido", http.StatusBadRequest)
			return
		}

		setDownloadHeaders(w, "application/octet-stream", "decoded.bin")
		// #nosec G705 -- resposta binária é attachment, application/octet-stream e nosniff.
		_, _ = w.Write(decoded)
	default:
		http.Error(w, "Modo Base64 inválido", http.StatusBadRequest)
	}
}

func handleMinify(w http.ResponseWriter, r *http.Request) {
	if !parseMultipartForm(w, r, maxUploadSize) {
		return
	}
	defer cleanupMultipartForm(r)

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Arquivo não enviado", http.StatusBadRequest)
		return
	}
	defer file.Close()

	language := r.FormValue("lang")
	if language == "" {
		switch strings.ToLower(filepath.Ext(header.Filename)) {
		case ".html", ".htm":
			language = "text/html"
		case ".css":
			language = "text/css"
		case ".js", ".mjs":
			language = "application/javascript"
		case ".json":
			language = "application/json"
		}
	}

	allowed := map[string]bool{
		"text/html":              true,
		"text/css":               true,
		"application/javascript": true,
		"application/json":       true,
	}
	if !allowed[language] {
		http.Error(w, "Formato não suportado; use HTML, CSS, JS ou JSON", http.StatusBadRequest)
		return
	}

	minifier := minify.New()
	minifier.AddFunc("text/html", html.Minify)
	minifier.AddFunc("text/css", css.Minify)
	minifier.AddFunc("application/javascript", js.Minify)
	minifier.AddFunc("application/json", json.Minify)

	var output bytes.Buffer
	if err := minifier.Minify(language, &output, file); err != nil {
		http.Error(w, "Arquivo inválido para o formato selecionado", http.StatusBadRequest)
		return
	}

	setDownloadHeaders(w, "text/plain; charset=utf-8", "minified_"+safeFilename(header.Filename))
	_, _ = output.WriteTo(w)
}

func handlePdfRotate(w http.ResponseWriter, r *http.Request) {
	if !parseMultipartForm(w, r, maxPDFUploadSize) {
		return
	}
	defer cleanupMultipartForm(r)

	file, _, err := r.FormFile("pdf")
	if err != nil {
		http.Error(w, "PDF não enviado", http.StatusBadRequest)
		return
	}
	defer file.Close()

	degrees, err := strconv.Atoi(r.FormValue("degrees"))
	if err != nil || (degrees != 90 && degrees != 180 && degrees != 270) {
		http.Error(w, "Rotação deve ser 90, 180 ou 270 graus", http.StatusBadRequest)
		return
	}

	pages := strings.TrimSpace(r.FormValue("pages"))
	if len(pages) > 200 {
		http.Error(w, "Seleção de páginas muito longa", http.StatusBadRequest)
		return
	}
	var pageSelection []string
	if pages != "" {
		pageSelection = []string{pages}
	}

	output, err := os.CreateTemp("", "canivete-rotated-*.pdf")
	if err != nil {
		internalError(w, "Erro ao preparar PDF de saída", err)
		return
	}
	defer func() {
		_ = output.Close()
		_ = os.Remove(output.Name())
	}()

	if err := api.Rotate(file, output, degrees, pageSelection, nil); err != nil {
		log.Printf("Erro ao girar PDF: %v", err)
		http.Error(w, "PDF ou seleção de páginas inválida", http.StatusBadRequest)
		return
	}
	if err := serveDownloadFile(w, output, "application/pdf", "rotated.pdf"); err != nil {
		log.Printf("Erro ao enviar PDF rotacionado: %v", err)
	}
}

func writeICO(w io.Writer, images [][]byte) error {
	if len(images) == 0 || len(images) > int(^uint16(0)) {
		return fmt.Errorf("quantidade inválida de imagens no ICO: %d", len(images))
	}
	headerSize := 6 + 16*len(images)
	if uint64(headerSize) > uint64(^uint32(0)) {
		return fmt.Errorf("cabeçalho ICO excede o limite de 32 bits")
	}

	if err := binary.Write(w, binary.LittleEndian, uint16(0)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(1)); err != nil {
		return err
	}
	// #nosec G115 -- len(images) foi validado contra o limite de uint16 acima.
	if err := binary.Write(w, binary.LittleEndian, uint16(len(images))); err != nil {
		return err
	}

	// #nosec G115 -- headerSize foi validado contra o limite de uint32 acima.
	offset := uint32(headerSize)
	for _, data := range images {
		config, err := png.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return err
		}
		if config.Width < 1 || config.Width > 256 || config.Height < 1 || config.Height > 256 {
			return fmt.Errorf("dimensão inválida no ICO: %dx%d", config.Width, config.Height)
		}
		if uint64(len(data)) > uint64(^uint32(0)) {
			return fmt.Errorf("imagem interna do ICO excede o limite de 32 bits")
		}
		// O formato ICO representa a dimensão 256 pelo byte zero.
		// #nosec G115 -- dimensões validadas no intervalo 1..256.
		entry := []byte{byte(config.Width % 256), byte(config.Height % 256), 0, 0}
		if _, err := w.Write(entry); err != nil {
			return err
		}
		for _, value := range []uint16{1, 32} {
			if err := binary.Write(w, binary.LittleEndian, value); err != nil {
				return err
			}
		}
		// #nosec G115 -- tamanho validado contra o limite de uint32 acima.
		dataSize := uint32(len(data))
		if err := binary.Write(w, binary.LittleEndian, dataSize); err != nil {
			return err
		}
		if err := binary.Write(w, binary.LittleEndian, offset); err != nil {
			return err
		}
		if uint64(offset)+uint64(dataSize) > uint64(^uint32(0)) {
			return fmt.Errorf("arquivo ICO excede o limite de 32 bits")
		}
		offset += dataSize
	}

	for _, data := range images {
		if _, err := w.Write(data); err != nil {
			return err
		}
	}
	return nil
}

func handleImgToIco(w http.ResponseWriter, r *http.Request) {
	if !parseMultipartForm(w, r, maxUploadSize) {
		return
	}
	defer cleanupMultipartForm(r)

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Imagem não enviada", http.StatusBadRequest)
		return
	}
	defer file.Close()

	img, _, err := decodeImage(file)
	if err != nil {
		http.Error(w, "Imagem inválida ou acima dos limites permitidos", http.StatusBadRequest)
		return
	}

	sizes := []int{16, 32, 48}
	pngImages := make([][]byte, 0, len(sizes))
	for _, size := range sizes {
		resized := imaging.Fill(img, size, size, imaging.Center, imaging.Lanczos)
		var output bytes.Buffer
		if err := png.Encode(&output, resized); err != nil {
			internalError(w, "Erro ao gerar ícone", err)
			return
		}
		pngImages = append(pngImages, output.Bytes())
	}

	var ico bytes.Buffer
	if err := writeICO(&ico, pngImages); err != nil {
		internalError(w, "Erro ao gerar ícone", err)
		return
	}
	setDownloadHeaders(w, "image/x-icon", "favicon.ico")
	_, _ = ico.WriteTo(w)
}

func handlePdfWatermark(w http.ResponseWriter, r *http.Request) {
	if !parseMultipartForm(w, r, maxPDFUploadSize) {
		return
	}
	defer cleanupMultipartForm(r)

	file, _, err := r.FormFile("pdf")
	if err != nil {
		http.Error(w, "PDF não enviado", http.StatusBadRequest)
		return
	}
	defer file.Close()

	watermarkText := strings.TrimSpace(r.FormValue("text"))
	if watermarkText == "" {
		watermarkText = "Watermark"
	}
	if len([]rune(watermarkText)) > 200 {
		http.Error(w, "Texto da marca d'água deve ter no máximo 200 caracteres", http.StatusBadRequest)
		return
	}

	watermark, err := api.TextWatermark(
		watermarkText,
		"font:Helvetica, points:48, rot:45, scale:1.0 abs, op:0.3",
		true,
		false,
		types.POINTS,
	)
	if err != nil {
		http.Error(w, "Texto de marca d'água inválido", http.StatusBadRequest)
		return
	}

	output, err := os.CreateTemp("", "canivete-watermarked-*.pdf")
	if err != nil {
		internalError(w, "Erro ao preparar PDF de saída", err)
		return
	}
	defer func() {
		_ = output.Close()
		_ = os.Remove(output.Name())
	}()

	if err := api.AddWatermarks(file, output, nil, watermark, nil); err != nil {
		log.Printf("Erro ao adicionar marca d'água: %v", err)
		http.Error(w, "PDF inválido ou não suportado", http.StatusBadRequest)
		return
	}
	if err := serveDownloadFile(w, output, "application/pdf", "watermarked.pdf"); err != nil {
		log.Printf("Erro ao enviar PDF com marca d'água: %v", err)
	}
}
