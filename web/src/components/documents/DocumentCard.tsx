import { Link } from 'react-router-dom'
import { User, Calendar, Layers } from 'lucide-react'
import { DocTypeBadge, StatusBadge } from '../ui/Badge'
import { formatDate } from '../../lib/utils'
import type { DocumentSummary } from '../../api/types'

interface DocumentCardProps {
  document: DocumentSummary
}

export function DocumentCard({ document }: DocumentCardProps) {
  return (
    <Link to={`/documents/${document.id}`}>
      <div className="card-hover p-5 cursor-pointer group">
        {/* Header */}
        <div className="flex items-start justify-between mb-3">
          <DocTypeBadge docType={document.doc_type} />
          {document.version && (
            <span className="text-xs text-neutral-400 font-medium">
              v{document.version}
            </span>
          )}
        </div>

        {/* Title */}
        <h3 className="font-semibold text-neutral-900 group-hover:text-primary-600 transition-colors line-clamp-2 mb-2">
          {document.title}
        </h3>

        {/* ID */}
        <p className="text-sm text-neutral-500 mb-3">
          {document.id}
        </p>

        {/* Status */}
        {document.status && (
          <div className="mb-4">
            <StatusBadge status={document.status} />
          </div>
        )}

        {/* Meta */}
        <div className="flex items-center gap-4 pt-4 border-t border-neutral-100 text-xs text-neutral-500">
          {document.author && (
            <span className="flex items-center gap-1">
              <User className="w-3.5 h-3.5" />
              {document.author}
            </span>
          )}
          <span className="flex items-center gap-1">
            <Calendar className="w-3.5 h-3.5" />
            {formatDate(document.indexed_at)}
          </span>
          <span className="flex items-center gap-1">
            <Layers className="w-3.5 h-3.5" />
            {document.chunk_count} chunks
          </span>
        </div>
      </div>
    </Link>
  )
}
