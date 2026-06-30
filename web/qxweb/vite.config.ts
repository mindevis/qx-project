/// <reference types="vitest/config" />
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath, URL } from 'node:url';
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { apiProxyConfig, suppressProxyErrorLogs } from './vite.proxy';
import { seoStaticPlugin } from './vite.seo';

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
    site_url: 'VITE_SITE_URL',
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
  plugins: [suppressProxyErrorLogs(), seoStaticPlugin(), react()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  test: {
    environment: 'jsdom',
    testTimeout: 15000,
    fileParallelism: true,
    exclude: ['**/node_modules/**', '**/dist/**', 'e2e/**'],
    setupFiles: ['./src/test/setup.ts'],
    coverage: {
      provider: 'v8',
      include: ['src/**/*.{ts,tsx}'],
      exclude: ['src/vite-env.d.ts', 'src/main.tsx', 'src/hooks/useMessage.ts', 'src/test/skinview3d-mock.ts'],
      thresholds: {
        statements: 88,
        branches: 77,
        functions: 88,
        lines: 89,
      },
    },
  },
  server: {
    port: 5173,
    proxy: apiProxyConfig(),
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) {
            return;
          }
          if (id.includes('react-router') || /[/\\]react[/\\]/.test(id) || /[/\\]react-dom[/\\]/.test(id)) {
            return 'vendor-react';
          }
          if (id.includes('antd') || id.includes('@ant-design')) {
            return 'vendor-antd';
          }
          if (id.includes('skinview3d') || id.includes('three')) {
            return 'vendor-3d';
          }
        },
      },
    },
  },
});
