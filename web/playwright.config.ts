import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  timeout: 60_000,
  use: {
    baseURL: "http://127.0.0.1:4173",
    trace: "on-first-retry"
  },
  webServer: [
    {
      command: "/bin/zsh -lc 'CCP_STORAGE_DRIVER=memory CCP_ENV=development CCP_API_HOST=127.0.0.1 CCP_API_PORT=18085 go run ./cmd/api'",
      cwd: "..",
      url: "http://127.0.0.1:18085/readyz",
      reuseExistingServer: !process.env.CI,
      timeout: 120_000
    },
    {
      command: "/bin/zsh -lc 'VITE_API_BASE_URL=http://127.0.0.1:18085 pnpm dev --host 127.0.0.1 --port 4173'",
      url: "http://127.0.0.1:4173",
      reuseExistingServer: !process.env.CI,
      timeout: 120_000
    }
  ]
});
