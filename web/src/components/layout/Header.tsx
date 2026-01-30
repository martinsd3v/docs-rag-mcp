import { useLocation } from 'react-router-dom'
import { Bell, Settings } from 'lucide-react'
import { useHealth } from '../../hooks/useApi'

const pageTitles: Record<string, string> = {
  '/': 'Dashboard',
  '/documents': 'Documentos',
  '/search': 'Busca Semântica',
}

export function Header() {
  const location = useLocation()
  const { data: health } = useHealth()

  const title = location.pathname.startsWith('/documents/')
    ? 'Detalhes do Documento'
    : pageTitles[location.pathname] || 'Docs RAG'

  return (
    <header className="h-16 bg-white border-b border-neutral-200 px-6 flex items-center justify-between">
      <div>
        <h1 className="text-xl font-semibold text-neutral-900">{title}</h1>
      </div>

      <div className="flex items-center gap-4">
        {/* Status indicator */}
        <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-neutral-50">
          <span className={`w-2 h-2 rounded-full ${
            health?.status === 'healthy' ? 'bg-green-500' : 'bg-yellow-500'
          }`} />
          <span className="text-sm text-neutral-600">
            {health?.status === 'healthy' ? 'Online' : 'Verificando...'}
          </span>
        </div>

        {/* Actions */}
        <button className="p-2 text-neutral-500 hover:text-neutral-700 hover:bg-neutral-100 rounded-lg transition-colors">
          <Bell className="w-5 h-5" />
        </button>
        <button className="p-2 text-neutral-500 hover:text-neutral-700 hover:bg-neutral-100 rounded-lg transition-colors">
          <Settings className="w-5 h-5" />
        </button>

        {/* User */}
        <div className="flex items-center gap-3 pl-4 border-l border-neutral-200">
          <div className="w-8 h-8 bg-primary-100 rounded-full flex items-center justify-center">
            <span className="text-sm font-medium text-primary-600">U</span>
          </div>
          <div className="text-sm">
            <p className="font-medium text-neutral-900">Usuário</p>
            <p className="text-neutral-500 text-xs">Admin</p>
          </div>
        </div>
      </div>
    </header>
  )
}
