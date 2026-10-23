import * as React from 'react'
import { cn } from '@/lib/utils'

type DialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  children?: React.ReactNode
  /** Panel width/layout (default `max-w-lg`). */
  className?: string
}

export function Dialog({ open, onOpenChange, children, className }: DialogProps) {
  if (!open) return null
  return (
    <div className="fixed inset-0 z-[9000] flex items-center justify-center p-4">
      <button
        type="button"
        className="absolute inset-0 bg-black/50"
        aria-label="Close"
        onClick={() => onOpenChange(false)}
      />
      <div
        className={cn(
          'relative z-[9001] flex max-h-[90vh] w-full max-w-lg flex-col bg-[var(--kg-surface)] shadow-lg outline-none',
          className,
        )}
      >
        {children}
      </div>
    </div>
  )
}

export function DialogContent({
  className,
  children,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn('flex max-h-[90vh] w-full flex-col overflow-auto', className)} {...props}>
      {children}
    </div>
  )
}

export function DialogHeader({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('border-b border-[var(--kg-border)] px-5 py-4', className)} {...props} />
}

export function DialogTitle({ className, ...props }: React.HTMLAttributes<HTMLHeadingElement>) {
  return <h2 className={cn('text-lg font-semibold text-[var(--kg-text)]', className)} {...props} />
}

export function DialogFooter({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn('flex items-center justify-end gap-3 border-t border-[var(--kg-border)] px-5 py-4', className)}
      {...props}
    />
  )
}
