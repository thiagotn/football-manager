<script lang="ts">
  import { mcpTokens, ApiError } from '$lib/api';
  import type { MCPTokenCreated } from '$lib/api';
  import { t } from '$lib/i18n';
  import { Copy, Check } from 'lucide-svelte';

  type Props = {
    open: boolean;
    onCreated: (token: MCPTokenCreated) => void;
    onClose: () => void;
  };

  let { open = $bindable(), onCreated, onClose }: Props = $props();

  type Step = 'form' | 'reveal';

  let step = $state<Step>('form');
  let name = $state('');
  let expiresIn = $state<'24h' | '7d' | null>(null);
  let loading = $state(false);
  let error = $state('');
  let createdToken = $state<MCPTokenCreated | null>(null);
  let copied = $state(false);

  function reset() {
    step = 'form';
    name = '';
    expiresIn = null;
    loading = false;
    error = '';
    createdToken = null;
    copied = false;
  }

  function handleClose() {
    reset();
    onClose();
  }

  async function handleGenerate() {
    if (!name.trim()) return;
    loading = true;
    error = '';
    try {
      const token = await mcpTokens.create({ name: name.trim(), expires_in: expiresIn });
      createdToken = token;
      onCreated(token);
      step = 'reveal';
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Erro ao gerar token.';
    } finally {
      loading = false;
    }
  }

  async function copyToken() {
    if (!createdToken) return;
    await navigator.clipboard.writeText(createdToken.token);
    copied = true;
    setTimeout(() => { copied = false; }, 2000);
  }
</script>

{#if open}
  <!-- Backdrop -->
  <button
    class="fixed inset-0 z-40 bg-black/60"
    onclick={handleClose}
    aria-label={$t('aria.close')}
  ></button>

  <!-- Modal -->
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4">
    <div class="bg-primary-800 border border-primary-700 rounded-2xl w-full max-w-md shadow-2xl">

      {#if step === 'form'}
        <div class="p-6">
          <h2 class="text-lg font-bold text-white mb-4">{$t('mcp.generate')}</h2>

          <div class="space-y-4">
            <div>
              <label class="block text-sm font-medium text-primary-200 mb-1">
                {$t('mcp.token_name')}
              </label>
              <input
                type="text"
                bind:value={name}
                placeholder={$t('mcp.token_name_placeholder')}
                class="w-full bg-primary-900 border border-primary-600 rounded-lg px-3 py-2 text-white placeholder-primary-400 focus:outline-none focus:border-primary-400 text-sm"
                maxlength="100"
              />
            </div>

            <div>
              <label class="block text-sm font-medium text-primary-200 mb-2">
                {$t('mcp.expiration')}
              </label>
              <div class="space-y-2">
                {#each [
                  { value: '24h' as const, label: $t('mcp.expires_24h') },
                  { value: '7d' as const, label: $t('mcp.expires_7d') },
                  { value: null, label: $t('mcp.no_expiration') },
                ] as opt}
                  <label class="flex items-center gap-3 cursor-pointer">
                    <input
                      type="radio"
                      name="expires_in"
                      value={opt.value}
                      checked={expiresIn === opt.value}
                      onchange={() => { expiresIn = opt.value; }}
                      class="accent-primary-400"
                    />
                    <span class="text-sm text-primary-100">{opt.label}</span>
                  </label>
                {/each}
              </div>
            </div>

            {#if error}
              <p class="text-red-400 text-sm">{error}</p>
            {/if}
          </div>

          <div class="flex gap-3 mt-6">
            <button
              onclick={handleClose}
              class="flex-1 btn btn-ghost btn-sm text-primary-300"
            >
              {$t('groups.cancel')}
            </button>
            <button
              onclick={handleGenerate}
              disabled={!name.trim() || loading}
              class="flex-1 btn btn-primary btn-sm disabled:opacity-50"
            >
              {loading ? '…' : $t('mcp.generate')}
            </button>
          </div>
        </div>

      {:else if step === 'reveal' && createdToken}
        <div class="p-6">
          <h2 class="text-lg font-bold text-white mb-2">{$t('mcp.generate')}</h2>

          <div class="bg-yellow-900/40 border border-yellow-600/50 rounded-lg px-4 py-3 mb-4">
            <p class="text-yellow-300 text-sm font-medium">{$t('mcp.token_warning')}</p>
          </div>

          <div class="bg-primary-900 border border-primary-600 rounded-lg px-3 py-3 mb-4">
            <p class="text-primary-300 text-xs mb-1 font-mono break-all">{createdToken.token}</p>
          </div>

          <button
            onclick={copyToken}
            class="w-full flex items-center justify-center gap-2 btn btn-ghost btn-sm border border-primary-600 mb-4"
          >
            {#if copied}
              <Check size={14} class="text-green-400" /> <span class="text-green-400">{$t('mcp.token_copied')}</span>
            {:else}
              <Copy size={14} /> {$t('mcp.copy_token')}
            {/if}
          </button>

          <button
            onclick={handleClose}
            class="w-full btn btn-primary btn-sm"
          >
            {$t('mcp.token_understood')}
          </button>
        </div>
      {/if}

    </div>
  </div>
{/if}
