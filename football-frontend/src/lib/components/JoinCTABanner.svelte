<script lang="ts">
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { isLoggedIn } from '$lib/stores/auth';
  import { t } from '$lib/i18n';

  let next = $derived(encodeURIComponent($page.url.pathname));

  function goLogin() {
    goto(`/login?next=${next}`, { replaceState: true });
  }
  function goRegister() {
    goto(`/register?next=${next}`, { replaceState: true });
  }
</script>

{#if !$isLoggedIn}
  <div class="fixed bottom-0 left-0 right-0 z-50 bg-gray-900/95 backdrop-blur-sm border-t border-gray-700 px-4 py-3"
       style="padding-bottom: calc(0.75rem + env(safe-area-inset-bottom, 0px))">
    <div class="max-w-lg mx-auto flex items-center gap-3">
      <p class="flex-1 text-sm font-medium text-white leading-tight">
        🏆 {$t('cta.banner_title')}
      </p>
      <button onclick={goLogin} class="btn btn-sm btn-secondary shrink-0">{$t('cta.login')}</button>
      <button onclick={goRegister} class="btn btn-sm btn-primary shrink-0">{$t('cta.register')}</button>
    </div>
  </div>
{/if}
