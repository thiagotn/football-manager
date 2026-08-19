<script lang="ts">
  import { Scale, Shuffle } from 'lucide-svelte';
  import { t } from '$lib/i18n';
  import type { DrawStrategy } from '$lib/api';
  import Modal from './Modal.svelte';

  let {
    open = $bindable(false),
    mode = 'build',
    loading = false,
    onConfirm,
  }: {
    open?: boolean;
    mode?: 'build' | 'rebuild';
    loading?: boolean;
    onConfirm: (strategy: DrawStrategy) => void;
  } = $props();

  let strategy = $state<DrawStrategy>('balanced');
</script>

<Modal bind:open title={$t('teams.strategy_title')}>
  <div class="space-y-4">
    <p class="text-sm text-gray-600 dark:text-gray-300">{$t('teams.strategy_desc')}</p>

    <div class="space-y-3">
      <button
        type="button"
        onclick={() => strategy = 'balanced'}
        class="w-full text-left rounded-xl border-2 p-4 transition-colors {strategy === 'balanced'
          ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
          : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'}"
      >
        <div class="flex items-center gap-2 font-semibold text-gray-900 dark:text-gray-100">
          <Scale size={16} class="text-primary-500" /> {$t('teams.strategy_balanced')}
        </div>
        <p class="text-xs text-gray-600 dark:text-gray-400 mt-1">{$t('teams.strategy_balanced_desc')}</p>
      </button>

      <button
        type="button"
        onclick={() => strategy = 'simple'}
        class="w-full text-left rounded-xl border-2 p-4 transition-colors {strategy === 'simple'
          ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
          : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'}"
      >
        <div class="flex items-center gap-2 font-semibold text-gray-900 dark:text-gray-100">
          <Shuffle size={16} class="text-primary-500" /> {$t('teams.strategy_simple')}
        </div>
        <p class="text-xs text-gray-600 dark:text-gray-400 mt-1">{$t('teams.strategy_simple_desc')}</p>
      </button>
    </div>

    {#if mode === 'rebuild'}
      <p class="text-xs text-amber-600 dark:text-amber-400">{$t('teams.rebuild_confirm')}</p>
    {/if}

    <button
      onclick={() => onConfirm(strategy)}
      disabled={loading}
      class="btn-primary w-full justify-center disabled:opacity-50"
    >
      {loading
        ? $t('teams.strategy_drawing')
        : mode === 'rebuild' ? $t('teams.rebuild_label') : $t('teams.strategy_confirm')}
    </button>
  </div>
</Modal>
