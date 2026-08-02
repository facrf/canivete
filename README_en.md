# Canivete da Mata 🪓

Canivete da Mata is a lightweight, offline, and fast web application for processing images and PDFs, written purely in Go without external C dependencies (except `librsvg` for SVG conversions).

## 🌍 Supported Languages
The application supports multiple languages out-of-the-box:
- [Português](README_pt.md)
- [English](README_en.md)
- [Español](README_es.md)
- [Français](README_fr.md)
- [Deutsch](README_de.md)
- [Русский](README_ru.md)
- [中文](README_zh.md)

## Features
- **Background Removal:** Easily remove white backgrounds.
- **Cropping & Resizing:** Interactive crop and accurate resize.
- **Format Converter:** Convert between PNG, JPG, BMP.
- **Image Compressor:** Shrink JPG sizes significantly.
- **Palette Extractor:** Get HEX colors from images.
- **QR Code Tools:** Generate and decode QR codes.
- **PDF Tools:** Images to PDF, merge, split, optimize, password protect, and extract/rasterize pages.

## Deployment
Use Docker for zero-setup deployment:
```bash
docker build -t canivete-da-mata:latest .
docker run -d -p 7001:7001 canivete-da-mata:latest
```
Access at `http://localhost:7001`.
