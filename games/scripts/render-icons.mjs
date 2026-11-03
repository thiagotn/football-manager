/**
 * Gera os PNGs dos ícones a partir dos SVGs em scripts/icons/.
 * Uso: npm run icons  (precisa de Playwright com Chromium instalado)
 *
 * Convenção, para não existir tabela a manter:
 *   <slug>.svg           → static/<slug>/{icon-192,icon-512,apple-touch-icon}.png
 *   <slug>-maskable.svg  → static/<slug>/maskable-512.png
 *   hub.svg              → os mesmos quatro na raiz de static/ (é o hub, não um jogo)
 */
import { createRequire } from 'node:module';
import { mkdirSync, readFileSync, readdirSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const require = createRequire(import.meta.url);
const { chromium } = require('playwright');
const root = join(dirname(fileURLToPath(import.meta.url)), '..');

/** `['icon-192.png', 192]` etc. — o que cada SVG comum gera. */
const PLAIN = [
  ['icon-192.png', 192],
  ['icon-512.png', 512],
  ['apple-touch-icon.png', 180],
];

const jobs = [];
for (const file of readdirSync(join(root, 'scripts/icons')).sort()) {
  if (!file.endsWith('.svg')) continue;
  const name = file.slice(0, -4);
  const maskable = name.endsWith('-maskable');
  const slug = maskable ? name.slice(0, -'-maskable'.length) : name;
  const dir = slug === 'hub' ? 'static' : `static/${slug}`;
  const outputs = maskable ? [['maskable-512.png', 512]] : PLAIN;
  for (const [out, size] of outputs) jobs.push([file, join(dir, out), size]);
}

const browser = await chromium.launch();
const page = await browser.newPage();
for (const [src, out, size] of jobs) {
  const svg = readFileSync(join(root, 'scripts/icons', src), 'utf8')
    .replace(/width="512" height="512"/, `width="${size}" height="${size}"`);
  mkdirSync(dirname(join(root, out)), { recursive: true });
  await page.setViewportSize({ width: size, height: size });
  await page.setContent(`<html><body style="margin:0">${svg}</body></html>`);
  await page.locator('svg').first().screenshot({ path: join(root, out), omitBackground: false });
  console.log('✓', out);
}
await browser.close();
