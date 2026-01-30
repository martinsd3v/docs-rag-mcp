import { ReactNode } from 'react'
import { cn } from '../../lib/utils'
import { LucideIcon } from 'lucide-react'

interface StatCardProps {
  title: string
  value: string | number
  icon: LucideIcon
  subtitle?: string
  trend?: {
    value: number
    isPositive: boolean
  }
  color?: 'primary' | 'blue' | 'green' | 'purple' | 'amber'
}

const colorClasses = {
  primary: {
    bg: 'bg-primary-50',
    icon: 'text-primary-500',
  },
  blue: {
    bg: 'bg-blue-50',
    icon: 'text-blue-500',
  },
  green: {
    bg: 'bg-green-50',
    icon: 'text-green-500',
  },
  purple: {
    bg: 'bg-purple-50',
    icon: 'text-purple-500',
  },
  amber: {
    bg: 'bg-amber-50',
    icon: 'text-amber-500',
  },
}

export function StatCard({
  title,
  value,
  icon: Icon,
  subtitle,
  trend,
  color = 'primary',
}: StatCardProps) {
  const colors = colorClasses[color]

  return (
    <div className="card p-6">
      <div className="flex items-start justify-between">
        <div className={cn('p-3 rounded-xl', colors.bg)}>
          <Icon className={cn('w-6 h-6', colors.icon)} />
        </div>
        {trend && (
          <span
            className={cn(
              'text-sm font-medium px-2 py-0.5 rounded',
              trend.isPositive
                ? 'text-green-600 bg-green-50'
                : 'text-red-600 bg-red-50'
            )}
          >
            {trend.isPositive ? '+' : ''}{trend.value}%
          </span>
        )}
      </div>
      <div className="mt-4">
        <p className="text-3xl font-bold text-neutral-900">{value}</p>
        <p className="text-sm text-neutral-500 mt-1">{title}</p>
        {subtitle && (
          <p className="text-xs text-neutral-400 mt-1">{subtitle}</p>
        )}
      </div>
    </div>
  )
}

// Card wrapper component
interface CardProps {
  children: ReactNode
  className?: string
  hover?: boolean
}

export function Card({ children, className, hover = false }: CardProps) {
  return (
    <div className={cn(hover ? 'card-hover' : 'card', className)}>
      {children}
    </div>
  )
}
