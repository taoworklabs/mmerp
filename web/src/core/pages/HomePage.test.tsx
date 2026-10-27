import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter } from 'react-router'
import { expect, test } from 'vitest'
import type { Me } from '@/shared/api/core'
import type { AreaManifest } from '@/shared/area'
import { MeProvider } from '@/shared/auth/me'
import HomePage from './HomePage'

const me: Me = { id: 1, login: 'a', name: 'A', locale: 'en', timezone: 'UTC', authz_version: '1', must_change_password: false, permissions: {}, products: ['hrm'], products_with_data: [] }
const Icon = () => null
const area = (label: string, basePath: string, extra: Partial<AreaManifest> = {}): AreaManifest => ({
  product: 'hrm',
  label,
  basePath,
  nav: [{ label: 'x', path: 'p', icon: Icon }],
  homeIcon: Icon,
  i18n: { meta: { vi: async () => ({ default: {} }), en: async () => ({ default: {} }) } },
  ...extra,
})
const render = (areas: AreaManifest[]) =>
  renderToStaticMarkup(
    <MemoryRouter>
      <MeProvider me={me}>
        <HomePage areas={areas} />
      </MeProvider>
    </MemoryRouter>,
  )

test('useEntitled replaces the permission rule', () => {
  expect(render([area('yes', '/a', { useEntitled: () => true })])).toContain('href="/a/p"')
  expect(render([area('no', '/a', { useEntitled: () => false })])).not.toContain('href="/a/p"')
  expect(render([area('wait', '/a', { useEntitled: () => undefined })])).not.toContain('href="/a/p"')
})

test('areas without the hook keep the permission rule', () => {
  expect(render([area('none', '/a')])).not.toContain('href="/a/p"')
})

test('two areas of one product both render', () => {
  const html = render([area('one', '/a', { useEntitled: () => true }), area('two', '/b', { useEntitled: () => true })])
  expect(html).toContain('href="/a/p"')
  expect(html).toContain('href="/b/p"')
})

test('no tile and no pending verdict shows the empty state', () => {
  expect(render([area('no', '/a', { useEntitled: () => false })])).toContain('core.home.empty')
  expect(render([area('wait', '/a', { useEntitled: () => undefined })])).not.toContain('core.home.empty')
})
