import { useState } from 'react'
import { Link } from 'react-router-dom'
import { User, Calendar, Layers, Trash2, Loader2 } from 'lucide-react'
import { DocTypeBadge, StatusBadge } from '../ui/Badge'
import { ConfirmModal } from '../ui/ConfirmModal'
import { formatDate } from '../../lib/utils'
import { useDeleteDocument } from '../../hooks/useApi'
import type { DocumentSummary } from '../../api/types'

interface DocumentCardProps {
  document: DocumentSummary
}

export function DocumentCard({ document }: DocumentCardProps) {
  const deleteDocument = useDeleteDocument()
  const [showConfirm, setShowConfirm] = useState(false)

  const handleDeleteClick = (e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setShowConfirm(true)
  }

  const handleDeleteConfirm = async () => {
    try {
      await deleteDocument.mutateAsync(document.id)
      setShowConfirm(false)
    } catch (error) {
      console.error('Failed to delete document:', error)
    }
  }

  return (
    <>
      <Link to={`/documents/${document.id}`}>
        <div className="card-hover p-5 cursor-pointer group relative">
        {/* Delete button - visible on hover */}
        <button
          onClick={handleDeleteClick}
          disabled={deleteDocument.isPending}
          className="absolute top-3 right-3 p-2 bg-white/90 rounded-lg
                     opacity-0 group-hover:opacity-100 transition-opacity
                     hover:bg-red-50 hover:text-red-600 text-neutral-400
                     disabled:opacity-50 disabled:cursor-not-allowed z-10"
          title="Deletar documento"
        >
          {deleteDocument.isPending ? (
            <Loader2 className="w-4 h-4 animate-spin" />
          ) : (
            <Trash2 className="w-4 h-4" />
          )}
        </button>

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

    {/* Delete Confirmation Modal */}
    <ConfirmModal
      isOpen={showConfirm}
      onClose={() => setShowConfirm(false)}
      onConfirm={handleDeleteConfirm}
      title="Deletar Documento?"
      message={`Tem certeza que deseja deletar o documento "${document.title}"?\n\nEsta ação não pode ser desfeita.`}
      confirmText="Deletar"
      cancelText="Cancelar"
      variant="danger"
      isLoading={deleteDocument.isPending}
    />
  </>
  )
}
