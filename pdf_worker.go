package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const pdfJobTimeout = 3 * time.Minute

func pdfHandlers() map[string]http.HandlerFunc {
	return map[string]http.HandlerFunc{
		"img-to-pdf": handleImgToPdf, "pdf-to-img": handlePdfToImg,
		"pdf-rasterize": handlePdfRasterize, "pdf-merge": handlePdfMerge,
		"pdf-split": handlePdfSplit, "pdf-protect": handlePdfProtect,
		"pdf-optimize": handlePdfOptimize, "pdf-rotate": handlePdfRotate,
		"pdf-watermark": handlePdfWatermark,
	}
}

// Only the parent creates these paths. The protocol travels over stdin, so PDF
// passwords and form values never appear in process arguments or logs.
type pdfWorkerRequest struct {
	Operation   string
	ContentType string
	BodyPath    string
}

type pdfWorkerResponse struct {
	Status int
	Header http.Header
}

// PDF libraries do not expose interruptible CPU operations. A separate process
// lets the parent enforce deadlines even while the library is computing.
func isolatePDF(operation string, timeout time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Err() != nil {
			http.Error(w, "Requisição cancelada", http.StatusRequestTimeout)
			return
		}
		directory, err := os.MkdirTemp("", "canivete-job-*")
		if err != nil {
			internalError(w, "Erro ao preparar processamento", err)
			return
		}
		defer os.RemoveAll(directory)
		workDirectory := filepath.Join(directory, "work")
		if err := os.Mkdir(workDirectory, 0700); err != nil {
			internalError(w, "Erro ao preparar processamento", err)
			return
		}
		body, err := os.CreateTemp(directory, "request-*")
		if err != nil {
			internalError(w, "Erro ao preparar upload", err)
			return
		}
		maximum := maxPDFUploadSize
		if operation == "pdf-merge" || operation == "img-to-pdf" {
			maximum = maxBatchUploadSize
		}
		r.Body = http.MaxBytesReader(w, r.Body, maximum)
		_, copyErr := io.Copy(body, &contextReader{ctx: r.Context(), reader: r.Body})
		closeErr := body.Close()
		if copyErr != nil {
			var tooLarge *http.MaxBytesError
			status := http.StatusBadRequest
			if r.Context().Err() != nil {
				status = http.StatusRequestTimeout
			}
			if errors.As(copyErr, &tooLarge) {
				status = http.StatusRequestEntityTooLarge
			}
			http.Error(w, "Upload inválido, cancelado ou acima do limite permitido", status)
			return
		}
		if closeErr != nil {
			internalError(w, "Erro ao salvar upload", closeErr)
			return
		}
		result, err := os.CreateTemp(directory, "response-*")
		if err != nil {
			internalError(w, "Erro ao preparar resultado", err)
			return
		}
		defer result.Close()
		payload, err := json.Marshal(pdfWorkerRequest{operation, r.Header.Get("Content-Type"), body.Name()})
		if err != nil {
			internalError(w, "Erro ao preparar processamento", err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		executable, err := os.Executable()
		if err != nil {
			internalError(w, "Erro ao localizar processador", err)
			return
		}
		cmd := exec.CommandContext(ctx, executable, "--pdf-worker")
		cmd.Stdin = bytes.NewReader(payload)
		boundedOutput := &limitedWriter{writer: result, remaining: maxGeneratedBytes + (64 << 10)}
		cmd.Stdout = &cancelOnErrorWriter{writer: boundedOutput, cancel: cancel}
		var diagnostics bytes.Buffer
		boundedDiagnostics := &limitedWriter{writer: &diagnostics, remaining: 64 << 10}
		cmd.Stderr = &cancelOnErrorWriter{writer: boundedDiagnostics, cancel: cancel}
		for _, variable := range os.Environ() {
			if !strings.HasPrefix(variable, "TMPDIR=") {
				cmd.Env = append(cmd.Env, variable)
			}
		}
		cmd.Env = append(cmd.Env, "TMPDIR="+workDirectory)
		configureWorkerProcess(cmd)
		if err := cmd.Run(); err != nil {
			if errors.Is(boundedOutput.err, errGeneratedLimit) {
				http.Error(w, "Saída PDF excede o limite permitido", http.StatusBadRequest)
				return
			}
			if ctx.Err() != nil {
				status := http.StatusRequestTimeout
				if r.Context().Err() == nil {
					status = http.StatusGatewayTimeout
				}
				http.Error(w, "Processamento de PDF cancelado ou excedeu o tempo permitido", status)
				return
			}
			log.Printf("Falha no processador PDF %q: %v: %q", operation, err, diagnostics.String())
			http.Error(w, "Não foi possível concluir o processamento do PDF", http.StatusInternalServerError)
			return
		}
		if _, err := result.Seek(0, io.SeekStart); err != nil {
			internalError(w, "Erro ao ler resultado", err)
			return
		}
		reader := bufio.NewReader(result)
		metadata, err := reader.ReadBytes('\n')
		var response pdfWorkerResponse
		if err != nil || len(metadata) > 64<<10 || json.Unmarshal(metadata, &response) != nil || response.Status < 200 || response.Status > 599 {
			internalError(w, "Resposta inválida do processador", errors.New("invalid worker response"))
			return
		}
		for key, values := range response.Header {
			w.Header()[key] = values
		}
		w.WriteHeader(response.Status)
		if _, err := io.Copy(w, reader); err != nil {
			log.Printf("Erro ao enviar resultado PDF: %v", err)
		}
	})
}

type cancelOnErrorWriter struct {
	writer io.Writer
	cancel context.CancelFunc
}

func (w *cancelOnErrorWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	if err != nil {
		w.cancel()
	}
	return n, err
}

type workerResponseWriter struct {
	header http.Header
	writer io.Writer
	wrote  bool
	err    error
}

func (w *workerResponseWriter) Header() http.Header { return w.header }

func (w *workerResponseWriter) WriteHeader(status int) {
	if w.wrote {
		return
	}
	w.wrote = true
	w.err = json.NewEncoder(w.writer).Encode(pdfWorkerResponse{status, w.header})
}

func (w *workerResponseWriter) Write(p []byte) (int, error) {
	w.WriteHeader(http.StatusOK)
	if w.err != nil {
		return 0, w.err
	}
	return w.writer.Write(p)
}

func runPDFWorker(input io.Reader, output io.Writer) error {
	var job pdfWorkerRequest
	if err := json.NewDecoder(io.LimitReader(input, 64<<10)).Decode(&job); err != nil {
		return err
	}
	handler, ok := pdfHandlers()[job.Operation]
	if !ok {
		return errors.New("unknown PDF operation")
	}
	file, err := os.Open(job.BodyPath)
	if err != nil {
		return err
	}
	defer file.Close()
	r, err := http.NewRequest(http.MethodPost, "http://worker/process/"+job.Operation, file)
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", job.ContentType)
	w := &workerResponseWriter{header: make(http.Header), writer: output}
	handler(w, r)
	w.WriteHeader(http.StatusOK)
	return w.err
}

func pdfWorkerMain() int {
	if err := runPDFWorker(os.Stdin, os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
