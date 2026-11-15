/**
 * Copia para `static/be-a-ba/img/` só os desenhos da Twemoji que o Bê-á-Bá usa.
 *
 * Os códigos são lidos dos arquivos de conteúdo (`img: '1f347'`, a lista
 * `ITEMS` e as constantes de `art.ts`). Desenhos próprios (sem código
 * hexadecimal, como `mesa`) já moram em `static/be-a-ba/img/` e são ignorados.
 *
 * Uso: npm run twemoji   (precisa de `twemoji-svg` instalado — devDependency)
 * Licença dos desenhos: Twemoji, CC-BY 4.0 — créditos na área dos pais.
 */
import { copyFileSync, mkdirSync, readFileSync, existsSync } from 'node:fs';
import { createRequire } from 'node:module';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const lib = join(root, 'src/lib/be-a-ba');
const out = join(root, 'static/be-a-ba/img');
const require = createRequire(import.meta.url);
// O pacote só exporta `./*` → `dist/*.svg`: resolve um desenho qualquer para achar a pasta.
const dist = dirname(require.resolve('twemoji-svg/1f347'));

const codes = new Set();
for (const file of ['words.ts', 'stickers.ts', 'numbers.ts', 'art.ts']) {
  const src = readFileSync(join(lib, file), 'utf8');
  for (const m of src.matchAll(/'([0-9a-f]{4,5}(?:-[0-9a-f]{4,5})*)'/g)) codes.add(m[1]);
}

mkdirSync(out, { recursive: true });
let missing = 0;
for (const code of [...codes].sort()) {
  const from = join(dist, `${code}.svg`);
  if (!existsSync(from)) {
    console.error('✗ sem desenho na Twemoji:', code);
    missing++;
    continue;
  }
  copyFileSync(from, join(out, `${code}.svg`));
}
console.log(`✓ ${codes.size - missing} desenhos em static/be-a-ba/img/`);
if (missing) process.exit(1);
