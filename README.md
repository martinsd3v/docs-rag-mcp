# docs-rag-mcp

RAG (Retrieval Augmented Generation) MCP Server para documentação técnica em Golang.

## Visão Geral

Sistema que indexa documentação técnica (RFCs, ADRs, BDRs, Guidelines) usando embeddings vetoriais e expõe via MCP (Model Context Protocol) para Claude Code, permitindo acesso contextual à documentação sem saturar o contexto da conversa.

## Arquitetura

```
Claude Code (MCP Client)
    │
    │ HTTP/SSE
    ▼
┌─────────────────────────────────────────────────────────────┐
│  docs-rag-web (Go)                                          │
│                                                             │
│  /api/*           → REST API (projetos, docs, search)       │
│  /mcp/sse         → SSE stream para MCP (GET)               │
│  /mcp/message     → Receber mensagens MCP (POST)            │
│  /*               → Frontend React (SPA)                    │
└─────────────────────────────────────────────────────────────┘
    │
    ▼
SQLite + sqlite-vec ←→ OpenAI API / Ollama (embeddings)
```

## Estrutura do Projeto

```
docs-rag-mcp/
├── cmd/
│   └── web/                 # Entrypoint único do servidor
├── internal/
│   ├── api/                 # REST API + handlers + project manager
│   ├── mcp/                 # MCP protocol (HTTP/SSE transport)
│   ├── embeddings/          # Clients OpenAI/Ollama
│   ├── vector/              # SQLite-vec wrapper
│   └── parser/              # Markdown parser + chunking
├── web/                     # React frontend (Vite + TypeScript)
├── db/
│   └── schema.sql           # Schema do banco de dados
├── docs/                    # Documentação técnica
├── data/                    # Diretório de dados (projetos + DBs)
│   └── projects/            # Bancos de dados por projeto
├── go.mod
├── Makefile
└── README.md
```

## Instalação

### Requisitos

- Go 1.22+
- Node.js 18+ (para build do frontend)
- Ollama (opcional, para embeddings offline)

### Build

```bash
# Clone o repositório
git clone https://github.com/trxio/docs-rag-mcp.git
cd docs-rag-mcp

# Instalar dependências Go
go mod download

# Build completo (backend + frontend)
make build-all

# Ou apenas backend
make build
```

## Configuração de Embeddings

O sistema suporta dois providers de embeddings, configurados diretamente pela interface:

### Ollama (Offline/Local)

```bash
# Instalar Ollama
brew install ollama          # macOS
# ou
curl -fsSL https://ollama.com/install.sh | sh  # Linux

# Baixar modelo recomendado
ollama pull nomic-embed-text

# Iniciar serviço
ollama serve
```

### OpenAI API (ou compatível)

Qualquer API compatível com OpenAI embeddings:
- OpenAI oficial
- Azure OpenAI
- APIs self-hosted (vLLM, LocalAI, etc.)

## Uso

### 1. Iniciar o Servidor

```bash
# Modo padrão
./bin/docs-rag-web

# Com opções personalizadas
./bin/docs-rag-web --port 8080 --data-dir ./data --static-dir ./web/dist
```

**Flags disponíveis:**

| Flag | Default | Descrição |
|------|---------|-----------|
| `--port`, `-p` | `8080` | Porta do servidor HTTP |
| `--data-dir` | `./data` | Diretório para projetos e bancos de dados |
| `--static-dir` | `./web/dist` | Diretório dos arquivos estáticos do frontend |

### 2. Acessar a Interface

Abra http://localhost:8080 no navegador.

### 3. Criar um Projeto

Na interface web:

1. Clique em **"New Project"**
2. Preencha os campos:
   - **Name**: Nome do projeto (ex: "Minha Documentação")
   - **Host**: URL do serviço de embeddings
     - Ollama: `http://localhost:11434`
     - OpenAI: `https://api.openai.com`
   - **Token**: API key (deixe vazio para Ollama)
   - **Model**: Modelo de embeddings
     - Ollama: `nomic-embed-text`
     - OpenAI: `text-embedding-3-small`
3. Clique em **"Create"**

### 4. Ativar o Projeto

Clique em **RUN** no projeto para:
- Inicializar o banco de dados SQLite
- Conectar ao serviço de embeddings
- Ativar o endpoint MCP via HTTP/SSE

### 5. Upload de Documentos

Na aba **"Upload"**:
1. Arraste arquivos `.md` ou clique para selecionar
2. Os documentos são processados automaticamente:
   - Parse do Markdown
   - Extração de metadados (título, autor, status)
   - Chunking semântico
   - Geração de embeddings
   - Extração de cross-references

## Integração com Claude Code

### Adicionar MCP Server

```bash
claude mcp add --transport sse trx-docs http://localhost:8080/mcp/sse
```

### Verificar Conexão

```bash
claude mcp list

# Esperado:
# trx-docs: http://localhost:8080/mcp/sse - ✓ Connected
```

### Remover

```bash
claude mcp remove trx-docs
```

### Tools Disponíveis

Após configurar, os seguintes tools estarão disponíveis no Claude Code:

| Tool | Descrição |
|------|-----------|
| `search_docs` | Busca semântica em documentos indexados |
| `get_relevant_context` | Obtém contexto relevante para uma tarefa |
| `get_document` | Retorna documento completo por ID |
| `search_by_section` | Busca em seções específicas (Implementation, Examples, etc.) |
| `find_related_docs` | Encontra documentos relacionados via grafo de referências |

## API REST

### Endpoints de Projeto (sempre disponíveis)

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| `GET` | `/api/projects` | Lista todos os projetos |
| `POST` | `/api/projects` | Cria novo projeto |
| `DELETE` | `/api/projects/{id}` | Remove projeto |
| `POST` | `/api/projects/{id}/start` | Ativa projeto |
| `POST` | `/api/projects/stop` | Para projeto ativo |
| `GET` | `/api/projects/active` | Retorna projeto ativo |
| `GET` | `/api/health` | Health check |

### Endpoints de Documentos (requerem projeto ativo)

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| `GET` | `/api/stats` | Estatísticas do banco |
| `GET` | `/api/documents` | Lista documentos (paginado) |
| `GET` | `/api/documents/{id}` | Detalhes de um documento |
| `POST` | `/api/search` | Busca semântica |
| `POST` | `/api/upload` | Upload de arquivos .md |

### Endpoints MCP (requerem projeto ativo)

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| `GET` | `/mcp/sse` | SSE stream para conexão MCP |
| `POST` | `/mcp/message` | Receber mensagens JSON-RPC |

## Documentação

- [Arquitetura Técnica](docs/ARCHITECTURE.md) - Detalhes de implementação
- [Formato de Documentos](docs/DOCUMENT_FORMAT.md) - Guia de formatação

## Desenvolvimento

### Rodar em Desenvolvimento

```bash
# Backend + Frontend com hot reload
make run-web-dev

# Ou apenas backend
make run-web
```

### Testes

```bash
# Rodar todos os testes
make test

# Com coverage
make test-coverage
```

### Lint

```bash
make lint
```

### Build de Produção

```bash
make build-prod
```

## Formato de Documentos

O sistema reconhece documentos Markdown com frontmatter YAML:

```markdown
---
id: RFC-001
title: Título do Documento
type: RFC
status: Approved
version: "1.0"
author: Nome do Autor
created_date: 2024-01-15
last_updated: 2024-01-20
---

# Conteúdo do documento

## Seção 1

Texto...
```

### Tipos de Documento Suportados

- **RFC** - Request for Comments
- **ADR** - Architecture Decision Record
- **BDR** - Business Decision Record
- **Guideline** - Guias e padrões
- **Roadmap** - Planos e roadmaps

Veja [Formato de Documentos](docs/DOCUMENT_FORMAT.md) para guia completo.

## Tecnologias

- **Go 1.22+** - Backend
- **SQLite 3** com extensão `sqlite-vec` - Banco vetorial
- **OpenAI / Ollama** - Embeddings
- **goldmark** - Parser Markdown
- **chi** - HTTP router
- **React 18 + Vite + TypeScript** - Frontend
- **Tailwind CSS** - Estilos

## Licença

MIT

## Autor

trxio Team
