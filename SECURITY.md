# Política de Segurança

A segurança e estabilidade dos dados que passam pelo Canivete da Mata é uma prioridade, especialmente por ser uma ferramenta focada 100% no uso Offline.

## Versões Suportadas

Nós aplicamos patches de segurança para as versões da branch principal e para a versão mais recente em estabilidade.

| Versão | Status de Suporte |
| ------- | ------------------ |
| `main` / Latest | :white_check_mark: Suportado |
| < v1.0  | :x: Não Suportado |

## Reportando uma Vulnerabilidade

Se você descobrir qualquer problema relacionado à segurança ou vazamento de contexto na ferramenta (como falha no processamento isolado, estoiro de buffer, ou acesso a arquivos do sistema além da pasta temporária restrita), por favor, **não crie uma issue pública** imediatamente.

Em vez disso, notifique os mantenedores do projeto privadamente através dos canais diretos da comunidade. Forneça uma descrição detalhada da vulnerabilidade, bem como os passos necessários para reproduzi-la na versão mais recente.

Nós nos comprometemos a analisar, debater os impactos e disponibilizar um patch de correção da maneira mais rápida possível!

## Processamento de arquivos

Os limites, prazos, arquivos temporários e formatos de metadados tratados estão
documentados em [Processamento e privacidade](docs/PROCESSAMENTO_E_PRIVACIDADE.md).
O subprocesso de PDF permite interromper trabalho que exceda o prazo; ele usa
as mesmas permissões do contêiner e não constitui um isolamento de acesso ao
sistema de arquivos. Em produção, mantenha as restrições de usuário sem
privilégios, sistema de arquivos somente leitura e limites de recursos descritos
em `docker-compose.yml`.
