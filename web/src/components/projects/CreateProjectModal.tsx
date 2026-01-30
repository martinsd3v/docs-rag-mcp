import { useState } from 'react'
import { X, Loader2 } from 'lucide-react'
import { useCreateProject } from '../../hooks/useProjects'

interface CreateProjectModalProps {
  isOpen: boolean
  onClose: () => void
}

interface FormData {
  name: string
  host: string
  token: string
  model: string
}

const presets = {
  ollama: {
    host: 'http://localhost:11434',
    model: 'nomic-embed-text',
    token: '',
  },
  openai: {
    host: 'https://api.openai.com/v1',
    model: 'text-embedding-3-small',
    token: '',
  },
}

export function CreateProjectModal({ isOpen, onClose }: CreateProjectModalProps) {
  const createProject = useCreateProject()
  const [formData, setFormData] = useState<FormData>({
    name: '',
    host: 'http://localhost:11434',
    token: '',
    model: 'nomic-embed-text',
  })
  const [error, setError] = useState<string | null>(null)

  const handlePreset = (preset: 'ollama' | 'openai') => {
    setFormData((prev) => ({
      ...prev,
      ...presets[preset],
    }))
  }

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

    try {
      await createProject.mutateAsync({
        name: formData.name.trim(),
        host: formData.host.trim(),
        token: formData.token.trim() || undefined,
        model: formData.model.trim(),
      })
      // Reset form and close
      setFormData({
        name: '',
        host: 'http://localhost:11434',
        token: '',
        model: 'nomic-embed-text',
      })
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Falha ao criar projeto')
    }
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
      <div className="relative bg-white rounded-xl shadow-xl w-full max-w-md mx-4">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-neutral-200">
          <h2 className="text-lg font-semibold text-neutral-900">Criar Novo Projeto</h2>
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

          {/* Presets */}
          <div>
            <label className="block text-sm font-medium text-neutral-700 mb-2">
              Presets Rápidos
            </label>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => handlePreset('ollama')}
                className="px-4 py-2 text-sm border border-neutral-300 rounded-lg hover:bg-neutral-50 hover:border-neutral-400 transition-colors"
              >
                Ollama Local
              </button>
              <button
                type="button"
                onClick={() => handlePreset('openai')}
                className="px-4 py-2 text-sm border border-neutral-300 rounded-lg hover:bg-neutral-50 hover:border-neutral-400 transition-colors"
              >
                OpenAI
              </button>
            </div>
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
              Token (opcional para Ollama)
            </label>
            <input
              type="password"
              value={formData.token}
              onChange={(e) => setFormData({ ...formData, token: e.target.value })}
              placeholder="sk-..."
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
              disabled={createProject.isPending}
              className="flex items-center gap-2 px-4 py-2.5 bg-primary-500 text-white rounded-lg hover:bg-primary-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {createProject.isPending && <Loader2 className="w-4 h-4 animate-spin" />}
              Criar Projeto
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
