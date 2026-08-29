let recording = false

export function setKeymapRecording(active: boolean) {
  recording = active
  window.dispatchEvent(new CustomEvent('kaizengo:keymap-recording', { detail: { active } }))
}

export function isKeymapRecording(): boolean {
  return recording
}
