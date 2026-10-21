import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import { Group } from '../../primitives/layout/Group.tsx'
import { Text } from '../../primitives/typography/Text.tsx'
import styles from './FormLayout.module.css'

export type FormSectionLayout = 'side' | 'wide' | 'compact'

export interface FormSectionProps extends Omit<React.ComponentPropsWithRef<'fieldset'>, 'title'> {
  title: string
  description?: string
  icon?: React.ReactNode
  // The error count for the section, built by the caller: the kit only reserves the spot.
  badge?: React.ReactNode
  // 'side' (default) puts the title in a left column and the fields on the right from 80em;
  // 'wide'/'compact' keep the title on top at every width.
  layout?: FormSectionLayout
  children: React.ReactNode
}

function FormSectionRoot({ title, description, icon, badge, layout = 'side', children, className, ...props }: FormSectionProps) {
  return (
    <fieldset className={cx(styles.section, className)} {...props}>
      <div className={styles.layout} data-layout={layout}>
        <div className={styles.titleColumn}>
          <Group gap="xs" wrap={false}>
            {icon}
            <legend className={styles.legend}>{title}</legend>
            {badge}
          </Group>
          {/* Quieter than the field labels, so the fields lead. */}
          {description && (
            <Text size="xs" tone="dimmed" className={styles.description}>
              {description}
            </Text>
          )}
        </div>
        <div className={styles.fields} data-layout={layout}>
          {children}
        </div>
      </div>
    </fieldset>
  )
}

export type FormSectionShortProps = React.ComponentPropsWithRef<'div'>

// Short takes a third of a "side" row instead of half: a date, a choice, a phone number.
function Short({ className, ...props }: FormSectionShortProps) {
  return <div className={cx(styles.short, className)} {...props} />
}

export type FormSectionRowProps = React.ComponentPropsWithRef<'div'>

// Row spans the whole row of a FormSection, for what is not one field (a table, totals).
function Row({ className, ...props }: FormSectionRowProps) {
  return <div className={cx(styles.fullRow, className)} {...props} />
}

export type FormSectionInputRowProps = React.ComponentPropsWithRef<'div'>

// InputRow puts controls (a checkbox, a remove button) on one line level with the inputs beside
// it in a FormSection, rather than with their labels.
function InputRow({ className, ...props }: FormSectionInputRowProps) {
  return <Group justify="space-between" wrap={false} className={cx(styles.inputRow, className)} {...props} />
}

export const FormSection = Object.assign(FormSectionRoot, { Short, Row, InputRow })

export interface FormFieldsProps extends React.ComponentPropsWithRef<'div'> {
  children: React.ReactNode
}

// FormFields lays out fields without a section's title: two to a row from 48em, for a group of a
// record's values edited in place in its card.
export function FormFields({ children, className, ...props }: FormFieldsProps) {
  return (
    <div className={cx(styles.pairFields, className)} {...props}>
      {children}
    </div>
  )
}
