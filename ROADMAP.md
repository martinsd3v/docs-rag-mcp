# ROADMAP - docs-rag-mcp

## Status Atual

| Fase | Status | Descrição |
|------|--------|-----------|
| Fase 1 | ✅ Completa | Core Infrastructure |
| Fase 2 | ✅ Completa | Vector Search |
| Fase 3 | ✅ Completa | MCP Server |
| Fase 4 | ✅ Completa | Web UI |
| Fase 5 | ⏳ Pendente | Production Ready |

---

## Fase 4: Web UI ✅

Interface web moderna implementada com:

- [x] HTTP API com chi router (`internal/api/`)
- [x] React + Vite + TypeScript frontend (`web/`)
- [x] TailwindCSS para estilização
- [x] Dashboard com estatísticas
- [x] Listagem de documentos com paginação e filtros
- [x] Busca semântica com resultados em tempo real
- [x] Visualização detalhada de documentos
- [x] Design moderno com sidebar e cards
- [x] Suporte a Ollama (offline)

### Rodando a Web UI

```bash
# Build
make build-all

# Iniciar servidor (produção)
./bin/docs-rag-web --ollama --db ./data/docs.db --port 8080

# Desenvolvimento com hot reload
make run-web-dev
```

---

## Fase 5: Production Ready

Preparação para uso em produção.

- [ ] Testes de integração completos
- [ ] CI/CD pipeline (GitHub Actions)
- [ ] Docker image multi-stage
- [ ] Docker Compose para desenvolvimento
- [ ] Helm chart para Kubernetes
- [ ] Documentação de API (OpenAPI/Swagger)
- [ ] Health checks e métricas (Prometheus)

---

## Melhorias Identificadas

### Chunking & Indexação

- [ ] Melhorar chunking semântico
  - Respeitar blocos de código completos
  - Não quebrar tabelas no meio
  - Detectar limites de seções mais precisamente
- [ ] Suporte a mais formatos
  - `.rst` (reStructuredText)
  - `.txt` (plain text)
  - `.html` (páginas web)
- [ ] Indexação incremental mais eficiente
  - Detectar apenas arquivos modificados
  - Re-indexar apenas chunks afetados
- [ ] Detecção automática de tipo de documento
  - Inferir tipo pelo conteúdo quando não está no filename

### Busca & Retrieval

- [ ] Re-ranking com cross-encoder
  - Usar modelo de reranking para melhorar precisão
- [ ] Filtros avançados
  - Por data de criação/atualização
  - Por autor
  - Por status (Draft, Approved, Deprecated)
- [ ] Busca híbrida
  - Combinar BM25 (keyword) + vetorial (semântico)
  - Score ponderado configurável
- [ ] Highlight de trechos relevantes
  - Marcar partes do texto que matchearam a query

### MCP Tools Adicionais

- [ ] `list_all_docs` - Listar todos os documentos indexados
- [ ] `get_index_stats` - Estatísticas do índice
- [ ] Paginação nos resultados de busca
- [ ] `get_doc_summary` - Resumo gerado de um documento

---

## Nova Feature: Project Memory

Sistema para o Claude salvar e consultar "memórias" do projeto - decisões, arquitetura, contexto importante.

### Novos MCP Tools

| Tool | Descrição |
|------|-----------|
| `save_memory` | Salvar nova memória do projeto |
| `query_memory` | Buscar memórias por query semântica |
| `list_memories` | Listar memórias com filtros |
| `update_memory` | Atualizar memória existente |
| `delete_memory` | Remover memória |

### Categorias de Memória

- `decision` - Decisões tomadas durante o desenvolvimento
- `architecture` - Detalhes e decisões de arquitetura
- `implementation` - Notas de implementação específicas
- `context` - Contexto geral do projeto
- `note` - Notas e observações diversas

### Schema

```sql
CREATE TABLE project_memories (
    id TEXT PRIMARY KEY,
    category TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    tags TEXT,           -- JSON array
    related_docs TEXT,   -- JSON array de doc IDs
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

CREATE TABLE memory_embeddings (
    memory_id TEXT PRIMARY KEY,
    embedding BLOB NOT NULL,
    model TEXT NOT NULL
);
```

### Casos de Uso

1. **Continuidade entre sessões**: Claude salva decisões importantes e consulta em sessões futuras
2. **Documentação automática**: Registrar arquitetura e decisões enquanto implementa
3. **Contexto de projeto**: Armazenar informações que não estão nos RFCs/ADRs
4. **Troubleshooting**: Registrar problemas encontrados e soluções

---

## Prioridades Sugeridas

### Alta Prioridade
1. Project Memory (manter contexto entre sessões)
2. `list_all_docs` tool (navegação básica)
3. Filtros avançados de busca

### Média Prioridade
4. Web UI básica (dashboard + busca)
5. Re-ranking com cross-encoder
6. Busca híbrida

### Baixa Prioridade
7. Suporte a mais formatos
8. Docker/Kubernetes
9. CI/CD pipeline

---

## Changelog

### v0.1.0 (Atual)
- Core infrastructure completa
- Parser Markdown com goldmark
- Chunker semântico
- SQLite vector store
- OpenAI + Ollama embeddings
- MCP Server com 5 tools
- Integração com Claude Code
