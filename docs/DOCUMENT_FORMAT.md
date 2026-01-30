# Formato de Documentos - docs-rag-mcp

Este guia descreve os requisitos e formatos suportados para documentos que serão indexados no sistema docs-rag-mcp.

## Sumário

1. [Formato Básico](#formato-básico)
2. [Metadados](#metadados)
3. [Tipos de Documento](#tipos-de-documento)
4. [Estrutura de Seções](#estrutura-de-seções)
5. [Elementos Especiais](#elementos-especiais)
6. [Cross-References](#cross-references)
7. [Exemplos Completos](#exemplos-completos)
8. [Boas Práticas](#boas-práticas)

---

## Formato Básico

O sistema aceita arquivos **Markdown** (`.md`) com as seguintes características:

- **Encoding**: UTF-8
- **Extensão**: `.md`
- **Tamanho máximo**: 10MB por arquivo
- **Naming convention**: `{TYPE}-{NUMBER}-{slug}.md` (recomendado)

### Naming Convention

O sistema extrai o ID do documento automaticamente do nome do arquivo:

```
RFC-001-microservice-architecture.md  →  ID: RFC-001
ADR-042-database-selection.md         →  ID: ADR-042
GUIDELINE-003-api-design.md           →  ID: GUIDELINE-003
```

**Padrão suportado**: `{TYPE}-{NUMBER}` onde:
- `TYPE`: RFC, ADR, BDR, GUIDELINE, ROADMAP (case-insensitive)
- `NUMBER`: Número sequencial (1-999)

---

## Metadados

Os metadados podem ser fornecidos de duas formas:

### 1. Frontmatter YAML (Recomendado)

```yaml
---
id: RFC-001
title: Microservice Architecture
type: RFC
status: Approved
version: "1.0"
author: John Doe
created_date: 2024-01-15
last_updated: 2024-01-20
tags:
  - architecture
  - microservices
---

# Conteúdo do documento...
```

### 2. Metadados Inline

Se não houver frontmatter, o parser busca padrões nas primeiras 20 linhas:

```markdown
# RFC-001: Microservice Architecture

**Status**: Approved
**Version**: 1.0
**Author**: John Doe
**Created**: 2024-01-15
**Last Updated**: 2024-01-20
```

### Campos Suportados

| Campo | Frontmatter | Inline | Descrição |
|-------|-------------|--------|-----------|
| `id` | `id` | Extraído do título/filename | Identificador único |
| `title` | `title` | Primeiro H1 | Título do documento |
| `type` | `type` | Extraído do ID | Tipo (RFC, ADR, etc.) |
| `status` | `status` | `**Status**:` | Status atual |
| `version` | `version` | `**Version**:` | Versão |
| `author` | `author` | `**Author**:` | Autor principal |
| `created_date` | `created_date` | `**Created**:` | Data de criação |
| `last_updated` | `last_updated` | `**Last Updated**:` | Última atualização |

### Formatos de Data Suportados

- `2024-01-15` (ISO 8601 - recomendado)
- `2024/01/15`
- `January 15, 2024`
- `Jan 15, 2024`
- `15-01-2024`
- `15/01/2024`

---

## Tipos de Documento

### RFC (Request for Comments)

Propostas técnicas detalhadas para mudanças significativas.

```markdown
---
id: RFC-001
type: RFC
status: Draft | Under Review | Approved | Rejected | Superseded
---

# RFC-001: Título da Proposta

## Executive Summary
Resumo executivo da proposta.

## Problem & Motivation
Problema que está sendo resolvido.

## Proposed Solution
Solução proposta em detalhes.

## Implementation Plan
Plano de implementação.

## Risks & Mitigations
Riscos identificados e mitigações.

## Success Metrics
Como medir o sucesso.
```

### ADR (Architecture Decision Record)

Registros de decisões arquiteturais.

```markdown
---
id: ADR-001
type: ADR
status: Accepted | Deprecated | Superseded
---

# ADR-001: Título da Decisão

## Context
Contexto que levou à decisão.

## Decision
A decisão tomada.

## Consequences
Consequências da decisão.

## Alternatives Considered
Alternativas que foram avaliadas.
```

### BDR (Business Decision Record)

Registros de decisões de negócio.

```markdown
---
id: BDR-001
type: BDR
status: Approved | Pending | Rejected
---

# BDR-001: Título da Decisão de Negócio

## Background
Contexto de negócio.

## Decision
Decisão tomada.

## Impact
Impacto no negócio.

## Timeline
Cronograma de implementação.
```

### Guideline

Guias e padrões técnicos.

```markdown
---
id: GUIDELINE-001
type: Guideline
status: Active | Draft | Deprecated
---

# Guideline: Título do Guia

## Overview
Visão geral do guia.

## Guidelines
### Guideline 1
...

### Guideline 2
...

## Examples
Exemplos de aplicação.

## Exceptions
Quando não se aplica.
```

### Roadmap

Planos e roadmaps de produto/técnico.

```markdown
---
id: ROADMAP-001
type: Roadmap
status: Current | Archived
---

# Roadmap: Título

## Vision
Visão de longo prazo.

## Q1 2024
- Item 1
- Item 2

## Q2 2024
- Item 3
- Item 4

## Future
Itens para o futuro.
```

---

## Estrutura de Seções

O sistema reconhece a hierarquia de seções baseada em headings Markdown:

```markdown
# Título Principal (H1)

## Seção Nível 2 (H2)

### Subseção Nível 3 (H3)

#### Subseção Nível 4 (H4)
```

### Section Path

Cada seção recebe um `section_path` hierárquico:

```
# Executive Summary
  → path: "Executive Summary"

## Problem Statement
  → path: "Executive Summary > Problem Statement"

### Technical Details
  → path: "Executive Summary > Problem Statement > Technical Details"
```

### Impacto no Chunking

- Cada seção se torna um ou mais chunks
- Seções grandes (>1000 tokens) são divididas por parágrafos
- O `section_path` é incluído como contexto em cada chunk

---

## Elementos Especiais

### Code Blocks

Code blocks são preservados integralmente e marcados nos chunks:

````markdown
```javascript
function example() {
  return "Hello World";
}
```
````

**Metadados extraídos:**
- Linguagem de programação
- Linha inicial/final
- Flag `has_code_block` no chunk

### Tabelas

Tabelas Markdown são reconhecidas e preservadas:

```markdown
| Coluna 1 | Coluna 2 | Coluna 3 |
|----------|----------|----------|
| Valor A  | Valor B  | Valor C  |
| Valor D  | Valor E  | Valor F  |
```

**Metadados extraídos:**
- Headers
- Linhas de dados
- Flag `has_table` no chunk

### Listas

Listas ordenadas e não-ordenadas são preservadas:

```markdown
- Item 1
- Item 2
  - Sub-item 2.1
  - Sub-item 2.2

1. Primeiro
2. Segundo
3. Terceiro
```

### Blockquotes

```markdown
> Citação importante que será indexada
> junto com o conteúdo.
```

### Links

Links são extraídos para cross-referencing:

```markdown
[Texto do link](./RFC-002-outro-documento.md)
[Referência externa](https://example.com)
```

---

## Cross-References

O sistema detecta automaticamente referências entre documentos.

### Formatos Reconhecidos

```markdown
<!-- Link direto -->
See [RFC-001](./RFC-001-microservices.md) for details.

<!-- Menção no texto -->
As defined in RFC-001, the architecture should...

<!-- Link com ID no texto -->
This supersedes [ADR-042](../adr/ADR-042-old-decision.md).
```

### Padrões Detectados

O parser usa regex para encontrar IDs de documentos:

```
(RFC|ADR|BDR|GUIDELINE|ROADMAP)-(\d+)
```

### Grafo de Referências

Cross-references criam um grafo direcionado:

```
RFC-001 ──references──> ADR-042
RFC-001 ──references──> GUIDELINE-003
RFC-002 ──references──> RFC-001
```

Este grafo é usado pelo tool `find_related_docs` para navegar entre documentos relacionados.

---

## Exemplos Completos

### RFC Completo

```markdown
---
id: RFC-001
title: Microservice Architecture
type: RFC
status: Approved
version: "1.2"
author: Jane Smith
created_date: 2024-01-10
last_updated: 2024-02-15
tags:
  - architecture
  - microservices
  - kubernetes
---

# RFC-001: Microservice Architecture

## Executive Summary

This RFC proposes a microservice architecture for our platform, replacing the current monolithic approach. The goal is to improve scalability, deployment flexibility, and team autonomy.

## Problem & Motivation

### Current State

The existing monolithic architecture has several limitations:

- Deployment requires full system restart
- Scaling is all-or-nothing
- Team dependencies slow development

### Desired State

We want to achieve:

1. Independent service deployment
2. Horizontal scaling per service
3. Team ownership of services

## Proposed Solution

### Architecture Overview

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   API GW    │────▶│  Service A  │────▶│  Database   │
└─────────────┘     └─────────────┘     └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │  Service B  │
                    └─────────────┘
```

### Service Boundaries

| Service | Responsibility | Team |
|---------|---------------|------|
| User Service | Authentication, profiles | Platform |
| Order Service | Orders, payments | Commerce |
| Notification | Email, push, SMS | Growth |

### Technology Stack

- **Runtime**: Kubernetes 1.28+
- **Language**: Go 1.22+
- **Database**: PostgreSQL 15
- **Messaging**: Apache Kafka

## Implementation Plan

### Phase 1: Foundation (Q1)

1. Set up Kubernetes cluster
2. Implement API Gateway
3. Create service template

### Phase 2: Migration (Q2)

1. Extract User Service
2. Extract Order Service
3. Implement event sourcing

See [ADR-042](../adr/ADR-042-event-sourcing.md) for event sourcing decision.

## Risks & Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Network latency | Medium | High | Service mesh, caching |
| Data consistency | High | High | Saga pattern |
| Operational complexity | High | Medium | Platform team training |

## Success Metrics

- Deployment frequency: 10x increase
- Lead time: < 1 day
- Service availability: 99.9%

## References

- [GUIDELINE-003](../guidelines/GUIDELINE-003-api-design.md) - API Design Guidelines
- [ADR-042](../adr/ADR-042-event-sourcing.md) - Event Sourcing Decision
```

### Guideline Completo

```markdown
---
id: GUIDELINE-003
title: API Design Guidelines
type: Guideline
status: Active
version: "2.0"
author: API Team
created_date: 2023-06-01
last_updated: 2024-01-15
---

# Guideline: API Design

## Overview

This document defines standards for designing REST APIs in our platform.

## Guidelines

### 1. URL Structure

Use lowercase, hyphen-separated names:

```
✅ GET /api/v1/user-profiles
❌ GET /api/v1/UserProfiles
❌ GET /api/v1/user_profiles
```

### 2. HTTP Methods

| Method | Usage | Idempotent |
|--------|-------|------------|
| GET | Retrieve resource | Yes |
| POST | Create resource | No |
| PUT | Replace resource | Yes |
| PATCH | Partial update | No |
| DELETE | Remove resource | Yes |

### 3. Response Codes

```javascript
// Success
200 OK           // GET, PUT, PATCH success
201 Created      // POST success
204 No Content   // DELETE success

// Client Errors
400 Bad Request  // Invalid input
401 Unauthorized // Missing/invalid auth
403 Forbidden    // No permission
404 Not Found    // Resource not found

// Server Errors
500 Internal     // Unexpected error
503 Unavailable  // Service down
```

### 4. Error Format

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid email format",
    "details": [
      {
        "field": "email",
        "message": "Must be a valid email address"
      }
    ]
  }
}
```

## Examples

### Creating a Resource

```bash
curl -X POST https://api.example.com/v1/users \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "John Doe",
    "email": "john@example.com"
  }'
```

Response:
```json
{
  "id": "usr_123",
  "name": "John Doe",
  "email": "john@example.com",
  "created_at": "2024-01-15T10:30:00Z"
}
```

## Exceptions

These guidelines may not apply when:

- Integrating with legacy systems
- Third-party API requirements
- Performance-critical endpoints (document deviations)

## Related Documents

- [RFC-005](../rfc/RFC-005-api-versioning.md) - API Versioning Strategy
- [ADR-010](../adr/ADR-010-rest-vs-graphql.md) - REST vs GraphQL Decision
```

---

## Boas Práticas

### Estrutura Recomendada

```
docs/
├── rfc/
│   ├── RFC-001-microservices.md
│   ├── RFC-002-authentication.md
│   └── RFC-003-data-pipeline.md
├── adr/
│   ├── ADR-001-database-choice.md
│   ├── ADR-002-framework-selection.md
│   └── ADR-003-cloud-provider.md
├── guidelines/
│   ├── GUIDELINE-001-coding-standards.md
│   ├── GUIDELINE-002-testing.md
│   └── GUIDELINE-003-api-design.md
└── roadmap/
    └── ROADMAP-001-2024.md
```

### Checklist de Qualidade

- [ ] Nome do arquivo segue padrão `{TYPE}-{NUMBER}-{slug}.md`
- [ ] Frontmatter YAML com campos obrigatórios
- [ ] Título H1 claro e descritivo
- [ ] Seções com hierarquia lógica (H2, H3, H4)
- [ ] Code blocks com linguagem especificada
- [ ] Tabelas formatadas corretamente
- [ ] Links para documentos relacionados
- [ ] Status atualizado
- [ ] Data de última atualização

### O Que Evitar

1. **Títulos muito longos**: Máximo 80 caracteres
2. **Seções vazias**: Remover seções sem conteúdo
3. **Links quebrados**: Verificar referências
4. **Metadados inconsistentes**: Usar formatos padronizados
5. **Arquivos muito grandes**: Dividir em múltiplos documentos se > 5000 palavras
6. **Imagens embutidas**: Não são indexadas (usar texto descritivo)

### Otimização para Busca

Para melhorar a qualidade da busca semântica:

1. **Use títulos descritivos**: "Authentication Flow" > "Section 3"
2. **Inclua contexto**: Explique o "porquê", não só o "como"
3. **Mantenha parágrafos focados**: Um conceito por parágrafo
4. **Use terminologia consistente**: Mesmo termo para mesmo conceito
5. **Adicione exemplos**: Código e casos de uso melhoram relevância

---

## Troubleshooting

### Documento não aparece na busca

1. Verifique se o upload foi bem-sucedido
2. Confirme que o projeto está ativo
3. Verifique os logs do servidor

### ID não foi extraído

1. Verifique o nome do arquivo
2. Adicione `id` no frontmatter
3. Certifique que o padrão é `{TYPE}-{NUMBER}`

### Metadados não foram reconhecidos

1. Use frontmatter YAML (mais confiável)
2. Verifique formatação dos campos inline
3. Campos devem estar nas primeiras 20 linhas

### Chunks muito pequenos/grandes

O chunker usa configuração padrão:
- Mínimo: 500 tokens
- Máximo: 1000 tokens
- Overlap: 150 tokens

Documentos com seções muito curtas terão chunks menores.
