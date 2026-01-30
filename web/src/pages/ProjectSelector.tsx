import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Database, Folder, Play, Trash2, Plus, Loader2, Pencil } from 'lucide-react'
import { useProjects, useStartProject, useDeleteProject } from '../hooks/useProjects'
import { CreateProjectModal } from '../components/projects/CreateProjectModal'
import { EditProjectModal } from '../components/projects/EditProjectModal'
import { ConfirmModal } from '../components/ui/ConfirmModal'
import type { Project } from '../api/types'

function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  return date.toLocaleDateString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  })
}

function ProjectCard({
  project,
  onStart,
  onEdit,
  onDelete,
  isActive,
  isStarting,
  isDeleting,
}: {
  project: Project
  onStart: () => void
  onEdit: () => void
  onDelete: () => void
  isActive: boolean
  isStarting: boolean
  isDeleting: boolean
}) {
  return (
    <div className="bg-white rounded-xl border border-neutral-200 p-5 hover:border-primary-300 hover:shadow-md transition-all">
      <div className="flex items-start justify-between">
        <div className="flex items-start gap-4">
          <div className="w-10 h-10 bg-primary-50 rounded-lg flex items-center justify-center flex-shrink-0">
            <Folder className="w-5 h-5 text-primary-500" />
          </div>
          <div>
            <h3 className="font-semibold text-neutral-900">{project.name}</h3>
            <p className="text-sm text-neutral-500 mt-1">
              {project.host} &bull; {project.model}
            </p>
            <p className="text-xs text-neutral-400 mt-2">
              Criado em: {formatDate(project.created_at)}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={onStart}
            disabled={isStarting || isDeleting}
            className="flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isStarting ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Play className="w-4 h-4" />
            )}
            RUN
          </button>
          <button
            onClick={onEdit}
            disabled={isActive || isStarting || isDeleting}
            className="p-2 text-neutral-400 hover:text-primary-500 hover:bg-primary-50 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            title={isActive ? "Pare o projeto para editar" : "Editar projeto"}
          >
            <Pencil className="w-4 h-4" />
          </button>
          <button
            onClick={onDelete}
            disabled={isStarting || isDeleting}
            className="p-2 text-neutral-400 hover:text-red-500 hover:bg-red-50 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            title="Deletar projeto"
          >
            {isDeleting ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Trash2 className="w-4 h-4" />
            )}
          </button>
        </div>
      </div>
    </div>
  )
}

export function ProjectSelector() {
  const navigate = useNavigate()
  const { data, isLoading: isLoadingProjects } = useProjects()
  const startProject = useStartProject()
  const deleteProject = useDeleteProject()
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)
  const [projectToEdit, setProjectToEdit] = useState<Project | null>(null)
  const [startingId, setStartingId] = useState<string | null>(null)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [projectToDelete, setProjectToDelete] = useState<{ id: string; name: string } | null>(null)

  const handleStart = async (id: string) => {
    setStartingId(id)
    try {
      await startProject.mutateAsync(id)
      navigate('/')
    } catch (error) {
      console.error('Failed to start project:', error)
    } finally {
      setStartingId(null)
    }
  }

  const handleEditClick = (project: Project) => {
    setProjectToEdit(project)
  }

  const handleDeleteClick = (id: string, name: string) => {
    setProjectToDelete({ id, name })
  }

  const handleDeleteConfirm = async () => {
    if (!projectToDelete) return

    setDeletingId(projectToDelete.id)
    try {
      await deleteProject.mutateAsync(projectToDelete.id)
      setProjectToDelete(null)
    } catch (error) {
      console.error('Failed to delete project:', error)
    } finally {
      setDeletingId(null)
    }
  }

  return (
    <div className="min-h-screen bg-neutral-50 flex items-center justify-center p-6">
      <div className="w-full max-w-2xl">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="flex items-center justify-center gap-3 mb-4">
            <div className="w-12 h-12 bg-primary-500 rounded-xl flex items-center justify-center">
              <Database className="w-7 h-7 text-white" />
            </div>
          </div>
          <h1 className="text-2xl font-bold text-neutral-900">Docs RAG</h1>
          <p className="text-neutral-500 mt-2">Selecione um projeto para iniciar</p>
        </div>

        {/* Projects List */}
        <div className="space-y-4">
          {isLoadingProjects ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="w-8 h-8 text-primary-500 animate-spin" />
            </div>
          ) : data?.projects && data.projects.length > 0 ? (
            data.projects.map((project) => (
              <ProjectCard
                key={project.id}
                project={project}
                onStart={() => handleStart(project.id)}
                onEdit={() => handleEditClick(project)}
                onDelete={() => handleDeleteClick(project.id, project.name)}
                isActive={data.active_project_id === project.id}
                isStarting={startingId === project.id}
                isDeleting={deletingId === project.id}
              />
            ))
          ) : (
            <div className="text-center py-12 bg-white rounded-xl border border-dashed border-neutral-300">
              <Folder className="w-12 h-12 text-neutral-300 mx-auto mb-3" />
              <p className="text-neutral-500">Nenhum projeto encontrado</p>
              <p className="text-neutral-400 text-sm mt-1">
                Crie um novo projeto para começar
              </p>
            </div>
          )}
        </div>

        {/* Create Button */}
        <button
          onClick={() => setIsCreateModalOpen(true)}
          className="w-full mt-6 flex items-center justify-center gap-2 py-3 border-2 border-dashed border-neutral-300 rounded-xl text-neutral-600 hover:border-primary-400 hover:text-primary-600 hover:bg-primary-50 transition-colors"
        >
          <Plus className="w-5 h-5" />
          Criar Novo Projeto
        </button>

        {/* Create Modal */}
        <CreateProjectModal
          isOpen={isCreateModalOpen}
          onClose={() => setIsCreateModalOpen(false)}
        />

        {/* Edit Modal */}
        {projectToEdit && (
          <EditProjectModal
            isOpen={!!projectToEdit}
            onClose={() => setProjectToEdit(null)}
            project={projectToEdit}
          />
        )}

        {/* Delete Confirmation Modal */}
        <ConfirmModal
          isOpen={!!projectToDelete}
          onClose={() => setProjectToDelete(null)}
          onConfirm={handleDeleteConfirm}
          title="Deletar Projeto"
          message={`Tem certeza que deseja deletar o projeto "${projectToDelete?.name}"?\n\nIsso vai remover o banco de dados e todos os documentos indexados. Esta ação não pode ser desfeita.`}
          confirmText="Deletar"
          cancelText="Cancelar"
          variant="danger"
          isLoading={!!deletingId}
        />
      </div>
    </div>
  )
}
