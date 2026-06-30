import { gzipSync } from 'node:zlib';
import type { Plugin } from 'vite';

const GZIP_MIN_BYTES = 256;
const GZIP_EXTENSIONS = /\.(?:js|mjs|css|svg|xml|txt|json|html)$/i;

/** Emit pre-compressed .gz assets for nginx gzip_static. */
export function gzipStaticAssets(): Plugin {
  return {
    name: 'gzip-static-assets',
    apply: 'build',
    generateBundle(_options, bundle) {
      for (const [fileName, chunk] of Object.entries(bundle)) {
        if (chunk.type !== 'asset' && chunk.type !== 'chunk') {
          continue;
        }
        if (!GZIP_EXTENSIONS.test(fileName)) {
          continue;
        }
        const source = chunk.type === 'asset' ? chunk.source : chunk.code;
        if (typeof source !== 'string' || source.length < GZIP_MIN_BYTES) {
          continue;
        }
        this.emitFile({
          type: 'asset',
          fileName: `${fileName}.gz`,
          source: gzipSync(Buffer.from(source, 'utf8')),
        });
      }
    },
  };
}
