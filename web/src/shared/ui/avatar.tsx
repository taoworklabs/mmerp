import { Avatar, Group, Text, type AvatarProps } from '@mantine/core'

// Accent colours for avatars: decorative, never carrying meaning.
const colors = ['blue', 'teal', 'violet', 'orange', 'pink', 'cyan', 'grape', 'indigo']

// initials takes the first letters of the first and last words: "Phạm Bảo Hà" → "PH".
export function initials(name: string): string {
  const words = name.trim().split(/\s+/).filter(Boolean)
  const first = words[0]?.[0] ?? ''
  const last = words.length > 1 ? (words.at(-1)?.[0] ?? '') : ''
  return (first + last).toUpperCase()
}

// PersonAvatar shows a person's initials on a colour derived from the name, so it stays stable.
export function PersonAvatar({ name, size = 'sm' }: { name: string; size?: AvatarProps['size'] }) {
  let h = 0
  for (const ch of name) h = (h * 31 + ch.codePointAt(0)!) >>> 0
  return (
    <Avatar size={size} color={colors[h % colors.length]} radius={1000} aria-hidden>
      {initials(name)}
    </Avatar>
  )
}

// PersonName is an avatar and a name on one line, for table cells and lists.
export function PersonName({ name }: { name: string }) {
  return (
    <Group gap="xs" wrap="nowrap" component="span">
      <PersonAvatar name={name} size={24} />
      <Text component="span" inherit truncate>
        {name}
      </Text>
    </Group>
  )
}
