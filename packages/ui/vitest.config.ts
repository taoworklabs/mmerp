import { playwright } from '@vitest/browser-playwright'
import { storybookTest } from '@storybook/addon-vitest/vitest-plugin'
import { defineConfig } from 'vitest/config'

// Each project needs its own browser config object; a shared reference confuses project naming.
const browser = () => ({ enabled: true, provider: playwright(), headless: true, instances: [{ browser: 'chromium' as const }] })

export default defineConfig({
  test: {
    projects: [
      {
        test: {
          name: 'unit',
          include: ['src/**/*.test.{ts,tsx}'],
          browser: browser(),
        },
      },
      {
        plugins: [storybookTest({ configDir: '.storybook' })],
        test: {
          name: 'storybook',
          browser: browser(),
        },
      },
    ],
  },
})
