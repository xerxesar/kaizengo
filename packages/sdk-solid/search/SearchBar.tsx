import { createSignal, type JSX } from 'solid-js'
import { Input } from '../ui/src/lib/Input'
import { searchQuery, type SearchHit } from './client'

type Props = {
  collections?: string[]
  placeholder?: string
  debounceMs?: number
  onResults?: (hits: SearchHit[] | null) => void
  onQuery?: (q: string) => void
}

export function SearchBar(props: Props): JSX.Element {
  const [value, setValue] = createSignal('')
  let timer: ReturnType<typeof setTimeout> | undefined

  async function runSearch(q: string) {
    props.onQuery?.(q)
    const trimmed = q.trim()
    if (!trimmed) {
      props.onResults?.(null)
      return
    }
    try {
      const hits = await searchQuery(trimmed, props.collections?.length ? props.collections : undefined)
      props.onResults?.(hits)
    } catch {
      props.onResults?.([])
    }
  }

  function handleInput(v: string) {
    setValue(v)
    clearTimeout(timer)
    timer = setTimeout(() => void runSearch(v), props.debounceMs ?? 300)
  }

  return (
    <Input
      value={value()}
      placeholder={props.placeholder ?? 'Search…'}
      onInput={handleInput}
    />
  )
}
