import { Stack, Text } from '@mmerp/ui'
import { Notice } from '@/shared/ui/Notice'

// PasswordRules lists the broken password rules, one per line.
export function PasswordRules({ rules }: { rules: string[] }) {
  if (rules.length === 0) return null
  return (
    <Notice tone="danger">
      <Stack gap="xxs" role="list">
        {rules.map((r) => (
          <Text key={r} size="sm" role="listitem">
            {r}
          </Text>
        ))}
      </Stack>
    </Notice>
  )
}
