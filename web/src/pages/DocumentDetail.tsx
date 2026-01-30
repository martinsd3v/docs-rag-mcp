import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, FileText, User, Calendar, Layers, ExternalLink } from 'lucide-react'
import { useDocument } from '../hooks/useApi'
import { DocTypeBadge, StatusBadge } from '../components/ui/Badge'
import { Card } from '../components/ui/Card'
import { Skeleton } from '../components/ui/Skeleton'
import { formatDate } from '../lib/utils'

export function DocumentDetail() {
  const { id } = useParams<{ id: string }>()
  const { data, isLoading, error } = useDocument(id || '', {
    include_chunks: true,
    include_related: true,
  })

  if (error) {
    return (
      <div className="text-center py-12">
        <p className="text-red-600">Erro ao carregar documento</p>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Skeleton className="h-8 w-8" />
          <Skeleton className="h-8 w-48" />
        </div>
        <Card className="p-6 space-y-4">
          <Skeleton className="h-8 w-3/4" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-2/3" />
        </Card>
      </div>
    )
  }

  if (!data) {
    return (
      <div className="text-center py-12">
        <p className="text-neutral-600">Documento não encontrado</p>
      </div>
    )
  }

  const { document, chunks, related_docs } = data

  return (
    <div className="space-y-6">
      {/* Back button */}
      <Link
        to="/documents"
        className="inline-flex items-center gap-2 text-neutral-600 hover:text-neutral-900 transition-colors"
      >
        <ArrowLeft className="w-4 h-4" />
        Voltar para Documentos
      </Link>

      {/* Header Card */}
      <Card className="p-6">
        <div className="flex items-start justify-between mb-4">
          <div className="flex items-center gap-3">
            <DocTypeBadge docType={document.doc_type} size="md" />
            {document.status && <StatusBadge status={document.status} size="md" />}
            {document.version && (
              <span className="text-sm text-neutral-500 bg-neutral-100 px-2 py-1 rounded">
                v{document.version}
              </span>
            )}
          </div>
        </div>

        <h1 className="text-2xl font-bold text-neutral-900 mb-2">
          {document.id}: {document.title}
        </h1>

        <div className="flex items-center gap-6 text-sm text-neutral-500 mt-4">
          {document.author && (
            <span className="flex items-center gap-1.5">
              <User className="w-4 h-4" />
              {document.author}
            </span>
          )}
          {document.created_date && (
            <span className="flex items-center gap-1.5">
              <Calendar className="w-4 h-4" />
              Criado em {formatDate(document.created_date)}
            </span>
          )}
          {document.last_updated && (
            <span className="flex items-center gap-1.5">
              <Calendar className="w-4 h-4" />
              Atualizado em {formatDate(document.last_updated)}
            </span>
          )}
          <span className="flex items-center gap-1.5">
            <Layers className="w-4 h-4" />
            {chunks?.length || 0} chunks
          </span>
        </div>

        <div className="mt-4 pt-4 border-t border-neutral-100">
          <p className="text-sm text-neutral-400 font-mono truncate">
            {document.file_path}
          </p>
        </div>
      </Card>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Content */}
        <div className="lg:col-span-2 space-y-4">
          <h2 className="font-semibold text-neutral-900">Conteúdo</h2>
          {chunks && chunks.length > 0 ? (
            chunks.map((chunk) => (
              <Card key={chunk.id} className="p-5">
                <div className="flex items-center justify-between mb-3">
                  <span className="text-sm font-medium text-neutral-600">
                    {chunk.section_path || `Chunk ${chunk.chunk_index + 1}`}
                  </span>
                  <span className="text-xs text-neutral-400">
                    Linhas {chunk.start_line}-{chunk.end_line} • {chunk.token_count} tokens
                  </span>
                </div>
                <div className="prose prose-neutral prose-sm max-w-none">
                  <pre className="bg-neutral-50 p-4 rounded-lg overflow-x-auto text-sm whitespace-pre-wrap">
                    {chunk.content}
                  </pre>
                </div>
                <div className="flex items-center gap-2 mt-3">
                  {chunk.has_code_block && (
                    <span className="text-xs bg-blue-50 text-blue-600 px-2 py-0.5 rounded">
                      código
                    </span>
                  )}
                  {chunk.has_table && (
                    <span className="text-xs bg-green-50 text-green-600 px-2 py-0.5 rounded">
                      tabela
                    </span>
                  )}
                </div>
              </Card>
            ))
          ) : (
            <Card className="p-8 text-center">
              <FileText className="w-12 h-12 text-neutral-300 mx-auto mb-3" />
              <p className="text-neutral-500">Nenhum chunk disponível</p>
            </Card>
          )}
        </div>

        {/* Sidebar */}
        <div className="space-y-4">
          {/* Related Documents */}
          <Card className="p-5">
            <h3 className="font-semibold text-neutral-900 mb-4">
              Documentos Relacionados
            </h3>
            {related_docs && related_docs.length > 0 ? (
              <div className="space-y-2">
                {related_docs.map((rel) => (
                  <Link
                    key={rel.id}
                    to={`/documents/${rel.id}`}
                    className="flex items-center gap-3 p-3 rounded-lg hover:bg-neutral-50 transition-colors group"
                  >
                    <DocTypeBadge docType={rel.doc_type} />
                    <div className="flex-1 min-w-0">
                      <p className="font-medium text-neutral-900 group-hover:text-primary-600 truncate">
                        {rel.id}
                      </p>
                      <p className="text-sm text-neutral-500 truncate">
                        {rel.title}
                      </p>
                    </div>
                    <ExternalLink className="w-4 h-4 text-neutral-400 group-hover:text-primary-500" />
                  </Link>
                ))}
              </div>
            ) : (
              <p className="text-sm text-neutral-500">
                Nenhum documento relacionado
              </p>
            )}
          </Card>
        </div>
      </div>
    </div>
  )
}
