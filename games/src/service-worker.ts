/// <reference types="@sveltejs/kit" />
/// <reference no-default-lib="true"/>
/// <reference lib="esnext" />
/// <reference lib="webworker" />

/**
 * Service worker nativo do SvelteKit: pré-cacheia o build inteiro (JS, CSS,
 * fontes, ícones, páginas pré-renderizadas) e serve cache-first. Os jogos são
 * 100% client-side, então funcionam offline depois da primeira visita.
 * Cada deploy gera um `version` novo → cache novo, antigos são apagados.
 */
import { build, files, prerendered, version } from '$service-worker';

declare const self: ServiceWorkerGlobalScope;

const CACHE = `games-${version}`;

/**
 * Arquivos de configuração do Cloudflare Pages. Moram em `static/` para irem
 * parar na raiz do build, mas o Pages os consome como configuração e nunca os
 * serve — pedir por eles dá 404.
 */
const PAGES_CONFIG = ['/_headers', '/_redirects', '/_routes.json'];

const ASSETS = [...build, ...files.filter((f) => !PAGES_CONFIG.includes(f)), ...prerendered];

/**
 * Páginas pré-renderizadas da mais específica para a mais genérica — `/` casa
 * com tudo e fica por último. Jogo novo entra sozinho, sem editar nada aqui.
 */
const PAGES = [...prerendered].sort((a, b) => b.length - a.length);

self.addEventListener('install', (event) => {
  event.waitUntil(
    (async () => {
      const cache = await caches.open(CACHE);
      // Um `cache.addAll` falha inteiro se um único asset falhar — e sem
      // instalação não existe offline nenhum. Cacheia um a um: o que faltar
      // vira um aviso, não a perda do service worker.
      const results = await Promise.allSettled(ASSETS.map((asset) => cache.add(asset)));
      const failed = results.filter((r) => r.status === 'rejected').length;
      if (failed) console.warn(`[sw] ${failed} de ${ASSETS.length} assets ficaram fora do cache`);
      await self.skipWaiting();
    })()
  );
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) => Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k))))
      .then(() => self.clients.claim())
  );
});

self.addEventListener('fetch', (event) => {
  const { request } = event;
  if (request.method !== 'GET') return;
  const url = new URL(request.url);
  if (url.origin !== self.location.origin) return;

  event.respondWith(
    (async () => {
      const cache = await caches.open(CACHE);
      const cached = await cache.match(request, { ignoreSearch: true });
      if (cached) return cached;
      try {
        const response = await fetch(request);
        if (response.ok && response.type === 'basic') cache.put(request, response.clone());
        return response;
      } catch {
        // Sem rede: navegação cai na página do jogo ou no hub, que estão no cache.
        if (request.mode === 'navigate') {
          const page = PAGES.find((p) => url.pathname.startsWith(p.slice(0, -1))) ?? '/';
          const fallback = await cache.match(page);
          if (fallback) return fallback;
        }
        return new Response('Offline', { status: 503, headers: { 'Content-Type': 'text/plain' } });
      }
    })()
  );
});
