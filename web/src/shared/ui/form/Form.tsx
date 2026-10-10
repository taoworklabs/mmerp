import { Alert, Anchor, Badge, Box, Button, Grid, Group, SimpleGrid, Stack, Text } from '@mantine/core'
import { IconAlertCircle, type Icon as TablerIcon } from '@tabler/icons-react'
import { useEffect, useRef, useState, type ReactNode } from 'react'
import { FormProvider, useFormContext, useFormState, type FieldErrors, type FieldValues, type Path, type UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { useBlocker } from 'react-router'
import { ApiError } from '@/shared/api/error'
import { errorText } from '@/shared/i18n'
import { useConfirm } from '../confirm'
import { icon } from '../theme'
import classes from './Form.module.css'

export const fieldId = (name: string) => `field-${name.replaceAll('.', '-')}`

// errorNames lists the dotted names of every field with an error, lines of a list included.
function errorNames(errors: FieldErrors, prefix = ''): string[] {
  return Object.entries(errors).flatMap(([key, e]) => {
    if (!e || typeof e !== 'object') return []
    const name = prefix + key
    return typeof e.message === 'string' ? [name] : errorNames(e as FieldErrors, `${name}.`)
  })
}

// invalidInputs are the inputs of the fields with an error, in reading order.
function invalidInputs(errors: FieldErrors): HTMLElement[] {
  return errorNames(errors)
    .map((n) => document.getElementById(fieldId(n)))
    .filter((el): el is HTMLElement => el !== null)
    .sort((a, b) => (a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING ? -1 : 1))
}

// focusInvalid focuses the first invalid input after `after` (from the top when null, wrapping
// round at the end) and brings it to the middle of the screen. It returns the focused input.
function focusInvalid(errors: FieldErrors, after: HTMLElement | null): HTMLElement | undefined {
  const inputs = invalidInputs(errors)
  const next = (after && inputs.find((el) => after.compareDocumentPosition(el) & Node.DOCUMENT_POSITION_FOLLOWING)) || inputs[0]
  next?.focus({ preventScroll: true })
  next?.scrollIntoView({ block: 'center' })
  return next
}

type Props<T extends FieldValues> = {
  form: UseFormReturn<T>
  onSubmit: (values: T) => Promise<unknown>
  // Maps an API error to the field it belongs to; other errors show as a banner.
  fieldOf?: (err: ApiError) => Path<T> | undefined
  id?: string
  children: ReactNode
}

// Form wires a React Hook Form to the rules every form follows: a failed submit focuses the
// first invalid field, API errors at their field, Ctrl+S to save, and a confirmation before
// leaving with unsaved changes.
export function Form<T extends FieldValues>({ form, onSubmit, fieldOf, id, children }: Props<T>) {
  const { t } = useTranslation()
  const [banner, setBanner] = useState<string | null>(null)
  const ref = useRef<HTMLFormElement>(null)
  // dirtyFields, not isDirty: a field registered without a default value (an untouched
  // SensitiveField) makes isDirty true although nothing was edited.
  const { dirtyFields, isSubmitting } = form.formState

  const submit = form.handleSubmit(
    async (values) => {
      setBanner(null)
      try {
        await onSubmit(values)
        form.reset(values)
      } catch (err) {
        const field = err instanceof ApiError ? fieldOf?.(err) : undefined
        if (field) form.setError(field, { message: errorText(t, err) }, { shouldFocus: true })
        else setBanner(errorText(t, err))
      }
    },
    // After React Hook Form's own focus, which follows registration order rather than the page.
    (errors) => void requestAnimationFrame(() => focusInvalid(errors, null)),
  )

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 's' && ref.current?.contains(document.activeElement)) {
        e.preventDefault()
        void submit()
      }
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [submit])

  const dirty = Object.keys(dirtyFields).length > 0 && !isSubmitting
  const blocker = useBlocker(({ currentLocation, nextLocation }) => dirty && currentLocation.pathname + currentLocation.search !== nextLocation.pathname + nextLocation.search)
  const [ask, dialog] = useConfirm()
  useEffect(() => {
    if (blocker.state !== 'blocked') return
    void ask({ title: t('shared.form.leave_title'), message: t('shared.form.leave_unsaved'), confirmLabel: t('shared.form.leave'), danger: true }).then((ok) =>
      ok ? blocker.proceed() : blocker.reset(),
    )
  }, [blocker, ask, t])
  useEffect(() => {
    if (!dirty) return
    const onUnload = (e: BeforeUnloadEvent) => e.preventDefault()
    window.addEventListener('beforeunload', onUnload)
    return () => window.removeEventListener('beforeunload', onUnload)
  }, [dirty])

  return (
    <FormProvider {...form}>
      <form ref={ref} id={id} onSubmit={submit} noValidate>
        <Stack gap="md">
          {banner && (
            <Alert color="danger" icon={<IconAlertCircle {...icon.button} />} role="alert">
              {banner}
            </Alert>
          )}
          {children}
        </Stack>
      </form>
      {dialog}
    </FormProvider>
  )
}

// FormSection groups fields under a title. From 1280px the title and description sit in a
// left column and short fields two per row on the right; below that, everything is one column.
// After a failed submit its title carries the number of its fields still to fix.
export function FormSection({ title, description, icon: Icon, children }: { title: string; description?: string; icon?: TablerIcon; children: ReactNode }) {
  const { t } = useTranslation()
  const { errors, submitCount } = useFormState()
  const ref = useRef<HTMLFieldSetElement>(null)
  const [count, setCount] = useState(0)
  // Which fields are in this section is known only from the page, so count after rendering.
  useEffect(() => setCount(submitCount > 0 ? invalidInputs(errors).filter((el) => ref.current?.contains(el)).length : 0))
  return (
    <Box component="fieldset" className={classes.section} ref={ref}>
      <Grid gap={{ base: 'sm', lg: 'xl' }}>
        <Grid.Col span={{ base: 12, lg: 4 }}>
          <Group gap="xs" wrap="nowrap">
            {Icon && <Icon {...icon.text} aria-hidden />}
            <Text component="legend" fw={600}>
              {title}
            </Text>
            {count > 0 && (
              <Badge color="danger" size="sm" circle aria-label={t('shared.form.section_invalid', { count })}>
                {count}
              </Badge>
            )}
          </Group>
          {description && (
            <Text size="sm" c="dimmed" mt="xxs">
              {description}
            </Text>
          )}
        </Grid.Col>
        <Grid.Col span={{ base: 12, lg: 8 }}>
          <SimpleGrid cols={{ base: 1, lg: 2 }} spacing="md" verticalSpacing="md">
            {children}
          </SimpleGrid>
        </Grid.Col>
      </Grid>
    </Box>
  )
}

// FormRow spans the whole row of a FormSection, for what is not one field (a table, totals).
export function FormRow({ children }: { children: ReactNode }) {
  return <div className={classes.fullRow}>{children}</div>
}

// InputRow puts controls (a checkbox, a remove button) on one line level with the inputs beside
// it in a FormSection, rather than with their labels.
export function InputRow({ children }: { children: ReactNode }) {
  return (
    <Group justify="space-between" wrap="nowrap" className={classes.inputRow}>
      {children}
    </Group>
  )
}

// FormActions is a form's footer: its buttons on the right. The submit button shows its busy
// state only after 300ms, so quick saves do not flicker. After a failed submit a line beside
// the buttons counts the fields still to fix and steps through them. On a page (no onCancel,
// which only modals pass) the footer shows only once something is edited or a submit failed;
// it sticks to the bottom of the screen, says so and offers to discard the changes.
export function FormActions({ submitLabel, onCancel, cancelLabel }: { submitLabel: string; onCancel?: () => void; cancelLabel?: string }) {
  const { t } = useTranslation()
  const { formState, reset } = useFormContext()
  const { errors, submitCount, dirtyFields } = useFormState()
  const [slow, setSlow] = useState(false)
  const last = useRef<HTMLElement | null>(null)
  const [ask, dialog] = useConfirm()
  const invalid = submitCount > 0 ? errorNames(errors).length : 0
  const page = !onCancel
  const dirty = page && Object.keys(dirtyFields).length > 0
  useEffect(() => {
    if (!formState.isSubmitting) return setSlow(false)
    const timer = setTimeout(() => setSlow(true), 300)
    return () => clearTimeout(timer)
  }, [formState.isSubmitting])
  async function discard() {
    if (await ask({ title: t('shared.form.discard_title'), message: t('shared.form.discard_message'), confirmLabel: t('shared.form.discard'), danger: true })) reset()
  }
  const buttons = (
    <Group justify="flex-end">
      {invalid > 0 && (
        <Group gap="xxs" role="alert" wrap="nowrap" c="danger">
          <IconAlertCircle {...icon.text} aria-hidden />
          <Text size="sm">
            {t('shared.form.invalid', { count: invalid })} ·
          </Text>
          <Anchor component="button" type="button" size="sm" onClick={() => (last.current = focusInvalid(errors, last.current) ?? null)}>
            {t('shared.form.next_invalid')}
          </Anchor>
        </Group>
      )}
      {dirty && (
        <Button variant="subtle" color="danger" onClick={() => void discard()}>
          {t('shared.form.discard')}
        </Button>
      )}
      {onCancel && (
        <Button variant="default" onClick={onCancel}>
          {cancelLabel}
        </Button>
      )}
      <Button type="submit" loading={slow} disabled={formState.isSubmitting}>
        {submitLabel}
      </Button>
    </Group>
  )
  if (!page) return buttons
  if (!dirty && invalid === 0) return null
  return (
    <Group justify="space-between" wrap="nowrap" className={classes.bar} data-dirty={dirty || undefined}>
      {dirty ? (
        <Group gap="xs" wrap="nowrap" role="status">
          <span className={classes.dot} aria-hidden />
          <Text size="sm">{t('shared.form.unsaved')}</Text>
        </Group>
      ) : (
        <span />
      )}
      {buttons}
      {dialog}
    </Group>
  )
}
