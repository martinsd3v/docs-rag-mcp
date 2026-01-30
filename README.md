# docs-rag-mcp

RAG (Retrieval Augmented Generation) MCP Server para Documentação Técnica em Golang.

## Visão Geral

Sistema que indexa documentação técnica (RFCs, ADRs, BDRs, Guidelines) usando embeddings vetoriais e expõe via MCP (Model Context Protocol) para Claude Code, permitindo acesso contextual à documentação sem saturar o contexto da conversa.

## Arquitetura

```
Claude Code (MCP Client)
    ↓ (stdio)
docs-rag-mcp-server (Go)
    ↓
SQLite + sqlite-vec extension
    ↓ (indexação)
OpenAI API ou Ollama (embeddings locais)
```

## Componentes

1. **CLI Indexer** (`cmd/indexer`): Comando manual para indexar/reindexar documentos
2. **MCP Server** (`cmd/server`): Servidor MCP que expõe tools de busca para Claude Code
3. **Web UI** (`cmd/web`): Interface React para debug, visualização e testes

## Estrutura do Projeto

```
docs-rag-mcp/
├── cmd/
│   ├── server/              # MCP server entrypoint
│   ├── indexer/             # CLI de indexação
│   └── web/                 # Web UI server
├── internal/
│   ├── mcp/                 # MCP protocol implementation
│   ├── embeddings/          # OpenAI/Ollama clients + cache
│   ├── vector/              # SQLite-vec wrapper
│   ├── parser/              # Markdown parser + chunking
│   ├── indexer/             # Indexing pipeline
│   └── api/                 # HTTP API para UI
├── web/                     # React front-end
├── db/
│   ├── schema.sql           # Database schema
│   └── migrations/
├── configs/
│   ├── config.yaml          # Server config
│   └── mcp-config.json      # Claude Code config
├── data/                    # SQLite database files
├── go.mod
├── Makefile
└── README.md
```

## Instalação

```bash
# Clone o repositório
git clone https://github.com/trxio/docs-rag-mcp.git
cd docs-rag-mcp

# Instalar dependências
go mod download

# Build todos os binários
make build

# Ou build individual
go build -o bin/docs-rag-indexer cmd/indexer/main.go
go build -o bin/docs-rag-mcp-server cmd/server/main.go
go build -o bin/docs-rag-web cmd/web/main.go
```

## Ollama (Embeddings Offline)

O sistema suporta geração de embeddings localmente usando Ollama, permitindo operação 100% offline sem necessidade de API externa.

### Instalação do Ollama

```bash
# macOS
brew install ollama

# Linux
curl -fsSL https://ollama.com/install.sh | sh

# Windows
# Baixar de https://ollama.com/download
```

### Iniciar o Ollama

```bash
# Iniciar serviço
ollama serve

# Ou em segundo plano
ollama serve &
```

### Modelos de Embedding Suportados

| Modelo | Dimensões | Tamanho | Descrição |
|--------|-----------|---------|-----------|
| `nomic-embed-text` | 768 | ~274MB | **Recomendado** - Melhor custo-benefício |
| `mxbai-embed-large` | 1024 | ~669MB | Maior precisão, mais pesado |
| `all-minilm` | 384 | ~45MB | Mais leve, menor precisão |

### Baixar Modelo

```bash
# Modelo recomendado
ollama pull nomic-embed-text

# Verificar modelos instalados
ollama list
```

### Testar Ollama

```bash
# Testar embedding
curl http://localhost:11434/api/embeddings -d '{
  "model": "nomic-embed-text",
  "prompt": "Hello World"
}'
```

## Uso Rápido

### 1. Indexar Documentos (com OpenAI)

```bash
./bin/docs-rag-indexer index \
    --path /path/to/docs \
    --db ./data/docs.db \
    --openai-key $OPENAI_API_KEY \
    --verbose
```

### 1. Indexar Documentos (com Ollama - Offline)

```bash
./bin/docs-rag-indexer index \
    --path /path/to/docs \
    --db ./data/docs.db \
    --ollama \
    --verbose
```

### 2. Verificar Status

```bash
./bin/docs-rag-indexer status --db ./data/docs.db --show-stats
```

### 3. Testar Busca (com OpenAI)

```bash
./bin/docs-rag-indexer test-query \
    --db ./data/docs.db \
    --openai-key $OPENAI_API_KEY \
    --query "microservice architecture patterns" \
    --top-k 5
```

### 3. Testar Busca (com Ollama)

```bash
./bin/docs-rag-indexer test-query \
    --db ./data/docs.db \
    --ollama \
    --query "microservice architecture patterns" \
    --top-k 5
```

### 4. Iniciar MCP Server (com OpenAI)

```bash
# Via variáveis de ambiente
DB_PATH=./data/docs.db \
OPENAI_API_KEY=your-key \
OPENAI_BASE_URL=https://ai.v3m.ai/v1 \
./bin/docs-rag-mcp-server

# Ou via flags
./bin/docs-rag-mcp-server \
    --db ./data/docs.db \
    --openai-key your-key \
    --base-url https://ai.v3m.ai/v1
```

### 4. Iniciar MCP Server (com Ollama - Offline)

```bash
# Via variáveis de ambiente
DB_PATH=./data/docs.db \
USE_OLLAMA=true \
./bin/docs-rag-mcp-server

# Ou via flags
./bin/docs-rag-mcp-server \
    --db ./data/docs.db \
    --ollama
```

## Integração com Claude Code

A forma recomendada de adicionar o MCP server é usando o comando `claude mcp add`.

### Configuração com Ollama (Offline)

```bash
claude mcp add -s user \
  -e DB_PATH="/Users/marcelo/myspace/claude/docs-rag-mcp/data/docs.db" \
  -e USE_OLLAMA="true" \
  -- trx-docs /Users/marcelo/myspace/claude/docs-rag-mcp/bin/docs-rag-mcp-server
```

### Configuração com OpenAI Privado

```bash
claude mcp add -s user \
  -e DB_PATH="/Users/marcelo/myspace/claude/docs-rag-mcp/data/docs.db" \
  -e OPENAI_API_KEY="your-api-key" \
  -e OPENAI_BASE_URL="https://your-openai-server.com/v1" \
  -- trx-docs /Users/marcelo/myspace/claude/docs-rag-mcp/bin/docs-rag-mcp-server
```

### Com Servidor OpenAI Padrão

```bash
claude mcp add -s user \
  -e DB_PATH="/path/to/docs-rag-mcp/data/docs.db" \
  -e OPENAI_API_KEY="sk-..." \
  -- trx-docs /path/to/docs-rag-mcp/bin/docs-rag-mcp-server
```

### Verificar Integração

```bash
# Verificar se o servidor está conectado
claude mcp list

# Saída esperada:
# trx-docs: /path/to/bin/docs-rag-mcp-server - ✓ Connected
```

### Gerenciar MCP Servers

```bash
# Listar servidores configurados
claude mcp list

# Remover um servidor
claude mcp remove trx-docs

# Ver ajuda
claude mcp --help
```

### Escopos de Configuração

O parâmetro `-s` (scope) define onde a configuração será salva:

| Scope | Descrição | Arquivo |
|-------|-----------|---------|
| `local` | Apenas no diretório atual (default) | `./.claude.json` |
| `project` | Para o projeto atual | `./.claude/settings.json` |
| `user` | Global para o usuário | `~/.claude.json` |

Recomendamos usar `-s user` para disponibilizar o MCP server em todos os projetos.

### Tools Disponíveis

Após configurar, os seguintes tools estarão disponíveis no Claude Code:

- `search_docs` - Busca semântica em documentos
- `get_relevant_context` - Contexto para uma tarefa
- `get_document` - Detalhes de um documento
- `search_by_section` - Buscar em seções específicas
- `find_related_docs` - Documentos relacionados

## Configuração

Arquivo `configs/config.yaml`:

```yaml
database:
  path: ./data/docs.db

openai:
  api_key: ${OPENAI_API_KEY}
  model: text-embedding-3-small
  rate_limit: 1000  # requests per minute

chunking:
  min_size: 500
  max_size: 1000
  overlap: 150

search:
  default_top_k: 5
  min_score: 0.7
```

## Desenvolvimento

### Executar Testes

```bash
go test ./...
go test -v -race ./...
go test -bench=. ./...
```

### Linting

```bash
golangci-lint run
```

### Build para Produção

```bash
make build-prod
```

## Performance

- **Busca vetorial**: < 100ms
- **Indexação**: ~500 docs/min (limitado por OpenAI API)
- **Embedding generation**: ~20 docs/min

## Custos OpenAI

- **Model**: text-embedding-3-small ($0.02 / 1M tokens)
- **Estimativa**: 200 docs × 50K tokens = 10M tokens = **$0.20**
- **Cache**: Evita custos recorrentes em re-indexações

## Tecnologias

- **Golang 1.22+**
- **SQLite 3** com extensão `sqlite-vec`
- **OpenAI text-embedding-3-small**
- **goldmark** (Markdown parser)
- **chi** (HTTP router)
- **React + Vite** (Web UI)

## Roadmap

- [x] Fase 1: Core Infrastructure
  - [x] Parser Markdown (goldmark)
  - [x] Chunker semântico
  - [x] SQLite store
  - [x] Indexer pipeline
- [x] Fase 2: Vector Search
  - [x] OpenAI embeddings client
  - [x] Ollama embeddings client (offline)
  - [x] Busca por similaridade
  - [x] Cache de embeddings
- [x] Fase 3: MCP Server
  - [x] Protocolo JSON-RPC 2.0
  - [x] 5 tools implementados
  - [x] Integração com Claude Code
- [ ] Fase 4: Web UI
  - [ ] HTTP API (chi router)
  - [ ] React + Vite frontend
  - [ ] Visualização de documentos
  - [ ] Interface de busca
- [ ] Fase 5: Production Ready
  - [ ] Testes de integração
  - [ ] CI/CD pipeline
  - [ ] Docker image
  - [ ] Documentação completa

## Licença

MIT

## Autor

trxio Team
