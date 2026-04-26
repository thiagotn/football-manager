<script lang="ts">
  import { goto } from '$app/navigation';
  import { isLoggedIn, isAdmin } from '$lib/stores/auth';
  import { mcpTokens, ApiError } from '$lib/api';
  import type { MCPTokenResponse, MCPTokenCreated } from '$lib/api';
  import { t } from '$lib/i18n';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import MCPTokenCreateModal from '$lib/components/MCPTokenCreateModal.svelte';
  import { Key, Plus, Trash2 } from 'lucide-svelte';

  let tokens = $state<MCPTokenResponse[]>([]);
  let loading = $state(true);
  let error = $state('');
  let modalOpen = $state(false);
  let revokeTarget = $state<MCPTokenResponse | null>(null);
  let confirmOpen = $state(false);

  $effect(() => {
    if (!$isLoggedIn) { goto('/login'); return; }
    if ($isAdmin) { goto('/'); return; }
    mcpTokens.list()
      .then(d => { tokens = d; loading = false; })
      .catch(() => { error = 'Erro ao carregar tokens.'; loading = false; });
  });

  function formatDate(iso: string | null): string {
    if (!iso) return $t('mcp.never');
    return new Date(iso).toLocaleDateString('pt-BR', { day: '2-digit', month: 'short', year: 'numeric' });
  }

  function handleCreated(token: MCPTokenCreated) {
    tokens = [
      {
        id: token.id,
        name: token.name,
        token_prefix: token.token_prefix,
        expires_at: token.expires_at,
        created_at: token.created_at,
        last_used_at: null,
        is_expired: false,
      },
      ...tokens,
    ];
  }

  function openRevoke(token: MCPTokenResponse) {
    revokeTarget = token;
    confirmOpen = true;
  }

  async function confirmRevoke() {
    if (!revokeTarget) return;
    try {
      await mcpTokens.revoke(revokeTarget.id);
      tokens = tokens.filter(t => t.id !== revokeTarget!.id);
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Erro ao revogar token.';
    }
    revokeTarget = null;
  }
</script>

<PageBackground>
  <main class="relative z-10 max-w-4xl mx-auto px-4 py-8">
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-white flex items-center gap-2">
          <Key size={24} class="text-primary-400" /> {$t('mcp.title')}
        </h1>
        <p class="text-sm text-white/60 mt-0.5">{$t('mcp.subtitle')}</p>
      </div>
      <button
        onclick={() => { modalOpen = true; }}
        class="btn btn-primary btn-sm flex items-center gap-1.5"
      >
        <Plus size={16} /> {$t('mcp.generate')}
      </button>
    </div>

    {#if error}
      <p class="text-red-400 text-sm mb-4">{error}</p>
    {/if}

    {#if loading}
      <div class="text-primary-300 text-sm">Carregando…</div>
    {:else}
      <div class="bg-primary-800/60 border border-primary-700/50 rounded-2xl overflow-hidden">
        {#if tokens.length === 0}
          <div class="px-6 py-12 text-center text-primary-400 text-sm">
            {$t('mcp.empty_state')}
          </div>
        {:else}
          <!-- Desktop table -->
          <div class="hidden sm:block overflow-x-auto">
            <table class="w-full text-sm">
              <thead>
                <tr class="border-b border-primary-700/50 text-primary-400 text-xs uppercase tracking-wide">
                  <th class="px-4 py-3 text-left font-medium">Nome</th>
                  <th class="px-4 py-3 text-left font-medium">Token</th>
                  <th class="px-4 py-3 text-left font-medium hidden lg:table-cell">{$t('mcp.expires_at')}</th>
                  <th class="px-4 py-3 text-left font-medium hidden lg:table-cell">{$t('mcp.last_used')}</th>
                  <th class="px-4 py-3 text-right font-medium">Ações</th>
                </tr>
              </thead>
              <tbody>
                {#each tokens as token (token.id)}
                  <tr class="border-b border-primary-700/30 last:border-0 hover:bg-primary-700/20 transition-colors">
                    <td class="px-4 py-3 text-white font-medium">
                      {token.name}
                      {#if token.is_expired}
                        <span class="ml-2 text-[10px] bg-red-900/60 text-red-300 border border-red-700/40 rounded px-1.5 py-0.5">{$t('mcp.expired_badge')}</span>
                      {/if}
                    </td>
                    <td class="px-4 py-3 text-primary-300 font-mono text-xs">{token.token_prefix}…</td>
                    <td class="px-4 py-3 text-primary-300 hidden lg:table-cell">{formatDate(token.expires_at)}</td>
                    <td class="px-4 py-3 text-primary-300 hidden lg:table-cell">
                      {token.last_used_at ? formatDate(token.last_used_at) : $t('mcp.never_used')}
                    </td>
                    <td class="px-4 py-3 text-right">
                      <button
                        onclick={() => openRevoke(token)}
                        class="btn btn-ghost btn-sm text-red-400 hover:text-red-300 flex items-center gap-1 ml-auto"
                      >
                        <Trash2 size={14} /> {$t('mcp.revoke')}
                      </button>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>

          <!-- Mobile list -->
          <div class="sm:hidden divide-y divide-primary-700/30">
            {#each tokens as token (token.id)}
              <div class="px-4 py-4">
                <div class="flex items-start justify-between gap-3">
                  <div class="min-w-0">
                    <p class="text-white font-medium text-sm truncate">
                      {token.name}
                      {#if token.is_expired}
                        <span class="ml-1.5 text-[10px] bg-red-900/60 text-red-300 border border-red-700/40 rounded px-1.5 py-0.5">{$t('mcp.expired_badge')}</span>
                      {/if}
                    </p>
                    <p class="text-primary-400 font-mono text-xs mt-0.5">{token.token_prefix}…</p>
                    <p class="text-primary-500 text-xs mt-1">
                      {$t('mcp.expires_at')}: {formatDate(token.expires_at)}
                    </p>
                  </div>
                  <button
                    onclick={() => openRevoke(token)}
                    class="btn btn-ghost btn-sm text-red-400 hover:text-red-300 shrink-0 flex items-center gap-1"
                  >
                    <Trash2 size={14} /> {$t('mcp.revoke')}
                  </button>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </main>
</PageBackground>

<MCPTokenCreateModal
  bind:open={modalOpen}
  onCreated={handleCreated}
  onClose={() => { modalOpen = false; }}
/>

<ConfirmDialog
  bind:open={confirmOpen}
  message={$t('mcp.revoke_confirm')}
  confirmLabel={$t('mcp.revoke')}
  danger={true}
  onConfirm={confirmRevoke}
/>
