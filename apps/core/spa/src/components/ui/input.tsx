import * as React from 'react'
import { cn } from '@/lib/utils'

export const Input = React.forwardRef<HTMLInputElement, React.ComponentProps<'input'>>(
  ({ className, type, ...props }, ref) => (
    <input
      type={type}
      className={cn(
        'h-10 w-full border-0 border-b border-[var(--kg-border-strong)] bg-[var(--kg-field)] px-4 text-sm text-[var(--kg-text)] outline-none focus:border-[var(--kg-primary)] disabled:cursor-not-allowed disabled:bg-[var(--kg-surface-muted)] disabled:text-[var(--kg-text-muted)]',
        className,
      )}
      ref={ref}
      {...props}
    />
  ),
)
Input.displayName = 'Input'
