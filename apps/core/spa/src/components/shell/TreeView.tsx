import { useState, type ReactNode } from 'react'
import { ChevronDown, ChevronRight } from 'lucide-react'
import type { TreeNode } from '@/lib/types'
import { cn } from '@/lib/utils'
import { treeItemClass } from '@/lib/styles/classes'

type Props<T> = {
  nodes: TreeNode<T>[]
  selectedId?: string
  onSelect?: (node: TreeNode<T>) => void
  className?: string
  renderMeta?: (node: TreeNode<T>) => ReactNode
}

function NodeRow<T>({
  node,
  depth,
  selectedId,
  onSelect,
  renderMeta,
}: {
  node: TreeNode<T>
  depth: number
  selectedId?: string
  onSelect?: (node: TreeNode<T>) => void
  renderMeta?: (node: TreeNode<T>) => ReactNode
}) {
  const [open, setOpen] = useState(true)
  const hasChildren = Boolean(node.children?.length)

  return (
    <div>
      <button
        type="button"
        className={cn(treeItemClass, selectedId === node.id && 'data-selected')}
        style={{ paddingLeft: `calc(1.25rem + ${depth} * 1rem)` }}
        data-selected={selectedId === node.id ? '' : undefined}
        onClick={() => onSelect?.(node)}
      >
        {hasChildren ? (
          <span
            className="inline-flex"
            onClick={(e) => {
              e.stopPropagation()
              setOpen(!open)
            }}
          >
            {open ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
          </span>
        ) : (
          <span className="w-3.5" />
        )}
        <span className="min-w-0 flex-1 truncate text-left">{node.label}</span>
        {renderMeta?.(node) ??
          (node.meta ? <span className="text-xs text-[var(--kg-text-muted)]">{node.meta}</span> : null)}
      </button>
      {hasChildren && open
        ? node.children!.map((child) => (
            <NodeRow
              key={child.id}
              node={child}
              depth={depth + 1}
              selectedId={selectedId}
              onSelect={onSelect}
              renderMeta={renderMeta}
            />
          ))
        : null}
    </div>
  )
}

export function TreeView<T>({ nodes, selectedId, onSelect, className, renderMeta }: Props<T>) {
  return (
    <div className={cn('border border-[var(--kg-border)] bg-[var(--kg-surface)]', className)}>
      {nodes.map((node) => (
        <NodeRow
          key={node.id}
          node={node}
          depth={0}
          selectedId={selectedId}
          onSelect={onSelect}
          renderMeta={renderMeta}
        />
      ))}
    </div>
  )
}
