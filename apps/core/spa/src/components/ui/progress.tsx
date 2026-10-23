import * as React from 'react'
import * as ProgressPrimitive from '@radix-ui/react-progress'
import { cn } from '@/lib/utils'

export function Progress({
  className,
  value,
  indeterminate,
  ...props
}: React.ComponentPropsWithoutRef<typeof ProgressPrimitive.Root> & { indeterminate?: boolean }) {
  return (
    <ProgressPrimitive.Root
      className={cn('h-1 w-full overflow-hidden bg-[var(--kg-border)]', className)}
      value={indeterminate ? undefined : value}
      {...props}
    >
      <ProgressPrimitive.Indicator
        className={cn(
          'h-full bg-[var(--kg-primary)] transition-all',
          indeterminate && 'w-full animate-pulse',
        )}
        style={indeterminate ? undefined : { width: `${value ?? 0}%` }}
      />
    </ProgressPrimitive.Root>
  )
}
