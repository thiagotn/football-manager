<script lang="ts">
  import '../app.css';
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { authStore, isLoggedIn } from '$lib/stores/auth';
  import { themeStore } from '$lib/stores/theme';
  import { toastInfo } from '$lib/stores/toast';
  import { pwaInstall } from '$lib/stores/pwaInstall';
  import Navbar from '$lib/components/Navbar.svelte';
  import Toast from '$lib/components/Toast.svelte';

  const PUBLIC_ROUTES = ['/login', '/register', '/invite', '/match/', '/faq', '/lp', '/terms', '/privacy'];

  onMount(async () => {
    themeStore.init();
    pwaInstall.init();
    await authStore.init();

    const handleSessionExpired = () => {
      authStore.logout();
      toastInfo('Sua sessão expirou. Faça login novamente.');
      sessionStorage.setItem('session_expired', '1');
      goto('/login');
    };
    window.addEventListener('session-expired', handleSessionExpired);
    return () => window.removeEventListener('session-expired', handleSessionExpired);
  });

  let isAppPage = $derived(
    $isLoggedIn
    && !$page.url.pathname.startsWith('/login')
    && !$page.url.pathname.startsWith('/invite')
    && !$page.url.pathname.startsWith('/match/')
    && !$page.url.pathname.startsWith('/lp')
    && !$page.url.pathname.startsWith('/terms')
    && !$page.url.pathname.startsWith('/privacy')
  );

  $effect(() => {
    if (!$authStore.loading) {
      const isPublic = PUBLIC_ROUTES.some(r => $page.url.pathname.startsWith(r));
      if (!$isLoggedIn && !isPublic) {
        goto($page.url.pathname === '/' ? '/lp' : '/login');
      }
    }
  });
</script>

<Toast />
{#if isAppPage}
  <Navbar />
{/if}
<slot />
{#if isAppPage}
  <footer class="hidden min-[940px]:flex border-t border-gray-200 dark:border-gray-700 py-4 px-6 text-center text-xs text-gray-400 dark:text-gray-500 flex-wrap items-center justify-center gap-4">
    <a href="/terms" class="hover:text-gray-600 dark:hover:text-gray-300 transition-colors">Termos de Uso</a>
    <a href="/privacy" class="hover:text-gray-600 dark:hover:text-gray-300 transition-colors">Política de Privacidade</a>
    <a href="/faq" class="hover:text-gray-600 dark:hover:text-gray-300 transition-colors">FAQ</a>
    <a href="https://status.rachao.app" target="_blank" rel="noopener noreferrer" class="hover:text-gray-600 dark:hover:text-gray-300 transition-colors">Status</a>
  </footer>
{/if}
