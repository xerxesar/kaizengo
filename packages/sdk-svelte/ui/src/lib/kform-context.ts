import { getContext } from 'svelte'
import type { KFormContext } from './kform-types'

export const KFORM_CTX = Symbol('kg-kform')

const fallback: KFormContext = {
  model: '',
  fields: [],
  draft: {},
  saving: false,
  canSubmit: false,
  setValue() {},
  submit() {},
}

/** Form state inside `<KForm>`; use `{@const ctx = getKFormContext()}` when child markup is implicit. */
export function getKFormContext(): KFormContext {
  return getContext<KFormContext>(KFORM_CTX) ?? fallback
}
