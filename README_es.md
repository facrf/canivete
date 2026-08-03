# Navaja de la Selva 🪓

> Herramienta web offline, ligera y rápida para procesamiento de imágenes y PDF.  
> Construida en Go. El contenedor incluye `librsvg` para SVG y `poppler-utils` para rasterizar PDF.

---

## ✨ Funciones

| Categoría | Herramientas |
|-----------|-------------|
| 🖼️ **Imagen** | Quitar fondo, recorte interactivo, redimensionar, convertir (PNG/JPG/BMP), comprimir |
| 🎨 **Colores** | Extraer paleta de colores (HEX) |
| 📄 **PDF** | Imágenes → PDF, unir PDFs, dividir PDF, optimizar, proteger con contraseña |
| 🔁 **Conversión** | PDF → Imágenes, SVG → PNG |
| 📱 **QR Code** | Generar y decodificar códigos QR y de barras |

---

## 🚀 Despliegue

### Docker
```bash
docker run -d --name canivete-da-mata -p 7001:7001 --restart unless-stopped ghcr.io/facrf/canivete:latest
```

### Portainer Stack
Use el archivo [`docker-compose.yml`](docker-compose.yml) directamente en Portainer → Stacks.

---

## 🏗️ Arquitecturas soportadas

`linux/amd64` · `linux/arm64` · `linux/arm/v7` · `linux/riscv64`

---

## 📄 Licencia

MIT — ver [LICENSE](LICENSE).
