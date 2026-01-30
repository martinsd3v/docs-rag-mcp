import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatNumber(num: number): string {
  return new Intl.NumberFormat().format(num)
}

export function formatDate(date: string | Date): string {
  const d = new Date(date)
  return new Intl.DateTimeFormat('pt-BR', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
  }).format(d)
}

export function formatRelativeTime(date: string | Date): string {
  const d = new Date(date)
  const now = new Date()
  const diffMs = now.getTime() - d.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMs / 3600000)
  const diffDays = Math.floor(diffMs / 86400000)

  if (diffMins < 1) return 'agora'
  if (diffMins < 60) return `${diffMins}min atrás`
  if (diffHours < 24) return `${diffHours}h atrás`
  if (diffDays < 7) return `${diffDays}d atrás`
  return formatDate(date)
}

export function truncateText(text: string, maxLength: number): string {
  if (text.length <= maxLength) return text
  return text.slice(0, maxLength) + '...'
}

export function getDocTypeColor(docType: string): string {
  const colors: Record<string, string> = {
    RFC: 'bg-blue-100 text-blue-700',
    ADR: 'bg-green-100 text-green-700',
    BDR: 'bg-purple-100 text-purple-700',
    Guideline: 'bg-amber-100 text-amber-700',
    Roadmap: 'bg-pink-100 text-pink-700',
  }
  return colors[docType] || 'bg-neutral-100 text-neutral-700'
}

export function getStatusColor(status: string): string {
  const colors: Record<string, string> = {
    Draft: 'bg-yellow-100 text-yellow-700',
    Approved: 'bg-green-100 text-green-700',
    Deprecated: 'bg-red-100 text-red-700',
    Active: 'bg-green-100 text-green-700',
  }
  return colors[status] || 'bg-neutral-100 text-neutral-700'
}

export function getScoreColor(score: number): string {
  if (score >= 0.8) return 'text-green-600 bg-green-50'
  if (score >= 0.5) return 'text-amber-600 bg-amber-50'
  return 'text-neutral-600 bg-neutral-50'
}
