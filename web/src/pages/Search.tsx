import { useState } from 'react'
import { Clock, Zap } from 'lucide-react'
import { useSearch } from '../hooks/useApi'
import { SearchBar, FilterChips } from '../components/search/SearchBar'
import { SearchResultItem, SearchEmptyState, SearchInitialState } from '../components/search/SearchResults'
import { SearchResultSkeleton } from '../components/ui/Skeleton'

const DOC_TYPES = ['RFC', 'ADR', 'BDR', 'Guideline', 'Roadmap']

export function Search() {
  const [query, setQuery] = useState('')
  const [selectedTypes, setSelectedTypes] = useState<string[]>([])
  const { mutate: doSearch, data, isPending, isIdle } = useSearch()

  const handleSearch = () => {
    if (!query.trim()) return
    doSearch({
      query: query.trim(),
      doc_types: selectedTypes.length > 0 ? selectedTypes : undefined,
      top_k: 10,
      include_content: true,
    })
  }

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      {/* Search Bar */}
      <div className="space-y-4">
        <SearchBar
          value={query}
          onChange={setQuery}
          onSearch={handleSearch}
          isLoading={isPending}
          placeholder="Descreva o que você está procurando..."
        />
        <FilterChips
          options={DOC_TYPES}
          selected={selectedTypes}
          onChange={setSelectedTypes}
        />
      </div>

      {/* Results Header */}
      {data && (
        <div className="flex items-center justify-between">
          <p className="text-neutral-600">
            <span className="font-semibold">{data.total_results}</span> resultados para "
            <span className="font-medium">{data.query}</span>"
          </p>
          <div className="flex items-center gap-4 text-sm text-neutral-500">
            <span className="flex items-center gap-1">
              <Clock className="w-4 h-4" />
              {data.search_time_ms}ms
            </span>
            <span className="flex items-center gap-1">
              <Zap className="w-4 h-4 text-amber-500" />
              Busca semântica
            </span>
          </div>
        </div>
      )}

      {/* Results */}
      <div className="space-y-3">
        {isPending && (
          <>
            <SearchResultSkeleton />
            <SearchResultSkeleton />
            <SearchResultSkeleton />
          </>
        )}

        {!isPending && data && data.results.length > 0 && (
          data.results.map((result, index) => (
            <SearchResultItem key={`${result.doc_id}-${index}`} result={result} />
          ))
        )}

        {!isPending && data && data.results.length === 0 && (
          <SearchEmptyState />
        )}

        {isIdle && !data && (
          <SearchInitialState />
        )}
      </div>
    </div>
  )
}
