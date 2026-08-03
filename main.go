package main

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

//go:embed static/*
var staticFS embed.FS

//go:embed templates/*
var templatesFS embed.FS

var tmpl *template.Template

func init() {
	// A aplicação usa apenas fontes PDF padrão e não precisa gravar configuração
	// no diretório pessoal do usuário do contêiner.
	api.DisableConfigDir()

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

func newHandler() http.Handler {
	loadLocales()

	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.FileServer(http.FS(staticFS)))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		var page bytes.Buffer
		if err := tmpl.ExecuteTemplate(&page, "index.html", PageData{Lang: getLang(r)}); err != nil {
			internalError(w, "Erro ao renderizar página", err)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = page.WriteTo(w)
	})

	processingMux := http.NewServeMux()
	processingMux.HandleFunc("POST /process/remove-bg", handleRemoveBg)
	processingMux.HandleFunc("POST /process/resize", handleResize)
	processingMux.HandleFunc("POST /process/crop", handleCrop)
	processingMux.HandleFunc("POST /process/convert", handleConvert)
	processingMux.HandleFunc("POST /process/img-to-pdf", handleImgToPdf)
	processingMux.HandleFunc("POST /process/pdf-to-img", handlePdfToImg)
	processingMux.HandleFunc("POST /process/pdf-rasterize", handlePdfRasterize)
	processingMux.HandleFunc("POST /process/svg-to-img", handleSvgToImg)
	processingMux.HandleFunc("POST /process/pdf-merge", handlePdfMerge)
	processingMux.HandleFunc("POST /process/pdf-split", handlePdfSplit)
	processingMux.HandleFunc("POST /process/pdf-protect", handlePdfProtect)
	processingMux.HandleFunc("POST /process/pdf-optimize", handlePdfOptimize)
	processingMux.HandleFunc("POST /process/img-compress", handleImgCompress)
	processingMux.HandleFunc("POST /process/img-palette", handleImgPalette)
	processingMux.HandleFunc("POST /process/qr-generate", handleQrGenerate)
	processingMux.HandleFunc("POST /process/qr-read", handleQrRead)
	processingMux.HandleFunc("POST /process/img-exif-strip", handleImgExifStrip)
	processingMux.HandleFunc("POST /process/img-rotate", handleImgRotate)
	processingMux.HandleFunc("POST /process/img-to-ico", handleImgToIco)
	processingMux.HandleFunc("POST /process/pdf-rotate", handlePdfRotate)
	processingMux.HandleFunc("POST /process/pdf-watermark", handlePdfWatermark)
	processingMux.HandleFunc("POST /process/base64", handleBase64)
	processingMux.HandleFunc("POST /process/minify", handleMinify)
	mux.Handle("/process/", limitConcurrentJobs(processingMux, 2))

	return recoverMiddleware(securityHeaders(mux))
}

func limitConcurrentJobs(next http.Handler, maximum int) http.Handler {
	semaphore := make(chan struct{}, maximum)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case semaphore <- struct{}{}:
			defer func() { <-semaphore }()
			next.ServeHTTP(w, r)
		default:
			w.Header().Set("Retry-After", "2")
			http.Error(w, "Servidor ocupado; tente novamente em instantes", http.StatusServiceUnavailable)
		}
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' blob: data:; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				// #nosec G706 -- método e caminho usam %q, que escapa quebras e controles no log.
				log.Printf("Pânico recuperado em %q %q: %v", r.Method, r.URL.Path, recovered)
				http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func serverPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		return "7001"
	}
	value, err := strconv.Atoi(port)
	if err != nil || value < 1 || value > 65535 {
		log.Fatal("PORT inválida; use um número entre 1 e 65535")
	}
	return port
}

func main() {
	port := serverPort()
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           newHandler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       2 * time.Minute,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	shutdownContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-shutdownContext.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Erro durante desligamento do servidor: %v", err)
		}
	}()

	log.Printf("Servidor Canivete da Mata rodando na porta %s...", port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
