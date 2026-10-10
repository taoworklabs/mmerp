// Every internal/<tier>/<module>/queries.sql writes only its own schema and reads only
// its own schema, the modules of its tier and of the tiers below. A module in modules/
// also reads the other modules of its own product and of the products that product
// declares as dependencies in the composition root.
import { readFileSync, readdirSync, existsSync } from 'node:fs'

const tiers = ['core', 'shared', 'modules']

const dirsOf = (root, tier) =>
  existsSync(new URL(`${tier}/`, root))
    ? readdirSync(new URL(`${tier}/`, root), { withFileTypes: true }).filter((e) => e.isDirectory()).map((e) => e.name)
    : []

// productDeps reads the product dependency map the composition root declares, so the
// check and the running app can never disagree about who may read whom.
function productDeps(appFile) {
  const src = readFileSync(appFile, 'utf8')
  const start = src.indexOf('var products = map[string][]string{')
  if (start < 0) throw new Error(`check-queries: no product dependency map in ${appFile}`)
  const open = src.indexOf('{', start)
  let end = src.length
  for (let i = open, depth = 0; i < src.length; i++) {
    if (src[i] === '{') depth++
    else if (src[i] === '}' && --depth === 0) { end = i; break }
  }
  const deps = {}
  for (const [, product, list] of src.slice(open, end).matchAll(/"([a-z_]+)":\s*(nil|\{[^}]*\})/g)) {
    deps[product] = [...list.matchAll(/"([a-z_]+)"/g)].map(([, d]) => d)
  }
  return deps
}

// productOf names the product a module belongs to: its manifest declares it. A module with
// no routes has no manifest and is then read as a product of its own.
function productOf(root, mod, manifest) {
  const file = new URL(`modules/${mod}/${manifest}`, root)
  const found = existsSync(file) && readFileSync(file, 'utf8').match(/Product:\s*"([a-z_]+)"/)
  return found ? found[1] : mod
}

// scan returns one message per schema access outside a module's bounds. app is the file
// holding the product dependency map, manifest the basename of a module's manifest.
function scan(root, { app, manifest }) {
  const deps = productDeps(app)
  const all = Object.fromEntries(tiers.map((t) => [t, dirsOf(root, t)]))
  const schemas = new Set(Object.values(all).flat())
  const product = Object.fromEntries(all.modules.map((m) => [m, productOf(root, m, manifest)]))

  const errors = []
  for (const [i, tier] of tiers.entries()) {
    for (const mod of all[tier]) {
      const file = new URL(`${tier}/${mod}/queries.sql`, root)
      if (!existsSync(file)) continue
      const where = `internal/${tier}/${mod}/queries.sql`
      const sql = readFileSync(file, 'utf8').replace(/--.*$/gm, '')

      let readable
      if (tier === 'modules') {
        const mine = product[mod]
        if (!(mine in deps)) {
          errors.push(`${where}: product ${mine} is not in the dependency map of the composition root`)
        }
        const reachable = new Set([mine, ...(deps[mine] ?? [])])
        readable = new Set([...all.modules.filter((m) => reachable.has(product[m])), ...all.core, ...all.shared])
      } else {
        readable = new Set(tiers.slice(0, i + 1).flatMap((t) => all[t]))
      }

      for (const [, schema] of sql.matchAll(/\b(?:INSERT\s+INTO|UPDATE|DELETE\s+FROM)\s+([a-z_]+)\./gi)) {
        if (schema !== mod) errors.push(`${where}: writes schema ${schema}`)
      }
      for (const [, schema] of sql.matchAll(/\b([a-z_]+)\.[a-z_]+/gi)) {
        if (schemas.has(schema) && !readable.has(schema)) errors.push(`${where}: reads schema ${schema}`)
      }
    }
  }
  return errors
}

const root = new URL('../internal/', import.meta.url)
const real = scan(root, { app: new URL('app/app.go', root), manifest: 'module.go' })
for (const e of real) console.error(e)

// The fixtures plant one violation per rule, so the check cannot rot into always passing.
// Their Go files carry a .fixture suffix, so Go never compiles products that do not exist.
const fixtures = new URL('query-fixtures/internal/', import.meta.url)
const planted = scan(fixtures, { app: new URL('app/app.go.fixture', fixtures), manifest: 'module.go.fixture' })
const got = new Set(planted.map((e) => e.replace(/^internal\//, '')))
const want = new Set(JSON.parse(readFileSync(new URL('../expected.json', fixtures), 'utf8')))
const missed = [...want].filter((e) => !got.has(e))
const extra = [...got].filter((e) => !want.has(e))
for (const e of missed) console.error(`query-fixtures missed: ${e}`)
for (const e of extra) console.error(`query-fixtures unexpected: ${e}`)

if (real.length || missed.length || extra.length) process.exit(1)
console.log(`queries.sql: schema access within bounds; all ${want.size} planted violations reported`)
