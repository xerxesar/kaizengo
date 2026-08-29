import { createEffect, type JSX } from 'solid-js'

type Props = {
  value: string
  current: unknown
  setValue: (value: string) => void
}

export function SyncValue(props: Props): null {
  createEffect(() => {
    if (props.value && String(props.current ?? '') !== props.value) {
      props.setValue(props.value)
    }
  })
  return null
}
