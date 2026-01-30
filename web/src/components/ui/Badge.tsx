import { cn } from '../../lib/utils'

interface BadgeProps {
  children: React.ReactNode
  variant?: 'default' | 'primary' | 'success' | 'warning' | 'danger'
  size?: 'sm' | 'md'
  className?: string
}

const variantClasses = {
  default: 'bg-neutral-100 text-neutral-700',
  primary: 'bg-primary-100 text-primary-700',
  success: 'bg-green-100 text-green-700',
  warning: 'bg-yellow-100 text-yellow-700',
  danger: 'bg-red-100 text-red-700',
}

const sizeClasses = {
  sm: 'px-2 py-0.5 text-xs',
  md: 'px-2.5 py-1 text-sm',
}

export function Badge({
  children,
  variant = 'default',
  size = 'sm',
  className,
}: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center font-medium rounded-full',
        variantClasses[variant],
        sizeClasses[size],
        className
      )}
    >
      {children}
    </span>
  )
}

// DocType specific badge
interface DocTypeBadgeProps {
  docType: string
  size?: 'sm' | 'md'
}

const docTypeColors: Record<string, string> = {
  RFC: 'bg-blue-100 text-blue-700',
  ADR: 'bg-green-100 text-green-700',
  BDR: 'bg-purple-100 text-purple-700',
  Guideline: 'bg-amber-100 text-amber-700',
  Roadmap: 'bg-pink-100 text-pink-700',
}

export function DocTypeBadge({ docType, size = 'sm' }: DocTypeBadgeProps) {
  const colorClass = docTypeColors[docType] || 'bg-neutral-100 text-neutral-700'

  return (
    <span
      className={cn(
        'inline-flex items-center font-medium rounded-full',
        colorClass,
        sizeClasses[size]
      )}
    >
      {docType}
    </span>
  )
}

// Status badge
interface StatusBadgeProps {
  status: string
  size?: 'sm' | 'md'
}

const statusColors: Record<string, string> = {
  Draft: 'bg-yellow-100 text-yellow-700',
  Approved: 'bg-green-100 text-green-700',
  Deprecated: 'bg-red-100 text-red-700',
  Active: 'bg-green-100 text-green-700',
}

export function StatusBadge({ status, size = 'sm' }: StatusBadgeProps) {
  const colorClass = statusColors[status] || 'bg-neutral-100 text-neutral-700'

  return (
    <span
      className={cn(
        'inline-flex items-center font-medium rounded-full',
        colorClass,
        sizeClasses[size]
      )}
    >
      {status}
    </span>
  )
}

// Score badge
interface ScoreBadgeProps {
  score: number
}

export function ScoreBadge({ score }: ScoreBadgeProps) {
  const percentage = Math.round(score * 100)
  let colorClass = 'bg-neutral-100 text-neutral-600'

  if (score >= 0.8) {
    colorClass = 'bg-green-100 text-green-700'
  } else if (score >= 0.5) {
    colorClass = 'bg-amber-100 text-amber-700'
  }

  return (
    <span className={cn('inline-flex items-center font-bold rounded-lg px-2.5 py-1', colorClass)}>
      {percentage}%
    </span>
  )
}
