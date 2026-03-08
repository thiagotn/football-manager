import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import { VitePWA } from 'vite-plugin-pwa';

export default defineConfig({
  plugins: [
    sveltekit(),
    VitePWA({
      registerType: 'autoUpdate',
      manifest: {
        name: 'rachao.app',
        short_name: 'rachao.app',
        description: 'Organize grupos de futebol, convide jogadores e controle presenças em um clique.',
        theme_color: '#15803d',
        background_color: '#111827',
        display: 'standalone',
        start_url: '/',
        orientation: 'portrait',
        icons: [
          { src: '/logo-192.png', sizes: '192x192', type: 'image/png', purpose: 'any' },
          { src: '/logo-512.png', sizes: '512x512', type: 'image/png', purpose: 'any' },
          { src: '/logo-maskable-512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
        ],
      },
      workbox: {
        globPatterns: ['**/*.{js,css,html,ico,png,svg,webp,jpg,woff,woff2}'],
        globIgnores: ['**/background-login.png'],
        navigateFallback: '/offline',
        navigateFallbackDenylist: [/^\/api/],
        runtimeCaching: [
          {
            urlPattern: ({ url }) => url.pathname.startsWith('/api'),
            handler: 'NetworkOnly',
          },
        ],
      },
      devOptions: {
        enabled: false,
      },
    }),
  ],
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: process.env.API_URL || 'http://localhost:8000',
        changeOrigin: true
      }
    }
  }
});
