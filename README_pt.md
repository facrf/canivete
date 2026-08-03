# Canivete da Mata 🪓

> Ferramenta web offline, leve e rápida para processamento de imagens e PDFs.  
> Construída em Go. O contêiner inclui `librsvg` para SVG e `poppler-utils` para rasterização de PDF.

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

## 🌍 Interface Multi-Idiomas

A interface está disponível em 7 idiomas, selecionáveis pela bandeirinha no canto superior direito:  
🇧🇷 PT · 🇺🇸 EN · 🇪🇸 ES · 🇫🇷 FR · 🇩🇪 DE · 🇷🇺 RU · 🇨🇳 ZH

---

## 🚀 Deploy

### Docker (linha de comando)
```bash
docker run -d \
  --name canivete-da-mata \
  -p 7001:7001 \
  --restart unless-stopped \
  ghcr.io/facrf/canivete:latest
```
Acesse em: **http://localhost:7001**

### Portainer Stack
1. Stacks → **Add Stack** → Web editor
2. Cole o conteúdo de [`docker-compose.yml`](docker-compose.yml)
3. Clique em **Deploy the stack**

### Build local (código-fonte)
```bash
git clone http://192.168.0.10:3010/facrf/canivete.git
cd canivete
docker build -t canivete-da-mata:latest .
docker run -d -p 7001:7001 --name canivete-da-mata canivete-da-mata:latest
```

---

## 🏗️ Arquiteturas suportadas

| Plataforma | Hardware |
|---|---|
| `linux/amd64` | Servidores x86_64, PCs, VMs |
| `linux/arm64` | Raspberry Pi 4/5 (64-bit), Apple Silicon |
| `linux/arm/v7` | Raspberry Pi 2/3/4 (32-bit), Orange Pi |
| `linux/riscv64` | VisionFive 2, StarFive, Milk-V Mars |

---

## 🔒 Segurança

- Sem banco de dados — imune a SQL Injection
- Templates Go com escaping automático de HTML/XSS
- Upload limitado com `http.MaxBytesReader`
- Validação de dimensões (máximo de 8000px e 20 megapixels)
- Senhas de PDF com mínimo de 8 caracteres

---

## 📄 Licença

MIT — veja [LICENSE](LICENSE).
