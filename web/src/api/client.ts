import axios from 'axios'
import type {
  HealthResponse,
  StatsResponse,
  DocumentsResponse,
  DocumentResponse,
  SearchResponse,
  SearchRequest,
  UploadResponse,
  ProjectListResponse,
  ActiveProjectResponse,
  CreateProjectRequest,
  Project,
} from './types'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Projects
export async function listProjects(): Promise<ProjectListResponse> {
  const { data } = await api.get<ProjectListResponse>('/projects')
  return data
}

export async function createProject(request: CreateProjectRequest): Promise<Project> {
  const { data } = await api.post<Project>('/projects', request)
  return data
}

export async function deleteProject(id: string): Promise<void> {
  await api.delete(`/projects/${id}`)
}

export async function startProject(id: string): Promise<ActiveProjectResponse> {
  const { data } = await api.post<ActiveProjectResponse>(`/projects/${id}/start`)
  return data
}

export async function stopProject(): Promise<void> {
  await api.post('/projects/stop')
}

export async function getActiveProject(): Promise<ActiveProjectResponse> {
  const { data } = await api.get<ActiveProjectResponse>('/projects/active')
  return data
}

// Health
export async function getHealth(): Promise<HealthResponse> {
  const { data } = await api.get<HealthResponse>('/health')
  return data
}

// Stats
export async function getStats(): Promise<StatsResponse> {
  const { data } = await api.get<StatsResponse>('/stats')
  return data
}

// Documents
export async function getDocuments(params?: {
  page?: number
  limit?: number
  doc_type?: string
  status?: string
}): Promise<DocumentsResponse> {
  const { data } = await api.get<DocumentsResponse>('/documents', { params })
  return data
}

export async function getDocument(
  id: string,
  params?: {
    include_chunks?: boolean
    include_related?: boolean
  }
): Promise<DocumentResponse> {
  const { data } = await api.get<DocumentResponse>(`/documents/${id}`, { params })
  return data
}

// Search
export async function search(request: SearchRequest): Promise<SearchResponse> {
  const { data } = await api.post<SearchResponse>('/search', request)
  return data
}

// Upload
export async function uploadDocuments(formData: FormData): Promise<UploadResponse> {
  const { data } = await api.post<UploadResponse>('/upload', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 120000, // 2 minutes for large uploads
  })
  return data
}

export default api
