import { ActionIcon, Group, Stack, Text, UnstyledButton } from '@mmerp/ui'
import { TextInput, Tooltip } from '@mantine/core'
import { IconCheck, IconX } from '@tabler/icons-react'
import { useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { errorText } from '@/shared/i18n'
import classes from './InlineEditField.module.css'
import { icon } from './theme'

type Props = {
  label: string
  value: string
  // Saves the new value (the caller sends the record with its version); a rejection shows under the input.
  onSave: (value: string) => Promise<unknown>
  // A message (already translated) when the value cannot be saved, null when it can.
  validate?: (value: string) => string | null
  readOnly?: boolean
  maxLength?: number
}

// InlineEditField is one value of a record page that reads as text under its label and turns
// into an input when clicked (or Enter). Enter or ✓ saves, Esc or ✕ cancels; leaving the input
// keeps it open, so a half-typed value is neither saved nor lost by accident.
export function InlineEditField({ label, value, onSave, validate, readOnly, maxLength }: Props) {
  const { t } = useTranslation()
  const id = useId()
  const [draft, setDraft] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const shown = value || t('shared.inline.empty')

  const cancel = () => {
    setDraft(null)
    setError(null)
  }
  async function save() {
    if (draft === null || saving) return
    const next = draft.trim()
    if (next === value) return cancel()
    const invalid = validate?.(next) ?? null
    if (invalid) return setError(invalid)
    setSaving(true)
    try {
      await onSave(next)
      cancel()
    } catch (err) {
      setError(errorText(t, err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Stack gap="xxs">
      <Text render={<label htmlFor={id} />} size="xs" weight={600} tone="dimmed">
        {label}
      </Text>
      {draft === null ? (
        readOnly ? (
          <Text id={id} className={classes.view} tone={value ? undefined : 'dimmed'}>
            {shown}
          </Text>
        ) : (
          <UnstyledButton id={id} className={`${classes.view} ${classes.editable}`} onClick={() => setDraft(value)} aria-label={t('shared.inline.edit', { label })}>
            <Text tone={value ? undefined : 'dimmed'} truncate>
              {shown}
            </Text>
          </UnstyledButton>
        )
      ) : (
        <Group gap="xxs" wrap={false} align="start">
          <TextInput
            id={id}
            flex={1}
            value={draft}
            maxLength={maxLength}
            error={error}
            autoFocus
            onChange={(e) => setDraft(e.currentTarget.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault()
                void save()
              } else if (e.key === 'Escape') {
                e.stopPropagation()
                cancel()
              }
            }}
          />
          <Tooltip label={t('shared.inline.save')}>
            <ActionIcon variant="default" loading={saving} onClick={() => void save()} aria-label={t('shared.inline.save')}>
              <IconCheck {...icon.button} />
            </ActionIcon>
          </Tooltip>
          <Tooltip label={t('shared.inline.cancel')}>
            <ActionIcon variant="default" disabled={saving} onClick={cancel} aria-label={t('shared.inline.cancel')}>
              <IconX {...icon.button} />
            </ActionIcon>
          </Tooltip>
        </Group>
      )}
    </Stack>
  )
}
