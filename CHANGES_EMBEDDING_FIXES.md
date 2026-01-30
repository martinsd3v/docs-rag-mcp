# Correções de Embedding e Limpeza de Dados

## Resumo das Mudanças

Foram implementadas correções críticas para dois problemas no sistema RAG:

1. **Erros de embedding agora impedem indexação**: Upload falha se embeddings não forem gerados, documento é deletado
2. **Embeddings órfãos**: Foreign keys CASCADE habilitado no SQLite para limpeza automática

## Arquivos Modificados

### Backend

#### `internal/vector/store.go`
- **Linha 30**: Adicionado `_foreign_keys=ON` na connection string do SQLite
- **Linhas 304-327**: Simplificado `DeleteDocument()` para confiar no CASCADE
  - Removida deleção manual de chunks
  - CASCADE agora deleta automaticamente chunks e chunk_embeddings

#### `internal/api/upload.go`
- **Linha 4**: Adicionado import `fmt`
- **Linhas 180-210**: Lógica de embedding modificada para falhar o upload se embeddings falharem
  - **Falha na geração**: Deleta documento, retorna erro "Falha ao gerar embeddings"
  - **Falha no armazenamento**: Deleta documento, retorna erro com quantidade de falhas
  - **Sem cliente de embedding**: Deleta documento, retorna erro de configuração
  - Usa CASCADE implementado para limpar chunks automaticamente

### Frontend

Sem mudanças necessárias - erros de embedding aparecem como erros normais de upload.

## Comportamento

### Antes das Correções

❌ Upload com falha de embedding parecia sucesso
❌ Documento indexado sem embeddings (busca não funciona)
❌ Deletar documento deixava embeddings órfãos
❌ Banco de dados acumulava lixo ao longo do tempo

### Depois das Correções

✅ Upload falha se embeddings falharem
✅ Documento não é indexado se não tiver embeddings
✅ Mensagem de erro clara para o usuário
✅ Deleção automática via CASCADE (documento → chunks → embeddings)
✅ Integridade referencial mantida pelo SQLite

## Testes Recomendados

### Teste 1: Falha de Embedding - Serviço Offline

```bash
# 1. Parar serviço de embeddings (Ollama/API)
# 2. Fazer upload de documento .md
# 3. Verificar:
#    - Status: "error" (vermelho)
#    - Mensagem: "Falha ao gerar embeddings: ... Documento não foi indexado."
#    - Documento NÃO aparece na lista de documentos
#    - Banco de dados NÃO contém documento, chunks ou embeddings
```

### Teste 2: Foreign Keys CASCADE Funcionando

```bash
# 1. Fazer upload de documento com sucesso (Ollama funcionando)
# 2. Verificar criação de dados:
sqlite3 data/projects/{id}.db "SELECT COUNT(*) FROM documents;"  # 1
sqlite3 data/projects/{id}.db "SELECT COUNT(*) FROM chunks;"     # N chunks
sqlite3 data/projects/{id}.db "SELECT COUNT(*) FROM chunk_embeddings;"  # N embeddings

# 3. Deletar documento via API:
curl -X DELETE http://localhost:8080/api/documents/{doc_id}

# 4. Verificar CASCADE funcionou:
sqlite3 data/projects/{id}.db "SELECT COUNT(*) FROM documents;"  # 0
sqlite3 data/projects/{id}.db "SELECT COUNT(*) FROM chunks;"     # 0 ✓
sqlite3 data/projects/{id}.db "SELECT COUNT(*) FROM chunk_embeddings;"  # 0 ✓
```

### Teste 3: Validar Rollback em Falha

```bash
# 1. Verificar estado inicial do banco
sqlite3 data/projects/{id}.db "SELECT COUNT(*) FROM documents;"

# 2. Parar Ollama DEPOIS de upload bem-sucedido
# 3. Tentar novo upload (vai falhar no embedding)
# 4. Verificar que o novo documento NÃO ficou no banco
sqlite3 data/projects/{id}.db "SELECT id, title FROM documents;"
# Deve mostrar apenas documentos anteriores, não o que falhou
```

## Limpeza de Bancos Existentes

Para bancos de dados criados antes desta correção que podem ter embeddings órfãos:

```bash
# Execute o script de limpeza:
sqlite3 data/projects/{project-id}.db < scripts/cleanup_orphaned_embeddings.sql
```

O script:
1. Conta embeddings órfãos
2. Lista-os para auditoria
3. Deleta-os
4. Executa VACUUM para liberar espaço
5. Mostra estatísticas finais

## Considerações Técnicas

### Foreign Keys no SQLite

- SQLite desabilita foreign keys por padrão para compatibilidade legada
- `_foreign_keys=ON` na connection string habilita globalmente
- Apenas bancos **novos** criados após esta mudança têm CASCADE funcionando automaticamente
- Bancos existentes continuam funcionais mas podem ter dados órfãos (use script de limpeza)

### Design: Fail-Fast para Embeddings

**Decisão**: Embeddings são obrigatórios para upload bem-sucedido

**Razões:**
- Sistema RAG depende fundamentalmente de busca semântica
- Sem embeddings, funcionalidade principal não funciona
- Melhor falhar cedo e claro do que silenciosamente criar documento inútil
- Usuário sabe imediatamente que precisa corrigir configuração de embeddings

**Alternativa rejeitada**: Warnings não-fatais
- ❌ Permite documentos "meio quebrados" no sistema
- ❌ Confuso para usuário (sucesso verde + warning amarelo)
- ❌ Busca não funciona mas usuário não entende por quê

### Impacto na Performance

- **CASCADE**: Impacto mínimo, SQLite otimiza deleções em cascata
- **Foreign Keys ON**: Overhead negligível, valida integridade em cada operação
- **Rollback em falha**: Deleção única via CASCADE, muito rápida

### Casos de Uso

**Cenário 1: Ollama não está rodando**
```
Upload → Parse OK → Chunks OK → Embedding FAIL → DELETE doc → Retorna erro
Resultado: Nada no banco, mensagem clara de erro
```

**Cenário 2: Falha parcial no armazenamento**
```
Upload → Parse OK → Chunks OK → Embeddings OK → Store 50/100 FAIL → DELETE doc → Retorna erro
Resultado: Nada no banco, erro indica quantidade de falhas
```

**Cenário 3: Upload bem-sucedido**
```
Upload → Parse OK → Chunks OK → Embeddings OK → Store 100% OK → Cross-refs OK → SUCCESS
Resultado: Documento completamente indexado e buscável
```

## Próximos Passos (Opcional)

1. **Endpoint de Healthcheck**: Verificar se serviço de embeddings está disponível antes do upload
2. **Retry Logic**: Tentar gerar embeddings novamente em caso de falha temporária
3. **Métricas**: Log de razões de falha de embedding para debugging
4. **Testes Automatizados**: Unit tests para cenários de falha + rollback

## Compatibilidade

- **Backend**: Compatível com bancos de dados existentes
- **Frontend**: Sem mudanças necessárias (usa error field existente)
- **API**: Mudança breaking para uploads sem embeddings (agora falham)
- **Migração**: Não requer migração de dados, apenas limpeza opcional de órfãos
