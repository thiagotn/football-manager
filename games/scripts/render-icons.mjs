/**
 * Gera os PNGs dos ícones a partir dos SVGs em scripts/icons/.
 * Uso: npm run icons  (precisa de Playwright com Chromium instalado)
 */
import { createRequire } from 'node:module';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const require = createRequire(import.meta.url);
const { chromium } = require('playwright');
const root = join(dirname(fileURLToPath(import.meta.url)), '..');

const JOBS = [
  ['enche-ai.svg', 'static/enche-ai/icon-192.png', 192],
  ['enche-ai.svg', 'static/enche-ai/icon-512.png', 512],
  ['enche-ai.svg', 'static/enche-ai/apple-touch-icon.png', 180],
  ['enche-ai-maskable.svg', 'static/enche-ai/maskable-512.png', 512],
  ['hub.svg', 'static/icon-192.png', 192],
  ['hub.svg', 'static/icon-512.png', 512],
  ['hub.svg', 'static/maskable-512.png', 512],
  ['hub.svg', 'static/apple-touch-icon.png', 180],
];

const browser = await chromium.launch();
const page = await browser.newPage();
for (const [src, out, size] of JOBS) {
  const svg = readFileSync(join(root, 'scripts/icons', src), 'utf8')
    .replace(/width="512" height="512"/, `width="${size}" height="${size}"`);
  await page.setViewportSize({ width: size, height: size });
  await page.setContent(`<html><body style="margin:0">${svg}</body></html>`);
  await page.locator('svg').first().screenshot({ path: join(root, out), omitBackground: false });
  console.log('✓', out);
}
await browser.close();
