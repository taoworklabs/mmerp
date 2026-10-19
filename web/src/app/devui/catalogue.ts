// The catalogue's entries as data: the sidebar, the routes and each page's header and links read
// from here, and catalogue.test.ts checks it against the lint allowlist and the translations.

export type Layer = 'start' | 'foundations' | 'mantine' | 'mmerp' | 'patterns' | 'templates'

export const layers: Layer[] = ['start', 'foundations', 'mantine', 'mmerp', 'patterns', 'templates']

// An entry shows its component name as-is, or an i18n label. `path` is `<layer>/<slug>`.
export type Entry = {
  layer: Layer
  path: string
  name?: string
  label?: string
  // mmerp: where to import it from.
  source?: string
  // mantine: the @mantine/core names the page documents, when more than its name.
  covers?: readonly string[]
  // Paths of the entries a page links to: related components, or what a pattern or template uses.
  related?: readonly string[]
}

export const entries = [{ layer: 'mantine', path: 'mantine/button', name: 'Button' }] as const satisfies readonly Entry[]

export type Path = (typeof entries)[number]['path']

export const entryByPath = (path: string): Entry => {
  const e = (entries as readonly Entry[]).find((x) => x.path === path)
  if (!e) throw new Error(`devui: no entry ${path}`)
  return e
}
