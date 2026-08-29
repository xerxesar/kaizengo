import { createSignal, For, onMount, Show } from 'solid-js'
import {
  Alert,
  Button,
  Card,
  KAppStatus,
  Spinner,
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
} from '@kaizengo/sdk-solid/ui'

type RowProps = {
  binding: KeymapBinding
  recordingId: () => string | null
  overrides: () => Record<string, string>
  onStart: (id: string) => void
  onApply: (id: string) => void
  onCancel: () => void
  onReset: (id: string) => void
  recorder: ReturnType<typeof useSimpleHotkeyRecorder>
  formatHotkey: (hotkey: string) => string
}

function KeymapBindingRow(props: RowProps) {
  const current = () => props.overrides()[props.binding.id] ?? props.binding.keys
  const isRecording = () => props.recordingId() === props.binding.id
  const liveDisplay = () => {
    if (!isRecording()) return props.formatHotkey(current())
    const recorded = props.recorder.state().value
    if (recorded?.display) return recorded.display
    if (recorded?.value) return props.formatHotkey(recorded.value)
    return 'Press keys…'
  }

  return (
    <div class="flex flex-wrap items-center gap-3 border-b border-[var(--kg-border-subtle)] py-3 last:border-0">
      <div class="min-w-[12rem] flex-1">
        <div class="font-medium">{keymapLabelFor(props.binding)}</div>
        <div class="text-xs text-[var(--kg-text-muted)]">{props.binding.id}</div>
      </div>
      <kbd class="min-w-[6rem] rounded border border-[var(--kg-border)] px-2 py-1 text-center font-mono text-xs">
        {liveDisplay()}
      </kbd>
      <div class="flex gap-2">
        <Show
          when={isRecording()}
          fallback={
            <Button size="sm" variant="secondary" onClick={() => props.onStart(props.binding.id)}>
              Change
            </Button>
          }
        >
          <Button size="sm" onClick={() => props.onApply(props.binding.id)}>
            Save
          </Button>
          <Button size="sm" variant="secondary" onClick={props.onCancel}>
            Cancel
          </Button>
        </Show>
        <Show when={props.overrides()[props.binding.id]}>
          <Button size="sm" variant="ghost" onClick={() => props.onReset(props.binding.id)}>
            Reset
          </Button>
        </Show>
      </div>
    </div>
  )
}

export default function KeymapSettings() {
  const [bindings, setBindings] = createSignal<KeymapBinding[]>([])
  const [overrides, setOverrides] = createSignal(getKeymapOverrides())
  const [loading, setLoading] = createSignal(true)
  const [error, setError] = createSignal('')
  const [recordingId, setRecordingId] = createSignal<string | null>(null)
  const [saved, setSaved] = createSignal(false)

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
    if (recorder.state().recording) recorder.cancel()
    setRecordingId(id)
    setSaved(false)
    setKeymapRecording(true)
    recorder.start()
  }

  function applyRecord(id: string) {
    recorder.stop()
    const recorded = recorder.state().value
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

  onMount(() => {
    void load()
    const onKeymap = () => setOverrides(getKeymapOverrides())
    window.addEventListener('kaizengo:keymap', onKeymap)
    return () => window.removeEventListener('kaizengo:keymap', onKeymap)
  })

  return (
    <>
      <Show when={error()}>
        <Alert variant="danger">{error()}</Alert>
      </Show>
      <Show when={saved()}>
        <Alert variant="success">Keyboard shortcuts updated.</Alert>
      </Show>

      <Show when={!loading()} fallback={<Spinner />}>
        <Card title="Keyboard shortcuts">
          <p class="hint mb-4 text-sm text-[var(--kg-text-secondary)]">
            Hold <kbd>Alt</kbd> anywhere in the app to preview shortcut hints. Click Change, press a new
            combination, then Save.
          </p>
          <div class="flex flex-col gap-3">
            <For each={bindings()}>
              {(binding) => (
                <KeymapBindingRow
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
              )}
            </For>
          </div>
          <Show when={Object.keys(overrides()).length}>
            <div class="mt-4">
              <Button variant="secondary" onClick={resetAll}>
                Reset all overrides
              </Button>
            </div>
          </Show>
        </Card>
      </Show>

      <KAppStatus />
    </>
  )
}
