import type { ReactNode } from 'react'
import { Alert } from '@/components/ui/alert'
import { Progress } from '@/components/ui/progress'
import { cn } from '@/lib/utils'
import { KPagination } from './KPagination'
import { KSearch } from './KSearch'
import type { UseKQueryResult } from './useKQuery'

export type KQueryShellProps = {
  query: UseKQueryResult
  emptyMessage?: string
  /** Content when not loading and not empty. */
  children: ReactNode
  /** Optional toolbar row below search (e.g. view toggle). */
  toolbar?: ReactNode
  className?: string
  /** Treat as empty when total is 0 (paginated/search) or items empty. */
  isEmpty?: boolean
}

/**
 * Shared chrome for query-backed K views: error, KSearch, pagination, loading/empty.
 */
export function KQueryShell(props: KQueryShellProps) {
  const { query: kq } = props
  const empty =
    props.isEmpty ??
    (kq.items.length === 0 && (!(kq.paginated || kq.searchable) || kq.total === 0))

  return (
    <div className={cn('flex flex-col gap-5', props.className)}>
      {kq.error ? <Alert variant="danger">{kq.error}</Alert> : null}

      {kq.searchProps ? <KSearch {...kq.searchProps} /> : null}

      {props.toolbar}

      {kq.pagerProps ? <KPagination {...kq.pagerProps} /> : null}

      {kq.loading ? (
        <div className="flex items-center justify-center py-8" role="status" aria-label="Loading">
          <Progress indeterminate className="w-48 max-w-full" />
        </div>
      ) : empty ? (
        <div className="rounded border border-dashed border-[var(--kg-border)] bg-[var(--kg-surface)] px-6 py-10 text-center">
          <p className="text-base font-medium text-[var(--kg-text)]">
            {props.emptyMessage ?? 'No records found'}
          </p>
        </div>
      ) : (
        props.children
      )}

      {kq.pagerProps && !kq.loading && !empty ? <KPagination {...kq.pagerProps} /> : null}
    </div>
  )
}
