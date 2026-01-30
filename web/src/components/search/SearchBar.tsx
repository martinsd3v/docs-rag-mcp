import { Search, X, SlidersHorizontal } from 'lucide-react'
import { cn } from '../../lib/utils'

interface SearchBarProps {
  value: string
  onChange: (value: string) => void
  onSearch: () => void
  placeholder?: string
  isLoading?: boolean
}

export function SearchBar({
  value,
  onChange,
  onSearch,
  placeholder = 'Buscar na documentação...',
  isLoading = false,
}: SearchBarProps) {
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      onSearch()
    }
  }

  return (
    <div className="relative">
      <div className="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
        <Search className={cn(
          'w-5 h-5',
          isLoading ? 'text-primary-500 animate-pulse' : 'text-neutral-400'
        )} />
      </div>
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        className="w-full pl-12 pr-24 py-4 text-lg bg-white border border-neutral-200 rounded-xl
                   focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent
                   placeholder:text-neutral-400 transition-all"
      />
      <div className="absolute inset-y-0 right-0 flex items-center gap-2 pr-3">
        {value && (
          <button
            onClick={() => onChange('')}
            className="p-1.5 text-neutral-400 hover:text-neutral-600 hover:bg-neutral-100 rounded-lg transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        )}
        <button
          onClick={onSearch}
          disabled={!value || isLoading}
          className="btn-primary disabled:opacity-50 disabled:cursor-not-allowed"
        >
          Buscar
        </button>
      </div>
    </div>
  )
}

// Filter chips
interface FilterChipsProps {
  options: string[]
  selected: string[]
  onChange: (selected: string[]) => void
}

export function FilterChips({ options, selected, onChange }: FilterChipsProps) {
  const toggleOption = (option: string) => {
    if (selected.includes(option)) {
      onChange(selected.filter((s) => s !== option))
    } else {
      onChange([...selected, option])
    }
  }

  return (
    <div className="flex items-center gap-2 flex-wrap">
      <span className="text-sm text-neutral-500 flex items-center gap-1">
        <SlidersHorizontal className="w-4 h-4" />
        Filtros:
      </span>
      {options.map((option) => (
        <button
          key={option}
          onClick={() => toggleOption(option)}
          className={cn(
            'px-3 py-1.5 rounded-full text-sm font-medium transition-all',
            selected.includes(option)
              ? 'bg-primary-100 text-primary-700 ring-2 ring-primary-500'
              : 'bg-neutral-100 text-neutral-600 hover:bg-neutral-200'
          )}
        >
          {option}
        </button>
      ))}
      {selected.length > 0 && (
        <button
          onClick={() => onChange([])}
          className="text-sm text-neutral-400 hover:text-neutral-600 underline"
        >
          Limpar
        </button>
      )}
    </div>
  )
}
