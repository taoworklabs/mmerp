import { Select, type ComboboxItem, type ComboboxLikeRenderOptionInput } from '@mantine/core'
import { IconSitemap } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useController, useFormContext } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type OrgUnit } from '@/shared/api/core'
import { OrgUnitName } from '../OrgUnitName'
import { icon } from '../theme'
import { fieldId } from './Form'
import { belowError, type Option } from './fields'

// Without a permission it lists the whole tree (administrators only).
type Scope = { product?: string; permission?: string }

// orgUnitOptions orders the tree depth-first; depth drives the indent in lists.
export function orgUnitOptions(units: OrgUnit[]): (Option & { depth: number })[] {
  const ids = new Set(units.map((u) => u.id))
  const children = new Map<number | null, OrgUnit[]>()
  for (const u of units) {
    // A unit whose parent is out of scope is shown as a root.
    const parent = u.parent_id !== null && ids.has(u.parent_id) ? u.parent_id : null
    children.set(parent, [...(children.get(parent) ?? []), u])
  }
  const out: (Option & { depth: number })[] = []
  const walk = (parent: number | null, depth: number) => {
    for (const u of children.get(parent) ?? []) {
      out.push({ value: String(u.id), label: u.name, depth })
      walk(u.id, depth + 1)
    }
  }
  walk(null, 0)
  return out
}

// The indent shows only in the dropdown, never in the chosen value.
function useTreeSelect(units: OrgUnit[]) {
  const options = orgUnitOptions(units)
  const byId = new Map(units.map((u) => [String(u.id), u]))
  const byValue = new Map(options.map((o) => [o.value, { depth: o.depth, unit: byId.get(o.value) }]))
  return {
    data: options.map(({ value, label }) => ({ value, label })),
    renderOption: ({ option }: ComboboxLikeRenderOptionInput<ComboboxItem>) => {
      const row = byValue.get(option.value)
      return row?.unit ? <OrgUnitName unit={row.unit} depth={row.depth} /> : option.label
    },
  }
}

// orgUnitKeys: every org-unit list, whole or scoped, sits under all(); invalidate that after a tree write.
export const orgUnitKeys = {
  all: () => ['core', 'org-units'] as const,
  list: (scope: Scope) => ['core', 'org-units', scope] as const,
}

export function useOrgUnits({ product, permission }: Scope) {
  return useQuery({
    queryKey: orgUnitKeys.list({ product, permission }),
    queryFn: async () => (await unwrap(api.GET('/org-units', { params: { query: { product, permission } } }))) ?? [],
  })
}

type SelectProps = Scope & { label: string; value: string | null; onChange: (v: string | null) => void; clearable?: boolean }

// OrgUnitSelect is the unbound picker of a list's filter bar: the label names it for
// assistive technology and shows as the placeholder.
export function OrgUnitSelect({ product, permission, label, value, onChange, clearable }: SelectProps) {
  const { data = [] } = useOrgUnits({ product, permission })
  const tree = useTreeSelect(data)
  return (
    <Select
      aria-label={label}
      placeholder={label}
      leftSection={<IconSitemap {...icon.text} />}
      {...tree}
      value={value}
      onChange={onChange}
      searchable
      clearable={clearable}
      w={{ base: '100%', md: 240 }}
    />
  )
}

// OrgUnitField shows the tree, limited to the units where the user has the permission.
export function OrgUnitField({
  name,
  label,
  required,
  product,
  permission,
  clearable,
  readOnly,
}: Scope & { name: string; label: string; required?: boolean; clearable?: boolean; readOnly?: boolean }) {
  const { t } = useTranslation()
  const { control } = useFormContext()
  const { field, fieldState } = useController({ name, control, rules: { required: required ? t('shared.form.required') : false } })
  const { data = [] } = useOrgUnits({ product, permission })
  const tree = useTreeSelect(data)
  return (
    <Select
      {...tree}
      id={fieldId(name)}
      ref={field.ref}
      label={label}
      withAsterisk={required}
      error={fieldState.error?.message}
      value={field.value ?? null}
      onChange={field.onChange}
      onBlur={field.onBlur}
      searchable
      clearable={clearable}
      readOnly={readOnly}
      comboboxProps={belowError(fieldState.error?.message)}
    />
  )
}
