# Stage 1: Builder
FROM golang:1.26.5-alpine AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG TARGETVARIANT

WORKDIR /app

# Copiar os arquivos de definição do módulo para cache de dependências
COPY go.mod go.sum ./
RUN go mod download

# Copiar o código fonte restante
COPY . .

# Compilar estaticamente para a plataforma solicitada pelo Docker Buildx.
RUN CGO_ENABLED=0 \
    GOOS="${TARGETOS}" \
    GOARCH="${TARGETARCH}" \
    GOARM="${TARGETVARIANT#v}" \
    go build -trimpath -ldflags="-w -s" -o canivete .

# Stage 2: imagem mínima com os renderizadores usados pela aplicação.
FROM alpine:3.24.1

WORKDIR /app

# Instalar dependências de runtime e criar um usuário sem privilégios.
RUN apk --no-cache add ca-certificates tzdata poppler-utils rsvg-convert \
    && addgroup -S canivete \
    && adduser -S -D -H -G canivete canivete

# Copiar apenas o binário compilado do Stage 1
COPY --from=builder --chown=canivete:canivete /app/canivete .

USER canivete

# Expor a porta 7001
EXPOSE 7001

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:7001/healthz || exit 1

# Executar a aplicação
CMD ["./canivete"]
