# Canivete da Mata 🪓

O Canivete da Mata é uma aplicação web leve, 100% offline e rápida para processamento de imagens e PDFs, desenvolvida puramente em Go.

## Funcionalidades
- **Remover Fundo:** Retira fundos brancos.
- **Recortar e Redimensionar:** Corte interativo e redimensionamento preciso.
- **Conversor de Formatos:** Converta entre PNG, JPG e BMP.
- **Compressor de Imagem:** Reduza o tamanho de JPGs em MB/KB.
- **Extrator de Paleta:** Obtenha cores HEX das imagens.
- **Ferramentas QR:** Gerador e leitor de QR Code.
- **Ferramentas PDF:** Juntar, dividir, otimizar, proteger com senha, além de conversão de páginas e imagens para PDF.

## Como Executar
Use o Docker para não precisar configurar nada:
```bash
docker build -t canivete-da-mata:latest .
docker run -d -p 7001:7001 canivete-da-mata:latest
```
Acesse em `http://localhost:7001`.
