import { ActionIcon, Button, Group, Input, Text, TextInput, Tooltip, VisuallyHidden } from '@mantine/core'
import { IconEye, IconEyeOff } from '@tabler/icons-react'
import { useState } from 'react'
import { useController, useFormContext } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { errorText } from '@/shared/i18n'
import { icon } from '../theme'
import { fieldId } from './Form'

type Props = {
  name: string
  label: string
  // The record holds a value; the value itself is fetched only on "Show".
  present: boolean
  canView: boolean
  canEdit: boolean
  // Each call is audited by the backend.
  reveal: () => Promise<string | null>
}

// SensitiveField is masked until the user asks to see it. Without the permission it
// says so instead of showing anything. The form value stays untouched (undefined)
// unless the user types, so only edited sensitive fields are sent.
export function SensitiveField({ name, label, present, canView, canEdit, reveal }: Props) {
  const { t } = useTranslation()
  const { control } = useFormContext()
  const { field, fieldState } = useController({ name, control })
  const [shown, setShown] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!canView) {
    return (
      <Input.Wrapper label={label} id={fieldId(name)}>
        <Text c="dimmed">{t('shared.sensitive.no_permission')}</Text>
      </Input.Wrapper>
    )
  }

  if (present && shown === null && field.value === undefined) {
    const show = async () => {
      setBusy(true)
      setError(null)
      try {
        setShown((await reveal()) ?? '')
      } catch (err) {
        setError(errorText(t, err))
      } finally {
        setBusy(false)
      }
    }
    return (
      <Input.Wrapper label={label} id={fieldId(name)} error={error}>
        <Group gap="sm" wrap="nowrap">
          <Text aria-hidden>••••••</Text>
          <VisuallyHidden>{t('shared.sensitive.masked')}</VisuallyHidden>
          <Button size="xs" variant="subtle" leftSection={<IconEye {...icon.text} />} loading={busy} onClick={() => void show()} aria-label={t('shared.sensitive.show_field', { field: label })}>
            {t('shared.sensitive.show')}
          </Button>
        </Group>
      </Input.Wrapper>
    )
  }

  if (!canEdit && !present) {
    return (
      <Input.Wrapper label={label} id={fieldId(name)}>
        <Text c="dimmed">—</Text>
      </Input.Wrapper>
    )
  }

  return (
    <TextInput
      id={fieldId(name)}
      ref={field.ref}
      label={label}
      readOnly={!canEdit}
      value={field.value ?? shown ?? ''}
      onChange={(e) => field.onChange(e.currentTarget.value)}
      onBlur={field.onBlur}
      error={fieldState.error?.message}
      autoComplete="off"
      // Shown but not edited: it can be masked again. Showing it again is another audited read.
      rightSection={
        shown !== null && field.value === undefined ? (
          <Tooltip label={t('shared.sensitive.hide')}>
            {/* Inside the input, so smaller than the input itself. */}
            <ActionIcon variant="subtle" color="gray" size="md" onClick={() => setShown(null)} aria-label={t('shared.sensitive.hide_field', { field: label })}>
              <IconEyeOff {...icon.button} />
            </ActionIcon>
          </Tooltip>
        ) : undefined
      }
    />
  )
}
