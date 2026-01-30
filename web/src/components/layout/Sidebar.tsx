import { NavLink } from 'react-router-dom'
import { LayoutDashboard, FileText, Search, Database, Upload } from 'lucide-react'
import { cn } from '../../lib/utils'

const navItems = [
  { icon: LayoutDashboard, label: 'Dashboard', path: '/' },
  { icon: FileText, label: 'Documentos', path: '/documents' },
  { icon: Search, label: 'Busca', path: '/search' },
  { icon: Upload, label: 'Upload', path: '/upload' },
]

export function Sidebar() {
  return (
    <aside className="w-64 bg-white border-r border-neutral-200 flex flex-col">
      {/* Logo */}
      <div className="h-16 px-6 flex items-center border-b border-neutral-100">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 bg-primary-500 rounded-lg flex items-center justify-center">
            <Database className="w-5 h-5 text-white" />
          </div>
          <span className="font-semibold text-neutral-900">Docs RAG</span>
        </div>
      </div>

      {/* Navigation */}
      <nav className="flex-1 p-4 space-y-1">
        <p className="text-xs font-medium text-neutral-400 uppercase tracking-wider px-3 mb-3">
          Menu Principal
        </p>
        {navItems.map((item) => (
          <NavLink
            key={item.path}
            to={item.path}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-3 px-3 py-2.5 rounded-lg font-medium transition-colors',
                isActive
                  ? 'bg-primary-50 text-primary-600 border-l-4 border-primary-500 -ml-1 pl-4'
                  : 'text-neutral-600 hover:bg-neutral-50 hover:text-neutral-900'
              )
            }
          >
            <item.icon className="w-5 h-5" />
            {item.label}
          </NavLink>
        ))}
      </nav>

      {/* Footer */}
      <div className="p-4 border-t border-neutral-100">
        <div className="bg-neutral-50 rounded-lg p-3">
          <p className="text-xs text-neutral-500">Versão 0.1.0</p>
          <p className="text-xs text-neutral-400 mt-1">
            RAG MCP Server
          </p>
        </div>
      </div>
    </aside>
  )
}
