import { FilterBar } from '@mmerp/ui'
import { useEffect, useRef, useState } from 'react'

// SearchInput is a list's search box. It applies on Enter or when leaving the box, so typing
// does not flood history; "/" focuses it from anywhere on the page.
export function SearchInput({ label, placeholder, value, onSearch }: { label: string; placeholder?: string; value: string; onSearch: (v: string) => void }) {
  const [draft, setDraft] = useState(value)
  const ref = useRef<HTMLInputElement>(null)
  // What was last sent: the URL re-renders in a transition, so a blur right after Enter would
  // still see the old value and push the same search twice.
  const sent = useRef(value)
  useEffect(() => {
    setDraft(value)
    sent.current = value
  }, [value])
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement
      if (e.key === '/' && !target.closest('input,textarea,[contenteditable]')) {
        e.preventDefault()
        ref.current?.focus()
      }
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [])
  const apply = () => {
    const v = draft.trim()
    if (v === sent.current) return
    sent.current = v
    onSearch(v)
  }
  return (
    <FilterBar.Search
      ref={ref}
      aria-label={label}
      placeholder={placeholder ?? label}
      value={draft}
      onChange={(e) => setDraft(e.currentTarget.value)}
      onKeyDown={(e) => e.key === 'Enter' && apply()}
      onBlur={apply}
    />
  )
}
