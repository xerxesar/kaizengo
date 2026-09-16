import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  initialName?: string
  initialShared?: boolean
  initialDefault?: boolean
  initialPredefined?: boolean
  pageSize?: number
  onSave: (opts: {
    name: string
    shared: boolean
    isDefault: boolean
    predefined: boolean
  }) => void | Promise<void>
}

export function KSaveCriteriaDialog({
  open,
  onOpenChange,
  initialName = '',
  initialShared = false,
  initialDefault = false,
  initialPredefined = false,
  pageSize,
  onSave,
}: Props) {
  const [name, setName] = useState(initialName)
  const [shared, setShared] = useState(initialShared)
  const [isDefault, setIsDefault] = useState(initialDefault)
  const [predefined, setPredefined] = useState(initialPredefined)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!open) return
    setName(initialName)
    setShared(initialShared)
    setIsDefault(initialDefault)
    setPredefined(initialPredefined)
  }, [open, initialName, initialShared, initialDefault, initialPredefined])

  async function submit() {
    const trimmed = name.trim()
    if (!trimmed) return
    setSaving(true)
    try {
      await onSave({ name: trimmed, shared, isDefault, predefined })
      onOpenChange(false)
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="gap-0 p-0">
        <DialogHeader>
          <DialogTitle>Save criteria</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 px-5 py-4">
          <div className="space-y-2">
            <Label htmlFor="criteria-name">Name</Label>
            <Input
              id="criteria-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="My filter…"
              onKeyDown={(e) => {
                if (e.key === 'Enter') void submit()
              }}
            />
          </div>
          <div className="space-y-3">
            <label className="flex cursor-pointer items-start gap-2 text-sm">
              <Checkbox checked={shared} onCheckedChange={(v) => setShared(v === true)} />
              <span>
                <span className="font-medium">Share with others</span>
                <span className="mt-0.5 block text-xs text-[var(--kg-text-muted)]">
                  Visible to everyone in your organization
                </span>
              </span>
            </label>
            <label className="flex cursor-pointer items-start gap-2 text-sm">
              <Checkbox checked={isDefault} onCheckedChange={(v) => setIsDefault(v === true)} />
              <span>
                <span className="font-medium">Set as default</span>
                <span className="mt-0.5 block text-xs text-[var(--kg-text-muted)]">
                  Apply automatically when you open this list
                  {pageSize ? ` (page size ${pageSize} included)` : ''}
                </span>
              </span>
            </label>
            <label className="flex cursor-pointer items-start gap-2 text-sm">
              <Checkbox checked={predefined} onCheckedChange={(v) => setPredefined(v === true)} />
              <span>
                <span className="font-medium">Show in Filters menu</span>
                <span className="mt-0.5 block text-xs text-[var(--kg-text-muted)]">
                  Appears as a quick filter preset alongside built-in filters
                </span>
              </span>
            </label>
          </div>
        </div>
        <DialogFooter>
          <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="button" disabled={!name.trim() || saving} onClick={() => void submit()}>
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
