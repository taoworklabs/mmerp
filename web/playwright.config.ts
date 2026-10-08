import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: 'e2e',
  // Tests share one admin user and change its language, so they run one at a time.
  workers: 1,
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
    { name: 'chromium', dependencies: ['setup'] },
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
