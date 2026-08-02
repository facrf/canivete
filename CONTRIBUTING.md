# Como Contribuir para o Canivete da Mata

Primeiramente, obrigado por se interessar em contribuir para o **Canivete da Mata**! 🎉

## Processo de Contribuição

1. Faça o **Fork** do repositório e clone para sua máquina local.
2. Crie uma branch para a sua feature ou correção (`git checkout -b feature/nova-funcionalidade`).
3. Faça suas modificações garantindo que o código está limpo, bem documentado e formatado usando o padrão oficial da linguagem (`go fmt`).
4. Teste as modificações localmente:
   - Via Docker: `docker build -t canivete-da-mata:latest .`
   - Ou via Go local: `go run .`
5. Faça o commit de forma descritiva (`git commit -m 'feat: adiciona nova funcionalidade'`).
6. Faça o push para a sua branch origin (`git push origin feature/nova-funcionalidade`).
7. Abra um **Pull Request (PR)** explicando detalhadamente o que foi resolvido ou adicionado.

## Padrões de Código
- O projeto usa a funcionalidade `go:embed` nativa da linguagem extensivamente para garantir a proposta offline da ferramenta. Portanto, lembre-se sempre de embutir os assets, fontes e JS necessários se adicionar novos recursos no frontend.
- O Frontend é Server-Side Rendered nativo via `html/template`. Evite a introdução de frameworks Javascript pesados.
- Use CSS limpo preferencialmente utilizando as classes do *Pico.css*.
- **CGO_ENABLED=0**: Nenhuma dependência externa CGO (dependência em bibliotecas C instaladas no host) deve ser introduzida na camada backend para garantir e facilitar a cross-compilação limpa para múltiplas arquiteturas (ARM, RISC-V, etc). O poppler (`pdftoppm`) roda externamente via `os/exec` justamente para preservar a compilação cruzada do binário principal!

Agradecemos imensamente por ajudar a melhorar a ferramenta!
