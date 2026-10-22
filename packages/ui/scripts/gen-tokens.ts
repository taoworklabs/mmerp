import { writeFileSync } from 'node:fs'
import { renderTokensCss } from '../src/tokens/render.ts'

writeFileSync(new URL('../src/tokens/tokens.css', import.meta.url), renderTokensCss())
