<script lang="ts">
  import { goto } from '$app/navigation';
  import { authStore, isAdmin } from '$lib/stores/auth';
  import { apiV2Admin, ApiError } from '$lib/api';
  import type { ApiV2UserItem } from '$lib/api';
  import { toastSuccess, toastError } from '$lib/stores/toast';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import { Zap, Search } from 'lucide-svelte';
  import { t } from '$lib/i18n';

  let users = $state<ApiV2UserItem[]>([]);
  let totalEnabled = $state(0);
  let search = $state('');
  let loading = $state(true);
  let error = $state('');
  let toggling = $state<Record<string, boolean>>({});

  let filtered = $derived(
    users.filter(u => {
      if (!search) return true;
      const q = search.toLowerCase();
      return u.name.toLowerCase().includes(q) || u.whatsapp.includes(q);
    })
  );

  async function load() {
    loading = true;
    error = '';
    try {
      const res = await apiV2Admin.listUsers();
      users = res.users;
      totalEnabled = res.total_enabled;
    } catch (e) {
      error = e instanceof ApiError ? e.message : $t('admin.apiv2.error_load');
    }
    loading = false;
  }

  async function toggleAccess(user: ApiV2UserItem) {
    toggling = { ...toggling, [user.id]: true };
    try {
      const updated = await apiV2Admin.updateAccess(user.id, !user.api_v2_enabled);
      users = users.map(u => u.id === updated.id ? updated : u);
      totalEnabled = users.filter(u => u.api_v2_enabled).length;
      toastSuccess(updated.api_v2_enabled ? $t('admin.apiv2.success_enabled') : $t('admin.apiv2.success_disabled'));
    } catch (e) {
      toastError(e instanceof ApiError ? e.message : $t('admin.apiv2.error_load'));
    }
    toggling = { ...toggling, [user.id]: false };
  }

  let loaded = false;
  $effect(() => {
    if ($authStore.loading) return;
    if (!$isAdmin) { goto('/dashboard', { replaceState: true }); return; }
    if (loaded) return;
    loaded = true;
    load();
  });

  function fmtDate(iso: string) {
    return new Date(iso).toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric' });
  }
</script>

<svelte:head><title>API v2 — Admin | rachao.app</title></svelte:head>

<PageBackground>
<main class="relative z-10 max-w-7xl mx-auto px-4 py-8">
  <div class="flex items-center justify-between mb-6">
    <div>
      <h1 class="text-2xl font-bold text-white flex items-center gap-2">
        <Zap size={24} class="text-primary-400" /> {$t('admin.apiv2.title')}
      </h1>
      <p class="text-sm text-white/60 mt-0.5">
        {$t('admin.apiv2.enabled_count', { count: totalEnabled, total: users.length })}
      </p>
    </div>
  </div>

  <!-- Search -->
  <div class="relative mb-4 max-w-sm">
    <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
    <input
      class="input pl-9"
      placeholder={$t('admin.apiv2.search_placeholder')}
      bind:value={search}
    />
  </div>

  {#if error}
    <div class="alert-error mb-4">{error}</div>
  {/if}

  {#if loading}
    <div class="card overflow-hidden">
      {#each [1,2,3,4,5] as _}
        <div class="px-6 py-4 border-b border-gray-100 dark:border-gray-700 animate-pulse">
          <div class="h-4 bg-gray-100 dark:bg-gray-700 rounded w-1/3"></div>
        </div>
      {/each}
    </div>
  {:else}
    <div class="card overflow-x-auto">
      <table class="table">
        <thead>
          <tr>
            <th>{$t('admin.apiv2.col_player')}</th>
            <th class="hidden sm:table-cell">{$t('admin.apiv2.col_whatsapp')}</th>
            <th class="hidden sm:table-cell">{$t('admin.apiv2.col_since')}</th>
            <th>{$t('admin.apiv2.col_access')}</th>
          </tr>
        </thead>
        <tbody>
          {#each filtered as user}
            <tr>
              <td class="font-medium">{user.name}</td>
              <td class="font-mono text-xs text-gray-600 hidden sm:table-cell">{user.whatsapp}</td>
              <td class="text-xs text-gray-500 hidden sm:table-cell">{fmtDate(user.created_at)}</td>
              <td>
                <button
                  onclick={() => toggleAccess(user)}
                  disabled={toggling[user.id]}
                  class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none disabled:opacity-50
                    {user.api_v2_enabled ? 'bg-green-500' : 'bg-gray-300 dark:bg-gray-600'}"
                  aria-label={user.api_v2_enabled ? $t('admin.apiv2.toggle_disable') : $t('admin.apiv2.toggle_enable')}
                >
                  <span class="inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform
                    {user.api_v2_enabled ? 'translate-x-6' : 'translate-x-1'}">
                  </span>
                </button>
              </td>
            </tr>
          {/each}
          {#if filtered.length === 0}
            <tr><td colspan="4" class="text-center text-gray-400 py-8">{$t('admin.apiv2.empty')}</td></tr>
          {/if}
        </tbody>
      </table>
    </div>
  {/if}
</main>
</PageBackground>
