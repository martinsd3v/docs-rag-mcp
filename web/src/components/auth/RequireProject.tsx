import { useEffect } from 'react'
import { Outlet, useNavigate } from 'react-router-dom'
import { useActiveProject } from '../../hooks/useProjects'
import { Loader2 } from 'lucide-react'

export function RequireProject() {
  const { data, isLoading } = useActiveProject()
  const navigate = useNavigate()

  useEffect(() => {
    if (!isLoading && data?.status !== 'running') {
      navigate('/projects')
    }
  }, [data, isLoading, navigate])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-screen bg-neutral-50">
        <div className="flex flex-col items-center gap-4">
          <Loader2 className="w-8 h-8 text-primary-500 animate-spin" />
          <p className="text-neutral-600">Carregando...</p>
        </div>
      </div>
    )
  }

  if (data?.status !== 'running') {
    return null
  }

  return <Outlet />
}
