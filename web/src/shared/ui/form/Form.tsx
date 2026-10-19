import { Anchor, Badge, Box, Button, Grid, Group, Stack, Text } from '@mantine/core'
import { IconAlertCircle, type Icon as TablerIcon } from '@tabler/icons-react'
import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from 'react'
import { FormProvider, useFormContext, useFormState, type FieldErrors, type FieldValues, type UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Link, useBlocker } from 'react-router'
import { ApiError } from '@/shared/api/error'
import { errorText } from '@/shared/i18n'
import { useConfirm } from '../confirm'
import { ModalFooter } from '../ModalFooter'
import { Notice } from '../Notice'
import { icon } from '../theme'
import { fieldOfError, type ErrorFields } from './errorFields'
import classes from './Form.module.css'

export const fieldId = (name: string) => `field-${name.replaceAll('.', '-')}`

// Unsaved tells a page whether a form inside it holds edits not yet saved, so the page's own
// actions (Submit for approval) can ask to save first instead of dropping them.
type Unsaved = { dirty: boolean; set: (dirty: boolean) => void }
const UnsavedContext = createContext<Unsaved | null>(null)

export function UnsavedProvider({ children }: { children: ReactNode }) {
  const [dirty, set] = useState(false)
  return <UnsavedContext.Provider value={{ dirty, set }}>{children}</UnsavedContext.Provider>
}

// useUnsaved is true while a form of the page has unsaved edits; false outside a provider.
export function useUnsaved() {
  return useContext(UnsavedContext)?.dirty ?? false
}

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
  // API error codes shown at a field; other errors show as a banner.
  errorFields?: ErrorFields<T>
  id?: string
  children: ReactNode
}

// Form wires a React Hook Form to the rules every form follows: a failed submit focuses the
// first invalid field, API errors at their field, Ctrl+S to save, and a confirmation before
// leaving with unsaved changes.
export function Form<T extends FieldValues>({ form, onSubmit, errorFields, id, children }: Props<T>) {
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
        const field = err instanceof ApiError ? fieldOfError(errorFields, err) : undefined
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
  const report = useContext(UnsavedContext)?.set
  useEffect(() => {
    report?.(dirty)
    return () => report?.(false)
  }, [report, dirty])
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
            <Notice tone="danger">
              {banner}
            </Notice>
          )}
          {children}
        </Stack>
      </form>
      {dialog}
    </FormProvider>
  )
}

// FormSection groups fields under a title. From 1280px the title and description sit in a
// left column and the fields on six columns on the right: half a row each, a third when short,
// the whole row when wide; below that, everything is one column.
// wide puts the title on top and gives the content the whole width, for a table of lines.
// compact puts the title on top too, with four fields to a row, for a short request of a few fields.
// After a failed submit its title carries the number of its fields still to fix.
export function FormSection({ title, description, icon: Icon, wide, compact, children }: { title: string; description?: string; icon?: TablerIcon; wide?: boolean; compact?: boolean; children: ReactNode }) {
  const { t } = useTranslation()
  const { errors, submitCount } = useFormState()
  const ref = useRef<HTMLFieldSetElement>(null)
  const [count, setCount] = useState(0)
  // Which fields are in this section is known only from the page, so count after rendering.
  useEffect(() => setCount(submitCount > 0 ? invalidInputs(errors).filter((el) => ref.current?.contains(el)).length : 0))
  const top = wide || compact
  return (
    <Box component="fieldset" className={classes.section} ref={ref}>
      <Grid gap={{ base: 'sm', lg: 'xl' }}>
        <Grid.Col span={{ base: 12, lg: top ? 12 : 4 }}>
          <Group gap="xs" wrap="nowrap">
            {Icon && <Icon {...icon.text} aria-hidden />}
            <Text component="legend" fz="h2" lh="h2" fw="var(--mantine-h2-font-weight)">
              {title}
            </Text>
            {count > 0 && (
              <Badge color="danger" size="sm" circle aria-label={t('shared.form.section_invalid', { count })}>
                {count}
              </Badge>
            )}
          </Group>
          {/* Quieter than the field labels, so the fields lead. */}
          {description && (
            <Text size="xs" c="dimmed" mt="xxs">
              {description}
            </Text>
          )}
        </Grid.Col>
        <Grid.Col span={{ base: 12, lg: top ? 12 : 8 }}>
          <div className={classes.fields} data-wide={wide || undefined} data-compact={compact || undefined}>
            {children}
          </div>
        </Grid.Col>
      </Grid>
    </Box>
  )
}

// FormFields lays out fields without a section's title: two to a row from 768px, for a group of a
// record's values edited in place in its card.
export function FormFields({ children }: { children: ReactNode }) {
  return (
    <div className={classes.fields} data-pair>
      {children}
    </div>
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
// which only modals pass) the footer sticks to the bottom of the screen, casting a shadow while
// the form runs on under it. Editing a record, it shows only once something is edited or a
// submit failed, says so and offers to discard the changes; creating one (cancelTo, the list to
// go back to) it shows from the start, with Cancel.
export function FormActions({ submitLabel, onCancel, cancelLabel, cancelTo }: { submitLabel: string; onCancel?: () => void; cancelLabel?: string; cancelTo?: string }) {
  const { t } = useTranslation()
  const { formState, reset } = useFormContext()
  const { errors, submitCount, dirtyFields } = useFormState()
  const [slow, setSlow] = useState(false)
  const [stuck, setStuck] = useState(false)
  const last = useRef<HTMLElement | null>(null)
  const [ask, dialog] = useConfirm()
  const invalid = submitCount > 0 ? errorNames(errors).length : 0
  const page = !onCancel
  const dirty = page && !cancelTo && Object.keys(dirtyFields).length > 0
  // The footer sits 1px past the screen's edge while pinned, so it is then never fully in view.
  const pinned = useCallback((el: HTMLDivElement | null) => {
    if (!el) return
    const io = new IntersectionObserver(([e]) => setStuck((e?.intersectionRatio ?? 1) < 1), { threshold: 1 })
    io.observe(el)
    return () => io.disconnect()
  }, [])
  useEffect(() => {
    if (!formState.isSubmitting) return setSlow(false)
    const timer = setTimeout(() => setSlow(true), 300)
    return () => clearTimeout(timer)
  }, [formState.isSubmitting])
  async function discard() {
    if (await ask({ title: t('shared.form.discard_title'), message: t('shared.form.discard_message'), confirmLabel: t('shared.form.discard'), danger: true })) reset()
  }
  // Cancel is quiet beside the primary button; the buttons keep together, the error line wraps first.
  const alert = invalid > 0 && (
    <Group gap="xxs" role="alert" wrap="nowrap" c="danger">
      <IconAlertCircle {...icon.text} aria-hidden />
      <Text size="sm">
        {t('shared.form.invalid', { count: invalid })} ·
      </Text>
      <Anchor component="button" type="button" size="sm" onClick={() => (last.current = focusInvalid(errors, last.current) ?? null)}>
        {t('shared.form.next_invalid')}
      </Anchor>
    </Group>
  )
  const actions = (
    <Group gap="sm" wrap="nowrap">
      {dirty && (
        <Button variant="default" onClick={() => void discard()}>
          {t('shared.form.discard')}
        </Button>
      )}
      {onCancel && (
        <Button variant="subtle" color="gray" onClick={onCancel}>
          {cancelLabel}
        </Button>
      )}
      {page && cancelTo && (
        <Button variant="subtle" color="gray" component={Link} to={cancelTo}>
          {t('shared.form.cancel')}
        </Button>
      )}
      <Button type="submit" loading={slow} disabled={formState.isSubmitting}>
        {submitLabel}
      </Button>
    </Group>
  )
  if (!page)
    return (
      <ModalFooter>
        {alert}
        {actions}
      </ModalFooter>
    )
  const buttons = (
    <Group justify="flex-end">
      {alert}
      {actions}
    </Group>
  )
  if (!cancelTo && !dirty && invalid === 0) return null
  // The state sits beside the buttons it is about; the error line, when shown, says more.
  return (
    <Group ref={pinned} justify="flex-end" gap="lg" className={classes.bar} data-stuck={stuck || undefined}>
      {dirty && invalid === 0 && (
        <Group gap="xs" wrap="nowrap" role="status">
          <span className={classes.dot} aria-hidden />
          <Text size="sm">{t('shared.form.unsaved')}</Text>
        </Group>
      )}
      {buttons}
      {dialog}
    </Group>
  )
}
