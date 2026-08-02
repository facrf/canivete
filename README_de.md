# Dschungelmesser 🪓

> Offline, leichtes und schnelles Web-Tool zur Bild- und PDF-Verarbeitung.  
> In reinem Go gebaut. Keine externen Laufzeitabhängigkeiten (außer `librsvg` für SVG).

---

## ✨ Funktionen

| Kategorie | Werkzeuge |
|-----------|-----------|
| 🖼️ **Bild** | Hintergrund entfernen, interaktives Zuschneiden, Größe ändern, konvertieren (PNG/JPG/BMP), komprimieren |
| 🎨 **Farben** | Farbpalette extrahieren (HEX) |
| 📄 **PDF** | Bilder → PDF, PDFs zusammenführen, PDF teilen, optimieren, passwortgeschützt |
| 🔁 **Konvertierung** | PDF → Bilder, SVG → PNG |
| 📱 **QR-Code** | QR-Codes und Barcodes generieren und dekodieren |

---

## 🚀 Bereitstellung

### Docker
```bash
docker run -d --name canivete-da-mata -p 7001:7001 --restart unless-stopped ghcr.io/facrf/canivete:latest
```

### Portainer Stack
Verwenden Sie die Datei [`docker-compose.yml`](docker-compose.yml) in Portainer → Stacks.

---

## 🏗️ Unterstützte Architekturen

`linux/amd64` · `linux/arm64` · `linux/arm/v7` · `linux/riscv64`

---

## 📄 Lizenz

MIT — siehe [LICENSE](LICENSE).
