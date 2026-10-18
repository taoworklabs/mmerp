import { IconBuildingSkyscraper } from '@tabler/icons-react'
import { useEffect } from 'react'
import { useFormContext } from 'react-hook-form'
import { SelectField, type Option } from './fields'
import { FilterChip } from './FilterChip'
import { useOrgUnits } from './OrgUnitField'

type Scope = { product?: string; permission?: string }

// useLegalEntities lists the legal entities where the user holds the permission, as options;
// `chosen` is the one picked on the URL, or the first when none is.
export function useLegalEntities(scope: Scope, picked?: string) {
  const units = useOrgUnits(scope)
  const options: Option[] = (units.data ?? []).filter((u) => u.kind === 'company').map((u) => ({ value: String(u.id), label: u.name }))
  const chosen = Number(picked) || Number(options[0]?.value) || undefined
  return { options, chosen, isPending: units.isPending, isError: units.isError, error: units.error }
}

// LegalEntityFilter is the legal-entity chip of a page that always shows one legal entity.
export function LegalEntityFilter({ label, options, value, onChange }: { label: string; options: Option[]; value: number | undefined; onChange: (v: string) => void }) {
  return (
    <FilterChip
      label={label}
      icon={IconBuildingSkyscraper}
      data={options}
      value={value ? String(value) : null}
      onChange={(v) => onChange(v ?? '')}
      clearable={false}
    />
  )
}

// LegalEntityField picks a legal entity where the user holds the permission; the only one is
// picked already.
export function LegalEntityField({ name, label, required, ...scope }: Scope & { name: string; label: string; required?: boolean }) {
  const { options } = useLegalEntities(scope)
  const { getValues, setValue } = useFormContext()
  const only = options.length === 1 ? options[0]!.value : undefined
  useEffect(() => {
    if (only && !getValues(name)) setValue(name, only)
  }, [only, name, getValues, setValue])
  return <SelectField name={name} label={label} data={options} required={required} />
}
