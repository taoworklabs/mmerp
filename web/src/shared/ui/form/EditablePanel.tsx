import { Button, Group, Panel } from '@mmerp/ui'
import { IconPencil, type Icon as TablerIcon } from '@tabler/icons-react'
import { useRef, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { icon } from '../theme'

type Props = {
  title: string
  icon?: TablerIcon
  // Shows Edit; false while the user may not edit.
  editable?: boolean
  // Before Edit on the title line: what else the card offers (show all its masked values).
  actions?: ReactNode
  // The group's form, given how to close it (Cancel, or after saving): fields and FormActions with onCancel.
  edit: (close: () => void) => ReactNode
  // The read view: label–value pairs; as a function, given how to open the form (an empty value's "+ Add").
  children: ReactNode | ((open: () => void) => ReactNode)
}

// EditablePanel is one group of a record's values in a card: read as label–value pairs, and on
// Edit the same card holds the group's form, with Save and Cancel, until it is saved or cancelled.
export function EditablePanel({ title, icon: Icon, editable, actions, edit, children }: Props) {
  const { t } = useTranslation()
  const [editing, setEditing] = useState(false)
  const button = useRef<HTMLButtonElement>(null)
  const body = useRef<HTMLDivElement>(null)
  const open = () => {
    setEditing(true)
    // Into the first field, so the keyboard carries on where the eye went.
    requestAnimationFrame(() => body.current?.querySelector<HTMLElement>('input:not([type=hidden]), textarea, [role=combobox]')?.focus())
  }
  const close = () => {
    setEditing(false)
    requestAnimationFrame(() => button.current?.focus())
  }
  return (
    <Panel
      title={title}
      icon={Icon && <Icon {...icon.text} aria-hidden />}
      actions={
        !editing && (
          <Group gap="xs" wrap={false}>
            {actions}
            {editable && (
              <Button ref={button} variant="subtle" leftSection={<IconPencil {...icon.button} />} onClick={open} aria-label={t('shared.inline.edit', { label: title })}>
                {t('shared.section.edit')}
              </Button>
            )}
          </Group>
        )
      }
    >
      <div ref={body}>{editing ? edit(close) : typeof children === 'function' ? children(open) : children}</div>
    </Panel>
  )
}
