# Stage 1: Builder
FROM golang:1.22-alpine AS builder

# Instalar dependências para compilação (git é comum para baixar módulos)
RUN apk add --no-cache git

WORKDIR /app

# Copiar os arquivos de definição do módulo para cache de dependências
COPY go.mod go.sum ./
RUN go mod download

# Copiar o código fonte restante
COPY . .

# Compilar o binário de forma totalmente estática e otimizada
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -ldflags="-w -s" -o canivete .

# Stage 2: Final (Imagem Mínima com poppler-utils para manipulação de PDF)
FROM alpine:latest

WORKDIR /app

# Instalar pacotes necessários (poppler-utils provê o comando 'pdftoppm' para rasterizar PDF)
RUN apk --no-cache add ca-certificates tzdata poppler-utils

# Copiar apenas o binário compilado do Stage 1
COPY --from=builder /app/canivete .

# Expor a porta 7001
EXPOSE 7001

# Executar a aplicação
CMD ["./canivete"]
