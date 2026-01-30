import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getHealth, getStats, getDocuments, getDocument, search, uploadDocuments } from '../api/client'
import type { SearchRequest } from '../api/types'

export function useHealth() {
  return useQuery({
    queryKey: ['health'],
    queryFn: getHealth,
    refetchInterval: 30000,
  })
}

export function useStats() {
  return useQuery({
    queryKey: ['stats'],
    queryFn: getStats,
    staleTime: 10000,
  })
}

export function useDocuments(params?: {
  page?: number
  limit?: number
  doc_type?: string
  status?: string
}) {
  return useQuery({
    queryKey: ['documents', params],
    queryFn: () => getDocuments(params),
  })
}

export function useDocument(id: string, options?: {
  include_chunks?: boolean
  include_related?: boolean
}) {
  return useQuery({
    queryKey: ['document', id, options],
    queryFn: () => getDocument(id, options),
    enabled: !!id,
  })
}

export function useSearch() {
  return useMutation({
    mutationFn: (request: SearchRequest) => search(request),
  })
}

export function useUpload() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (formData: FormData) => uploadDocuments(formData),
    onSuccess: () => {
      // Invalidate documents and stats queries to refresh the UI
      queryClient.invalidateQueries({ queryKey: ['documents'] })
      queryClient.invalidateQueries({ queryKey: ['stats'] })
    },
  })
}
