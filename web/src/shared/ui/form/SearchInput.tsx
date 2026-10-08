import { TextInput } from '@mantine/core'
import { IconSearch } from '@tabler/icons-react'
import { useEffect, useRef, useState } from 'react'
import { icon } from '../theme'

// SearchInput is a list's search box. It applies on Enter or when leaving the box, so typing
// does not flood history; "/" focuses it from anywhere on the page.
export function SearchInput({ label, placeholder, value, onSearch }: { label: string; placeholder?: string; value: string; onSearch: (v: string) => void }) {
  const [draft, setDraft] = useState(value)
  const ref = useRef<HTMLInputElement>(null)
  useEffect(() => setDraft(value), [value])
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
  const apply = () => draft.trim() !== value && onSearch(draft.trim())
  return (
    <TextInput
      ref={ref}
      type="search"
      aria-label={label}
      placeholder={placeholder ?? label}
      w={{ base: '100%', md: 280 }}
      leftSection={<IconSearch {...icon.text} />}
      value={draft}
      onChange={(e) => setDraft(e.currentTarget.value)}
      onKeyDown={(e) => e.key === 'Enter' && apply()}
      onBlur={apply}
    />
  )
}
