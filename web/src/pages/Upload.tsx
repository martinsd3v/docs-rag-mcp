import { useState, useCallback } from 'react'
import { FileText, X, Loader2 } from 'lucide-react'
import { useUpload } from '../hooks/useApi'
import { DropZone } from '../components/upload/DropZone'
import { UploadProgress } from '../components/upload/UploadProgress'
import { Card } from '../components/ui/Card'
import { cn } from '../lib/utils'

export function Upload() {
  const [files, setFiles] = useState<File[]>([])
  const [isDragging, setIsDragging] = useState(false)
  const upload = useUpload()

  const handleFilesSelected = useCallback((newFiles: File[]) => {
    setFiles((prev) => {
      // Avoid duplicates by filename
      const existingNames = new Set(prev.map((f) => f.name))
      const uniqueNewFiles = newFiles.filter((f) => !existingNames.has(f.name))
      return [...prev, ...uniqueNewFiles]
    })
  }, [])

  const handleRemoveFile = useCallback((index: number) => {
    setFiles((prev) => prev.filter((_, i) => i !== index))
  }, [])

  const handleClearFiles = useCallback(() => {
    setFiles([])
  }, [])

  const handleUpload = async () => {
    if (files.length === 0) return

    const formData = new FormData()
    files.forEach((file) => formData.append('files', file))

    try {
      await upload.mutateAsync(formData)
      // Clear files on success
      setFiles([])
    } catch (error) {
      // Error is handled by react-query
      console.error('Upload failed:', error)
    }
  }

  const handleReset = useCallback(() => {
    upload.reset()
    setFiles([])
  }, [upload])

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-neutral-900">Upload de Documentos</h1>
        <p className="text-neutral-500 mt-1">
          Faça upload de arquivos Markdown para indexar na base de conhecimento
        </p>
      </div>

      {/* Drop Zone */}
      {!upload.data && (
        <DropZone
          onFilesSelected={handleFilesSelected}
          isDragging={isDragging}
          onDragEnter={() => setIsDragging(true)}
          onDragLeave={() => setIsDragging(false)}
          disabled={upload.isPending}
        />
      )}

      {/* Selected Files */}
      {files.length > 0 && !upload.data && (
        <Card className="p-4">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-medium text-neutral-700">
              Arquivos selecionados ({files.length})
            </h3>
            <button
              onClick={handleClearFiles}
              className="text-sm text-neutral-500 hover:text-neutral-700"
              disabled={upload.isPending}
            >
              Limpar tudo
            </button>
          </div>

          <div className="space-y-2 max-h-64 overflow-y-auto">
            {files.map((file, index) => (
              <div
                key={index}
                className="flex items-center gap-3 p-2 bg-neutral-50 rounded-lg"
              >
                <FileText className="w-4 h-4 text-neutral-400 flex-shrink-0" />
                <span className="flex-1 text-sm text-neutral-700 truncate">
                  {file.name}
                </span>
                <span className="text-xs text-neutral-400">
                  {formatFileSize(file.size)}
                </span>
                <button
                  onClick={() => handleRemoveFile(index)}
                  className="p-1 hover:bg-neutral-200 rounded"
                  disabled={upload.isPending}
                >
                  <X className="w-4 h-4 text-neutral-400" />
                </button>
              </div>
            ))}
          </div>
        </Card>
      )}

      {/* Upload Button */}
      {files.length > 0 && !upload.data && (
        <button
          onClick={handleUpload}
          disabled={upload.isPending}
          className={cn(
            'w-full py-3 px-4 rounded-lg font-medium transition-colors',
            'bg-primary-500 text-white hover:bg-primary-600',
            'disabled:opacity-50 disabled:cursor-not-allowed',
            'flex items-center justify-center gap-2'
          )}
        >
          {upload.isPending ? (
            <>
              <Loader2 className="w-5 h-5 animate-spin" />
              Indexando documentos...
            </>
          ) : (
            `Enviar ${files.length} arquivo${files.length > 1 ? 's' : ''}`
          )}
        </button>
      )}

      {/* Error State */}
      {upload.isError && (
        <Card className="p-4 border-red-200 bg-red-50">
          <p className="text-sm text-red-700">
            Erro ao fazer upload: {(upload.error as Error)?.message || 'Erro desconhecido'}
          </p>
        </Card>
      )}

      {/* Results */}
      {upload.data && (
        <div className="space-y-4">
          <UploadProgress
            results={upload.data.results}
            summary={upload.data.summary}
          />

          <button
            onClick={handleReset}
            className={cn(
              'w-full py-3 px-4 rounded-lg font-medium transition-colors',
              'bg-neutral-100 text-neutral-700 hover:bg-neutral-200'
            )}
          >
            Fazer novo upload
          </button>
        </div>
      )}
    </div>
  )
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}
