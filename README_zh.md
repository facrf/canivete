# 丛林刀 🪓

> 离线、轻量级、快速的图像和 PDF 处理 Web 工具。  
> 用纯 Go 编写。无外部运行时依赖（SVG 转换除外需要 `librsvg`）。

---

## ✨ 功能

| 类别 | 工具 |
|------|------|
| 🖼️ **图像** | 移除背景、交互式裁剪、调整大小、格式转换（PNG/JPG/BMP）、压缩 |
| 🎨 **颜色** | 提取调色板（HEX 代码）|
| 📄 **PDF** | 图像 → PDF、合并 PDF、拆分 PDF、优化、密码保护 |
| 🔁 **转换** | PDF → 图像、SVG → PNG |
| 📱 **二维码** | 生成和解码二维码及条形码 |

---

## 🚀 部署

### Docker
```bash
docker run -d --name canivete-da-mata -p 7001:7001 --restart unless-stopped ghcr.io/facrf/canivete:latest
```

### Portainer Stack
在 Portainer → Stacks 中使用 [`docker-compose.yml`](docker-compose.yml) 文件。

---

## 🏗️ 支持的架构

`linux/amd64` · `linux/arm64` · `linux/arm/v7` · `linux/riscv64`

---

## 📄 许可证

MIT — 见 [LICENSE](LICENSE)。
