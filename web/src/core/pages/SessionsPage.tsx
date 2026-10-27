import { Button } from '@mmerp/ui'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type SessionInfo as Session } from '@/shared/api/core'
import { useMe } from '@/shared/auth/me'
import { errorText, formatDateTime } from '@/shared/i18n'
import { type Column } from '@/shared/ui/DataTable'
import { useConfirm } from '@/shared/ui/confirm'
import { Notice } from '@/shared/ui/Notice'
import { notifySuccess } from '@/shared/ui/notify'
import { ListPage } from '@/shared/ui/page'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { deviceLabel } from '../deviceLabel'
import { coreKeys } from '../keys'

const key = coreKeys.sessions()


// The signed-in user's own sessions: end one, or all but this one.
export default function SessionsPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const me = useMe()
  const [ask, dialog] = useConfirm()
  const [error, setError] = useState<string | null>(null)
  const list = useQuery({
    queryKey: key,
    queryFn: () => unwrap(api.GET('/me/sessions')),
    select: (d) => ({ items: d ?? [], total: d?.length ?? 0 }),
  })
  const others = list.data?.items.some((s) => !s.current) ?? false

  async function run(action: () => Promise<unknown>, done: string) {
    setError(null)
    try {
      await action()
      notifySuccess(done)
    } catch (err) {
      setError(errorText(t, err))
    }
    await qc.invalidateQueries({ queryKey: key })
  }

  async function endAll() {
    const ok = await ask({ title: t('core.sessions.end_all'), message: t('core.sessions.end_all_confirm'), confirmLabel: t('core.sessions.end_all'), danger: true })
    if (ok) await run(() => unwrap(api.DELETE('/me/sessions')), t('core.sessions.ended_all'))
  }

  const columns: Column<Session>[] = [
    {
      key: 'device',
      header: t('core.sessions.device'),
      role: 'title',
      render: (s) => (
        <>
          {deviceLabel(s.user_agent) || t('core.sessions.unknown')}
          {s.current && <StatusBadge tone="info">{t('core.sessions.this_device')}</StatusBadge>}
        </>
      ),
    },
    { key: 'created_at', header: t('core.sessions.created_at'), role: 'meta', render: (s) => formatDateTime(s.created_at, me.timezone) },
    { key: 'expires_at', header: t('core.sessions.expires_at'), role: 'meta', render: (s) => formatDateTime(s.expires_at, me.timezone) },
    {
      key: 'end',
      header: t('core.sessions.end'),
      role: 'meta',
      render: (s) =>
        s.current ? null : (
          <Button variant="default" size="sm" onClick={() => void run(() => unwrap(api.DELETE('/me/sessions/{id}', { params: { path: { id: s.id } } })), t('core.sessions.ended'))}>
            {t('core.sessions.end')}
          </Button>
        ),
    },
  ]

  return (
    <ListPage
      title={t('core.sessions.title')}
      description={t('core.sessions.description')}
      action={
        others && (
          <Button variant="default" onClick={() => void endAll()}>
            {t('core.sessions.end_all')}
          </Button>
        )
      }
      list={{
        query: list,
        table: { label: t('core.sessions.title'), columns, rowKey: (s) => s.id },
        empty: { title: t('core.sessions.empty'), description: '' },
        noResults: t('core.sessions.empty'),
      }}
    >
      {error && <Notice tone="danger">{error}</Notice>}
      {dialog}
    </ListPage>
  )
}
