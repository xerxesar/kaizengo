import { cva, type VariantProps } from 'class-variance-authority'
import { Slot } from '@radix-ui/react-slot'
import * as React from 'react'
import { cn } from '@/lib/utils'

const buttonVariants = cva(
  'inline-flex items-center justify-center gap-2 font-medium transition outline-none disabled:pointer-events-none disabled:opacity-50',
  {
    variants: {
      variant: {
        primary: 'bg-[var(--kg-primary)] text-white hover:opacity-90',
        secondary:
          'border border-[var(--kg-border-strong)] bg-[var(--kg-surface)] text-[var(--kg-text)] hover:bg-[var(--kg-surface-hover)]',
        ghost: 'bg-transparent text-[var(--kg-text)] hover:bg-[var(--kg-surface-hover)]',
        danger: 'bg-[var(--kg-danger)] text-[var(--kg-text-on-color)] hover:opacity-90',
      },
      size: {
        sm: 'h-8 px-3 text-sm',
        md: 'h-10 px-4 text-sm',
      },
    },
    defaultVariants: { variant: 'primary', size: 'md' },
  },
)

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : 'button'
    return <Comp className={cn(buttonVariants({ variant, size }), className)} ref={ref} {...props} />
  },
)
Button.displayName = 'Button'

export { buttonVariants }
