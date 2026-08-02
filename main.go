package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
)

//go:embed static/*
var staticFS embed.FS

//go:embed templates/*
var templatesFS embed.FS

var tmpl *template.Template

func init() {
	var err error
	tmpl, err = template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		log.Fatalf("Erro ao carregar templates: %v", err)
	}
}

type PageData struct {
	Lang string
}

func (p PageData) T(k string) string {
	return translate(p.Lang, k)
}

func main() {
	mux := http.NewServeMux()

	// Servir assets estáticos embutidos
	mux.Handle("/static/", http.FileServer(http.FS(staticFS)))

	// Carregar traduções
	loadLocales()

	tmpl := template.Must(template.ParseFS(templatesFS, "templates/index.html"))

	// Rota principal (UI)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		lang := getLang(r)
		data := PageData{Lang: lang}

		err := tmpl.Execute(w, data)
		if err != nil {
			log.Printf("Erro ao executar template: %v", err)
		}
	})

	// Rotas de processamento
	mux.HandleFunc("/process/remove-bg", handleRemoveBg)
	mux.HandleFunc("/process/resize", handleResize)
	mux.HandleFunc("/process/crop", handleCrop)
	mux.HandleFunc("/process/convert", handleConvert)
	mux.HandleFunc("/process/img-to-pdf", handleImgToPdf)
	mux.HandleFunc("/process/pdf-to-img", handlePdfToImg)
	mux.HandleFunc("/process/pdf-rasterize", handlePdfRasterize) // Converte as páginas do PDF para Imagem
	mux.HandleFunc("/process/svg-to-img", handleSvgToImg) // Renderiza SVG para PNG

	// Novas rotas extras
	mux.HandleFunc("/process/pdf-merge", handlePdfMerge)
	mux.HandleFunc("/process/pdf-split", handlePdfSplit)
	mux.HandleFunc("/process/pdf-protect", handlePdfProtect)
	mux.HandleFunc("/process/pdf-optimize", handlePdfOptimize)
	mux.HandleFunc("/process/img-compress", handleImgCompress)
	mux.HandleFunc("/process/img-palette", handleImgPalette)
	mux.HandleFunc("/process/qr-generate", handleQrGenerate)
	mux.HandleFunc("/process/qr-read", handleQrRead)

	// Extras 2.0 (Security & Dev)
	mux.HandleFunc("/process/img-exif-strip", handleImgExifStrip)
	mux.HandleFunc("/process/img-rotate", handleImgRotate)
	mux.HandleFunc("/process/img-to-ico", handleImgToIco)
	mux.HandleFunc("/process/pdf-rotate", handlePdfRotate)
	mux.HandleFunc("/process/pdf-watermark", handlePdfWatermark)
	mux.HandleFunc("/process/base64", handleBase64)
	mux.HandleFunc("/process/minify", handleMinify)

	// Iniciar servidor
	port := "7001"
	log.Printf("Servidor Canivete da Mata rodando na porta %s...", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
