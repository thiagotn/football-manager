<script lang="ts">
  import '../app.css';
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { authStore, isLoggedIn } from '$lib/stores/auth';
  import { themeStore } from '$lib/stores/theme';
  import Navbar from '$lib/components/Navbar.svelte';
  import Toast from '$lib/components/Toast.svelte';

  const PUBLIC_ROUTES = ['/login', '/invite', '/match', '/faq', '/lp'];

  let betaDismissed = $state(false);

  onMount(async () => {
    themeStore.init();
    betaDismissed = sessionStorage.getItem('beta-dismissed') === '1';
    await authStore.init();
  });

  function dismissBeta() {
    betaDismissed = true;
    sessionStorage.setItem('beta-dismissed', '1');
  }

  let isAppPage = $derived(
    $isLoggedIn
    && !$page.url.pathname.startsWith('/login')
    && !$page.url.pathname.startsWith('/invite')
    && !$page.url.pathname.startsWith('/match')
    && !$page.url.pathname.startsWith('/lp')
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
  {#if !betaDismissed}
    <div class="bg-yellow-50 dark:bg-yellow-900/20 border-b border-yellow-200 dark:border-yellow-800 px-4 py-2 flex items-center justify-between gap-4 text-xs text-yellow-800 dark:text-yellow-300">
      <span>
        <strong>Versão Beta:</strong> este produto ainda esta em desenvolvimento. Funcionalidades podem mudar e dados podem ser resetados sem aviso.
      </span>
      <button onclick={dismissBeta} class="shrink-0 text-yellow-600 hover:text-yellow-900 font-semibold">Dispensar</button>
    </div>
  {/if}
{/if}
<slot />
