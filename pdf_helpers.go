package main

import (
	"io"
	"log"
	"net/http"
	"os"
)

// The caller parses and cleans up the multipart form. Every single-PDF rewrite
// shares the same page validation and bounded output path.
func transformPDF(w http.ResponseWriter, r *http.Request, filename string, transform func(io.ReadSeeker, io.Writer) error) {
	file, _, err := r.FormFile("pdf")
	if err != nil {
		http.Error(w, "PDF não enviado", http.StatusBadRequest)
		return
	}
	defer file.Close()
	input := contextReadSeeker{ctx: r.Context(), ReadSeeker: file}
	if _, err := validatePDFPages(input); err != nil {
		http.Error(w, "PDF inválido ou excede o limite de 50 páginas", http.StatusBadRequest)
		return
	}
	output, err := os.CreateTemp("", "canivete-output-*.pdf")
	if err != nil {
		internalError(w, "Erro ao preparar PDF de saída", err)
		return
	}
	defer os.Remove(output.Name())
	defer output.Close()
	if err := writeLimitedPDF(output, func(writer io.Writer) error { return transform(input, writer) }); err != nil {
		log.Printf("Erro ao processar PDF: %v", err)
		http.Error(w, "PDF inválido, opção não suportada ou saída excede 64 MiB", http.StatusBadRequest)
		return
	}
	if r.Context().Err() != nil {
		http.Error(w, "Processamento cancelado", http.StatusRequestTimeout)
		return
	}
	if err := serveDownloadFile(w, output, "application/pdf", filename); err != nil {
		log.Printf("Erro ao enviar PDF: %v", err)
	}
}
