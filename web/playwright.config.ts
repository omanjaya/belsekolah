import { defineConfig, devices } from '@playwright/test'

const port = Number(process.env.PORT ?? 18080)
export const DATA_DIR = `.e2e-data-${port}`

export default defineConfig({
  testDir: 'e2e',
  fullyParallel: false,
  workers: 1,
  timeout: 60_000,
  expect: { timeout: 10_000 },
  reporter: [['list']],
  use: {
    baseURL: process.env.BEL_URL ?? `http://127.0.0.1:${port}`,
    locale: 'id-ID',
    trace: 'retain-on-failure',
  },
  projects: [
    { name: 'siapkan', testMatch: /siapkan\.setup\.ts/ },
    {
      name: 'hp',
      use: { ...devices['Pixel 7'], storageState: 'e2e/.auth.json' },
      dependencies: ['siapkan'],
      testIgnore: /siapkan\.setup\.ts/,
    },
  ],
  webServer: process.env.BEL_URL
    ? undefined
    : {
        command: `rm -rf ${DATA_DIR} && npm run build && cd .. && go run ./cmd/belsekolah -data web/${DATA_DIR} -addr 127.0.0.1:${port} -tz Asia/Jakarta`,
        url: `http://127.0.0.1:${port}/api/sesi`,
        reuseExistingServer: false,
        timeout: 180_000,
      },
})
