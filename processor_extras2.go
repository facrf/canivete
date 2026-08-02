package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/disintegration/imaging"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/json"
)

// handleImgExifStrip reads an image and re-encodes it to strip EXIF
func handleImgExifStrip(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Erro ao processar form", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Imagem não enviada", http.StatusBadRequest)
		return
	}
	defer file.Close()

	img, format, err := image.Decode(file)
	if err != nil {
		http.Error(w, "Erro ao decodificar imagem", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"stripped_%s\"", header.Filename))
	
	if format == "jpeg" || format == "jpg" {
		w.Header().Set("Content-Type", "image/jpeg")
		jpeg.Encode(w, img, &jpeg.Options{Quality: 95})
	} else {
		w.Header().Set("Content-Type", "image/png")
		png.Encode(w, img)
	}
}

// handleImgRotate rotates or flips an image
func handleImgRotate(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Erro ao processar form", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Imagem não enviada", http.StatusBadRequest)
		return
	}
	defer file.Close()

	action := r.FormValue("action")
	
	img, format, err := image.Decode(file)
	if err != nil {
		http.Error(w, "Erro ao decodificar imagem", http.StatusBadRequest)
		return
	}

	var res image.Image
	switch action {
	case "rot90":
		res = imaging.Rotate90(img)
	case "rot180":
		res = imaging.Rotate180(img)
	case "rot270":
		res = imaging.Rotate270(img)
	case "fliph":
		res = imaging.FlipH(img)
	case "flipv":
		res = imaging.FlipV(img)
	default:
		res = img
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"rotated_%s\"", header.Filename))
	if format == "jpeg" || format == "jpg" {
		w.Header().Set("Content-Type", "image/jpeg")
		jpeg.Encode(w, res, &jpeg.Options{Quality: 95})
	} else {
		w.Header().Set("Content-Type", "image/png")
		png.Encode(w, res)
	}
}

// handleBase64 process Base64 encode and decode
func handleBase64(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Erro ao processar form", http.StatusBadRequest)
		return
	}
	
	mode := r.FormValue("mode")
	if mode == "encode" {
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Arquivo não enviado", http.StatusBadRequest)
			return
		}
		defer file.Close()
		
		data, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "Erro ao ler arquivo", http.StatusInternalServerError)
			return
		}
		encoded := base64.StdEncoding.EncodeToString(data)
		
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(encoded))
		
	} else if mode == "decode" {
		text := r.FormValue("text")
		decoded, err := base64.StdEncoding.DecodeString(text)
		if err != nil {
			http.Error(w, "Texto Base64 inválido", http.StatusBadRequest)
			return
		}
		
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename=\"decoded.bin\"")
		w.Write(decoded)
	}
}

// handleMinify minifies code
func handleMinify(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Erro ao processar form", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Arquivo não enviado", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename)
	lang := r.FormValue("lang")
	if lang == "" {
		switch ext {
		case ".html": lang = "text/html"
		case ".css": lang = "text/css"
		case ".js": lang = "application/javascript"
		case ".json": lang = "application/json"
		}
	}
	
	m := minify.New()
	m.AddFunc("text/html", html.Minify)
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("application/javascript", js.Minify)
	m.AddFunc("application/json", json.Minify)
	
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"minified_%s\"", header.Filename))
	w.Header().Set("Content-Type", "text/plain")
	
	if err := m.Minify(lang, w, file); err != nil {
		http.Error(w, "Erro ao minificar. Suporte apenas a HTML, CSS, JS e JSON.", http.StatusInternalServerError)
	}
}

// handlePdfRotate rotates PDF pages
func handlePdfRotate(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(50 << 20)
	if err != nil {
		http.Error(w, "Erro ao ler form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("pdf")
	if err != nil {
		http.Error(w, "PDF não enviado", http.StatusBadRequest)
		return
	}
	defer file.Close()
	
	pages := r.FormValue("pages") // Ex: "1,2,3" ou vazio para todas
	degreesStr := r.FormValue("degrees")
	degrees, _ := strconv.Atoi(degreesStr)
	if degrees == 0 {
		degrees = 90
	}
	
	var pageSelection []string
	if pages != "" {
		pageSelection = []string{pages}
	}
	
	var buf bytes.Buffer
	err = api.Rotate(file, &buf, degrees, pageSelection, nil)
	if err != nil {
		http.Error(w, "Erro ao girar PDF: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Disposition", "attachment; filename=\"rotated.pdf\"")
	w.Header().Set("Content-Type", "application/pdf")
	w.Write(buf.Bytes())
}

// writeICOHeader generates ICO
func writeICOHeader(w io.Writer, pngs [][]byte) error {
	// ICO Header
	w.Write([]byte{0, 0, 1, 0})
	count := len(pngs)
	w.Write([]byte{byte(count), 0})
	
	offset := 6 + (16 * count)
	
	for _, p := range pngs {
		cfg, err := png.DecodeConfig(bytes.NewReader(p))
		if err != nil { return err }
		w.Write([]byte{byte(cfg.Width), byte(cfg.Height), 0, 0, 1, 0, 32, 0})
		size := len(p)
		w.Write([]byte{byte(size), byte(size >> 8), byte(size >> 16), byte(size >> 24)})
		w.Write([]byte{byte(offset), byte(offset >> 8), byte(offset >> 16), byte(offset >> 24)})
		offset += size
	}
	for _, p := range pngs {
		w.Write(p)
	}
	return nil
}

// handleImgToIco
func handleImgToIco(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Erro form", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Imagem vazia", http.StatusBadRequest)
		return
	}
	defer file.Close()
	
	img, _, err := image.Decode(file)
	if err != nil {
		http.Error(w, "Erro decode", http.StatusBadRequest)
		return
	}
	
	sizes := []int{16, 32, 48}
	var pngs [][]byte
	for _, s := range sizes {
		resized := imaging.Resize(img, s, s, imaging.Lanczos)
		var buf bytes.Buffer
		png.Encode(&buf, resized)
		pngs = append(pngs, buf.Bytes())
	}
	
	w.Header().Set("Content-Disposition", "attachment; filename=\"favicon.ico\"")
	w.Header().Set("Content-Type", "image/x-icon")
	writeICOHeader(w, pngs)
}

// handlePdfWatermark
func handlePdfWatermark(w http.ResponseWriter, r *http.Request) {
    err := r.ParseMultipartForm(50 << 20)
	if err != nil {
		http.Error(w, "Erro ao ler form", http.StatusBadRequest)
		return
	}
    file, _, err := r.FormFile("pdf")
	if err != nil {
		http.Error(w, "PDF não enviado", http.StatusBadRequest)
		return
	}
	defer file.Close()
    
    text := r.FormValue("text")
    if text == "" { text = "Watermark" }
    
    wm, _ := api.TextWatermark(text, "font:Helvetica, points:48, rot:45, scale:1.0 abs, op:0.3", true, false, 1) // 1 = POINTS
    var buf bytes.Buffer
    err = api.AddWatermarks(file, &buf, nil, wm, nil)
    if err != nil {
		http.Error(w, "Erro ao adicionar marca d'água: "+err.Error(), http.StatusInternalServerError)
		return
	}
    
    w.Header().Set("Content-Disposition", "attachment; filename=\"watermarked.pdf\"")
	w.Header().Set("Content-Type", "application/pdf")
	w.Write(buf.Bytes())
}
