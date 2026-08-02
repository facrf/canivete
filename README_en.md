# Jungle Knife 🪓

> Lightweight, offline, and fast web tool for image and PDF processing.  
> Built in pure Go. Zero external runtime dependencies (except `librsvg` for SVG).

---

## ✨ Features

| Category | Tools |
|----------|-------|
| 🖼️ **Image** | Remove background, interactive crop, resize, convert (PNG/JPG/BMP), compress |
| 🎨 **Colors** | Extract color palette (HEX) |
| 📄 **PDF** | Images → PDF, merge PDFs, split PDF, optimize, password protect |
| 🔁 **Convert** | PDF → Images, SVG → PNG |
| 📱 **QR Code** | Generate and decode QR codes and barcodes |

---

## 🌍 Multi-Language Interface

The interface is available in 7 languages, selectable via the flag icon in the top-right corner:  
🇧🇷 PT · 🇺🇸 EN · 🇪🇸 ES · 🇫🇷 FR · 🇩🇪 DE · 🇷🇺 RU · 🇨🇳 ZH

---

## 🚀 Deployment

### Docker (command line)
```bash
docker run -d \
  --name canivete-da-mata \
  -p 7001:7001 \
  --restart unless-stopped \
  ghcr.io/facrf/canivete:latest
```
Access at: **http://localhost:7001**

### Portainer Stack
1. Stacks → **Add Stack** → Web editor
2. Paste the contents of [`docker-compose.yml`](docker-compose.yml)
3. Click **Deploy the stack**

### Build from source
```bash
git clone http://192.168.0.10:3010/facrf/canivete.git
cd canivete
docker build -t canivete-da-mata:latest .
docker run -d -p 7001:7001 --name canivete-da-mata canivete-da-mata:latest
```

---

## 🏗️ Supported Architectures

| Platform | Hardware |
|---|---|
| `linux/amd64` | x86_64 servers, PCs, VMs |
| `linux/arm64` | Raspberry Pi 4/5 (64-bit), Apple Silicon |
| `linux/arm/v7` | Raspberry Pi 2/3/4 (32-bit), Orange Pi |
| `linux/riscv64` | VisionFive 2, StarFive, Milk-V Mars |

---

## 🔒 Security

- No database — immune to SQL Injection
- Go templates with automatic HTML/XSS escaping
- Upload limited via `http.MaxBytesReader`
- Maximum dimension validation (8000px)
- PDF passwords require at least 8 characters

---

## 📄 License

MIT — see [LICENSE](LICENSE).
