# Jungle Knife 🪓

> Lightweight, offline, and fast web tool for image and PDF processing.  
> Built in Go. The container includes `librsvg` for SVG and `poppler-utils` for PDF rasterization.

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

### 🐳 Portainer (Stack / YAML)

1. In your **Portainer** dashboard, select your **Environment** and navigate to **Stacks** ➔ **Add stack**.
2. Enter a stack name in the **Name** field (e.g., `canivete`).
3. Under **Web editor**, paste the following YAML:

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

4. Click **Deploy the stack**.
5. Access the app at: `http://<YOUR-SERVER-IP>:7001`.

### Container health and processing limits

The image and Compose stack run `/app/canivete --healthcheck`. In Portainer,
open **Containers → canivete-da-mata** to see `starting`, `healthy` or `unhealthy`
and the check history. The probe honors `PORT`, checks the PDF/SVG renderers,
and calls `/healthz`, which also checks temporary storage access. An unhealthy
status alone does not trigger a restart under `restart: unless-stopped`.

The default concurrency is 2. Every PDF operation accepts at most 50 pages per
document; a merge accepts 50 pages in total. A PDF job has a three-minute limit.
Rasterized pages are limited to 4000 px, and generated PDF/ZIP files to 64 MiB.
Image-to-PDF batches are limited to 40 megapixels in total. SVG output is
limited to 8000 px per dimension and 20 megapixels. Shutdown handles SIGTERM
and allows 15 seconds for active requests; Compose allows 20 seconds before
forcing termination. See the detailed [processing and privacy policy](docs/PROCESSAMENTO_E_PRIVACIDADE.md)
(in Portuguese).

### Build from source
```bash
git clone https://github.com/facrf/canivete.git
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
- Dimension validation (maximum 8000px and 20 megapixels)
- PDF passwords require at least 8 characters

---

## 📄 License

MIT — see [LICENSE](LICENSE).
