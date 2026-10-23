import * as React from 'react'
import { cn } from '@/lib/utils'

export const Textarea = React.forwardRef<HTMLTextAreaElement, React.ComponentProps<'textarea'>>(
  ({ className, ...props }, ref) => (
    <textarea
      className={cn(
        'min-h-24 w-full resize-y border-0 border-b border-[var(--kg-border-strong)] bg-[var(--kg-field)] px-4 py-3 text-sm text-[var(--kg-text)] outline-none focus:border-[var(--kg-primary)]',
        className,
      )}
      ref={ref}
      {...props}
    />
  ),
)
Textarea.displayName = 'Textarea'
