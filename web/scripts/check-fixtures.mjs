// Fails unless dependency-cruiser reports exactly the violations planted in lint-fixtures.
import { execFileSync } from 'node:child_process'
import { readFileSync } from 'node:fs'

const dir = new URL('../lint-fixtures/', import.meta.url)
const out = execFileSync(
  '../node_modules/.bin/depcruise',
  ['src', '--config', '../.dependency-cruiser.cjs', '--ts-config', 'tsconfig.json', '--output-type', 'json'],
  { cwd: dir, encoding: 'utf8', maxBuffer: 1 << 26 },
)
const got = new Set(JSON.parse(out).summary.violations.map((v) => `${v.rule.name} ${v.from.replace(/^src\//, '')}`))
const want = new Set(JSON.parse(readFileSync(new URL('expected.json', dir), 'utf8')).map(([rule, from]) => `${rule} ${from}`))

const missed = [...want].filter((v) => !got.has(v))
const extra = [...got].filter((v) => !want.has(v))
for (const v of missed) console.error(`missed: ${v}`)
for (const v of extra) console.error(`unexpected: ${v}`)
if (missed.length || extra.length) process.exit(1)
console.log(`lint-fixtures: all ${want.size} planted violations reported`)
