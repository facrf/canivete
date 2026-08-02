# Canivete da Mata 🔪

Sua ferramenta offline, leve e rápida para processamento de imagens e PDFs, 100% conteinerizada e sem dependências externas.

## Funcionalidades

- **Remoção de Fundo (Transparência):** Detecta fundos claros/brancos usando distância euclidiana de cor e aplica transparência. O usuário controla a tolerância diretamente na interface.
- **Recorte Quadrado Interativo:** Usando o `Cropper.js`, permite selecionar exatamente a parte da imagem desejada, gerando recortes nos tamanhos padrões ideais para ícones (16x16, 32x32, 144x144, 256x256, etc).
- **Redimensionamento:** Altere resolução (Largura x Altura) com algoritmo Lanczos para alta qualidade.
- **Conversão de Formato:** Converte imagens entre PNG, JPG, e BMP, além de possuir slider de qualidade de compressão.
- **Manipulação de PDF:**
  - Junte múltiplas imagens para formar um documento PDF único.
  - Extraia imagens nativas de dentro de um PDF.
  - (NOVO) **Rasterize** as páginas de um documento PDF, convertendo-as em formato de imagem compactado usando `pdftoppm`.

## Arquitetura

- **Linguagem Backend:** Go (1.22+)
- **Bibliotecas:** `imaging`, `pdfcpu`, pacotes nativos do Go (`image`). Sem uso de CGO no core!
- **Frontend:** Server-side Rendering com `html/template`, utilizando Pico.css para o design e `Cropper.js` para recortes interativos. Tudo embarcado no executável final através do `go:embed`.
- **Containers:** Multi-stage build no Docker. O executável Go roda numa imagem `alpine:latest` que já conta com pacote `poppler-utils` para rasterização de PDFs.

## Como Rodar (Padrão Docker)

O projeto já vem pronto com um arquivo `Dockerfile` otimizado para o padrão *Multi-Stage*. 

### Pré-requisitos
- Docker instalado na sua máquina.

### Construir a Imagem
A partir do diretório raiz do repositório, rode:
```bash
docker build -t canivete-da-mata:latest .
```

### Executar o Container
A aplicação utiliza a porta **7001**.
```bash
docker run -d --name canivete-app-1 -p 7001:7001 canivete-da-mata:latest
```

Após iniciar o container, basta acessar no seu navegador:
**http://localhost:7001**

---

### Executar Localmente (Ambiente de Desenvolvimento)
Caso queira rodar diretamente usando o Go instalado na máquina:
```bash
go mod download
go run .
```

*(Lembre-se que para a funcionalidade de "Rasterizar" o PDF localmente, seu sistema operacional deve ter instalado o pacote de binários `poppler-utils` no path)*
