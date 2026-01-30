import { useState } from 'react'
import { FileText, ChevronLeft, ChevronRight } from 'lucide-react'
import { useDocuments } from '../hooks/useApi'
import { DocumentCard } from '../components/documents/DocumentCard'
import { DocumentCardSkeleton } from '../components/ui/Skeleton'
import { Card } from '../components/ui/Card'
import { cn } from '../lib/utils'

const DOC_TYPES = ['RFC', 'ADR', 'BDR', 'Guideline', 'Roadmap']

export function Documents() {
  const [page, setPage] = useState(1)
  const [docType, setDocType] = useState<string>('')
  const limit = 12

  const { data, isLoading } = useDocuments({ page, limit, doc_type: docType || undefined })

  return (
    <div className="space-y-6">
      {/* Filters */}
      <div className="flex items-center gap-2 flex-wrap">
        <button
          onClick={() => setDocType('')}
          className={cn(
            'px-4 py-2 rounded-lg font-medium transition-colors',
            !docType
              ? 'bg-primary-500 text-white'
              : 'bg-white text-neutral-600 hover:bg-neutral-50 border border-neutral-200'
          )}
        >
          Todos
        </button>
        {DOC_TYPES.map((type) => (
          <button
            key={type}
            onClick={() => setDocType(type)}
            className={cn(
              'px-4 py-2 rounded-lg font-medium transition-colors',
              docType === type
                ? 'bg-primary-500 text-white'
                : 'bg-white text-neutral-600 hover:bg-neutral-50 border border-neutral-200'
            )}
          >
            {type}
          </button>
        ))}
      </div>

      {/* Results count */}
      {data && (
        <p className="text-sm text-neutral-500">
          Mostrando {data.documents.length} de {data.pagination.total} documentos
          {docType && ` do tipo ${docType}`}
        </p>
      )}

      {/* Documents Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        {isLoading ? (
          Array.from({ length: 8 }).map((_, i) => (
            <DocumentCardSkeleton key={i} />
          ))
        ) : (
          data?.documents.map((doc) => (
            <DocumentCard key={doc.id} document={doc} />
          ))
        )}
      </div>

      {/* Empty State */}
      {!isLoading && (!data?.documents || data.documents.length === 0) && (
        <Card className="p-12 text-center">
          <FileText className="w-16 h-16 text-neutral-300 mx-auto mb-4" />
          <h3 className="text-lg font-medium text-neutral-900 mb-2">
            Nenhum documento encontrado
          </h3>
          <p className="text-neutral-500">
            {docType
              ? `Não há documentos do tipo ${docType}.`
              : 'Execute o indexador para começar a indexar documentos.'}
          </p>
        </Card>
      )}

      {/* Pagination */}
      {data && data.pagination.total_pages > 1 && (
        <div className="flex items-center justify-center gap-2">
          <button
            onClick={() => setPage(page - 1)}
            disabled={page === 1}
            className="p-2 rounded-lg border border-neutral-200 text-neutral-600 hover:bg-neutral-50 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <ChevronLeft className="w-5 h-5" />
          </button>

          <div className="flex items-center gap-1">
            {Array.from({ length: Math.min(5, data.pagination.total_pages) }).map((_, i) => {
              const pageNum = i + 1
              return (
                <button
                  key={pageNum}
                  onClick={() => setPage(pageNum)}
                  className={cn(
                    'w-10 h-10 rounded-lg font-medium transition-colors',
                    page === pageNum
                      ? 'bg-primary-500 text-white'
                      : 'text-neutral-600 hover:bg-neutral-100'
                  )}
                >
                  {pageNum}
                </button>
              )
            })}
            {data.pagination.total_pages > 5 && (
              <>
                <span className="text-neutral-400">...</span>
                <button
                  onClick={() => setPage(data.pagination.total_pages)}
                  className={cn(
                    'w-10 h-10 rounded-lg font-medium transition-colors',
                    page === data.pagination.total_pages
                      ? 'bg-primary-500 text-white'
                      : 'text-neutral-600 hover:bg-neutral-100'
                  )}
                >
                  {data.pagination.total_pages}
                </button>
              </>
            )}
          </div>

          <button
            onClick={() => setPage(page + 1)}
            disabled={page === data.pagination.total_pages}
            className="p-2 rounded-lg border border-neutral-200 text-neutral-600 hover:bg-neutral-50 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <ChevronRight className="w-5 h-5" />
          </button>
        </div>
      )}
    </div>
  )
}
