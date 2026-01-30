import { useCallback } from 'react'
import { Upload, FileText } from 'lucide-react'
import { cn } from '../../lib/utils'

interface DropZoneProps {
  onFilesSelected: (files: File[]) => void
  isDragging: boolean
  onDragEnter: () => void
  onDragLeave: () => void
  disabled?: boolean
}

export function DropZone({
  onFilesSelected,
  isDragging,
  onDragEnter,
  onDragLeave,
  disabled = false,
}: DropZoneProps) {
  const handleDrop = useCallback(
    (e: React.DragEvent<HTMLDivElement>) => {
      e.preventDefault()
      e.stopPropagation()
      onDragLeave()

      if (disabled) return

      const droppedFiles = Array.from(e.dataTransfer.files).filter((f) =>
        f.name.toLowerCase().endsWith('.md')
      )

      if (droppedFiles.length > 0) {
        onFilesSelected(droppedFiles)
      }
    },
    [onFilesSelected, onDragLeave, disabled]
  )

  const handleDragOver = useCallback((e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    e.stopPropagation()
  }, [])

  const handleDragEnter = useCallback(
    (e: React.DragEvent<HTMLDivElement>) => {
      e.preventDefault()
      e.stopPropagation()
      if (!disabled) {
        onDragEnter()
      }
    },
    [onDragEnter, disabled]
  )

  const handleDragLeave = useCallback(
    (e: React.DragEvent<HTMLDivElement>) => {
      e.preventDefault()
      e.stopPropagation()
      onDragLeave()
    },
    [onDragLeave]
  )

  const handleFileInput = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const files = e.target.files
      if (files) {
        const mdFiles = Array.from(files).filter((f) =>
          f.name.toLowerCase().endsWith('.md')
        )
        if (mdFiles.length > 0) {
          onFilesSelected(mdFiles)
        }
      }
      // Reset input value to allow selecting the same files again
      e.target.value = ''
    },
    [onFilesSelected]
  )

  return (
    <div
      onDrop={handleDrop}
      onDragOver={handleDragOver}
      onDragEnter={handleDragEnter}
      onDragLeave={handleDragLeave}
      className={cn(
        'border-2 border-dashed rounded-xl p-12 text-center transition-colors cursor-pointer',
        isDragging
          ? 'border-primary-500 bg-primary-50'
          : 'border-neutral-300 hover:border-primary-400 hover:bg-neutral-50',
        disabled && 'opacity-50 cursor-not-allowed'
      )}
    >
      <input
        type="file"
        id="file-input"
        multiple
        accept=".md"
        onChange={handleFileInput}
        className="hidden"
        disabled={disabled}
      />
      <label
        htmlFor="file-input"
        className={cn('cursor-pointer', disabled && 'cursor-not-allowed')}
      >
        <div className="flex flex-col items-center gap-4">
          <div
            className={cn(
              'w-16 h-16 rounded-full flex items-center justify-center',
              isDragging ? 'bg-primary-100' : 'bg-neutral-100'
            )}
          >
            {isDragging ? (
              <Upload className="w-8 h-8 text-primary-500" />
            ) : (
              <FileText className="w-8 h-8 text-neutral-400" />
            )}
          </div>

          <div>
            <p className="text-lg font-medium text-neutral-700">
              {isDragging
                ? 'Solte os arquivos aqui'
                : 'Arraste arquivos .md aqui'}
            </p>
            <p className="text-sm text-neutral-500 mt-1">
              ou clique para selecionar
            </p>
          </div>

          <p className="text-xs text-neutral-400">
            Apenas arquivos Markdown (.md) são aceitos
          </p>
        </div>
      </label>
    </div>
  )
}
