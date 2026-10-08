import { Menu, Stack, Text } from '@mantine/core'
import { IconPencil } from '@tabler/icons-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type PrintTemplate, type PrintTemplateBlocks } from '@/shared/api/core'
import { useMe } from '@/shared/auth/me'
import { errorText, formatDateTime } from '@/shared/i18n'
import { DataTable } from '@/shared/ui/DataTable'
import { Form, FormActions, TextField } from '@/shared/ui/form'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { ContentSkeleton, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { coreKeys } from '../keys'

// PrintTemplatesPage: the templates products print with. Their layout is fixed; tenant
// administrators edit the wording of their text blocks, each save a new version.
export function PrintTemplatesPage() {
  const { t } = useTranslation()
  const me = useMe()
  const [editing, setEditing] = useState<string | null>(null)
  const templates = useQuery({ queryKey: coreKeys.printTemplates.all(), queryFn: () => unwrap(api.GET('/print-templates')) })

  let body
  if (templates.isPending) body = <ContentSkeleton />
  else if (templates.isError) body = <ErrorState message={errorText(t, templates.error)} onRetry={() => void templates.refetch()} />
  else
    body = (
      <DataTable<PrintTemplate>
        label={t('core.print.title')}
        rows={templates.data}
        rowKey={(p) => p.code}
        columns={[
          { key: 'name', header: t('core.print.name'), role: 'title', render: (p) => p.name },
          {
            key: 'version',
            header: t('core.print.version'),
            role: 'meta',
            render: (p) => (p.version === 0 ? t('core.print.default') : t('core.print.version_n', { n: p.version })),
          },
          {
            key: 'saved',
            header: t('core.print.saved'),
            role: 'meta',
            render: (p) => (p.saved_at ? t('core.print.saved_by', { at: formatDateTime(p.saved_at, me.timezone), name: p.saved_by_name }) : '—'),
          },
        ]}
        menu={(p) => (
          <Menu.Item leftSection={<IconPencil {...icon.button} />} onClick={() => setEditing(p.code)}>
            {t('core.print.edit')}
          </Menu.Item>
        )}
      />
    )

  return (
    <ListPage title={t('core.print.title')} description={t('core.print.description')}>
      {body}
      {editing && <BlocksModal code={editing} onClose={() => setEditing(null)} />}
    </ListPage>
  )
}

type Values = { blocks: Record<string, { vi: string; en: string }> }

function BlocksModal({ code, onClose }: { code: string; onClose: () => void }) {
  const { t } = useTranslation()
  const template = useQuery({ queryKey: coreKeys.printTemplates.detail(code), queryFn: () => unwrap(api.GET('/print-templates/{code}', { params: { path: { code } } })) })
  return (
    <FormModal opened title={template.data ? t('core.print.edit_title', { name: template.data.name }) : t('core.print.edit')} onClose={onClose}>
      {template.isPending ? (
        <ContentSkeleton />
      ) : template.isError ? (
        <ErrorState message={errorText(t, template.error)} onRetry={() => void template.refetch()} />
      ) : (
        <BlocksForm template={template.data} onClose={onClose} />
      )}
    </FormModal>
  )
}

function BlocksForm({ template, onClose }: { template: PrintTemplateBlocks; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const form = useForm<Values>({
    mode: 'onBlur',
    defaultValues: { blocks: Object.fromEntries(template.blocks.map((b) => [b.key, { vi: b.vi, en: b.en }])) },
  })

  async function save(v: Values) {
    const blocks = template.blocks.map((b) => ({ key: b.key, vi: v.blocks[b.key]?.vi ?? '', en: v.blocks[b.key]?.en ?? '' }))
    await unwrap(api.PUT('/print-templates/{code}/blocks', { params: { path: { code: template.code } }, body: { blocks } }))
    await qc.invalidateQueries({ queryKey: coreKeys.printTemplates.all() })
    notifySuccess(t('core.print.saved_notice'))
    onClose()
  }

  return (
    <Form form={form} onSubmit={save}>
      <Text size="sm" c="dimmed">
        {t('core.print.hint')}
      </Text>
      {template.blocks.map((b) => {
        const placeholders = b.placeholders.map((p) => `{${p}}`).join(', ')
        const description = placeholders ? t('core.print.placeholders', { list: placeholders }) : undefined
        return (
          <Stack key={b.key} gap="xs">
            <TextField name={`blocks.${b.key}.vi`} label={t('core.print.block_vi', { label: b.label })} description={description} multiline maxLength={4000} />
            <TextField name={`blocks.${b.key}.en`} label={t('core.print.block_en', { label: b.label })} description={description} multiline maxLength={4000} />
          </Stack>
        )
      })}
      <FormActions submitLabel={t('core.print.save')} onCancel={onClose} cancelLabel={t('core.common.cancel')} />
    </Form>
  )
}
