import { renderToStaticMarkup } from 'react-dom/server'
import { expect, test } from 'vitest'
import type { Me } from '@/shared/api/core'
import { MeProvider } from '@/shared/auth/me'
import type { AreaManifest } from './index'
import { RecordTypesProvider, useRecordLink } from './records'

const me = (permissions: Me['permissions']): Me => ({ id: 1, login: 'a', name: 'A', locale: 'en', timezone: 'UTC', authz_version: '1', must_change_password: false, permissions, products: ['hrm'], products_with_data: [] })
const Icon = () => null
const area = (basePath: string, path: string, extra: Partial<AreaManifest> = {}): AreaManifest => ({
  product: 'hrm',
  label: basePath,
  basePath,
  nav: [],
  homeIcon: Icon,
  i18n: { meta: { vi: async () => ({ default: {} }), en: async () => ({ default: {} }) } },
  recordTypes: { 'hrm.leave_request': { path: (id) => `${path}/${id}`, invalidate: () => [] } },
  ...extra,
})
const main = area('/hrm', 'leaves')
const personal = area('/me', 'requests/leave', { useEntitled: () => true })

function Link() {
  return <>{useRecordLink('hrm.leave_request', 7) ?? 'none'}</>
}
const render = (permissions: Me['permissions'], areas: AreaManifest[]) =>
  renderToStaticMarkup(
    <MeProvider me={me(permissions)}>
      <RecordTypesProvider areas={areas}>
        <Link />
      </RecordTypesProvider>
    </MeProvider>,
  )

test('a record opens in the first area the user may enter', () => {
  expect(render({ hrm: ['hrm.leave.view'] }, [main, personal])).toBe('/hrm/leaves/7')
  expect(render({}, [main, personal])).toBe('/me/requests/leave/7')
})

test('no area the user may enter: no link', () => {
  expect(render({}, [main])).toBe('none')
})
