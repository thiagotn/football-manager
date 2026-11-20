/**
 * Copia para `static/<jogo>/img/` só os desenhos da Twemoji que cada jogo usa.
 *
 * Os códigos são lidos dos arquivos de conteúdo listados em `JOBS` (todo
 * literal como `'1f347'` ou `'1f9b8-200d-2640-fe0f'`). Desenhos próprios (sem
 * código hexadecimal, como `mesa`) já moram em `static/<jogo>/img/` e são
 * ignorados.
 *
 * Uso: npm run twemoji   (precisa de `twemoji-svg` instalado — devDependency)
 * Licença dos desenhos: Twemoji, CC-BY 4.0 — créditos nos ajustes de cada jogo.
 */
import { copyFileSync, mkdirSync, readFileSync, existsSync } from 'node:fs';
import { createRequire } from 'node:module';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const require = createRequire(import.meta.url);
// O pacote só exporta `./*` → `dist/*.svg`: resolve um desenho qualquer para achar a pasta.
const dist = dirname(require.resolve('twemoji-svg/1f347'));

const JOBS = [
  { slug: 'be-a-ba', files: ['words.ts', 'stickers.ts', 'numbers.ts', 'art.ts', 'english.ts'] },
  { slug: 'quebra-cuca', files: ['scenes.ts', 'art.ts'] },
];

let missing = 0;
for (const { slug, files } of JOBS) {
  const lib = join(root, 'src/lib', slug);
  const out = join(root, 'static', slug, 'img');
  const codes = new Set();
  for (const file of files) {
    const src = readFileSync(join(lib, file), 'utf8');
    for (const m of src.matchAll(/'([0-9a-f]{4,5}(?:-[0-9a-f]{4,5})*)'/g)) codes.add(m[1]);
  }
  mkdirSync(out, { recursive: true });
  let copied = 0;
  for (const code of [...codes].sort()) {
    const from = join(dist, `${code}.svg`);
    if (!existsSync(from)) {
      console.error(`✗ ${slug}: sem desenho na Twemoji:`, code);
      missing++;
      continue;
    }
    copyFileSync(from, join(out, `${code}.svg`));
    copied++;
  }
  console.log(`✓ ${copied} desenhos em static/${slug}/img/`);
}
if (missing) process.exit(1);
