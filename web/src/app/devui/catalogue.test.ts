import { describe, expect, it } from 'vitest'
import eslintConfig from '../../../eslint.config.js?raw'
import viJson from '../../shared/i18n/locales/vi.json'
import { type Entry, entries, layers } from './catalogue'

const all = entries as readonly Entry[]
const vi = viJson as Record<string, string>
// The catalogue's own sources, to check the links inside its pages.
const pages = import.meta.glob<string>('./*.tsx', { query: '?raw', import: 'default', eager: true })
const shared = import.meta.glob<string>('../../shared/**/*.{ts,tsx}', { query: '?raw', import: 'default', eager: true })
const key = (e: Entry) => e.path.slice(e.path.indexOf('/') + 1).replaceAll('-', '_')
const byName = (a: string, b: string) => a.localeCompare(b, undefined, { sensitivity: 'base' })

describe('devui catalogue', () => {
  it('paths are unique and start with their layer', () => {
    for (const e of all) expect(e.path.startsWith(`${e.layer}/`), e.path).toBe(true)
    expect(new Set(all.map((e) => e.path)).size).toBe(all.length)
  })

  it('mantine layer covers exactly the area allowlist', () => {
    const match = /allowImportNames:\s*\[([^\]]*)\]/.exec(eslintConfig)
    const allowed = [...match![1]!.matchAll(/'([A-Za-z]+)'/g)].map((m) => m[1]!)
    const covered = all.filter((e) => e.layer === 'mantine').flatMap((e) => e.covers ?? [e.name!])
    expect([...covered].sort()).toEqual([...allowed].sort())
  })

  it('every mmerp entry names its import source', () => {
    for (const e of all.filter((x) => x.layer === 'mmerp')) expect(e.source, e.path).toMatch(/^@\/shared\//)
  })

  it('every mmerp import line names real exports of its source', () => {
    const missing: string[] = []
    for (const e of all.filter((x) => x.layer === 'mmerp')) {
      const dir = e.source!.replace('@/', '../../')
      const files = Object.entries(shared).filter(([f]) => f.startsWith(`${dir}/`) || f.startsWith(`${dir}.`))
      for (const name of e.exports ?? [e.name!]) {
        const exported = new RegExp(`\\bexport\\b[^\\n]*\\b${name}\\b`)
        if (!/^\w+$/.test(name) || !files.some(([, src]) => exported.test(src))) missing.push(`${e.path}: ${name}`)
      }
    }
    expect(missing).toEqual([])
  })

  it('component layers are alphabetical', () => {
    for (const layer of ['mantine', 'mmerp', 'templates'] as const) {
      const names = all.filter((e) => e.layer === layer).map((e) => e.name!)
      expect(names, layer).toEqual([...names].sort(byName))
    }
  })

  it('every entry has its i18n keys', () => {
    const missing: string[] = []
    const need = (k: string) => vi[k] === undefined && missing.push(k)
    for (const l of layers) {
      need(`shared.devui.layer_${l}`)
      need(`shared.devui.layer_${l}_note`)
    }
    for (const e of all) {
      if (e.label) need(e.label)
      if (!e.name && !e.label) missing.push(`${e.path}: no name or label`)
      if (['mantine', 'mmerp', 'patterns', 'templates'].includes(e.layer)) for (const p of ['about', 'use', 'avoid']) need(`shared.devui.${p}_${key(e)}`)
    }
    expect(missing).toEqual([])
  })

  it('a page frame named in the texts is a catalogued template', () => {
    const known = new Set(all.flatMap((e) => [...(e.layer === 'templates' ? [e.name!] : []), ...(e.exports ?? [])]))
    const texts = Object.entries(vi).filter(([k]) => /^shared\.devui\.(about|use|avoid)_/.test(k))
    const unknown = texts.flatMap(([k, v]) => [...v.matchAll(/\b[A-Z]\w*Page\b/g)].filter((m) => !known.has(m[0])).map((m) => `${k}: ${m[0]}`))
    expect(unknown).toEqual([])
  })

  it('related points at real entries', () => {
    const paths = new Set(all.map((e) => e.path))
    for (const e of all) for (const r of e.related ?? []) expect(paths.has(r), `${e.path} → ${r}`).toBe(true)
  })

  it('every /dev/ui link in the catalogue resolves', () => {
    const paths = new Set(all.map((e) => e.path))
    const broken: string[] = []
    for (const [file, source] of Object.entries(pages)) {
      for (const m of source.matchAll(/['"`]\/dev\/ui\/([a-z0-9/-]+)/g)) if (!paths.has(m[1]!)) broken.push(`${file}: ${m[1]}`)
    }
    expect(broken).toEqual([])
  })
})
