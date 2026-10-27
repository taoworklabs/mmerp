import { Avatar, Group, Text, type DecorativeColor } from '@mmerp/ui'

// Accent colours for avatars: decorative, never carrying meaning.
const colors: DecorativeColor[] = ['blue', 'teal', 'violet', 'orange', 'pink', 'cyan', 'grape', 'indigo']

const sizes = { sm: 26, md: 38, lg: 56 }

// initials takes the first letters of the first and last words: "Phạm Bảo Hà" → "PH".
export function initials(name: string): string {
  const words = name.trim().split(/\s+/).filter(Boolean)
  const first = words[0]?.[0] ?? ''
  const last = words.length > 1 ? (words.at(-1)?.[0] ?? '') : ''
  return (first + last).toUpperCase()
}

// PersonAvatar shows a person's initials on a colour derived from the name, so it stays stable.
export function PersonAvatar({ name, size = 'sm' }: { name: string; size?: keyof typeof sizes | number }) {
  let h = 0
  for (const ch of name) h = (h * 31 + ch.codePointAt(0)!) >>> 0
  return (
    <Avatar size={typeof size === 'number' ? size : sizes[size]} color={colors[h % colors.length]} alt={name} aria-hidden>
      {initials(name)}
    </Avatar>
  )
}

// PersonName is an avatar and a name on one line, for table cells and lists.
export function PersonName({ name }: { name: string }) {
  return (
    <Group gap="xs" wrap={false} render={<span />}>
      <PersonAvatar name={name} size={24} />
      <Text render={<span />} size="inherit" truncate>
        {name}
      </Text>
    </Group>
  )
}
