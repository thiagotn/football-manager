<script lang="ts">
  import '../app.css';
  import '@fontsource/inter/400.css';
  import '@fontsource/inter/500.css';
  import '@fontsource/inter/600.css';
  import { onMount } from 'svelte';
  import { browser } from '$app/environment';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { authStore, isLoggedIn } from '$lib/stores/auth';
  import { themeStore } from '$lib/stores/theme';
  import { toastInfo } from '$lib/stores/toast';
  import { pwaInstall } from '$lib/stores/pwaInstall';
  import { sessionExpiredStore } from '$lib/stores/sessionExpired';
  import Navbar from '$lib/components/Navbar.svelte';
  import Toast from '$lib/components/Toast.svelte';
  import AndroidBetaBanner from '$lib/components/AndroidBetaBanner.svelte';
  import { initLocale, t } from '$lib/i18n';

  const PUBLIC_ROUTES = ['/login', '/register', '/invite', '/match/', '/faq', '/lp', '/terms', '/privacy', '/ranking', '/discover', '/players/', '/draw', '/simulator', '/tetris'];

  onMount(() => {
    themeStore.init();
    pwaInstall.init();
    authStore.init();
    initLocale();
  });

  // $effect é registrado na inicialização do componente (antes de qualquer onMount),
  // então detecta a mudança no store independente da ordem de onMount pai/filho.
  $effect(() => {
    if ($sessionExpiredStore) {
      sessionExpiredStore.set(false);
      toastInfo($t('layout.session_expired'));
      goto('/login?expired=1');
    }
  });

  let isAppPage = $derived(
    $isLoggedIn
    && !$page.url.pathname.startsWith('/login')
    && !$page.url.pathname.startsWith('/invite')
    && !$page.url.pathname.startsWith('/lp')
    && !$page.url.pathname.startsWith('/terms')
    && !$page.url.pathname.startsWith('/privacy')
  );

  $effect(() => {
    if (!$authStore.loading) {
      const isPublic = PUBLIC_ROUTES.some(r => $page.url.pathname.startsWith(r));
      if (!$isLoggedIn && !isPublic) {
        if ($page.url.pathname === '/') {
          const isMobile = window.innerWidth < 768;
          goto(isMobile ? '/login' : '/lp');
        } else {
          goto('/login');
        }
      }
    }
  });
</script>

<Toast />
{#if isAppPage}
  <Navbar />
{/if}
<slot />
{#if browser && !$page.url.pathname.startsWith('/admin')}
  <AndroidBetaBanner />
{/if}
{#if isAppPage}
  <footer class="hidden min-[940px]:flex border-t border-gray-200 dark:border-gray-700 py-4 px-6 text-center text-xs text-gray-400 dark:text-gray-500 flex-wrap items-center justify-center gap-4">
    <a href="/terms" class="hover:text-gray-600 dark:hover:text-gray-300 transition-colors">{$t('footer.terms')}</a>
    <a href="/privacy" class="hover:text-gray-600 dark:hover:text-gray-300 transition-colors">{$t('footer.privacy')}</a>
    <a href="/faq" class="hover:text-gray-600 dark:hover:text-gray-300 transition-colors">{$t('footer.faq')}</a>
    <a href="https://status.rachao.app" target="_blank" rel="noopener noreferrer" class="hover:text-gray-600 dark:hover:text-gray-300 transition-colors">{$t('footer.status')}</a>
  </footer>
{/if}
