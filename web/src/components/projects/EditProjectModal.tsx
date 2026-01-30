import { useState } from 'react'
import { X, Loader2, ChevronDown, ChevronUp, Info } from 'lucide-react'
import { useUpdateProject } from '../../hooks/useProjects'
import type { Project, ChunkConfig } from '../../api/types'

interface EditProjectModalProps {
  isOpen: boolean
  onClose: () => void
  project: Project
}

interface FormData {
  name: string
  host: string
  token: string
  model: string
  chunkConfig: ChunkConfig | null
}

const defaultChunkConfig: ChunkConfig = {
  min_size: 500,
  max_size: 1000,
  overlap: 150,
  respect_boundaries: true,
}

export function EditProjectModal({ isOpen, onClose, project }: EditProjectModalProps) {
  const updateProject = useUpdateProject()
  const [showAdvanced, setShowAdvanced] = useState(!!project.chunk_config)
  const [formData, setFormData] = useState<FormData>({
    name: project.name,
    host: project.host,
    token: '',
    model: project.model,
    chunkConfig: project.chunk_config || null,
  })
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    if (!formData.name.trim()) {
      setError('Nome do projeto é obrigatório')
      return
    }
    if (!formData.host.trim()) {
      setError('Host é obrigatório')
      return
    }
    if (!formData.model.trim()) {
      setError('Modelo é obrigatório')
      return
    }

    // Validate chunk config if provided
    if (formData.chunkConfig) {
      const cfg = formData.chunkConfig
      if (cfg.min_size < 200 || cfg.min_size > 1000) {
        setError('Tamanho mínimo deve estar entre 200 e 1000 tokens')
        return
      }
      if (cfg.max_size < 500 || cfg.max_size > 2000) {
        setError('Tamanho máximo deve estar entre 500 e 2000 tokens')
        return
      }
      if (cfg.min_size >= cfg.max_size) {
        setError('Tamanho mínimo deve ser menor que o máximo')
        return
      }
      if (cfg.overlap < 0 || cfg.overlap > 500) {
        setError('Overlap deve estar entre 0 e 500 tokens')
        return
      }
      if (cfg.overlap >= cfg.min_size) {
        setError('Overlap deve ser menor que o tamanho mínimo')
        return
      }
    }

    try {
      await updateProject.mutateAsync({
        id: project.id,
        data: {
          name: formData.name.trim(),
          host: formData.host.trim(),
          token: formData.token.trim() || undefined,
          model: formData.model.trim(),
          chunk_config: formData.chunkConfig || undefined,
        },
      })
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Falha ao atualizar projeto')
    }
  }

  const enableAdvancedConfig = () => {
    setShowAdvanced(true)
    if (!formData.chunkConfig) {
      setFormData({ ...formData, chunkConfig: defaultChunkConfig })
    }
  }

  const disableAdvancedConfig = () => {
    setShowAdvanced(false)
    setFormData({ ...formData, chunkConfig: null })
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50"
        onClick={onClose}
      />

      {/* Modal */}
      <div className="relative bg-white rounded-xl shadow-xl w-full max-w-md mx-4 max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-neutral-200 sticky top-0 bg-white rounded-t-xl">
          <h2 className="text-lg font-semibold text-neutral-900">Editar Projeto</h2>
          <button
            onClick={onClose}
            className="p-1 text-neutral-400 hover:text-neutral-600 rounded-lg hover:bg-neutral-100 transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="p-6 space-y-5">
          {/* Error */}
          {error && (
            <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-600 text-sm">
              {error}
            </div>
          )}

          {/* Name */}
          <div>
            <label className="block text-sm font-medium text-neutral-700 mb-2">
              Nome do Projeto
            </label>
            <input
              type="text"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              placeholder="Minha Documentação"
              className="w-full px-4 py-2.5 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none transition-colors"
            />
          </div>

          {/* Host */}
          <div>
            <label className="block text-sm font-medium text-neutral-700 mb-2">
              Host (URL da API)
            </label>
            <input
              type="text"
              value={formData.host}
              onChange={(e) => setFormData({ ...formData, host: e.target.value })}
              placeholder="http://localhost:11434"
              className="w-full px-4 py-2.5 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none transition-colors"
            />
          </div>

          {/* Token */}
          <div>
            <label className="block text-sm font-medium text-neutral-700 mb-2">
              Token {project.has_token && '(deixe em branco para manter o atual)'}
            </label>
            <input
              type="password"
              value={formData.token}
              onChange={(e) => setFormData({ ...formData, token: e.target.value })}
              placeholder={project.has_token ? '••••••••' : 'sk-...'}
              className="w-full px-4 py-2.5 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none transition-colors"
            />
          </div>

          {/* Model */}
          <div>
            <label className="block text-sm font-medium text-neutral-700 mb-2">
              Modelo
            </label>
            <input
              type="text"
              value={formData.model}
              onChange={(e) => setFormData({ ...formData, model: e.target.value })}
              placeholder="nomic-embed-text"
              className="w-full px-4 py-2.5 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none transition-colors"
            />
          </div>

          {/* Advanced Configuration */}
          <div className="border-t border-neutral-200 pt-4">
            <button
              type="button"
              onClick={() => showAdvanced ? disableAdvancedConfig() : enableAdvancedConfig()}
              className="flex items-center justify-between w-full px-4 py-2.5 text-sm font-medium text-neutral-700 hover:bg-neutral-50 rounded-lg transition-colors"
            >
              <span>Configuração Avançada (opcional)</span>
              {showAdvanced ? (
                <ChevronUp className="w-4 h-4" />
              ) : (
                <ChevronDown className="w-4 h-4" />
              )}
            </button>

            {showAdvanced && formData.chunkConfig && (
              <div className="mt-4 p-4 bg-neutral-50 rounded-lg space-y-4">
                <p className="text-xs text-neutral-600">
                  Configure como os documentos são divididos em chunks para melhor busca semântica.
                </p>

                {/* Min Size */}
                <div>
                  <div className="flex items-center gap-2 mb-2">
                    <label className="block text-sm font-medium text-neutral-700">
                      Tamanho Mínimo (tokens)
                    </label>
                    <div className="group relative">
                      <Info className="w-4 h-4 text-neutral-400" />
                      <div className="hidden group-hover:block absolute left-0 top-6 w-64 p-2 bg-neutral-800 text-white text-xs rounded-lg z-10">
                        Tamanho mínimo em tokens para cada chunk. Chunks muito pequenos podem perder contexto. Recomendado: 500
                      </div>
                    </div>
                  </div>
                  <input
                    type="number"
                    min="200"
                    max="1000"
                    value={formData.chunkConfig.min_size}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        chunkConfig: formData.chunkConfig
                          ? { ...formData.chunkConfig, min_size: parseInt(e.target.value) || 500 }
                          : null,
                      })
                    }
                    className="w-full px-4 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none transition-colors"
                  />
                </div>

                {/* Max Size */}
                <div>
                  <div className="flex items-center gap-2 mb-2">
                    <label className="block text-sm font-medium text-neutral-700">
                      Tamanho Máximo (tokens)
                    </label>
                    <div className="group relative">
                      <Info className="w-4 h-4 text-neutral-400" />
                      <div className="hidden group-hover:block absolute left-0 top-6 w-64 p-2 bg-neutral-800 text-white text-xs rounded-lg z-10">
                        Tamanho máximo em tokens. Chunks muito grandes podem diluir relevância. Recomendado: 1000
                      </div>
                    </div>
                  </div>
                  <input
                    type="number"
                    min="500"
                    max="2000"
                    value={formData.chunkConfig.max_size}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        chunkConfig: formData.chunkConfig
                          ? { ...formData.chunkConfig, max_size: parseInt(e.target.value) || 1000 }
                          : null,
                      })
                    }
                    className="w-full px-4 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none transition-colors"
                  />
                </div>

                {/* Overlap */}
                <div>
                  <div className="flex items-center gap-2 mb-2">
                    <label className="block text-sm font-medium text-neutral-700">
                      Overlap (tokens)
                    </label>
                    <div className="group relative">
                      <Info className="w-4 h-4 text-neutral-400" />
                      <div className="hidden group-hover:block absolute left-0 top-6 w-64 p-2 bg-neutral-800 text-white text-xs rounded-lg z-10">
                        Quantos tokens do chunk anterior são incluídos no próximo. Melhora busca semântica mas aumenta tamanho. Recomendado: 150
                      </div>
                    </div>
                  </div>
                  <input
                    type="number"
                    min="0"
                    max="500"
                    value={formData.chunkConfig.overlap}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        chunkConfig: formData.chunkConfig
                          ? { ...formData.chunkConfig, overlap: parseInt(e.target.value) || 150 }
                          : null,
                      })
                    }
                    className="w-full px-4 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 outline-none transition-colors"
                  />
                </div>

                {/* Respect Boundaries */}
                <div className="flex items-center gap-3">
                  <input
                    type="checkbox"
                    id="respect_boundaries"
                    checked={formData.chunkConfig.respect_boundaries}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        chunkConfig: formData.chunkConfig
                          ? { ...formData.chunkConfig, respect_boundaries: e.target.checked }
                          : null,
                      })
                    }
                    className="w-4 h-4 text-primary-500 border-neutral-300 rounded focus:ring-primary-500"
                  />
                  <div className="flex items-center gap-2">
                    <label htmlFor="respect_boundaries" className="text-sm font-medium text-neutral-700">
                      Preservar Code Blocks e Tabelas
                    </label>
                    <div className="group relative">
                      <Info className="w-4 h-4 text-neutral-400" />
                      <div className="hidden group-hover:block absolute left-0 top-6 w-64 p-2 bg-neutral-800 text-white text-xs rounded-lg z-10">
                        Evita quebrar code blocks e tabelas no meio. Mantém formatação de diagramas ASCII. Recomendado: ativado
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            )}
          </div>

          {/* Actions */}
          <div className="flex justify-end gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2.5 text-neutral-600 hover:bg-neutral-100 rounded-lg transition-colors"
            >
              Cancelar
            </button>
            <button
              type="submit"
              disabled={updateProject.isPending}
              className="flex items-center gap-2 px-4 py-2.5 bg-primary-500 text-white rounded-lg hover:bg-primary-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {updateProject.isPending && <Loader2 className="w-4 h-4 animate-spin" />}
              Salvar Alterações
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
