import { createHotkeyStore } from '@zag-js/hotkeys'

/** Shared hotkey store — must be module-scoped (not recreated per render). */
export const keymapHotkeyStore = createHotkeyStore({
  conflictBehavior: 'replace',
  defaultOptions: {
    preventDefault: true,
    enableOnFormTags: false,
  },
})
