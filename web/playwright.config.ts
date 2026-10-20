import { defineConfig, devices } from '@playwright/test'

// m6 and the approval-* files change approval rules other files rely on; m9 checks tenant-wide counts.
const exclusive = /\/(m6|m9|approval-flows|approval-actions|approval-lines|approval-delegation)\.spec\.ts$/

export default defineConfig({
  testDir: 'e2e',
  // Files run in parallel; each creates its own data under a unique suffix. Tests inside a
  // file share that data, so they stay in order. More workers starve the one server.
  workers: 4,
  // Some tests are a whole user journey; sharing the server, they outgrow the 30 s default.
  timeout: 60_000,
  forbidOnly: !!process.env.CI,
  reporter: process.env.CI ? 'github' : 'list',
  use: {
    // E2E_BASE_URL points at another server, e.g. make dev on :8080 for make seed.
    baseURL: process.env.E2E_BASE_URL ?? 'http://localhost:8090',
    locale: 'vi-VN',
    trace: 'retain-on-failure',
    ...devices['Desktop Chrome'],
  },
  projects: [
    { name: 'setup', testMatch: /\.setup\.ts$/ },
    { name: 'chromium', dependencies: ['setup'], testIgnore: exclusive },
    // Files that change or count tenant-wide state run alone, after the rest.
    { name: 'exclusive', testMatch: exclusive, dependencies: ['chromium'], workers: 1 },
    // Sample data for a dev database; only with E2E_SEED, so the e2e run never adds it.
    ...(process.env.E2E_SEED ? [{ name: 'seed', testMatch: /\.seed\.ts$/, dependencies: ['setup'] }] : []),
  ],
  // E2E_EXTERNAL=1 runs against servers already up on :8090 and :8091, e.g. on the dev database.
  webServer: process.env.E2E_EXTERNAL ? undefined : {
    command: '../scripts/e2e-server.sh',
    url: 'http://localhost:8090/healthz',
    timeout: 300_000,
    reuseExistingServer: false,
    stdout: 'pipe',
  },
})
