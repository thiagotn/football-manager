<script lang="ts">
  import type { VoteResultsResponse } from '$lib/api';

  interface Props { results: VoteResultsResponse }
  let { results }: Props = $props();

  const MEDALS: Record<number, string> = { 1: '🥇', 2: '🥈', 3: '🥉' };
</script>

<div class="space-y-4">
  <!-- Top 5 -->
  <div>
    <p class="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-3">🏆 Melhores da Partida</p>
    {#if results.top5.length === 0}
      <p class="text-xs text-gray-400 dark:text-gray-500 text-center py-2">Nenhum voto registrado.</p>
    {:else}
      <div class="space-y-1.5">
        {#each results.top5 as item}
          <div class="flex items-center gap-2 px-3 py-2 rounded-lg bg-gray-50 dark:bg-gray-700/50">
            <span class="w-6 text-lg shrink-0">{MEDALS[item.position] ?? '  '}</span>
            <span class="flex-1 text-sm font-medium text-gray-800 dark:text-gray-200 truncate">{item.name}</span>
            <span class="text-xs font-bold text-primary-600 dark:text-primary-400 shrink-0">{item.points} pts</span>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  <!-- Flop -->
  {#if results.flop.length > 0}
    <div>
      <p class="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">😬 Decepção do Jogo</p>
      <div class="space-y-1.5">
        {#each results.flop as item}
          <div class="flex items-center gap-2 px-3 py-2 rounded-lg bg-red-50 dark:bg-red-900/20">
            <span class="flex-1 text-sm text-gray-700 dark:text-gray-300 truncate">{item.name}</span>
            <span class="text-xs text-red-600 dark:text-red-400 shrink-0">{item.votes} voto{item.votes !== 1 ? 's' : ''}</span>
          </div>
        {/each}
      </div>
    </div>
  {/if}

  <p class="text-xs text-gray-400 dark:text-gray-500 text-center">
    {results.total_voters} de {results.eligible_voters} jogador{results.eligible_voters !== 1 ? 'es' : ''} votaram
  </p>
</div>
