import type * as React from 'react'
import { render } from 'vitest-browser-react'
import { page } from 'vitest/browser'
import { describe, expect, it, vi } from 'vitest'
import { DateFilterChip } from '../../blocks/list/DateFilterChip.tsx'
import { FilterChip } from '../../blocks/list/FilterChip.tsx'
import { MonthFilterChip } from '../../blocks/list/MonthFilterChip.tsx'
import { Calendar } from '../dates/Calendar.tsx'
import { DateInput } from '../dates/DateInput.tsx'
import { MonthPicker } from '../dates/MonthPicker.tsx'
import { MonthPickerInput } from '../dates/MonthPickerInput.tsx'
import { formatDate, formatMonth, monthLabels, parseDate, viLocale, yearLabels } from '../dates/storySamples.ts'
import { Tabs } from '../navigation/Tabs.tsx'
import { Button } from './Button.tsx'
import { Combobox } from './Combobox.tsx'
import { ComboboxButton } from './ComboboxButton.tsx'
import { FileInput } from './FileInput.tsx'
import { MultiSelect } from './MultiSelect.tsx'
import { NumberInput } from './NumberInput.tsx'
import { SegmentedControl } from './SegmentedControl.tsx'
import { Select } from './Select.tsx'
import '../../styles/index.css'

const data = [
  { value: 'a', label: 'Alpha' },
  { value: 'b', label: 'Beta' },
]
const icon = <span />
const file = new File(['x'], 'contract.pdf')

// Each input starts from defaultValue, changes on interaction without a value prop, and reports the change.
const cases: {
  name: string
  ui(onChange: (value: never) => void): React.ReactElement
  before(): Promise<void>
  act(): Promise<void>
  after(): Promise<void>
  called: unknown
}[] = [
  {
    name: 'NumberInput',
    ui: (onChange) => <NumberInput label="Qty" defaultValue={5} onChange={onChange} />,
    before: () => expect.element(page.getByRole('textbox')).toHaveValue('5'),
    act: () => page.getByRole('textbox').fill('7'),
    after: () => expect.element(page.getByRole('textbox')).toHaveValue('7'),
    called: 7,
  },
  {
    name: 'Select',
    ui: (onChange) => <Select label="Pick" data={data} defaultValue="a" onChange={onChange} />,
    before: () => expect.element(page.getByRole('combobox')).toHaveTextContent('Alpha'),
    act: async () => {
      await page.getByRole('combobox').click()
      await page.getByRole('option', { name: 'Beta' }).click()
    },
    after: () => expect.element(page.getByRole('combobox')).toHaveTextContent('Beta'),
    called: 'b',
  },
  {
    name: 'Combobox',
    ui: (onChange) => <Combobox label="Pick" data={data} defaultValue="a" onChange={onChange} nothingFound="None" clearable clearLabel="Clear" />,
    before: () => expect.element(page.getByRole('combobox')).toHaveValue('Alpha'),
    act: () => page.getByRole('button', { name: 'Clear' }).click(),
    after: async () => {
      await expect.element(page.getByRole('combobox')).toHaveValue('')
      await expect.element(page.getByRole('button', { name: 'Clear' })).not.toBeInTheDocument()
    },
    called: null,
  },
  {
    name: 'MultiSelect',
    ui: (onChange) => <MultiSelect label="Pick" data={data} defaultValue={['a', 'b']} onChange={onChange} nothingFound="None" removeLabel={(l) => `Remove ${l}`} />,
    before: () => expect.element(page.getByText('Alpha')).toBeInTheDocument(),
    act: () => page.getByRole('button', { name: 'Remove Alpha' }).click(),
    after: () => expect.element(page.getByText('Alpha')).not.toBeInTheDocument(),
    called: ['b'],
  },
  {
    name: 'FileInput',
    ui: (onChange) => <FileInput label="File" defaultValue={file} onChange={onChange} placeholder="No file" clearLabel="Clear" />,
    before: () => expect.element(page.getByText('contract.pdf')).toBeInTheDocument(),
    act: () => page.getByRole('button', { name: 'Clear' }).click(),
    after: () => expect.element(page.getByText('No file')).toBeInTheDocument(),
    called: null,
  },
  {
    name: 'DateInput',
    ui: (onChange) => (
      <DateInput label="Day" defaultValue="2026-03-15" onChange={onChange} parse={parseDate} format={formatDate} locale={viLocale} labels={monthLabels} clearable clearLabel="Clear" />
    ),
    before: () => expect.element(page.getByRole('textbox')).toHaveValue('15/03/2026'),
    act: () => page.getByRole('button', { name: 'Clear' }).click(),
    after: async () => {
      await expect.element(page.getByRole('textbox')).toHaveValue('')
      await expect.element(page.getByRole('button', { name: 'Clear' })).not.toBeInTheDocument()
    },
    called: null,
  },
  {
    name: 'MonthPickerInput',
    ui: (onChange) => (
      <MonthPickerInput label="Month" defaultValue="2026-03" onChange={onChange} format={formatMonth} locale={viLocale} labels={yearLabels} placeholder="Any" clearable clearLabel="Clear" />
    ),
    before: () => expect.element(page.getByText('03/2026')).toBeInTheDocument(),
    act: () => page.getByRole('button', { name: 'Clear' }).click(),
    after: () => expect.element(page.getByText('Any')).toBeInTheDocument(),
    called: null,
  },
  {
    name: 'SegmentedControl',
    ui: (onChange) => <SegmentedControl aria-label="View" data={data} defaultValue="a" onChange={onChange} />,
    before: () => expect.element(page.getByRole('button', { name: 'Alpha' })).toHaveAttribute('aria-pressed', 'true'),
    act: () => page.getByRole('button', { name: 'Beta' }).click(),
    after: () => expect.element(page.getByRole('button', { name: 'Beta' })).toHaveAttribute('aria-pressed', 'true'),
    called: 'b',
  },
  {
    name: 'Tabs',
    ui: (onChange) => (
      <Tabs defaultValue="a" onChange={onChange}>
        <Tabs.List>
          <Tabs.Tab value="a">Alpha</Tabs.Tab>
          <Tabs.Tab value="b">Beta</Tabs.Tab>
        </Tabs.List>
      </Tabs>
    ),
    before: () => expect.element(page.getByRole('tab', { name: 'Alpha' })).toHaveAttribute('aria-selected', 'true'),
    act: () => page.getByRole('tab', { name: 'Beta' }).click(),
    after: () => expect.element(page.getByRole('tab', { name: 'Beta' })).toHaveAttribute('aria-selected', 'true'),
    called: 'b',
  },
  {
    name: 'Calendar',
    ui: (onChange) => <Calendar defaultValue="2026-03-15" onChange={onChange} locale={viLocale} labels={monthLabels} />,
    before: () => expect.element(page.getByRole('gridcell', { name: '15', exact: true })).toHaveAttribute('aria-selected', 'true'),
    act: () => page.getByRole('gridcell', { name: '20', exact: true }).click(),
    after: () => expect.element(page.getByRole('gridcell', { name: '20', exact: true })).toHaveAttribute('aria-selected', 'true'),
    called: '2026-03-20',
  },
  {
    name: 'MonthPicker',
    ui: (onChange) => <MonthPicker defaultValue="2026-03" onChange={onChange} locale={viLocale} labels={yearLabels} />,
    before: () => expect.element(page.getByRole('gridcell', { name: 'Tháng 3', exact: true })).toHaveAttribute('aria-selected', 'true'),
    act: () => page.getByRole('gridcell', { name: 'Tháng 5', exact: true }).click(),
    after: () => expect.element(page.getByRole('gridcell', { name: 'Tháng 5', exact: true })).toHaveAttribute('aria-selected', 'true'),
    called: '2026-05',
  },
  {
    name: 'ComboboxButton',
    ui: (onChange) => <ComboboxButton trigger={<Button>Pick</Button>} data={data} defaultValue="a" onChange={onChange} nothingFound="None" />,
    before: async () => {
      await page.getByRole('combobox').click()
      await expect.element(page.getByRole('option', { name: 'Alpha' })).toHaveAttribute('aria-selected', 'true')
    },
    act: () => page.getByRole('option', { name: 'Beta' }).click(),
    after: async () => {
      await page.getByRole('combobox').click()
      await expect.element(page.getByRole('option', { name: 'Beta' })).toHaveAttribute('aria-selected', 'true')
    },
    called: 'b',
  },
  {
    name: 'FilterChip',
    ui: (onChange) => (
      <FilterChip label="Status" icon={icon} data={data} defaultValue="a" onChange={onChange} valueText={(l, v) => `${l}: ${v}`} clearLabel="Clear" nothingFound="None" />
    ),
    before: () => expect.element(page.getByText('Status: Alpha')).toBeInTheDocument(),
    act: () => page.getByRole('button', { name: 'Clear' }).click(),
    after: () => expect.element(page.getByText('Status', { exact: true })).toBeInTheDocument(),
    called: null,
  },
  {
    name: 'DateFilterChip',
    ui: (onChange) => (
      <DateFilterChip label="Due" icon={icon} defaultValue="2026-03-15" onChange={onChange} valueText={(l, v) => `${l}: ${v}`} clearLabel="Clear" locale={viLocale} labels={monthLabels} />
    ),
    before: () => expect.element(page.getByText('Due: 2026-03-15')).toBeInTheDocument(),
    act: () => page.getByRole('button', { name: 'Clear' }).click(),
    after: () => expect.element(page.getByText('Due', { exact: true })).toBeInTheDocument(),
    called: null,
  },
  {
    name: 'MonthFilterChip',
    ui: (onChange) => (
      <MonthFilterChip label="Month" icon={icon} defaultValue="2026-03" onChange={onChange} valueText={(l, v) => `${l}: ${v}`} clearLabel="Clear" locale={viLocale} labels={yearLabels} />
    ),
    before: () => expect.element(page.getByText('Month: 2026-03')).toBeInTheDocument(),
    act: () => page.getByRole('button', { name: 'Clear' }).click(),
    after: () => expect.element(page.getByText('Month', { exact: true })).toBeInTheDocument(),
    called: null,
  },
]

describe('inputs work uncontrolled from defaultValue', () => {
  it.each(cases)('$name', async ({ ui, before, act, after, called }) => {
    const onChange = vi.fn()
    await render(ui(onChange))
    await before()
    await act()
    await after()
    expect(onChange).toHaveBeenLastCalledWith(called)
  })
})
