-- Script para limpar embeddings órfãos em bancos de dados existentes
-- Execute este script em cada banco de dados do projeto que foi criado
-- antes da correção de foreign keys

-- 1. Verificar quantidade de embeddings órfãos
SELECT COUNT(*) as orphaned_embeddings
FROM chunk_embeddings
WHERE chunk_id NOT IN (SELECT id FROM chunks);

-- 2. Listar embeddings órfãos (para auditoria)
SELECT ce.chunk_id, ce.model, ce.created_at
FROM chunk_embeddings ce
WHERE ce.chunk_id NOT IN (SELECT id FROM chunks)
ORDER BY ce.created_at DESC;

-- 3. Deletar embeddings órfãos
DELETE FROM chunk_embeddings
WHERE chunk_id NOT IN (SELECT id FROM chunks);

-- 4. Verificar que não há mais órfãos
SELECT COUNT(*) as remaining_orphaned_embeddings
FROM chunk_embeddings
WHERE chunk_id NOT IN (SELECT id FROM chunks);

-- 5. Liberar espaço no banco
VACUUM;

-- 6. Verificar estatísticas finais
SELECT
    (SELECT COUNT(*) FROM documents) as total_documents,
    (SELECT COUNT(*) FROM chunks) as total_chunks,
    (SELECT COUNT(*) FROM chunk_embeddings) as total_embeddings;
