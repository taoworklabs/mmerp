import { Anchor, Text } from '@mmerp/ui'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'

// MissingValue stands for an empty value: a link to fill it in for who may (onAdd opens the form,
// to goes where it is entered), otherwise a quiet "not filled in".
export function MissingValue({ onAdd, to, label }: { onAdd?: () => void; to?: string; label: string }) {
  const { t } = useTranslation()
  const text = t('shared.field.add')
  const name = t('shared.field.add_label', { label })
  if (to)
    return (
      <Anchor render={<Link to={to} />} size="sm" aria-label={name}>
        {text}
      </Anchor>
    )
  if (onAdd)
    return (
      <Anchor render={<button />} type="button" size="sm" onClick={onAdd} aria-label={name}>
        {text}
      </Anchor>
    )
  return (
    <Text render={<span />} size="sm" tone="dimmed">
      {t('shared.field.missing')}
    </Text>
  )
}
