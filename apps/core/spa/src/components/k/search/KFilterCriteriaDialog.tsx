import { useEffect, useState, type ReactNode } from 'react'
import { Minus, Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { cn } from '@/lib/utils'
import {
  cloneFilterTree,
  emptyFilterTree,
  filterTreeIsEmpty,
  newFilterGroup,
  newFilterLeaf,
  updateFilterNode,
  type FilterGroup,
  type FilterNode,
} from './domain-tree'
import { opsForFieldType, type DomainOp, type SearchField } from './types'

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  fields: SearchField[]
  root: FilterGroup
  onApply: (root: FilterGroup) => void
  onSaveCriteria?: () => void
}

function fieldLabel(fields: SearchField[], key: string) {
  return fields.find((f) => f.key === key)?.label ?? key
}

function QueryPreview({
  node,
  fields,
}: {
  node: FilterNode
  fields: SearchField[]
}) {
  if (node.kind === 'leaf') {
    const label = fieldLabel(fields, node.field)
    if (node.op === 'is set' || node.op === 'is not set') {
      return (
        <>
          <span className="font-semibold text-[var(--kg-text)]">{label}</span>
          <span className="text-[var(--kg-text-muted)]"> {node.op}</span>
        </>
      )
    }
    const raw = node.value == null ? '' : String(node.value)
    const display = raw.replace(/^%|%$/g, '')
    return (
      <>
        <span className="font-semibold text-[var(--kg-text)]">{label}</span>
        <span className="text-[var(--kg-text-muted)]"> {node.op} </span>
        <span className="font-medium text-[var(--kg-primary)]">&quot;{display}&quot;</span>
      </>
    )
  }
  if (!node.children.length) return null
  return (
    <>
      <span className="text-[var(--kg-text-muted)]">(</span>
      {node.children.map((child, i) => (
        <span key={child.id}>
          {i > 0 ? (
            <span className="mx-1.5 font-bold uppercase text-[var(--kg-text-secondary)]">
              {node.junction}
            </span>
          ) : null}
          <QueryPreview node={child} fields={fields} />
        </span>
      ))}
      <span className="text-[var(--kg-text-muted)]">)</span>
    </>
  )
}

function QueryOutputBar({ children }: { children: ReactNode }) {
  return (
    <div className="border border-[var(--kg-border)] bg-[var(--kg-surface-muted,var(--kg-field-hover))] px-3 py-2 text-sm leading-relaxed">
      <div className="mb-1 text-xxs font-semibold uppercase tracking-wide text-[var(--kg-text-muted)]">
        Generated query
      </div>
      <div className="font-mono text-[13px] text-[var(--kg-text)]">{children}</div>
    </div>
  )
}

function LeafRow({
  fields,
  leaf,
  onChange,
  onRemove,
}: {
  fields: SearchField[]
  leaf: Extract<FilterNode, { kind: 'leaf' }>
  onChange: (next: Extract<FilterNode, { kind: 'leaf' }>) => void
  onRemove: () => void
}) {
  const field = fields.find((f) => f.key === leaf.field) ?? fields[0]
  const ops = opsForFieldType(field?.type)
  const displayValue =
    leaf.value == null
      ? ''
      : String(leaf.value).replace(/^%|%$/g, '')

  return (
    <div className="flex flex-wrap items-center gap-2 border border-[var(--kg-border)] bg-[var(--kg-surface)] px-2 py-1.5">
      <Select
        value={leaf.field}
        onValueChange={(key) => {
          const f = fields.find((x) => x.key === key)
          const nextOps = opsForFieldType(f?.type)
          onChange({
            ...leaf,
            field: key,
            op: nextOps.includes(leaf.op as DomainOp) ? leaf.op : (nextOps[0] ?? '='),
            value: '',
          })
        }}
      >
        <SelectTrigger className="h-8 w-[9rem]" aria-label="Field">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {fields.map((f) => (
            <SelectItem key={f.key} value={f.key}>
              {f.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Select value={String(leaf.op)} onValueChange={(op) => onChange({ ...leaf, op })}>
        <SelectTrigger className="h-8 w-[5.5rem]" aria-label="Operator">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {ops.map((op) => (
            <SelectItem key={op} value={op}>
              {op}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {leaf.op !== 'is set' && leaf.op !== 'is not set' ? (
        field?.values?.length ? (
          <Select value={String(leaf.value ?? '')} onValueChange={(value) => onChange({ ...leaf, value })}>
            <SelectTrigger className="h-8 min-w-[6rem] flex-1" aria-label="Value">
              <SelectValue placeholder="Value" />
            </SelectTrigger>
            <SelectContent>
              {field.values.map((v) => (
                <SelectItem key={v} value={v}>
                  {v}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        ) : (
          <Input
            className="h-8 min-w-[6rem] flex-1"
            value={displayValue}
            onChange={(e) => onChange({ ...leaf, value: e.target.value })}
            placeholder="Value"
          />
        )
      ) : null}
      <button
        type="button"
        className="ml-auto flex h-7 w-7 shrink-0 items-center justify-center text-[var(--kg-danger)] hover:bg-[var(--kg-danger-bg)]"
        aria-label="Remove condition"
        onClick={onRemove}
      >
        <Minus className="h-3.5 w-3.5" />
      </button>
    </div>
  )
}

function QueryGroup({
  fields,
  group,
  depth,
  onChange,
  onRemove,
}: {
  fields: SearchField[]
  group: FilterGroup
  depth: number
  onChange: (next: FilterGroup) => void
  onRemove?: () => void
}) {
  function patch(id: string, updater: (node: FilterNode) => FilterNode | null) {
    onChange(updateFilterNode(group, id, updater))
  }

  function addCondition() {
    const first = fields[0]
    if (!first) return
    onChange({
      ...group,
      children: [
        ...group.children,
        newFilterLeaf(first.key, opsForFieldType(first.type)[0] ?? 'ilike'),
      ],
    })
  }

  return (
    <div
      className={cn(
        'space-y-2 border border-[var(--kg-border)] bg-[var(--kg-surface)] p-3',
        depth > 0 && 'border-l-2 border-l-[var(--kg-border-strong)] bg-[var(--kg-surface-muted,var(--kg-field-hover))]',
      )}
    >
      <div className="flex flex-wrap items-center gap-2">
        <Select
          value={group.junction}
          onValueChange={(v) => onChange({ ...group, junction: v as 'and' | 'or' })}
        >
          <SelectTrigger className="h-8 w-[5.5rem] font-semibold" aria-label="Junction">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="and">AND</SelectItem>
            <SelectItem value="or">OR</SelectItem>
          </SelectContent>
        </Select>
        <Button type="button" size="sm" variant="secondary" className="h-8" onClick={addCondition}>
          <Plus className="mr-1 h-3.5 w-3.5" />
          Condition
        </Button>
        <Button
          type="button"
          size="sm"
          variant="secondary"
          className="h-8"
          onClick={() => onChange({ ...group, children: [...group.children, newFilterGroup('and', [])] })}
        >
          <Plus className="mr-1 h-3.5 w-3.5" />
          Group
        </Button>
        {depth > 0 ? (
          <Button type="button" size="sm" variant="ghost" className="h-8 text-[var(--kg-danger)]" onClick={onRemove}>
            <Minus className="mr-1 h-3.5 w-3.5" />
            Remove group
          </Button>
        ) : null}
      </div>

      <div className="space-y-2 pl-1">
        {group.children.map((child) =>
          child.kind === 'leaf' ? (
            <LeafRow
              key={child.id}
              fields={fields}
              leaf={child}
              onChange={(next) => patch(child.id, () => next)}
              onRemove={() => patch(child.id, () => null)}
            />
          ) : (
            <QueryGroup
              key={child.id}
              fields={fields}
              group={child}
              depth={depth + 1}
              onChange={(next) => patch(child.id, () => next)}
              onRemove={() => patch(child.id, () => null)}
            />
          ),
        )}
      </div>
    </div>
  )
}

export function KFilterCriteriaDialog({
  open,
  onOpenChange,
  fields,
  root,
  onApply,
  onSaveCriteria,
}: Props) {
  const [draft, setDraft] = useState(() => cloneFilterTree(root))

  useEffect(() => {
    if (open) setDraft(cloneFilterTree(root))
  }, [open, root])

  return (
    <Dialog open={open} onOpenChange={onOpenChange} className="max-w-3xl">
      <DialogContent className="gap-0 p-0">
        <DialogHeader>
          <DialogTitle>Custom criteria</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 px-5 py-4">
          <QueryOutputBar>
            {filterTreeIsEmpty(draft) ? (
              <span className="text-[var(--kg-text-muted)]">No conditions yet</span>
            ) : (
              <QueryPreview node={draft} fields={fields} />
            )}
          </QueryOutputBar>

          <QueryGroup fields={fields} group={draft} depth={0} onChange={setDraft} />
        </div>

        <DialogFooter className="justify-between sm:justify-between">
          <div>
            {onSaveCriteria ? (
              <Button type="button" variant="ghost" onClick={onSaveCriteria}>
                Save criteria…
              </Button>
            ) : null}
          </div>
          <div className="flex gap-2">
            <Button
              type="button"
              variant="ghost"
              onClick={() => {
                setDraft(emptyFilterTree())
              }}
            >
              Clear all
            </Button>
            <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button
              type="button"
              onClick={() => {
                onApply(draft)
                onOpenChange(false)
              }}
            >
              Apply filters
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
