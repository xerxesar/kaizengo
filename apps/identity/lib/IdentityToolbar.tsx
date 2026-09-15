import { useIdentityState, selectOrg } from './state'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

export function IdentityToolbar() {
  const identity = useIdentityState()
  const options = identity.orgs.map((o) => ({ value: o.id, label: o.name }))
  const selectedId = identity.selectedOrg?.id

  if (identity.orgs.length <= 1) {
    if (!identity.selectedOrg) return null
    return (
      <div className="mb-5 flex justify-end">
        <span className="inline-flex items-center bg-[var(--kg-primary-subtle)] px-5 py-2 text-sm font-semibold text-[var(--kg-primary)]">
          {identity.selectedOrg.name}
        </span>
      </div>
    )
  }

  return (
    <div className="mb-5 flex justify-end">
      <Select value={selectedId} onValueChange={(v) => selectOrg(v)}>
        <SelectTrigger className="w-auto min-w-[12rem]" aria-label="Organization">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {options.map((opt) => (
            <SelectItem key={opt.value} value={opt.value}>
              {opt.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}
