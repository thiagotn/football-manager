<script lang="ts">
  import { CheckCircle, XCircle, Clock3, UserCheck } from 'lucide-svelte';
  import type { WaitlistEntry } from '$lib/api';

  interface Props {
    entries: WaitlistEntry[];
    accepting?: string | null;
    rejecting?: string | null;
    onaccept: (entryId: string) => void;
    onreject: (entryId: string) => void;
  }

  let { entries, accepting = null, rejecting = null, onaccept, onreject }: Props = $props();

  function fmtDate(s: string) {
    const d = new Date(s);
    return d.toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' });
  }
</script>

<div class="card overflow-hidden">
  <div class="px-4 py-3 bg-amber-50 dark:bg-amber-900/20 border-b border-gray-100 dark:border-gray-700">
    <h3 class="text-sm font-semibold text-amber-800 dark:text-amber-300 flex items-center gap-1.5">
      <UserCheck size={14} /> Lista de espera ({entries.length})
    </h3>
  </div>

  {#if entries.length === 0}
    <div class="px-4 py-6 text-center text-sm text-gray-400 dark:text-gray-500">
      Nenhum candidato aguardando aprovação.
    </div>
  {:else}
    <ul class="divide-y divide-gray-100 dark:divide-gray-700">
      {#each entries as entry}
        <li class="px-4 py-3 space-y-2">
          <div class="flex items-start justify-between gap-2">
            <div class="min-w-0">
              <p class="text-sm font-medium text-gray-900 dark:text-gray-100">
                {entry.player_nickname || entry.player_name}
              </p>
              <p class="text-xs text-gray-400 dark:text-gray-500 flex items-center gap-1 mt-0.5">
                <Clock3 size={10} /> {fmtDate(entry.created_at)}
              </p>
            </div>
            <div class="flex gap-1 shrink-0">
              <button
                class="btn-sm btn-ghost text-green-600 dark:text-green-400 flex items-center gap-1 disabled:opacity-40"
                onclick={() => onaccept(entry.id)}
                disabled={accepting === entry.id || rejecting === entry.id}>
                <CheckCircle size={13} /> {accepting === entry.id ? 'Aceitando…' : 'Aceitar'}
              </button>
              <button
                class="btn-sm btn-ghost text-red-500 dark:text-red-400 flex items-center gap-1 disabled:opacity-40"
                onclick={() => onreject(entry.id)}
                disabled={accepting === entry.id || rejecting === entry.id}>
                <XCircle size={13} /> {rejecting === entry.id ? 'Rejeitando…' : 'Rejeitar'}
              </button>
            </div>
          </div>
          {#if entry.intro}
            <p class="text-xs text-gray-500 dark:text-gray-400 bg-gray-50 dark:bg-gray-700/50 rounded-lg px-3 py-2 italic">
              "{entry.intro}"
            </p>
          {/if}
        </li>
      {/each}
    </ul>
  {/if}
</div>
