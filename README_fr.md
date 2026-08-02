# Couteau de la Jungle 🪓

> Outil web hors ligne, léger et rapide pour le traitement d'images et de PDF.  
> Construit en Go pur. Aucune dépendance externe au runtime (sauf `librsvg` pour SVG).

---

## ✨ Fonctionnalités

| Catégorie | Outils |
|-----------|--------|
| 🖼️ **Image** | Supprimer l'arrière-plan, recadrage interactif, redimensionner, convertir (PNG/JPG/BMP), compresser |
| 🎨 **Couleurs** | Extraire la palette de couleurs (HEX) |
| 📄 **PDF** | Images → PDF, fusionner PDFs, diviser PDF, optimiser, protéger par mot de passe |
| 🔁 **Conversion** | PDF → Images, SVG → PNG |
| 📱 **QR Code** | Générer et décoder des QR codes et codes-barres |

---

## 🚀 Déploiement

### Docker
```bash
docker run -d --name canivete-da-mata -p 7001:7001 --restart unless-stopped ghcr.io/facrf/canivete:latest
```

### Portainer Stack
Utilisez le fichier [`docker-compose.yml`](docker-compose.yml) dans Portainer → Stacks.

---

## 🏗️ Architectures supportées

`linux/amd64` · `linux/arm64` · `linux/arm/v7` · `linux/riscv64`

---

## 📄 Licence

MIT — voir [LICENSE](LICENSE).
