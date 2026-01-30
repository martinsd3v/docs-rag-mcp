import { FileText, Layers, Link2, Database, Clock } from 'lucide-react'
import { Link } from 'react-router-dom'
import { useStats, useDocuments } from '../hooks/useApi'
import { StatCard, Card } from '../components/ui/Card'
import { StatCardSkeleton, DocumentCardSkeleton } from '../components/ui/Skeleton'
import { DocumentCard } from '../components/documents/DocumentCard'
import { formatNumber, formatRelativeTime } from '../lib/utils'

export function Dashboard() {
  const { data: stats, isLoading: statsLoading } = useStats()
  const { data: docsData, isLoading: docsLoading } = useDocuments({ limit: 6 })

  return (
    <div className="space-y-6">
      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {statsLoading ? (
          <>
            <StatCardSkeleton />
            <StatCardSkeleton />
            <StatCardSkeleton />
            <StatCardSkeleton />
          </>
        ) : (
          <>
            <StatCard
              title="Documentos"
              value={formatNumber(stats?.documents || 0)}
              icon={FileText}
              color="blue"
              subtitle="Indexados no sistema"
            />
            <StatCard
              title="Chunks"
              value={formatNumber(stats?.chunks || 0)}
              icon={Layers}
              color="green"
              subtitle={`Média de ${stats?.documents ? Math.round((stats.chunks || 0) / stats.documents) : 0} por doc`}
            />
            <StatCard
              title="Embeddings"
              value={formatNumber(stats?.embeddings || 0)}
              icon={Database}
              color="purple"
              subtitle="Vetores gerados"
            />
            <StatCard
              title="Cross-refs"
              value={formatNumber(stats?.cross_references || 0)}
              icon={Link2}
              color="amber"
              subtitle="Referências entre docs"
            />
          </>
        )}
      </div>

      {/* Two Column Layout */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Doc Types */}
        <Card className="p-6">
          <h3 className="font-semibold text-neutral-900 mb-4">Por Tipo</h3>
          {statsLoading ? (
            <div className="space-y-3">
              {[1, 2, 3, 4].map((i) => (
                <div key={i} className="flex items-center justify-between">
                  <div className="h-4 w-20 bg-neutral-200 rounded animate-pulse" />
                  <div className="h-4 w-10 bg-neutral-200 rounded animate-pulse" />
                </div>
              ))}
            </div>
          ) : (
            <div className="space-y-3">
              {stats?.doc_types && Object.entries(stats.doc_types).map(([type, count]) => (
                <div key={type} className="flex items-center justify-between">
                  <span className="text-neutral-600">{type || 'Outros'}</span>
                  <span className="font-semibold text-neutral-900">{count}</span>
                </div>
              ))}
              {(!stats?.doc_types || Object.keys(stats.doc_types).length === 0) && (
                <p className="text-neutral-500 text-sm">Nenhum documento indexado</p>
              )}
            </div>
          )}
        </Card>

        {/* Recent Activity */}
        <Card className="p-6 lg:col-span-2">
          <div className="flex items-center justify-between mb-4">
            <h3 className="font-semibold text-neutral-900">Atividade Recente</h3>
            {stats?.last_indexed && (
              <span className="text-sm text-neutral-500 flex items-center gap-1">
                <Clock className="w-4 h-4" />
                Última indexação: {formatRelativeTime(stats.last_indexed)}
              </span>
            )}
          </div>
          <div className="space-y-3">
            <div className="flex items-center gap-3 text-sm">
              <div className="w-2 h-2 bg-green-500 rounded-full" />
              <span className="text-neutral-600">Sistema operacional</span>
            </div>
            <div className="flex items-center gap-3 text-sm">
              <div className="w-2 h-2 bg-blue-500 rounded-full" />
              <span className="text-neutral-600">
                {stats?.documents || 0} documentos disponíveis para busca
              </span>
            </div>
            <div className="flex items-center gap-3 text-sm">
              <div className="w-2 h-2 bg-purple-500 rounded-full" />
              <span className="text-neutral-600">
                {stats?.embeddings || 0} embeddings prontos
              </span>
            </div>
          </div>
        </Card>
      </div>

      {/* Recent Documents */}
      <div>
        <div className="flex items-center justify-between mb-4">
          <h3 className="font-semibold text-neutral-900">Documentos Recentes</h3>
          <Link
            to="/documents"
            className="text-sm text-primary-600 hover:text-primary-700 font-medium"
          >
            Ver todos →
          </Link>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {docsLoading ? (
            <>
              <DocumentCardSkeleton />
              <DocumentCardSkeleton />
              <DocumentCardSkeleton />
            </>
          ) : (
            docsData?.documents.map((doc) => (
              <DocumentCard key={doc.id} document={doc} />
            ))
          )}
          {!docsLoading && (!docsData?.documents || docsData.documents.length === 0) && (
            <Card className="p-8 col-span-3 text-center">
              <FileText className="w-12 h-12 text-neutral-300 mx-auto mb-3" />
              <p className="text-neutral-500">Nenhum documento indexado ainda.</p>
              <p className="text-sm text-neutral-400 mt-1">
                Execute o indexador para começar.
              </p>
            </Card>
          )}
        </div>
      </div>
    </div>
  )
}
