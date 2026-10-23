import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/utils'

const alertVariants = cva(
  'kg-alert flex items-start justify-between gap-3 border px-4 py-3 text-sm',
  {
    variants: {
      variant: {
        info: 'border-[var(--kg-info)] bg-[var(--kg-info-bg)] text-[var(--kg-info)]',
        success: 'border-[var(--kg-success)] bg-[var(--kg-success-bg)] text-[var(--kg-success)]',
        warning: 'border-[var(--kg-warning)] bg-[var(--kg-warning-bg)] text-[var(--kg-warning)]',
        danger: 'border-[var(--kg-danger)] bg-[var(--kg-danger-bg)] text-[var(--kg-danger)]',
      },
    },
    defaultVariants: { variant: 'info' },
  },
)

export function Alert({
  className,
  variant,
  ...props
}: React.HTMLAttributes<HTMLDivElement> & VariantProps<typeof alertVariants>) {
  return <div role="alert" className={cn(alertVariants({ variant }), className)} {...props} />
}

export { alertVariants }
