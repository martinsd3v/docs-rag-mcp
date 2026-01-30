import { CheckCircle, XCircle, RefreshCw } from 'lucide-react'
import type { UploadResult, UploadSummary } from '../../api/types'
import { cn } from '../../lib/utils'

interface UploadProgressProps {
  results: UploadResult[]
  summary: UploadSummary
}

export function UploadProgress({ results, summary }: UploadProgressProps) {
  return (
    <div className="space-y-4">
      {/* Summary */}
      <div className="flex items-center gap-4 p-4 bg-neutral-50 rounded-lg">
        <div className="flex-1">
          <p className="text-sm font-medium text-neutral-700">
            Resumo do Upload
          </p>
          <div className="flex items-center gap-4 mt-2 text-sm">
            <span className="text-neutral-500">
              Total: <span className="font-medium">{summary.total}</span>
            </span>
            {summary.succeeded > 0 && (
              <span className="text-green-600">
                Sucesso: <span className="font-medium">{summary.succeeded}</span>
              </span>
            )}
            {summary.replaced > 0 && (
              <span className="text-blue-600">
                Substituídos: <span className="font-medium">{summary.replaced}</span>
              </span>
            )}
            {summary.failed > 0 && (
              <span className="text-red-600">
                Falhas: <span className="font-medium">{summary.failed}</span>
              </span>
            )}
          </div>
        </div>
      </div>

      {/* Results List */}
      <div className="border border-neutral-200 rounded-lg overflow-hidden">
        <div className="divide-y divide-neutral-200">
          {results.map((result, index) => (
            <ResultRow key={index} result={result} />
          ))}
        </div>
      </div>
    </div>
  )
}

function ResultRow({ result }: { result: UploadResult }) {
  const statusConfig = {
    success: {
      icon: CheckCircle,
      color: 'text-green-500',
      bg: 'bg-green-50',
      label: 'Novo',
    },
    replaced: {
      icon: RefreshCw,
      color: 'text-blue-500',
      bg: 'bg-blue-50',
      label: 'Substituído',
    },
    error: {
      icon: XCircle,
      color: 'text-red-500',
      bg: 'bg-red-50',
      label: 'Erro',
    },
  }

  const config = statusConfig[result.status]
  const Icon = config.icon

  return (
    <div className={cn('flex items-center gap-4 px-4 py-3', config.bg)}>
      <Icon className={cn('w-5 h-5 flex-shrink-0', config.color)} />

      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium text-neutral-900 truncate">
          {result.doc_id || result.filename}
        </p>
        {result.filename !== result.doc_id && (
          <p className="text-xs text-neutral-500 truncate">{result.filename}</p>
        )}
      </div>

      {result.status !== 'error' ? (
        <div className="flex items-center gap-4 text-sm">
          <span className="text-neutral-500">
            {result.chunk_count} chunks
          </span>
          <span
            className={cn(
              'px-2 py-0.5 rounded text-xs font-medium',
              result.status === 'replaced'
                ? 'bg-blue-100 text-blue-700'
                : 'bg-green-100 text-green-700'
            )}
          >
            {config.label}
          </span>
        </div>
      ) : (
        <div className="text-sm text-red-600 max-w-xs truncate" title={result.error}>
          {result.error}
        </div>
      )}
    </div>
  )
}
