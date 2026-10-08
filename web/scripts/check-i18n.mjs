// Every vi translation file must have an en sibling with exactly the same keys, and vice versa.
// Covers frontend (vi.json, i18n/vi/*.json) and backend module files anywhere in the repo.
import { readFileSync, readdirSync } from 'node:fs'
import { join, relative } from 'node:path'

const root = new URL('../../', import.meta.url).pathname
const skip = new Set(['node_modules', '.git', 'dist', '.backup', 'lint-fixtures'])

function* walk(dir) {
  for (const e of readdirSync(dir, { withFileTypes: true })) {
    if (skip.has(e.name)) continue
    const p = join(dir, e.name)
    if (e.isDirectory()) yield* walk(p)
    else if (e.name.endsWith('.json')) yield p
  }
}

// vi.json ↔ en.json, or …/vi/x.json ↔ …/en/x.json
const sibling = (p, from, to) =>
  p.endsWith(`/${from}.json`) ? p.slice(0, -`${from}.json`.length) + `${to}.json`
  : p.includes(`/${from}/`) ? p.replace(`/${from}/`, `/${to}/`) : null

const keys = (p) => {
  try { return new Set(Object.keys(JSON.parse(readFileSync(p, 'utf8')))) } catch { return null }
}

let errors = 0
let files = 0
for (const p of walk(root)) {
  for (const [from, to] of [['vi', 'en'], ['en', 'vi']]) {
    const other = sibling(p, from, to)
    if (!other) continue
    if (from === 'vi') files++
    const a = keys(p)
    const b = keys(other)
    if (!a) { console.error(`${relative(root, p)}: invalid JSON`); errors++; continue }
    if (!b) { console.error(`${relative(root, other)}: missing or invalid`); errors++; continue }
    for (const k of a) if (!b.has(k)) { console.error(`${relative(root, other)}: missing key ${k}`); errors++ }
  }
}
if (errors) process.exit(1)
console.log(`i18n: ${files} translation files complete in vi and en`)
