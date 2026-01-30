import { Link } from 'react-router-dom'
import { ChevronRight, FileText } from 'lucide-react'
import { DocTypeBadge, ScoreBadge } from '../ui/Badge'
import type { SearchResult } from '../../api/types'

interface SearchResultItemProps {
  result: SearchResult
}

export function SearchResultItem({ result }: SearchResultItemProps) {
  return (
    <Link to={`/documents/${result.doc_id}`}>
      <div className="card-hover p-5 group">
        <div className="flex items-start gap-4">
          {/* Score */}
          <div className="flex-shrink-0">
            <ScoreBadge score={result.score} />
          </div>

          {/* Content */}
          <div className="flex-1 min-w-0">
            {/* Header */}
            <div className="flex items-center gap-2 mb-2">
              <DocTypeBadge docType={result.doc_type} />
              <h3 className="font-semibold text-neutral-900 group-hover:text-primary-600 transition-colors truncate">
                {result.doc_id}: {result.doc_title}
              </h3>
            </div>

            {/* Section path */}
            <div className="flex items-center gap-1 text-sm text-neutral-500 mb-3">
              <FileText className="w-4 h-4" />
              <span className="truncate">{result.chunk.section_path}</span>
            </div>

            {/* Content preview */}
            <p className="text-sm text-neutral-600 line-clamp-3">
              {result.chunk.content}
            </p>

            {/* Related docs */}
            {result.related_docs && result.related_docs.length > 0 && (
              <div className="flex items-center gap-2 mt-3 pt-3 border-t border-neutral-100">
                <span className="text-xs text-neutral-400">Relacionados:</span>
                <div className="flex gap-1">
                  {result.related_docs.slice(0, 3).map((docId) => (
                    <span
                      key={docId}
                      className="text-xs px-2 py-0.5 bg-neutral-100 text-neutral-600 rounded"
                    >
                      {docId}
                    </span>
                  ))}
                  {result.related_docs.length > 3 && (
                    <span className="text-xs text-neutral-400">
                      +{result.related_docs.length - 3}
                    </span>
                  )}
                </div>
              </div>
            )}
          </div>

          {/* Arrow */}
          <ChevronRight className="w-5 h-5 text-neutral-300 group-hover:text-primary-500 transition-colors flex-shrink-0" />
        </div>
      </div>
    </Link>
  )
}

// Empty state
export function SearchEmptyState() {
  return (
    <div className="text-center py-12">
      <div className="w-16 h-16 bg-neutral-100 rounded-full flex items-center justify-center mx-auto mb-4">
        <FileText className="w-8 h-8 text-neutral-400" />
      </div>
      <h3 className="text-lg font-medium text-neutral-900 mb-2">
        Nenhum resultado encontrado
      </h3>
      <p className="text-neutral-500 max-w-md mx-auto">
        Tente ajustar sua busca ou usar termos diferentes.
        A busca semântica funciona melhor com descrições do que você está procurando.
      </p>
    </div>
  )
}

// Initial state
export function SearchInitialState() {
  return (
    <div className="text-center py-12">
      <div className="w-16 h-16 bg-primary-50 rounded-full flex items-center justify-center mx-auto mb-4">
        <FileText className="w-8 h-8 text-primary-500" />
      </div>
      <h3 className="text-lg font-medium text-neutral-900 mb-2">
        Busque na documentação
      </h3>
      <p className="text-neutral-500 max-w-md mx-auto">
        Use linguagem natural para encontrar RFCs, ADRs e outros documentos.
        Por exemplo: "como implementar autenticação" ou "padrões de API REST".
      </p>
    </div>
  )
}
