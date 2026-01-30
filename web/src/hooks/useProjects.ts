import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listProjects,
  createProject,
  deleteProject,
  startProject,
  stopProject,
  getActiveProject,
} from '../api/client'
import type { CreateProjectRequest } from '../api/types'

export function useProjects() {
  return useQuery({
    queryKey: ['projects'],
    queryFn: listProjects,
  })
}

export function useActiveProject() {
  return useQuery({
    queryKey: ['activeProject'],
    queryFn: getActiveProject,
    refetchInterval: 5000, // Poll every 5 seconds to detect changes
  })
}

export function useCreateProject() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (request: CreateProjectRequest) => createProject(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
  })
}

export function useDeleteProject() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => deleteProject(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
  })
}

export function useStartProject() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => startProject(id),
    onSuccess: (data) => {
      // Update activeProject cache immediately with the response
      queryClient.setQueryData(['activeProject'], data)
      queryClient.invalidateQueries({ queryKey: ['projects'] })
      queryClient.invalidateQueries({ queryKey: ['health'] })
      queryClient.invalidateQueries({ queryKey: ['stats'] })
      queryClient.invalidateQueries({ queryKey: ['documents'] })
    },
  })
}

export function useStopProject() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => stopProject(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
      queryClient.invalidateQueries({ queryKey: ['activeProject'] })
      queryClient.invalidateQueries({ queryKey: ['health'] })
    },
  })
}
