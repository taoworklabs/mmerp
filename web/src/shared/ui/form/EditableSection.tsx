import { Button, Group, Stack, Text } from '@mantine/core'
import { IconPencil, type Icon as TablerIcon } from '@tabler/icons-react'
import { useRef, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { icon } from '../theme'
import classes from './EditableSection.module.css'

type Props = {
  title: string
  icon?: TablerIcon
  // Shows the edit affordance; false while the user may not edit.
  editable?: boolean
  // Opens the group's fields, in a FormModal; without it the group only reads.
  onEdit?: () => void
  // The read view: label–value pairs.
  children: ReactNode
}

// EditableSection is one group of a record's values, read as label–value pairs and edited as a
// group. "Edit" is always shown, grey until the group is hovered or focused, which also tints it; a
// click anywhere in it (except on a button or link) asks to edit too.
export function EditableSection({ title, icon: Icon, editable = true, onEdit, children }: Props) {
  const { t } = useTranslation()
  const edit = editable ? onEdit : undefined
  const button = useRef<HTMLButtonElement>(null)
  return (
    <section
      className={classes.section}
      data-editable={edit ? true : undefined}
      aria-label={title}
      onClick={(e) => {
        if (!edit) return
        // Selecting text to copy it, or using a control inside, is not asking to edit.
        if ((e.target as HTMLElement).closest('button, a, input') || window.getSelection()?.toString()) return
        // Focus on Edit first, so closing the modal returns there.
        button.current?.focus()
        edit()
      }}
    >
      <Stack gap="sm">
        <Group justify="space-between" wrap="nowrap" gap="xs">
          <Group gap="xs" wrap="nowrap">
            {Icon && <Icon {...icon.text} aria-hidden />}
            <Text fw={600}>{title}</Text>
          </Group>
          {edit && (
            <Button ref={button} variant="subtle" className={classes.edit} leftSection={<IconPencil {...icon.button} />} onClick={edit} aria-label={t('shared.inline.edit', { label: title })}>
              {t('shared.section.edit')}
            </Button>
          )}
        </Group>
        {children}
      </Stack>
    </section>
  )
}
