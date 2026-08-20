import type { Plugin, ProxyOptions } from 'vite';

const API_TARGET = 'http://localhost:3000';

function writeBackendUnavailable(res: unknown) {
  if (!res || typeof res !== 'object' || !('writeHead' in res)) {
    return;
  }

  const response = res as {
    headersSent?: boolean;
    writeHead: (status: number, headers?: Record<string, string>) => void;
    end: (chunk?: string) => void;
  };
  if (response.headersSent) {
    return;
  }

  response.writeHead(502, { 'Content-Type': 'application/json' });
  response.end(
    JSON.stringify({
      error: { code: 'BACKEND_UNAVAILABLE', message: 'Backend unavailable' },
    }),
  );
}

function upstreamProxy(
  target: string,
  prefix: string,
  extraHeaders?: Record<string, string>,
): ProxyOptions {
  return {
    target,
    changeOrigin: true,
    secure: true,
    rewrite: (path) => path.replace(new RegExp(`^/upstream/${prefix}`), ''),
    ...(extraHeaders ? { headers: extraHeaders } : {}),
  };
}

export function apiProxyConfig(): Record<string, ProxyOptions> {
  return {
    '/api': {
      target: API_TARGET,
      changeOrigin: true,
      ws: true,
      configure(proxy) {
        proxy.on('error', (_err, _req, res) => {
          writeBackendUnavailable(res);
        });
      },
    },
    '/upstream/forge': upstreamProxy('https://files.minecraftforge.net', 'forge'),
    '/upstream/mavenforge': upstreamProxy('https://maven.minecraftforge.net', 'mavenforge'),
    '/upstream/papermc': upstreamProxy('https://fill.papermc.io', 'papermc', {
      // Fill v3 rejects generic User-Agent values.
      'User-Agent': 'QXProject/1.0 (https://github.com/qxproject/qx)',
    }),
    '/upstream/purpur': upstreamProxy('https://api.purpurmc.org', 'purpur'),
    '/upstream/neoforge': upstreamProxy('https://maven.neoforged.net', 'neoforge'),
    '/upstream/fabric': upstreamProxy('https://meta.fabricmc.net', 'fabric'),
    '/upstream/quilt': upstreamProxy('https://meta.quiltmc.org', 'quilt'),
    '/upstream/mohist': upstreamProxy('https://mohistmc.com', 'mohist'),
    '/upstream/magma': upstreamProxy('https://magmafoundation.org', 'magma'),
    '/upstream/arclight': upstreamProxy('https://files.hypertention.cn', 'arclight'),
  };
}

/** Dev-only: Vite logs ECONNREFUSED on every proxied request when the API is down. */
export function suppressProxyErrorLogs(): Plugin {
  return {
    name: 'qxweb-suppress-proxy-errors',
    apply: 'serve',
    configureServer(server) {
      const logger = server.config.logger;
      const logError = logger.error.bind(logger);

      logger.error = (msg, options) => {
        if (
          typeof msg === 'string' &&
          (msg.includes('http proxy error') || msg.includes('ws proxy error'))
        ) {
          return;
        }
        logError(msg, options);
      };
    },
  };
}
