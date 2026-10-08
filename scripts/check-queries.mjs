// Every internal/<tier>/<module>/queries.sql writes only its own schema and reads
// only its own schema and the modules of its tier and the tiers below.
// ponytail: product modules may not read other products yet; add declared product dependencies with the second product.
import { readFileSync, readdirSync, existsSync } from 'node:fs'

const root = new URL('../internal/', import.meta.url)
const tiers = ['core', 'shared', 'modules']
const modulesOf = (tier) =>
  existsSync(new URL(`${tier}/`, root))
    ? readdirSync(new URL(`${tier}/`, root), { withFileTypes: true }).filter((e) => e.isDirectory()).map((e) => e.name)
    : []
const all = Object.fromEntries(tiers.map((t) => [t, modulesOf(t)]))
const schemas = new Set(Object.values(all).flat())

let errors = 0
for (const [i, tier] of tiers.entries()) {
  for (const mod of all[tier]) {
    const file = new URL(`${tier}/${mod}/queries.sql`, root)
    if (!existsSync(file)) continue
    const sql = readFileSync(file, 'utf8').replace(/--.*$/gm, '')
    const readable = new Set(tier === 'modules' ? [mod, ...all.core, ...all.shared] : tiers.slice(0, i + 1).flatMap((t) => all[t]))
    const where = `internal/${tier}/${mod}/queries.sql`
    for (const [, schema] of sql.matchAll(/\b(?:INSERT\s+INTO|UPDATE|DELETE\s+FROM)\s+([a-z_]+)\./gi)) {
      if (schema !== mod) { console.error(`${where}: writes schema ${schema}`); errors++ }
    }
    for (const [, schema] of sql.matchAll(/\b([a-z_]+)\.[a-z_]+/gi)) {
      if (schemas.has(schema) && !readable.has(schema)) { console.error(`${where}: reads schema ${schema}`); errors++ }
    }
  }
}
if (errors) process.exit(1)
console.log('queries.sql: schema access within bounds')
