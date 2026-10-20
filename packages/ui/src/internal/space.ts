import type { Space } from '../tokens/tokens.ts'

// Resolves a spacing prop to its token variable; an explicit 0 (no token at that key) renders as a literal zero.
export function spaceVar(value: Space | 0 | undefined, fallback: Space): string {
  if (value === 0) return '0px'
  return `var(--mmerp-space-${value ?? fallback})`
}
