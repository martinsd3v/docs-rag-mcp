import { useLocation, useNavigate } from 'react-router-dom'
import { Power, Folder, Loader2 } from 'lucide-react'
import { useHealth } from '../../hooks/useApi'
import { useActiveProject, useStopProject } from '../../hooks/useProjects'

const pageTitles: Record<string, string> = {
  '/': 'Dashboard',
  '/documents': 'Documentos',
  '/search': 'Busca Semântica',
  '/upload': 'Upload',
}

export function Header() {
  const location = useLocation()
  const navigate = useNavigate()
  const { data: health } = useHealth()
  const { data: activeProject } = useActiveProject()
  const stopProject = useStopProject()

  const title = location.pathname.startsWith('/documents/')
    ? 'Detalhes do Documento'
    : pageTitles[location.pathname] || 'Docs RAG'

  const handleExit = async () => {
    try {
      await stopProject.mutateAsync()
      navigate('/projects')
    } catch (error) {
      console.error('Failed to stop project:', error)
    }
  }

  return (
    <header className="h-16 bg-white border-b border-neutral-200 px-6 flex items-center justify-between">
      <div>
        <h1 className="text-xl font-semibold text-neutral-900">{title}</h1>
      </div>

      <div className="flex items-center gap-4">
        {/* Active Project */}
        {activeProject?.project && (
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-primary-50 border border-primary-200">
            <Folder className="w-4 h-4 text-primary-500" />
            <span className="text-sm font-medium text-primary-700">
              {activeProject.project.name}
            </span>
          </div>
        )}

        {/* Status indicator */}
        <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-neutral-50">
          <span className={`w-2 h-2 rounded-full ${
            health?.status === 'healthy' ? 'bg-green-500' : 'bg-yellow-500'
          }`} />
          <span className="text-sm text-neutral-600">
            {health?.status === 'healthy' ? 'Online' : 'Verificando...'}
          </span>
        </div>

        {/* Stop Button */}
        <button
          onClick={handleExit}
          disabled={stopProject.isPending}
          className="flex items-center gap-2 px-3 py-2 text-neutral-600 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors disabled:opacity-50"
          title="Parar servidor MCP"
        >
          {stopProject.isPending ? (
            <Loader2 className="w-4 h-4 animate-spin" />
          ) : (
            <Power className="w-4 h-4" />
          )}
          <span className="text-sm font-medium">STOP</span>
        </button>
      </div>
    </header>
  )
}
