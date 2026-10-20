// Fails unless dependency-cruiser and ESLint report exactly the violations planted in lint-fixtures.
import { execFileSync } from 'node:child_process'
import { readdirSync, readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { ESLint } from 'eslint'

const dir = new URL('../lint-fixtures/', import.meta.url)
const expected = (file) => new Set(JSON.parse(readFileSync(new URL(file, dir), 'utf8')).map(([rule, from]) => `${rule} ${from}`))
let failed = false

function compare(tool, got, want) {
  const missed = [...want].filter((v) => !got.has(v))
  const extra = [...got].filter((v) => !want.has(v))
  for (const v of missed) console.error(`${tool} missed: ${v}`)
  for (const v of extra) console.error(`${tool} unexpected: ${v}`)
  if (missed.length || extra.length) failed = true
  else console.log(`lint-fixtures: all ${want.size} planted ${tool} violations reported`)
}

const out = execFileSync(
  '../node_modules/.bin/depcruise',
  ['src', '--config', '../.dependency-cruiser.cjs', '--ts-config', 'tsconfig.json', '--output-type', 'json'],
  { cwd: dir, encoding: 'utf8', maxBuffer: 1 << 26 },
)
compare('dependency-cruiser', new Set(JSON.parse(out).summary.violations.map((v) => `${v.rule.name} ${v.from.replace(/^src\//, '')}`)), expected('expected.json'))

// Each ESLint fixture is linted as if it sat at the same path under src, so path-scoped rules apply as in the app.
const eslintDir = new URL('eslint/', dir)
const eslint = new ESLint({ cwd: fileURLToPath(new URL('..', dir)) })
const got = new Set()
for (const rel of readdirSync(eslintDir, { recursive: true })) {
  if (!/\.tsx?$/.test(rel)) continue
  const [result] = await eslint.lintText(readFileSync(new URL(rel, eslintDir), 'utf8'), { filePath: `src/${rel}` })
  for (const m of result.messages) got.add(`${m.ruleId ?? m.message} ${rel}`)
}
compare('ESLint', got, expected('eslint/expected.json'))

if (failed) process.exit(1)
