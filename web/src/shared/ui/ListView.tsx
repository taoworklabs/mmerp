import { Button, Checkbox, Group, Menu, Popover, Stack, Text } from '@mantine/core'
import { IconColumns3, IconDeviceFloppy, IconLayoutList, IconTrash } from '@tabler/icons-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router'
import { api, unwrap, type ListSettings } from '@/shared/api/core'
import type { Column } from './DataTable'
import { Form, FormActions, TextField } from './form'
import { listViewKeys, shownColumns, viewQuery, withColumn } from './listView'
import { FormModal } from './page'
import { icon } from './theme'

const empty: ListSettings = { views: [], columns: [] }

// useListView gives a list the user's own settings for it (saved views, column choices) and the
// two menus that change them. listKey names the list for good, e.g. "sales.customers".
export function useListView<T>(listKey: string, columns: Column<T>[]) {
  const qc = useQueryClient()
  const key = listViewKeys.one(listKey)
  const settings = useQuery({ queryKey: key, queryFn: () => unwrap(api.GET('/me/lists/{key}', { params: { path: { key: listKey } } })) })
  // Display preferences change at once; a failed save puts the previous ones back.
  const save = useMutation({
    mutationFn: (next: ListSettings) => unwrap(api.PUT('/me/lists/{key}', { params: { path: { key: listKey } }, body: next })),
    onMutate: async (next) => {
      await qc.cancelQueries({ queryKey: key })
      const before = qc.getQueryData<ListSettings>(key)
      qc.setQueryData(key, next)
      return { before }
    },
    onError: (_e, _next, ctx) => qc.setQueryData(key, ctx?.before),
    onSettled: () => qc.invalidateQueries({ queryKey: key }),
  })
  const current = settings.data ?? empty
  return {
    columns: shownColumns(columns, settings.data),
    onResize: (column: string, width: number) => save.mutate(withColumn(current, column, { width })),
    tools: (
      <Group gap="xs" wrap="nowrap">
        <ViewsMenu settings={current} onChange={(next) => save.mutateAsync(next)} />
        <ColumnsMenu columns={columns} settings={current} onChange={(next) => save.mutate(next)} />
      </Group>
    ),
  }
}

// ViewsMenu lists the saved views of a list; picking one opens the list as it was saved.
function ViewsMenu({ settings, onChange }: { settings: ListSettings; onChange: (next: ListSettings) => Promise<unknown> }) {
  const { t } = useTranslation()
  const [search, setSearch] = useSearchParams()
  const [naming, setNaming] = useState(false)
  const query = viewQuery(search)
  const active = settings.views.find((v) => v.query === query)
  return (
    <>
      <Menu position="bottom-end">
        <Menu.Target>
          <Button variant="default" leftSection={<IconLayoutList {...icon.button} />}>
            {active?.name ?? t('shared.views.label')}
          </Button>
        </Menu.Target>
        <Menu.Dropdown>
          {settings.views.length === 0 && <Menu.Label>{t('shared.views.none')}</Menu.Label>}
          {settings.views.map((v) => (
            <Menu.Item key={v.name} data-active={v === active || undefined} onClick={() => setSearch(new URLSearchParams(v.query))}>
              {v.name}
            </Menu.Item>
          ))}
          <Menu.Divider />
          <Menu.Item leftSection={<IconDeviceFloppy {...icon.text} />} onClick={() => setNaming(true)}>
            {t('shared.views.save')}
          </Menu.Item>
          {active && (
            <Menu.Item color="danger" leftSection={<IconTrash {...icon.text} />} onClick={() => void onChange({ ...settings, views: settings.views.filter((v) => v !== active) })}>
              {t('shared.views.delete', { name: active.name })}
            </Menu.Item>
          )}
        </Menu.Dropdown>
      </Menu>
      <FormModal title={t('shared.views.save')} opened={naming} onClose={() => setNaming(false)}>
        <SaveView
          onCancel={() => setNaming(false)}
          onSave={async (name) => {
            await onChange({ ...settings, views: [...settings.views.filter((v) => v.query !== query), { name, query }] })
            setNaming(false)
          }}
        />
      </FormModal>
    </>
  )
}

function SaveView({ onSave, onCancel }: { onSave: (name: string) => Promise<unknown>; onCancel: () => void }) {
  const { t } = useTranslation()
  const form = useForm<{ name: string }>({ defaultValues: { name: '' }, mode: 'onBlur' })
  return (
    <Form form={form} onSubmit={(v) => onSave(v.name.trim())} fieldOf={(err) => (err.code === 'list_view_name_taken' ? 'name' : undefined)}>
      <TextField name="name" label={t('shared.views.name')} required maxLength={100} />
      <FormActions submitLabel={t('shared.views.save_submit')} onCancel={onCancel} />
    </Form>
  )
}

// ColumnsMenu shows or hides the columns after the first, and puts back the designed layout.
function ColumnsMenu<T>({ columns, settings, onChange }: { columns: Column<T>[]; settings: ListSettings; onChange: (next: ListSettings) => void }) {
  const { t } = useTranslation()
  const hidden = new Set(settings.columns.filter((c) => c.hidden).map((c) => c.key))
  return (
    <Popover position="bottom-end">
      <Popover.Target>
        <Button variant="default" leftSection={<IconColumns3 {...icon.button} />}>
          {t('shared.views.columns')}
        </Button>
      </Popover.Target>
      <Popover.Dropdown>
        <Stack gap="xs">
          <Text size="xs" fw={600} c="dimmed">
            {t('shared.views.columns_hint')}
          </Text>
          {columns.slice(1).map((c) => (
            <Checkbox key={c.key} label={c.header} checked={!hidden.has(c.key)} onChange={(e) => onChange(withColumn(settings, c.key, { hidden: !e.currentTarget.checked }))} />
          ))}
          <Group>
            <Button variant="subtle" onClick={() => onChange({ ...settings, columns: [] })}>
              {t('shared.views.reset')}
            </Button>
          </Group>
        </Stack>
      </Popover.Dropdown>
    </Popover>
  )
}
