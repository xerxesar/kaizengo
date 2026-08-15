<script lang="ts">
  import SearchInput from '../ui/src/lib/SearchInput.svelte'
  import { searchQuery, type SearchHit } from './client'

  type Props = {
    collections?: string[]
    placeholder?: string
    debounceMs?: number
    onResults?: (hits: SearchHit[] | null) => void
    onQuery?: (q: string) => void
  }

  let {
    collections = [],
    placeholder = 'Search…',
    debounceMs = 300,
    onResults,
    onQuery,
  }: Props = $props()

  let value = $state('')
  let timer: ReturnType<typeof setTimeout> | undefined

  async function runSearch(q: string) {
    onQuery?.(q)
    const trimmed = q.trim()
    if (!trimmed) {
      onResults?.(null)
      return
    }
    try {
      const hits = await searchQuery(trimmed, collections.length ? collections : undefined)
      onResults?.(hits)
    } catch {
      onResults?.([])
    }
  }

  function handleInput(v: string) {
    value = v
    clearTimeout(timer)
    timer = setTimeout(() => void runSearch(v), debounceMs)
  }
</script>

<SearchInput bind:value {placeholder} oninput={handleInput} />
