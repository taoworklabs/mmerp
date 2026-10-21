import { describe, expect, it } from 'vitest'

// Raw source of every screen story, so the checks below run on the text itself rather than on
// what it renders: a screen may only be built from the kit, proving @mmerp/ui needs nothing else.
const files = import.meta.glob('./*.stories.tsx', { query: '?raw', import: 'default', eager: true }) as Record<string, string>

const allowedPrefixes = ['@storybook/', 'storybook/']
const allowedSpecifiers = new Set(['react', '@tabler/icons-react', '../index.ts', './samples.ts'])

function isAllowedSpecifier(specifier: string) {
  return allowedSpecifiers.has(specifier) || allowedPrefixes.some((prefix) => specifier.startsWith(prefix))
}

describe('screen stories', () => {
  it('has at least one screen story file', () => {
    expect(Object.keys(files).length).toBeGreaterThan(0)
  })

  for (const [path, source] of Object.entries(files)) {
    it(`${path} has no CSS module and no inline style`, () => {
      expect(source).not.toContain('.module.css')
      expect(source).not.toMatch(/style=/)
      expect(source).not.toMatch(/style:/)
    })

    it(`${path} imports only from the kit, react, tabler icons or storybook`, () => {
      const specifiers = [...source.matchAll(/^import[^'"]*from\s+['"]([^'"]+)['"]/gm)].map((match) => match[1]!)
      expect(specifiers.length).toBeGreaterThan(0)
      for (const specifier of specifiers) {
        expect(isAllowedSpecifier(specifier)).toBe(true)
      }
    })
  }
})
