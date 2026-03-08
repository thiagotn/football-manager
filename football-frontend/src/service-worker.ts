/// <reference types="@sveltejs/kit/types/ambient" />
/// <reference no-default-lib="true"/>
/// <reference lib="esnext" />
/// <reference lib="webworker" />

import { cleanupOutdatedCaches, precacheAndRoute } from 'workbox-precaching';
import { registerRoute, NavigationRoute } from 'workbox-routing';
import { NetworkOnly } from 'workbox-strategies';

declare const self: ServiceWorkerGlobalScope;

cleanupOutdatedCaches();
precacheAndRoute(self.__WB_MANIFEST);

// API: sempre rede, nunca cache
registerRoute(
  ({ url }) => url.pathname.startsWith('/api'),
  new NetworkOnly()
);

// Push notifications
self.addEventListener('push', (event: PushEvent) => {
  if (!event.data) return;
  const data = event.data.json() as { title: string; body: string; url?: string };
  event.waitUntil(
    self.registration.showNotification(data.title, {
      body: data.body,
      icon: '/logo-192.png',
      badge: '/logo-192.png',
      data: { url: data.url ?? '/' },
    })
  );
});

self.addEventListener('notificationclick', (event: NotificationEvent) => {
  event.notification.close();
  const url: string = event.notification.data?.url ?? '/';
  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
      const existing = clientList.find((c) => c.url.includes(url)) as WindowClient | undefined;
      if (existing) return existing.focus();
      return self.clients.openWindow(url);
    })
  );
});

// Navegação: tenta rede; em erro de rede OU resposta 5xx → exibe /offline
registerRoute(
  new NavigationRoute(async ({ request }) => {
    try {
      const response = await fetch(request);
      if (response.ok || response.type === 'opaqueredirect') return response;
      // 502, 503, 504, etc.
      const cached = await caches.match('/offline');
      return cached ?? response;
    } catch {
      // sem rede
      const cached = await caches.match('/offline');
      return cached ?? new Response('Sem conexão', { status: 503, headers: { 'Content-Type': 'text/plain' } });
    }
  }, { denylist: [/^\/api/] })
);
