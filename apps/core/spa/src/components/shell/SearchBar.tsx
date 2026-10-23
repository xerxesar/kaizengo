import { useEffect, useState } from 'react'
import { Input } from '@/components/ui/input'
import { searchQuery, type SearchHit } from '@/lib/search/client'

type Props = {
  collections?: string[]
  placeholder?: string
  debounceMs?: number
  onResults?: (hits: SearchHit[] | null) => void
  onQuery?: (q: string) => void
}

export function SearchBar({
  collections,
  placeholder = 'Search…',
  debounceMs = 300,
  onResults,
  onQuery,
}: Props) {
  const [q, setQ] = useState('')

  useEffect(() => {
    onQuery?.(q)
    if (!q.trim()) {
      onResults?.(null)
      return
    }
    const t = window.setTimeout(() => {
      void searchQuery(q, collections)
        .then((hits) => onResults?.(hits))
        .catch(() => onResults?.([]))
    }, debounceMs)
    return () => window.clearTimeout(t)
  }, [q, collections, debounceMs, onResults, onQuery])

  return (
    <Input
      value={q}
      placeholder={placeholder}
      onChange={(e) => setQ(e.target.value)}
      aria-label={placeholder}
    />
  )
}

export default SearchBar
