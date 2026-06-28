/// <reference types="vitest/config" />
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath, URL } from 'node:url';
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { apiProxyConfig, suppressProxyErrorLogs } from './vite.proxy';

const repoRoot = fileURLToPath(new URL('../../', import.meta.url));

function parseSimpleToml(src: string): Record<string, string> {
  const out: Record<string, string> = {};
  for (const line of src.split('\n')) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) continue;
    const eq = trimmed.indexOf('=');
    if (eq < 0) continue;
    const key = trimmed.slice(0, eq).trim();
    let val = trimmed.slice(eq + 1).trim();
    if (
      (val.startsWith('"') && val.endsWith('"')) ||
      (val.startsWith("'") && val.endsWith("'"))
    ) {
      val = val.slice(1, -1);
    }
    out[key] = val;
  }
  return out;
}

function applyWebToml() {
  const tomlPath = path.join(repoRoot, 'web.toml');
  if (!fs.existsSync(tomlPath)) return;
  const parsed = parseSimpleToml(fs.readFileSync(tomlPath, 'utf8'));
  const map: Record<string, string> = {
    api_base_url: 'VITE_API_BASE_URL',
    log_level: 'VITE_LOG_LEVEL',
    launcher_download_url: 'VITE_LAUNCHER_DOWNLOAD_URL',
  };
  for (const [tomlKey, envKey] of Object.entries(map)) {
    const val = parsed[tomlKey];
    if (val && !process.env[envKey]) {
      process.env[envKey] = val;
    }
  }
}

applyWebToml();

export default defineConfig({
  plugins: [suppressProxyErrorLogs(), react()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  test: {
    environment: 'jsdom',
    testTimeout: 15000,
    fileParallelism: false,
    exclude: ['**/node_modules/**', '**/dist/**', 'e2e/**'],
    setupFiles: ['./src/test/setup.ts'],
    coverage: {
      provider: 'v8',
      include: ['src/**/*.{ts,tsx}'],
      exclude: ['src/vite-env.d.ts', 'src/main.tsx', 'src/hooks/useMessage.ts'],
      thresholds: {
        statements: 94,
        branches: 86,
        functions: 97,
        lines: 95,
      },
    },
  },
  server: {
    port: 5173,
    proxy: apiProxyConfig(),
  },
});
