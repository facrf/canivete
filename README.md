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
| `MAX_CONCURRENT_JOBS` | `2` | Limite de processamentos simultâneos; ajuste conforme a memória disponível. |

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

### 🐳 Portainer (Stack / YAML)

1. No painel do **Portainer**, acesse o seu ambiente (**Environment**) e vá em **Stacks** ➔ **Add stack**.
2. Defina o nome da stack no campo **Name** (ex: `canivete`).
3. Selecione a opção **Web editor** e cole o seguinte YAML:

```yaml
version: '3.8'

services:
  canivete:
    image: ghcr.io/facrf/canivete:latest
    container_name: canivete-da-mata
    restart: unless-stopped
    stop_grace_period: 20s
    ports:
      - "7001:7001"
    environment:
      - TZ=America/Sao_Paulo
      - MAX_CONCURRENT_JOBS=2
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    read_only: true
    tmpfs:
      - /tmp:size=512M,mode=1777
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 768M
        reservations:
          memory: 64M
    healthcheck:
      test: ["CMD", "/app/canivete", "--healthcheck"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 10s
```

4. *(Opcional)* Ajuste as portas ou variáveis de ambiente se necessário.
5. Clique em **Deploy the stack**.
6. Acesse a ferramenta em: `http://<IP-DO-SEU-SERVIDOR>:7001`.

> 💡 **Dica:** O arquivo completo com exemplos e integração Traefik também pode ser consultado em [`docker-compose.yml`](docker-compose.yml).

### Saúde no Portainer e limites de processamento

O contêiner possui healthcheck nativo, executado com `/app/canivete --healthcheck`.
No Portainer, abra **Containers → canivete-da-mata** para consultar o estado
`starting`, `healthy` ou `unhealthy` e o histórico das verificações.
A verificação usa a variável `PORT`, exige os renderizadores `pdftoppm` e
`rsvg-convert` e consulta `/healthz`, que verifica também o acesso ao diretório temporário.
O healthcheck sinaliza falhas; `restart: unless-stopped` reinicia processos que
encerram, mas não reinicia automaticamente um contêiner apenas por estar `unhealthy`.

O padrão é de **2 processamentos simultâneos**. A extração, divisão e rasterização
aceitam PDFs de até **50 páginas**; páginas rasterizadas têm dimensão máxima de
**4000 px**, e saídas de conversão/ZIP de PDF têm limite de **64 MiB**.
SVGs são validados antes da renderização, com até **8000 px por dimensão** e
**20 megapixels**. O desligamento trata `SIGTERM`, com até 15 segundos para
concluir requisições; a stack concede 20 segundos antes de forçar a parada.

### Build local (a partir do código-fonte)
```bash
git clone https://github.com/facrf/canivete.git
cd canivete
docker build -t canivete-da-mata:latest .
docker run -d -p 7001:7001 --name canivete-da-mata canivete-da-mata:latest
```

---

### Raspberry Pi 400: `exec ./canivete: exec format error`

Esse erro indica que o executável não é compatível com a arquitetura do sistema.
O build usa a plataforma de destino do Docker BuildKit, sem fixar `amd64`.

Confira o sistema instalado (um Pi 400 também pode usar um sistema de 32 bits):

```bash
uname -m
getconf LONG_BIT
docker info --format '{{.Architecture}}'
```

Com sistema de 64 bits (`aarch64`, `64` e Docker `aarch64`/`arm64`),
compile uma imagem local com uma tag própria:

```bash
docker buildx build --platform linux/arm64 --load -t canivete-da-mata:arm64 .
docker run -d --name canivete-pi400 -p 7001:7001 canivete-da-mata:arm64
```

Em sistemas ARM de 32 bits, use `--platform linux/arm/v7` e uma tag
`canivete-da-mata:armv7` em ambos os comandos. Se o comando `buildx` não
existir, instale o plugin Docker Buildx. Não sobrescreva `TARGETARCH` com
`--build-arg`: deixe o Docker selecionar a arquitetura.

Para Portainer, publique a correção no GHCR antes de atualizar a stack.
Depois, habilite a opção de baixar a imagem novamente ao recriar o serviço.
Para usar o build local, altere `image:` para `canivete-da-mata:arm64` na stack.
Uma imagem antiga em cache ainda pode conter o executável incompatível.

Sem Docker, confira o executável com `file ./canivete`. Para gerar um binário
ARM64 a partir do código, use:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o canivete-linux-arm64 .
```

---

## 🏗️ Arquiteturas suportadas (Docker)

A imagem no GHCR é publicada para múltiplas arquiteturas automaticamente via GitHub Actions:

| Plataforma | Hardware |
|---|---|
| `linux/amd64` | Servidores x86_64, PCs, VMs |
| `linux/arm64` | Raspberry Pi 4/400/5 (64-bit), Apple Silicon (M1/M2/M3) |
| `linux/arm/v7` | Raspberry Pi 2/3/4/400 (32-bit), Orange Pi |
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
