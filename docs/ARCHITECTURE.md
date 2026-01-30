# Arquitetura Técnica - docs-rag-mcp

Este documento descreve em detalhes a arquitetura e funcionamento técnico do sistema docs-rag-mcp.

## Sumário

1. [Visão Geral](#visão-geral)
2. [Componentes Principais](#componentes-principais)
3. [Fluxo de Dados](#fluxo-de-dados)
4. [MCP Protocol](#mcp-protocol)
5. [Sistema de Embeddings](#sistema-de-embeddings)
6. [Banco de Dados](#banco-de-dados)
7. [Pipeline de Indexação](#pipeline-de-indexação)
8. [API REST](#api-rest)

---

## Visão Geral

O docs-rag-mcp é um sistema RAG (Retrieval Augmented Generation) que permite indexar documentação técnica e expô-la via MCP (Model Context Protocol) para Claude Code.

### Arquitetura em Camadas

```
┌────────────────────────────────────────────────────────────────┐
│                        CAMADA DE APRESENTAÇÃO                  │
│                                                                │
│   React Frontend (SPA)          Claude Code (MCP Client)       │
│        ↓                              ↓                        │
│   REST API (/api/*)           MCP HTTP/SSE (/mcp/*)            │
└────────────────────────────────────────────────────────────────┘
                              ↓
┌────────────────────────────────────────────────────────────────┐
│                        CAMADA DE APLICAÇÃO                     │
│                                                                │
│   ┌──────────────┐    ┌──────────────┐    ┌──────────────┐     │
│   │  API Server  │    │  MCP Handler │    │   Project    │     │
│   │   (chi)      │    │  (HTTP/SSE)  │    │   Manager    │     │
│   └──────────────┘    └──────────────┘    └──────────────┘     │
└────────────────────────────────────────────────────────────────┘
                              ↓
┌────────────────────────────────────────────────────────────────┐
│                        CAMADA DE SERVIÇOS                      │
│                                                                │
│   ┌──────────────┐    ┌──────────────┐    ┌──────────────┐     │
│   │   Parser     │    │   Chunker    │    │  Embeddings  │     │
│   │  (goldmark)  │    │  (semântico) │    │   Client     │     │
│   └──────────────┘    └──────────────┘    └──────────────┘     │
└────────────────────────────────────────────────────────────────┘
                              ↓
┌────────────────────────────────────────────────────────────────┐
│                        CAMADA DE DADOS                         │
│                                                                │
│   ┌──────────────────────────────────────────────────────┐     │
│   │              SQLite + sqlite-vec                     │     │
│   │                                                      │     │
│   │   documents │ chunks │ embeddings │ cross_references │     │
│   └──────────────────────────────────────────────────────┘     │
└────────────────────────────────────────────────────────────────┘
```

---

## Componentes Principais

### 1. Web Server (`cmd/web`)

Ponto de entrada único do sistema. Responsável por:

- Inicializar o servidor HTTP na porta configurada (default: 8080)
- Gerenciar o ProjectManager para CRUD de projetos
- Servir arquivos estáticos do frontend React
- Configurar graceful shutdown

**Flags:**
- `--port`, `-p`: Porta HTTP (default: 8080)
- `--data-dir`: Diretório de dados (default: `./data`)
- `--static-dir`: Diretório do frontend (default: `./web/dist`)

### 2. API Server (`internal/api`)

Gerencia todas as rotas HTTP e lógica de negócio.

**Estrutura:**
```
internal/api/
├── server.go           # Router principal e middleware
├── project_manager.go  # CRUD de projetos (JSON store)
├── project_handlers.go # Handlers de projeto (start/stop)
├── upload.go          # Handler de upload de arquivos
├── handlers/          # Handlers adicionais
└── models/            # Tipos de request/response
```

**Middleware Stack:**
1. RequestID - Adiciona ID único a cada request
2. RealIP - Extrai IP real do cliente
3. Logger - Log de requests
4. Recoverer - Recupera de panics
5. Compress - Compressão gzip
6. CORS - Cross-Origin Resource Sharing

### 3. MCP Handler (`internal/mcp`)

Implementa o protocolo MCP sobre HTTP/SSE.

**Estrutura:**
```
internal/mcp/
├── server.go          # Tipos MCP e servidor stdio (legado)
├── http_transport.go  # Transport HTTP/SSE
└── tools.go          # Implementação dos tools
```

**Componentes:**
- `MCPHTTPHandler`: Gerencia sessões SSE e processa mensagens JSON-RPC
- `SSESession`: Representa uma conexão SSE ativa
- `ToolsHandler`: Implementa os 6 tools disponíveis

### 4. Parser (`internal/parser`)

Parse de documentos Markdown usando goldmark.

**Estrutura:**
```
internal/parser/
├── parser.go    # Parse de Markdown + extração de metadados
└── chunker.go   # Chunking semântico
```

**Funcionalidades:**
- Extração de frontmatter YAML
- Parse de metadados inline (Status, Version, Author)
- Identificação de seções hierárquicas
- Extração de code blocks, tabelas e links
- Cross-references para outros documentos

### 5. Embeddings (`internal/embeddings`)

Clientes para geração de embeddings vetoriais.

**Providers suportados:**
- OpenAI API (e compatíveis)
- Ollama (local)

**Funcionalidades:**
- Detecção automática de provider baseado na URL
- Batch processing para múltiplos textos
- Cache de embeddings (em desenvolvimento)

### 6. Vector Store (`internal/vector`)

Wrapper para SQLite com extensão sqlite-vec.

**Funcionalidades:**
- Busca por similaridade vetorial (KNN)
- CRUD de documentos, chunks e embeddings
- Gerenciamento de cross-references
- Estatísticas do banco

---

## Fluxo de Dados

### Upload de Documento

```
1. Cliente envia POST /api/upload com arquivo .md
                    ↓
2. Upload Handler recebe arquivo
                    ↓
3. Parser.ParseFile()
   - Extrai frontmatter/metadados
   - Identifica seções
   - Extrai links e referências
                    ↓
4. Chunker.ChunkDocument()
   - Divide em chunks semânticos
   - Respeita limites de tamanho (500-1000 tokens)
   - Adiciona overlap entre chunks
                    ↓
5. EmbeddingClient.EmbedBatch()
   - Gera embeddings para cada chunk
   - Usa OpenAI ou Ollama
                    ↓
6. Store.Insert*()
   - Salva documento
   - Salva chunks
   - Salva embeddings
   - Salva cross-references
                    ↓
7. Response com status de sucesso
```

### Busca Semântica

```
1. Cliente envia POST /api/search ou MCP tools/call
                    ↓
2. EmbeddingClient.Embed()
   - Gera embedding da query
                    ↓
3. Store.SearchSimilar()
   - Busca K nearest neighbors
   - Filtra por score mínimo
   - Aplica filtros (doc_types)
                    ↓
4. Enriquecimento de resultados
   - Carrega metadados do documento
   - Inclui cross-references
                    ↓
5. Response com resultados ordenados por score
```

---

## MCP Protocol

### Transporte HTTP/SSE

O sistema implementa MCP sobre HTTP usando Server-Sent Events (SSE):

```
┌─────────────────┐                    ┌─────────────────┐
│  Claude Code    │                    │  docs-rag-web   │
│  (MCP Client)   │                    │  (MCP Server)   │
└────────┬────────┘                    └────────┬────────┘
         │                                      │
         │  1. GET /mcp/sse                     │
         │ ─────────────────────────────────────>
         │                                      │
         │  2. SSE: event:endpoint              │
         │     data:/mcp/message?sessionId=xxx  │
         │ <─────────────────────────────────────
         │                                      │
         │  3. POST /mcp/message?sessionId=xxx  │
         │     {"method":"initialize",...}      │
         │ ─────────────────────────────────────>
         │                                      │
         │  4. SSE: event:message               │
         │     data:{"result":{...}}            │
         │ <─────────────────────────────────────
         │                                      │
         │  5. POST /mcp/message                │
         │     {"method":"tools/call",...}      │
         │ ─────────────────────────────────────>
         │                                      │
         │  6. SSE: event:message               │
         │     data:{"result":{...}}            │
         │ <─────────────────────────────────────
```

### Sessões SSE

Cada conexão SSE cria uma sessão:

```go
type SSESession struct {
    ID        string           // UUID da sessão
    Writer    http.ResponseWriter
    Flusher   http.Flusher
    Created   time.Time
    Messages  chan *Response   // Canal de mensagens (buffer: 100)
    Done      chan struct{}    // Canal de encerramento
    Ctx       context.Context
    CancelFn  context.CancelFunc
}
```

**Lifecycle:**
1. Cliente conecta em GET /mcp/sse
2. Servidor cria sessão e envia evento `endpoint`
3. Cliente envia requests via POST /mcp/message
4. Servidor processa e envia respostas via SSE
5. Keepalive a cada 30 segundos
6. Cleanup automático ao desconectar

### Tools Disponíveis

| Tool | Descrição | Parâmetros |
|------|-----------|------------|
| `search_docs` | Busca semântica | `query`, `doc_types[]`, `top_k`, `min_score` |
| `get_relevant_context` | Contexto para tarefa | `task`, `max_chunks`, `expand_graph` |
| `get_document` | Documento por ID | `doc_id`, `include_chunks`, `include_related` |
| `search_by_section` | Busca em seções | `query`, `section_names[]`, `doc_types[]`, `top_k` |
| `find_related_docs` | Docs relacionados | `doc_id`, `max_depth`, `include_incoming` |

---

## Sistema de Embeddings

### Provider Detection

O sistema detecta automaticamente o provider baseado na URL:

```go
func DetectProvider(baseURL, apiKey string) Provider {
    if strings.Contains(baseURL, "11434") {
        return ProviderOllama
    }
    if strings.Contains(baseURL, "ollama") {
        return ProviderOllama
    }
    if apiKey == "ollama" || apiKey == "" {
        return ProviderOllama
    }
    return ProviderOpenAI
}
```

### Ollama Client

- Endpoint: `POST /api/embeddings`
- Modelo padrão: `nomic-embed-text`
- Dimensões: 768 (nomic-embed-text)

```json
{
  "model": "nomic-embed-text",
  "prompt": "texto para embedar"
}
```

### OpenAI Client

- Endpoint: `POST /v1/embeddings`
- Modelo padrão: `text-embedding-3-small`
- Dimensões: 1536

```json
{
  "model": "text-embedding-3-small",
  "input": ["texto1", "texto2"]
}
```

### Batch Processing

Para múltiplos textos, o sistema usa batch processing:

```go
func (c *Client) EmbedBatch(ctx context.Context, texts []string) (*BatchResult, error)
```

- OpenAI: Envia todos os textos em uma única request
- Ollama: Processa um texto por vez (limitação da API)

---

## Banco de Dados

### Schema

```sql
-- Documentos indexados
CREATE TABLE documents (
    id TEXT PRIMARY KEY,
    doc_type TEXT NOT NULL,
    title TEXT NOT NULL,
    file_path TEXT,
    status TEXT,
    version TEXT,
    author TEXT,
    created_date TEXT,
    last_updated TEXT,
    file_hash TEXT,
    metadata TEXT,
    indexed_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Chunks de documentos
CREATE TABLE chunks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    doc_id TEXT NOT NULL REFERENCES documents(id),
    chunk_index INTEGER NOT NULL,
    section_path TEXT,
    section_level INTEGER,
    content TEXT NOT NULL,
    content_hash TEXT,
    token_count INTEGER,
    start_line INTEGER,
    end_line INTEGER,
    has_code_block BOOLEAN DEFAULT FALSE,
    has_table BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Embeddings vetoriais (sqlite-vec)
CREATE VIRTUAL TABLE chunk_embeddings USING vec0(
    chunk_id INTEGER PRIMARY KEY,
    embedding FLOAT[1536]  -- ou 768 para Ollama
);

-- Cross-references entre documentos
CREATE TABLE cross_references (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_doc_id TEXT NOT NULL,
    target_doc_id TEXT NOT NULL,
    reference_type TEXT,
    context TEXT,
    line_number INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Busca Vetorial

O sqlite-vec permite busca por K-Nearest Neighbors:

```sql
SELECT
    c.id, c.doc_id, c.content,
    e.distance
FROM chunk_embeddings e
INNER JOIN chunks c ON c.id = e.chunk_id
WHERE e.embedding MATCH ?
ORDER BY e.distance
LIMIT ?
```

---

## Pipeline de Indexação

### 1. Parse

```go
parser := parser.NewParser()
doc, err := parser.ParseFile(filename, content)
```

**Extração:**
- ID do documento (padrão: `{TYPE}-{NUMBER}`)
- Título (primeiro H1 ou filename)
- Metadados (Status, Version, Author, datas)
- Seções hierárquicas
- Code blocks e tabelas
- Links e cross-references

### 2. Chunking

```go
chunker := parser.NewChunker(parser.DefaultChunkerConfig())
chunks, err := chunker.ChunkDocument(doc)
```

**Configuração padrão:**
- MinSize: 500 tokens
- MaxSize: 1000 tokens
- Overlap: 150 tokens
- RespectBoundaries: true

**Estratégia:**
1. Cada seção vira um ou mais chunks
2. Seções grandes são divididas por parágrafos
3. Overlap é adicionado entre chunks consecutivos
4. Contexto (section path) é prefixado em cada chunk

### 3. Embedding

```go
result, err := embeddingClient.EmbedBatch(ctx, chunkContents)
```

### 4. Storage

```go
store.InsertDocument(doc)
store.InsertChunk(chunk)
store.InsertChunkEmbedding(chunkID, embedding, model)
store.InsertCrossReference(ref)
```

---

## API REST

### Autenticação

Atualmente não há autenticação implementada. Todas as rotas são públicas.

### Endpoints

#### Projetos

| Método | Rota | Descrição |
|--------|------|-----------|
| GET | `/api/projects` | Lista projetos |
| POST | `/api/projects` | Cria projeto |
| DELETE | `/api/projects/{id}` | Remove projeto |
| POST | `/api/projects/{id}/start` | Ativa projeto |
| POST | `/api/projects/stop` | Para projeto ativo |
| GET | `/api/projects/active` | Projeto ativo atual |

#### Documentos (requer projeto ativo)

| Método | Rota | Descrição |
|--------|------|-----------|
| GET | `/api/documents` | Lista documentos (paginado) |
| GET | `/api/documents/{id}` | Detalhes de documento |
| POST | `/api/upload` | Upload de arquivos .md |
| POST | `/api/search` | Busca semântica |
| GET | `/api/stats` | Estatísticas |
| GET | `/api/health` | Health check |

### Formato de Erro

```json
{
  "error": "mensagem de erro",
  "details": "detalhes técnicos (opcional)"
}
```

### Paginação

```json
{
  "documents": [...],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

---

## Considerações de Performance

### Busca Vetorial

- sqlite-vec usa busca linear (scan completo)
- Para datasets grandes (>100k chunks), considerar FAISS ou Milvus
- Índices auxiliares em doc_type e section_path aceleram filtros

### Embedding Generation

- Ollama: ~50-100ms por texto (local)
- OpenAI: ~100-500ms por batch (rede)
- Batch processing reduz overhead de rede

### Memory

- Embeddings são carregados sob demanda
- SQLite usa mmap para acesso eficiente
- Frontend usa lazy loading para documentos

---

## Limitações Conhecidas

1. **Sem autenticação**: API pública, adequada apenas para uso local/interno
2. **Single tenant**: Um projeto ativo por vez
3. **Busca linear**: sqlite-vec não usa índices aproximados
4. **Sem cache de embeddings**: Cada busca gera novo embedding
5. **Ollama single-thread**: Batch processing é sequencial

---

## Extensibilidade

### Adicionar novo Provider de Embeddings

1. Implementar interface `EmbeddingClient`
2. Adicionar detecção em `DetectProvider()`
3. Registrar no factory `NewEmbeddingClient()`

### Adicionar novo Tool MCP

1. Definir schema em `tools.go`
2. Implementar handler function
3. Registrar em `registerTools()`
4. Atualizar documentação

### Adicionar novo tipo de Documento

1. Atualizar regex em `parser.go`
2. Adicionar ao enum de `DocTypes`
3. Atualizar filtros no frontend
