# Processamento e privacidade

Os uploads são processados localmente. A aplicação não mantém uma biblioteca de
arquivos enviados: os temporários usados por uma requisição são removidos ao seu
fim. Resultados de PDF só são oferecidos para download depois que a operação
termina e a saída é verificada.

## Limites de processamento

| Operação | Limite aplicado |
| --- | --- |
| Upload de imagem individual | 20 MiB por requisição; no lote, 20 MiB por imagem |
| Imagens para PDF | até 50 imagens, 100 MiB de formulário, 40 megapixels somados e 64 MiB de PDF gerado |
| Upload de PDF individual | 50 MiB por requisição, incluindo os campos do formulário |
| Juntar PDFs | até 50 arquivos, 100 MiB de formulário e 50 páginas somadas |
| Outras operações de PDF | de 1 a 50 páginas por documento |
| Saída de PDF ou ZIP de PDF | 64 MiB, inclusive após a compactação |
| Rasterização de PDF | até 4.000 px na maior dimensão de cada página |
| Processamento de PDF | até 3 minutos depois do recebimento do upload |
| SVG | 20 MiB de requisição, 20 megapixels e 8.000 px por dimensão |

As operações de PDF rodam em um subprocesso do próprio binário. Quando o cliente
cancela a requisição ou o prazo de três minutos termina, o servidor encerra esse
processo e, no Linux, seus subprocessos de renderização. O diretório temporário
do trabalho é apagado em seguida. O servidor limita os trabalhos simultâneos com
`MAX_CONCURRENT_JOBS` (padrão: 2). A memória e o espaço temporário disponíveis
continuam sendo os limites do ambiente em que a aplicação roda; a stack de
exemplo reserva 512 MiB para `/tmp` e limita a memória do contêiner a 768 MiB.

Erros de tamanho de upload retornam HTTP 413. Um documento, seleção de páginas
ou saída fora dos limites retorna HTTP 400. O prazo do subprocesso retorna HTTP
504, e uma requisição cancelada retorna HTTP 408 quando ainda há conexão para
receber a resposta.

## Nomes de arquivo no ZIP

As imagens extraídas de PDF recebem nomes gerados pelo servidor, como
`page-001-image-000012.png`. O nome de um recurso dentro do PDF não é usado como
caminho no ZIP. Isso impede nomes repetidos entre páginas e componentes como
`../` vindos do PDF. A extensão é escolhida de uma lista permitida; formatos
desconhecidos usam `.bin`.

## Remoção de metadados de imagem

JPEG: remove APP1 (Exif/XMP), APP11 (JUMBF/C2PA), APP13 (IPTC), comentários e
dados anexados depois do marcador final. Arquivos sem quadro, varredura ou
marcador final são recusados. A remoção é feita sem recomprimir os pixels.

PNG: mantém os blocos estruturais, de cor e de animação conhecidos e remove os
demais, incluindo textos e dados anexados depois de `IEND`.

WebP: respeita o tamanho declarado pelo contêiner RIFF, remove EXIF, XMP, C2PA,
blocos desconhecidos no nível superior e dados anexados. Os indicadores EXIF/XMP do bloco `VP8X`
são limpos. Os blocos de cor, transparência, pixels e animação são preservados.

Essa operação atua nos metadados reconhecidos pelo parser e não detecta
informações desenhadas nos pixels ou escondidas em dados de imagem. Em JPEG, a
orientação que dependia de Exif também pode deixar de ser aplicada pelo leitor;
confira o resultado visual antes de compartilhar a foto.

## Verificação e publicação

O projeto exige Go 1.26.9 ou superior. Os dois fluxos de CI chamam a mesma
verificação de formatação, módulos, testes com detector de concorrência, `go vet`
e `govulncheck`. O fluxo Docker só testa e publica a imagem depois da aprovação
dessas verificações. Os testes de integração usam `pdftoppm` e `rsvg-convert` no
CI; localmente, os casos que exigem esses programas são ignorados se não
estiverem instalados.

Para executar os mesmos checks localmente:

```bash
go mod verify
test -z "$(gofmt -l $(git ls-files '*.go'))"
go test -race ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
```
