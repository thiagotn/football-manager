<script lang="ts">
  import { X, Clock, MapPin, Users } from 'lucide-svelte';
  import type { MatchDetail } from '$lib/api';

  interface Props {
    open: boolean;
    match: MatchDetail;
    submitting?: boolean;
    onsubmit: (data: { agreed: boolean; intro: string }) => void;
    onclose: () => void;
  }

  let { open = $bindable(), match, submitting = false, onsubmit, onclose }: Props = $props();

  let intro = $state('');
  let agreed = $state(false);

  const COURT_LABELS: Record<string, string> = { campo: 'Campo', sintetico: 'Sintético', terrao: 'Terrão', quadra: 'Quadra' };

  function fmtDate(d: string) {
    const dt = new Date(d + 'T00:00');
    return dt.toLocaleDateString('pt-BR', { weekday: 'long', day: '2-digit', month: 'long' });
  }

  function fmtTime(t: string) {
    return t.slice(0, 5);
  }

  function handleSubmit(e: Event) {
    e.preventDefault();
    if (!agreed) return;
    onsubmit({ agreed, intro });
  }

  function handleBackdrop(e: MouseEvent) {
    if (e.target === e.currentTarget) onclose();
  }
</script>

{#if open}
  <div
    class="fixed inset-0 z-50 bg-black/60 flex items-end sm:items-center justify-center"
    role="dialog"
    aria-modal="true"
    onclick={handleBackdrop}>

    <div class="bg-white dark:bg-gray-800 w-full sm:max-w-lg rounded-t-2xl sm:rounded-2xl shadow-2xl flex flex-col max-h-[90dvh]">

      <!-- Header -->
      <div class="flex items-center justify-between px-5 py-4 border-b border-gray-100 dark:border-gray-700 shrink-0">
        <p class="font-bold text-gray-800 dark:text-gray-100">⚽ Entrar na lista de espera</p>
        <button
          onclick={onclose}
          class="p-1.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 transition-colors"
          aria-label="Fechar">
          <X size={18} />
        </button>
      </div>

      <!-- Body (scrollable) -->
      <div class="overflow-y-auto p-5 space-y-4">

        <!-- Match details (terms) -->
        <div class="bg-gray-50 dark:bg-gray-700/50 rounded-xl p-4 space-y-2 text-sm">
          <p class="font-semibold text-gray-700 dark:text-gray-200 text-xs uppercase tracking-wide mb-3">Detalhes do rachão</p>
          <div class="flex items-center gap-2 text-gray-600 dark:text-gray-300">
            <Clock size={14} class="text-primary-500 shrink-0" />
            <span class="capitalize">{fmtDate(match.match_date)}</span>
          </div>
          <div class="flex items-center gap-2 text-gray-600 dark:text-gray-300">
            <Clock size={14} class="shrink-0 opacity-0" />
            <span>
              {fmtTime(match.start_time)}
              {#if match.end_time} – {fmtTime(match.end_time)}{/if}
            </span>
          </div>
          <div class="flex items-center gap-2 text-gray-600 dark:text-gray-300">
            <MapPin size={14} class="text-primary-500 shrink-0" />
            <span>{match.location}{#if match.address} — {match.address}{/if}</span>
          </div>
          {#if match.court_type || match.players_per_team || match.max_players}
            <div class="flex items-center gap-2 text-gray-600 dark:text-gray-300">
              <Users size={14} class="text-primary-500 shrink-0" />
              <span>
                {#if match.court_type}{COURT_LABELS[match.court_type]}{/if}
                {#if match.players_per_team} · {match.players_per_team} na linha{/if}
                {#if match.max_players} · {match.confirmed_count}/{match.max_players} vagas{/if}
              </span>
            </div>
          {/if}
          {#if match.group_per_match_amount != null || match.group_monthly_amount != null}
            <div class="flex flex-wrap gap-2 mt-1">
              {#if match.group_per_match_amount != null}
                <span class="bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-300 text-xs rounded px-2 py-0.5 font-medium">
                  R$ {Number(match.group_per_match_amount).toFixed(2).replace('.', ',')} por jogo
                </span>
              {/if}
              {#if match.group_monthly_amount != null}
                <span class="bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-300 text-xs rounded px-2 py-0.5 font-medium">
                  R$ {Number(match.group_monthly_amount).toFixed(2).replace('.', ',')} mensal
                </span>
              {/if}
            </div>
          {/if}
        </div>

        <!-- Intro field -->
        <form onsubmit={handleSubmit} id="waitlist-form" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-200 mb-1">
              Conte um pouco sobre você <span class="text-gray-400 font-normal">(opcional)</span>
            </label>
            <textarea
              bind:value={intro}
              maxlength="500"
              rows="3"
              placeholder="Jogo há 5 anos, posição: meia. Disponível toda quinta."
              class="input resize-none text-sm"
            ></textarea>
            <p class="text-xs text-gray-400 mt-1 text-right">{intro.length}/500</p>
          </div>

          <!-- Agreement checkbox -->
          <label class="flex items-start gap-3 cursor-pointer">
            <input type="checkbox" bind:checked={agreed} class="mt-0.5 shrink-0 accent-primary-600 w-4 h-4" />
            <span class="text-sm text-gray-600 dark:text-gray-300">
              Li e concordo com as condições acima (data, local, horário e valores do rachão)
            </span>
          </label>
        </form>
      </div>

      <!-- Footer -->
      <div class="px-5 py-4 border-t border-gray-100 dark:border-gray-700 shrink-0">
        <button
          type="submit"
          form="waitlist-form"
          class="btn btn-primary w-full justify-center"
          disabled={!agreed || submitting}>
          {submitting ? 'Enviando…' : 'Enviar candidatura'}
        </button>
      </div>

    </div>
  </div>
{/if}
