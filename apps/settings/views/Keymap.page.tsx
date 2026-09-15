import { useEffect, useState } from 'react'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import {
  clearAllKeymapOverrides,
  fetchKeymap,
  keymapLabelFor,
  setKeymapOverride,
  clearKeymapOverride,
  getKeymapOverrides,
  useFormatHotkey,
  setKeymapRecording,
  useSimpleHotkeyRecorder,
  type KeymapBinding,
  Card,
} from '@/lib'
import { KAppStatus } from '@/k'

type RowProps = {
  binding: KeymapBinding
  recordingId: string | null
  overrides: Record<string, string>
  onStart: (id: string) => void
  onApply: (id: string) => void
  onCancel: () => void
  onReset: (id: string) => void
  recorder: ReturnType<typeof useSimpleHotkeyRecorder>
  formatHotkey: (hotkey: string) => string
}

function KeymapBindingRow(props: RowProps) {
  const current = props.overrides[props.binding.id] ?? props.binding.keys
  const isRecording = props.recordingId === props.binding.id
  let liveDisplay = props.formatHotkey(current)
  if (isRecording) {
    const recorded = props.recorder.state.value
    if (recorded?.display) liveDisplay = recorded.display
    else if (recorded?.value) liveDisplay = props.formatHotkey(recorded.value)
    else liveDisplay = 'Press keys…'
  }

  return (
    <div className="flex flex-wrap items-center gap-3 border-b border-[var(--kg-border-subtle)] py-3 last:border-0">
      <div className="min-w-[12rem] flex-1">
        <div className="font-medium">{keymapLabelFor(props.binding)}</div>
        <div className="text-xs text-[var(--kg-text-muted)]">{props.binding.id}</div>
      </div>
      <kbd className="min-w-[6rem] rounded border border-[var(--kg-border)] px-2 py-1 text-center font-mono text-xs">
        {liveDisplay}
      </kbd>
      <div className="flex gap-2">
        {isRecording ? (
          <>
            <Button size="sm" onClick={() => props.onApply(props.binding.id)}>
              Save
            </Button>
            <Button size="sm" variant="secondary" onClick={props.onCancel}>
              Cancel
            </Button>
          </>
        ) : (
          <Button size="sm" variant="secondary" onClick={() => props.onStart(props.binding.id)}>
            Change
          </Button>
        )}
        {props.overrides[props.binding.id] ? (
          <Button size="sm" variant="ghost" onClick={() => props.onReset(props.binding.id)}>
            Reset
          </Button>
        ) : null}
      </div>
    </div>
  )
}

export default function KeymapSettings() {
  const [bindings, setBindings] = useState<KeymapBinding[]>([])
  const [overrides, setOverrides] = useState(getKeymapOverrides())
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [recordingId, setRecordingId] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  const formatHotkey = useFormatHotkey()
  const recorder = useSimpleHotkeyRecorder()

  async function load() {
    setLoading(true)
    setError('')
    try {
      const catalog = await fetchKeymap()
      setBindings(catalog.bindings)
      setOverrides(getKeymapOverrides())
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }

  function startRecord(id: string) {
    if (recorder.state.recording) recorder.cancel()
    setRecordingId(id)
    setSaved(false)
    setKeymapRecording(true)
    recorder.start()
  }

  function applyRecord(id: string) {
    recorder.stop()
    const recorded = recorder.state.value
    setRecordingId(null)
    setKeymapRecording(false)
    if (!recorded?.value) return
    setKeymapOverride(id, recorded.value)
    setOverrides(getKeymapOverrides())
    setSaved(true)
  }

  function cancelRecord() {
    recorder.cancel()
    setRecordingId(null)
    setKeymapRecording(false)
  }

  function resetOne(id: string) {
    clearKeymapOverride(id)
    setOverrides(getKeymapOverrides())
    setSaved(true)
  }

  function resetAll() {
    clearAllKeymapOverrides()
    setOverrides(getKeymapOverrides())
    setSaved(true)
  }

  useEffect(() => {
    void load()
    const onKeymap = () => setOverrides(getKeymapOverrides())
    window.addEventListener('kaizengo:keymap', onKeymap)
    return () => window.removeEventListener('kaizengo:keymap', onKeymap)
  }, [])

  return (
    <>
      {error ? (
        <Alert variant="danger">
          <div className="min-w-0 flex-1">{error}</div>
        </Alert>
      ) : null}
      {saved ? (
        <Alert variant="success">
          <div className="min-w-0 flex-1">Keyboard shortcuts updated.</div>
        </Alert>
      ) : null}

      {loading ? (
        <div className="flex items-center justify-center py-8" role="status" aria-label="Loading">
          <Progress indeterminate className="w-48 max-w-full" />
        </div>
      ) : (
        <Card title="Keyboard shortcuts">
          <p className="hint mb-4 text-sm text-[var(--kg-text-secondary)]">
            Hold <kbd>Alt</kbd> anywhere in the app to preview shortcut hints. Click Change, press a
            new combination, then Save.
          </p>
          <div className="flex flex-col gap-3">
            {bindings.map((binding) => (
              <KeymapBindingRow
                key={binding.id}
                binding={binding}
                recordingId={recordingId}
                overrides={overrides}
                onStart={startRecord}
                onApply={applyRecord}
                onCancel={cancelRecord}
                onReset={resetOne}
                recorder={recorder}
                formatHotkey={formatHotkey}
              />
            ))}
          </div>
          {Object.keys(overrides).length ? (
            <div className="mt-4">
              <Button variant="secondary" onClick={resetAll}>
                Reset all overrides
              </Button>
            </div>
          ) : null}
        </Card>
      )}

      <KAppStatus />
    </>
  )
}
