# Canivete da Mata 🪓

> Ferramenta web offline, leve e rápida para processamento de imagens e PDFs.  
> Construída em Go. O contêiner inclui `librsvg` para SVG e `poppler-utils` para rasterização de PDF.

[![Build & Push GHCR](https://github.com/facrf/canivete/actions/workflows/docker-ghcr.yml/badge.svg)](https://github.com/facrf/canivete/actions/workflows/docker-ghcr.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

---

## 🌍 Documentação por Idioma

| Idioma | Link |
|--------|------|
| 🇧🇷 Português | [README_pt.md](README_pt.md) |
| 🇺🇸 English | [README_en.md](README_en.md) |
| 🇪🇸 Español | [README_es.md](README_es.md) |
| 🇫🇷 Français | [README_fr.md](README_fr.md) |
| 🇩🇪 Deutsch | [README_de.md](README_de.md) |
| 🇷🇺 Русский | [README_ru.md](README_ru.md) |
| 🇨🇳 中文 | [README_zh.md](README_zh.md) |

---

## ✨ Funcionalidades

| Categoria | Ferramentas |
|-----------|------------|
| 🖼️ **Imagem** | Remover fundo, recorte interativo, redimensionar, converter (PNG/JPG/BMP), comprimir |
| 🎨 **Cores** | Extrair paleta de cores (HEX) |
| 📄 **PDF** | Imagens → PDF, juntar PDFs, dividir PDF, otimizar, proteger com senha |
| 🔁 **Conversão** | PDF → Imagens, SVG → PNG |
| 📱 **QR Code** | Gerar e decodificar QR Codes e códigos de barras |

---

## ⚙️ Configuração (Variáveis de Ambiente)

O sistema pode ser customizado através das seguintes variáveis de ambiente:

| Variável | Padrão | Descrição |
|----------|---------|-----------|
| `PORT` | `7001` | Porta em que o servidor web irá rodar. |
| `MAX_CONCURRENT_JOBS` | `100` | Limite de processamentos simultâneos (Rate Limiter) para evitar sobrecarga do servidor. |

---

## 🚀 Deploy Rápido

### Docker (linha de comando)
```bash
docker run -d \
  --name canivete-da-mata \
  -p 7001:7001 \
  --restart unless-stopped \
  ghcr.io/facrf/canivete:latest
```
Acesse em: **http://localhost:7001**

### Docker Compose / Portainer Stack
Consulte o arquivo [`docker-compose.yml`](docker-compose.yml) — compatível com Portainer.

No Portainer:
1. Stacks → **Add Stack** → Web editor
2. Cole o conteúdo de `docker-compose.yml`
3. Clique em **Deploy the stack**

### Build local (a partir do código-fonte)
```bash
git clone http://192.168.0.10:3010/facrf/canivete.git
cd canivete
docker build -t canivete-da-mata:latest .
docker run -d -p 7001:7001 --name canivete-da-mata canivete-da-mata:latest
```

---

## 🏗️ Arquiteturas suportadas (Docker)

A imagem no GHCR é publicada para múltiplas arquiteturas automaticamente via GitHub Actions:

| Plataforma | Hardware |
|---|---|
| `linux/amd64` | Servidores x86_64, PCs, VMs |
| `linux/arm64` | Raspberry Pi 4/5 (64-bit), Apple Silicon (M1/M2/M3) |
| `linux/arm/v7` | Raspberry Pi 2/3/4 (32-bit), Orange Pi |
| `linux/riscv64` | VisionFive 2, StarFive, Milk-V Mars |

---

## 🔒 Segurança

- Sem banco de dados — imune a SQL Injection
- Templates Go com escaping automático de HTML/XSS
- Upload limitado com `http.MaxBytesReader`
- Validação de dimensões (máximo de 8000px e 20 megapixels)
- Validação de argumentos antes de passar a processos externos
- Senhas de PDF exigem mínimo de 8 caracteres

Consulte [SECURITY.md](SECURITY.md) para reportar vulnerabilidades.

---

## 🧪 Testes e Qualidade de Código

O Canivete da Mata conta com uma cobertura completa de testes garantindo estabilidade e integridade:
- **Detector de Corrida (Race Detector)** ativo para evitar falhas de concorrência e uso paralelo na memória.
- Suíte cobrindo todos os utilitários, tratamento de imagens, limpeza de temporários e validação HTTP.
- Integração e verificação de formatação via Github Actions para assegurar a melhor performance com o *Go 1.26+*.

Para executar a suíte localmente (incluindo o race detector):
```bash
go test -race -count=1 ./...
```

---

## 🤝 Contribuindo

Veja [CONTRIBUTING.md](CONTRIBUTING.md) para instruções de como contribuir com o projeto.

---

## 📄 Licença

Distribuído sob a licença MIT. Veja [LICENSE](LICENSE) para mais informações.
