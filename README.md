# Perplexity CLI Go

Uma interface de linha de comando (CLI) em Go para interagir com a API do Perplexity AI, com suporte a streaming de respostas, autenticação via cookies e renderização de markdown no terminal.

## 🚀 Recursos

- **Múltiplos Modelos IA**: Suporte para pplx_pro, gpt5, claude45sonnet, grok41, gemini30pro e outros
- **Modos de Busca**: fast, pro, reasoning, deep-research
- **Streaming em Tempo Real**: Respostas fluem em tempo real enquanto são geradas
- **Autenticação Segura**: Usa cookies do navegador para autenticação
- **Renderização Markdown**: Saída formatada com Glamour/Lipgloss
- **Configuração Interativa**: Menu TUI para configuração fácil
- **Histórico de Consultas**: Salva e gerencia histórico de buscas
- **Suporte a Arquivos**: Permite anexar arquivos e ler consultas de arquivos
- **Múltiplos Idiomas**: Suporte a diferentes idiomas de resposta
- **Impersonificação TLS**: Emula fingerprint do Chrome para evitar detecção

## 📦 Instalação

### Via Go (Recomendado)

```bash
go install github.com/diogo/perplexity-go@latest
```

### Via Make (Build Local)

```bash
git clone https://github.com/diogo/perplexity-go.git
cd perplexity-cli-go
make install
```

### Binários Pré-compilados

Baixe os binários da página de [Releases](https://github.com/diogo/perplexity-go/releases).

## 🔧 Configuração

### 1. Autenticação

A CLI usa cookies do navegador para autenticação. Exporte os cookies do seu navegador:

#### Método 1: Extensão de Browser (Recomendado)
1. Instale uma extensão como "Get cookies.txt LOCALLY" no Chrome/Firefox
2. Acesse [perplexity.ai](https://perplexity.ai) e faça login
3. Use a extensão para exportar os cookies como JSON
4. Importe com: `perplexity import-cookies cookies.json`

#### Método 2: Exportação Manual
1. Faça login no [perplexity.ai](https://perplexity.ai)
2. Abra as ferramentas de desenvolvedor (F12)
3. Vá para Application/Storage > Cookies > https://perplexity.ai
4. Exporte o cookie `next-auth.csrf-token` e outros cookies necessários

### 2. Configuração Interativa

```bash
perplexity config
```

Isso abrirá um menu interativo para configurar:
- Modelo IA padrão
- Modo de busca padrão
- Idioma de resposta
- Fontes de busca
- Streaming
- Modo anônimo

### 3. Verificação

```bash
# Verificar status dos cookies
perplexity cookies status

# Verificar configuração atual
perplexity config path
```

## 📖 Uso

### Comandos Básicos

```bash
# Busca simples
perplexity "Qual a capital do Brasil?"

# Com modelo específico
perplexity "Explique computação quântica" --model gpt5 --mode pro

# Com streaming
perplexity "Latest news on AI" --stream

# Ler consulta de arquivo
perplexity -f pergunta.md --mode reasoning

# Salvar resposta em arquivo
perplexity "What is Go?" -o resposta.md

# Busca com fontes específicas
perplexity "Climate change research" --sources web,scholar --language pt-BR
```

### Modos de Busca

| Modo | Descrição | Modelo Padrão |
|------|-----------|---------------|
| `fast` | Respostas rápidas e concisas | turbo |
| `pro` | Busca profunda com raciocínio | (do modelo) |
| `reasoning` | Mode com raciocínio avançado | + is_pro_reasoning=true |
| `deep-research` | Pesquisa aprofundada | pplx_alpha |
| `default` | Modo padrão copilot | (do modelo) |

### Modelos Disponíveis

#### Pro Mode:
- `pplx_pro`
- `gpt51`
- `grok41nonreasoning`
- `experimental`
- `claude45sonnet`

#### Reasoning Mode:
- `gemini30pro`
- `gpt51_thinking`
- `grok41reasoning`
- `kimik2thinking`
- `claude45sonnetthinking`

### Comandos de Configuração

```bash
# Menu interativo de configuração
perplexity config

# Gerenciar cookies
perplexity import-cookies <arquivo>
perplexity cookies status
perplexity cookies clear
perplexity cookies path

# Ver histórico
perplexity history

# Versão
perplexity version
```

### Uso Avançado

```bash
# Modo anônimo (não salva no histórico)
perplexity "consulta sensível" --incognito

# Busca verbose
perplexity "consulta complexa" --verbose

# Usar arquivo de cookies específico
perplexity "consulta" --cookies /path/to/cookies.json

# Combinar múltiplas opções
perplexity -f pesquisa.txt -o resultado.md --model claude45sonnet --mode reasoning --stream --language pt-BR
```

## 🔒 Segurança

- Os cookies são armazenados localmente em `~/.perplexity-cli/cookies.json`
- A configuração fica em `~/.perplexity-cli/config.json`
- Use `--incognito` para consultas sensíveis que não devem ser salvas
- Os cookies nunca são compartilhados ou enviados para servidores de terceiros

## 🛠️ Desenvolvimento

### Estrutura do Projeto

```
perplexity-cli-go/
├── cmd/perplexity/         # CLI commands (Cobra)
│   ├── main.go            # Entry point
│   ├── root.go            # Main query command + flags
│   ├── config.go          # Interactive config menu
│   ├── cookies.go         # Cookie management
│   ├── history.go         # Query history
│   └── version.go         # Version info
├── pkg/
│   ├── client/            # API client (exportado)
│   │   ├── client.go      # Main client + Search methods
│   │   ├── http.go        # TLS-client wrapper
│   │   ├── search.go      # SSE parsing, payload building
│   │   └── upload.go      # S3 file upload
│   └── models/            # Data types (exportado)
│       ├── types.go       # Mode, Model, Source enums
│       ├── request.go     # SearchRequest, SearchOptions
│       └── response.go    # SearchResponse, StreamChunk
└── internal/
    ├── auth/              # Cookie loading
    ├── config/            # Viper-based config
    ├── history/           # JSONL history writer
    └── ui/                # Glamour/Lipgloss rendering
```

### Build e Testes

```bash
# Build
make build                    # Build para ./build/perplexity
make build-release            # Build otimizado

# Install
make install                  # Install system (/usr/local/bin)
make install-user             # Install user (~/.local/bin)

# Testes
make test                     # Run todos os testes
make test-coverage            # Com coverage
make test-coverage-html       # HTML coverage report

# Run direto
make run ARGS='"O que é Go?"'
./build/perplexity "consulta" --model gpt51 --mode pro --stream
```

### Dependências Principais

- `github.com/bogdanfinn/tls-client` + `fhttp`: Chrome TLS fingerprint impersonation
- `github.com/spf13/cobra` + `viper`: CLI framework e config
- `github.com/charmbracelet/glamour` + `lipgloss`: Terminal markdown rendering
- `github.com/charmbracelet/huh`: Interactive terminal UI forms

## 📝 Exemplos

### Exemplo 1: Pesquisa Rápida

```bash
perplexity "Como funciona a fotossíntese?" --mode fast
```

### Exemplo 2: Pesquisa Profunda com Streaming

```bash
perplexity "História da inteligência artificial" --mode deep-research --stream --sources web,scholar
```

### Exemplo 3: Consulta Técnica

```bash
perplexity "Implemente um quicksort em Go" --model gpt51 --mode reasoning --language pt-BR
```

### Exemplo 4: Usando Arquivos

```bash
# Criar arquivo de consulta
echo "Explique a relatividade de Einstein de forma simples" > pergunta.txt

# Fazer a busca e salvar resposta
perplexity -f pergunta.txt -o resposta.md --model claude45sonnet --mode pro
```

## 🐛 Troubleshooting

### Problemas Comuns

#### "cookies file not found"
```bash
# Importe os cookies primeiro
perplexity import-cookies cookies.json

# Verifique o caminho
perplexity cookies path
```

#### "failed to load cookies"
- Verifique se o arquivo JSON está válido
- Exporte os cookies novamente do navegador
- Use o formato correto (JSON da extensão)

#### Respostas vazias ou erros
- Verifique sua conexão com perplexity.ai no navegador
- Tente re-exportar os cookies
- Use `--verbose` para debug

### Logs e Debug

```bash
# Busca verbose
perplexity "consulta" --verbose

# Ver configuração atual
cat ~/.perplexity-cli/config.json

# Testar cookies
perplexity cookies status
```

## 📄 Licença

MIT License - veja o arquivo [LICENSE](LICENSE) para detalhes.

## 🤝 Contribuição

Contribuições são bem-vindas! Por favor:

1. Fork o repositório
2. Crie uma feature branch (`git checkout -b feature/amazing-feature`)
3. Commit suas mudanças (`git commit -m 'Add amazing feature'`)
4. Push para a branch (`git push origin feature/amazing-feature`)
5. Abra um Pull Request

### Requisitos para Contribuição

- Mantenha cobertura de testes > 80%
- Siga o estilo de código Go convencional
- Adicione testes para novas funcionalidades
- Documente funções exportadas

## 📞 Suporte

- Abra uma [issue](https://github.com/diogo/perplexity-go/issues) para bugs ou feature requests
- Consulte a [documentação](https://github.com/diogo/perplexity-go/wiki) para tutoriais
- Entre em contato via [discussions](https://github.com/diogo/perplexity-go/discussions)

## 🗺️ Roadmap

- [ ] Suporte a plugins
- [ ] Interface web opcional
- [ ] Mais modelos IA
- [ ] Exportação em múltiplos formatos
- [ ] Integração com outras APIs

---

**Feito com ❤️ usando Go**