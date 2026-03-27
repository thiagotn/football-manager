<script lang="ts">
  import { players as playersApi, ApiError } from '$lib/api';
  import type { PlayerFullStats } from '$lib/api';
  import { currentPlayer, isLoggedIn } from '$lib/stores/auth';
  import { goto } from '$app/navigation';
  import { playerDisplayName } from '$lib/utils.js';
  import { Users, Trophy, Clock, ThumbsDown, Flame, Shield, BarChart2, Share2 } from 'lucide-svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import AvatarImage from '$lib/components/AvatarImage.svelte';
  import { t } from '$lib/i18n';
  import { toastSuccess, toastError } from '$lib/stores/toast';

  let stats = $state<PlayerFullStats | null>(null);
  let loading = $state(true);
  let error = $state('');

  $effect(() => {
    if (!$isLoggedIn) { goto('/login'); return; }
    let cancelled = false;
    (async () => {
      try {
        const data = await playersApi.myFullStats();
        if (!cancelled) stats = data;
      } catch (e) {
        if (!cancelled) error = e instanceof ApiError ? e.message : 'Erro ao carregar estatísticas';
      }
      if (!cancelled) loading = false;
    })();
    return () => { cancelled = true; };
  });

  const MONTHS_SHORT = ['Jan', 'Fev', 'Mar', 'Abr', 'Mai', 'Jun', 'Jul', 'Ago', 'Set', 'Out', 'Nov', 'Dez'];

  function monthLabel(yyyymm: string): string {
    return MONTHS_SHORT[parseInt(yyyymm.split('-')[1], 10) - 1];
  }

  function formatMinutes(m: number): string {
    if (m === 0) return '0min';
    const h = Math.floor(m / 60);
    const min = m % 60;
    if (h === 0) return `${min}min`;
    return min > 0 ? `${h}h${String(min).padStart(2, '0')}` : `${h}h`;
  }

  function memberSince(created_at: string | null | undefined): string {
    if (!created_at) return '';
    const d = new Date(created_at);
    if (isNaN(d.getTime())) return '';
    return `${MONTHS_SHORT[d.getMonth()]} ${d.getFullYear()}`;
  }

  function hasGoalkeeper(groups: PlayerFullStats['groups']): boolean {
    return groups.some(g => g.is_goalkeeper);
  }

  async function shareScore() {
    const url = `https://rachao.app/players/${$currentPlayer?.id}`;
    if (navigator.share) {
      try {
        await navigator.share({
          title: 'Rachão Score — ' + ($currentPlayer?.nickname || $currentPlayer?.name),
          url,
        });
      } catch {
        /* user cancelled */
      }
    } else {
      try {
        await navigator.clipboard.writeText(url);
        toastSuccess($t('stats.share_copied'));
      } catch {
        toastError($t('stats.share_error'));
      }
    }
  }

  const MARKET_VALUE_TIERS = [
    ['1 camisa do Brasil made in thailand', '1 meião furado', '1 litrão de guaraná genérico', '1 bombom Sonho de Valsa'],
    ['3 litrão de Heineken', '1 par de Kichute novo', '1 frango assado + refri', '1 chuteira de couro legítimo'],
    ['1 Monza 1986', '1 kit completo da Nike', '1 PlayStation 4 usado', '1 espetinho de carne'],
    ['1 Fiat Uno 94 conservado', 'Contrato na Portuguesa B', '1 taça do Brasileirão falsificada', 'Cobiçado pelo Al-Nassr (fila de espera)'],
  ];

  function marketValue(s: PlayerFullStats, seed: string): string {
    const score = s.total_matches_confirmed * 2 + s.attendance_rate + s.total_vote_points * 3;
    const tier = score <= 30 ? 0 : score <= 70 ? 1 : score <= 120 ? 2 : 3;
    const idx = seed.split('').reduce((acc, c) => acc + c.charCodeAt(0), 0) % 4;
    return MARKET_VALUE_TIERS[tier][idx];
  }
</script>

<svelte:head>
  <title>Rachão Score — rachao.app</title>
</svelte:head>

<PageBackground>
  <main class="relative z-10 max-w-7xl mx-auto px-4 py-8">

    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-white flex items-center gap-2">
          <BarChart2 size={24} class="text-primary-400" /> {$t('stats.title')}
        </h1>
        <p class="text-sm text-white/60 mt-0.5">{$t('stats.subtitle')}</p>
      </div>
      <button
        onclick={shareScore}
        class="btn btn-sm btn-ghost text-white border border-white/20 hover:bg-white/10 flex items-center gap-1.5 shrink-0">
        <Share2 size={14} /> {$t('stats.share_score')}
      </button>
    </div>

    {#if loading}
      <div class="grid sm:grid-cols-2 lg:grid-cols-4 gap-3 mb-4">
        {#each [1, 2, 3, 4] as _}
          <div class="card animate-pulse h-20 bg-gray-100 dark:bg-gray-800"></div>
        {/each}
      </div>
      <div class="grid lg:grid-cols-3 gap-4">
        <div class="lg:col-span-2 space-y-4">
          {#each [1, 2] as _}
            <div class="card animate-pulse h-32 bg-gray-100 dark:bg-gray-800"></div>
          {/each}
        </div>
        <div class="space-y-4">
          {#each [1, 2] as _}
            <div class="card animate-pulse h-32 bg-gray-100 dark:bg-gray-800"></div>
          {/each}
        </div>
      </div>

    {:else if error}
      <div class="card card-body text-center text-red-500">{error}</div>

    {:else if stats}

      <!-- Bloco 1: Cartão de identidade (full width) -->
      <div class="card overflow-hidden mb-4">
        <div class="relative px-4 py-4 text-white overflow-hidden"
          style="background: linear-gradient(135deg, #166534 0%, #15803d 60%, #16a34a 100%);">
          <div class="absolute -right-6 -top-6 w-28 h-28 rounded-full opacity-10 bg-white"></div>
          <div class="absolute -right-2 top-10 w-16 h-16 rounded-full opacity-10 bg-white"></div>
          <div class="relative flex items-start justify-between gap-3">
            <div class="flex items-start gap-3 min-w-0">
              <AvatarImage
                name={$currentPlayer?.name ?? ''}
                avatarUrl={$currentPlayer?.avatar_url}
                updatedAt={$currentPlayer?.updated_at}
                size={48}
                class="shrink-0 ring-2 ring-white/30"
              />
              <div class="min-w-0">
                <p class="text-xl font-bold leading-tight truncate">
                  {playerDisplayName($currentPlayer?.name ?? '', $currentPlayer?.nickname)}
                </p>
                {#if $currentPlayer?.nickname}
                  <p class="text-sm text-green-200 truncate">{$currentPlayer.name}</p>
                {/if}
                {#if memberSince($currentPlayer?.created_at)}
                  <p class="text-xs text-green-300 mt-1.5">
                    {$t('stats.member_since').replace('{date}', memberSince($currentPlayer?.created_at))}
                  </p>
                {/if}
              </div>
            </div>
            <div class="flex flex-col items-end gap-1.5 shrink-0">
              {#if hasGoalkeeper(stats.groups)}
                <span class="px-2 py-0.5 rounded-full text-xs font-bold bg-amber-400 text-amber-900">{$t('stats.goalkeeper')}</span>
              {/if}
              <span class="px-2 py-0.5 rounded-full text-xs font-semibold bg-white/20 text-white">
                ⚽ {stats.total_matches_confirmed} {$t('stats.matches')}
              </span>
            </div>
          </div>
          <div class="mt-3 pt-3 border-t border-white/20 flex items-center gap-2 min-w-0">
            <span class="text-xs text-green-200 shrink-0">{$t('stats.market_value')}</span>
            <span class="text-xs font-bold text-white truncate min-w-0">{marketValue(stats, $currentPlayer?.id ?? $currentPlayer?.name ?? '')}</span>
          </div>
        </div>
      </div>

      <!-- Bloco 2: Métricas rápidas — 2 cols mobile, 4 cols desktop -->
      <div class="grid grid-cols-2 lg:grid-cols-4 gap-2 mb-4">
        <div class="card card-body flex items-center gap-3 py-3">
          <div class="w-9 h-9 rounded-xl bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center shrink-0">
            <Users size={18} class="text-blue-600 dark:text-blue-400" />
          </div>
          <div class="min-w-0">
            <p class="text-xl font-bold text-gray-900 dark:text-gray-100 leading-none">{stats.total_matches_confirmed}</p>
            <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5">{$t('stats.matches')}</p>
          </div>
        </div>

        <div class="card card-body flex items-center gap-3 py-3">
          <div class="w-9 h-9 rounded-xl bg-green-100 dark:bg-green-900/30 flex items-center justify-center shrink-0">
            <Clock size={18} class="text-green-600 dark:text-green-400" />
          </div>
          <div class="min-w-0">
            <p class="text-xl font-bold text-gray-900 dark:text-gray-100 leading-none">{formatMinutes(stats.total_minutes_played)}</p>
            <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5">{$t('stats.on_field')}</p>
          </div>
        </div>

        <div class="card card-body flex items-center gap-3 py-3">
          <div class="w-9 h-9 rounded-xl bg-amber-100 dark:bg-amber-900/30 flex items-center justify-center shrink-0">
            <Trophy size={18} class="text-amber-500 dark:text-amber-400" />
          </div>
          <div class="min-w-0">
            <p class="text-xl font-bold text-gray-900 dark:text-gray-100 leading-none">{stats.total_vote_points}</p>
            <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5">{$t('stats.points')}</p>
          </div>
        </div>

        <div class="card card-body flex items-center gap-3 py-3">
          <div class="w-9 h-9 rounded-xl bg-red-100 dark:bg-red-900/30 flex items-center justify-center shrink-0">
            <ThumbsDown size={17} class="text-red-500 dark:text-red-400" />
          </div>
          <div class="min-w-0">
            <p class="text-xl font-bold text-gray-900 dark:text-gray-100 leading-none">{stats.total_flop_votes}</p>
            <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5">{$t('stats.flops')}</p>
          </div>
        </div>
      </div>

      <!-- Blocos 3-7: single column mobile / two-column desktop -->
      <div class="grid lg:grid-cols-3 gap-4">

        <!-- Coluna principal (2/3 no desktop) -->
        <div class="lg:col-span-2 space-y-4">

          <!-- Bloco 3: Presença -->
          <div class="card card-body">
            <h2 class="text-sm font-semibold text-gray-700 dark:text-gray-200 mb-3">{$t('stats.section_attendance')}</h2>
            <div class="flex items-center gap-5">
              <div class="relative w-20 h-20 rounded-full shrink-0"
                style="background: conic-gradient(#22c55e {stats.attendance_rate}%, #e5e7eb 0%);">
                <div class="absolute inset-[16%] rounded-full bg-white dark:bg-gray-800 flex items-center justify-center">
                  <span class="text-sm font-bold text-gray-900 dark:text-gray-100">{stats.attendance_rate}%</span>
                </div>
              </div>
              <div class="flex-1 space-y-2.5">
                <div class="flex items-center justify-between">
                  <span class="text-xs text-gray-500 dark:text-gray-400">{$t('stats.current_streak')}</span>
                  <span class="text-sm font-bold text-gray-900 dark:text-gray-100 flex items-center gap-1">
                    {#if stats.current_streak >= 3}<Flame size={13} class="text-orange-500" />{/if}
                    {$t('stats.streak_consecutive').replace('{n}', String(stats.current_streak))}
                  </span>
                </div>
                <div class="flex items-center justify-between">
                  <span class="text-xs text-gray-500 dark:text-gray-400">{$t('stats.best_streak')}</span>
                  <span class="text-sm font-bold text-primary-600 dark:text-primary-400">{stats.best_streak}</span>
                </div>
                <div class="h-1.5 bg-gray-100 dark:bg-gray-700 rounded-full overflow-hidden">
                  <div class="h-full bg-green-500 rounded-full" style="width: {stats.attendance_rate}%;"></div>
                </div>
              </div>
            </div>
          </div>

          <!-- Bloco 5: Reputação -->
          {#if stats.total_matches_confirmed > 0}
            {@const approvalPct = Math.min(100, Math.round((stats.top5_count / Math.max(1, stats.total_matches_confirmed)) * 100))}
            <div class="card card-body">
              <h2 class="text-sm font-semibold text-gray-700 dark:text-gray-200 mb-3">{$t('stats.section_reputation')}</h2>
              <div class="grid grid-cols-3 gap-2 mb-3">
                <div class="text-center p-2.5 rounded-xl bg-amber-50 dark:bg-amber-900/20">
                  <p class="text-xl font-bold text-amber-600 dark:text-amber-400">{stats.top1_count}</p>
                  <p class="text-[10px] text-gray-500 dark:text-gray-400 leading-tight mt-0.5">{$t('stats.best_of_match')}</p>
                </div>
                <div class="text-center p-2.5 rounded-xl bg-blue-50 dark:bg-blue-900/20">
                  <p class="text-xl font-bold text-blue-600 dark:text-blue-400">{stats.top5_count}</p>
                  <p class="text-[10px] text-gray-500 dark:text-gray-400 leading-tight mt-0.5">{$t('stats.top5')}</p>
                </div>
                <div class="text-center p-2.5 rounded-xl bg-purple-50 dark:bg-purple-900/20">
                  <p class="text-xl font-bold text-purple-600 dark:text-purple-400">{stats.total_vote_points}</p>
                  <p class="text-[10px] text-gray-500 dark:text-gray-400 leading-tight mt-0.5">{$t('stats.total_points')}</p>
                </div>
              </div>
              <div>
                <div class="flex items-center justify-between mb-1">
                  <span class="text-xs text-gray-500 dark:text-gray-400">{$t('stats.approval')}</span>
                  <span class="text-xs font-semibold text-gray-700 dark:text-gray-300">{approvalPct}%</span>
                </div>
                <div class="h-2 bg-gray-100 dark:bg-gray-700 rounded-full overflow-hidden">
                  <div class="h-full rounded-full transition-all"
                    style="width: {approvalPct}%; background: linear-gradient(90deg, #3b82f6, #8b5cf6);"></div>
                </div>
              </div>
            </div>
          {/if}

          <!-- Bloco 6: Gráfico de barras mensal -->
          {#if stats.monthly_stats.some(m => m.matches_confirmed > 0)}
            {@const maxVal = Math.max(1, ...stats.monthly_stats.map(m => m.matches_confirmed))}
            <div class="card card-body">
              <h2 class="text-sm font-semibold text-gray-700 dark:text-gray-200 mb-4">{$t('stats.section_frequency')}</h2>
              <div class="flex items-end gap-1 h-24">
                {#each stats.monthly_stats as m}
                  <div class="flex-1 flex flex-col items-center gap-1">
                    <span class="text-[9px] text-gray-400 h-3 flex items-center">
                      {m.matches_confirmed > 0 ? m.matches_confirmed : ''}
                    </span>
                    <div class="w-full rounded-t transition-all"
                      style="height: {Math.max(3, (m.matches_confirmed / maxVal) * 52)}px; background-color: {m.matches_confirmed > 0 ? '#3b82f6' : '#e5e7eb'};"></div>
                    <span class="text-[9px] text-gray-400">{monthLabel(m.month)}</span>
                  </div>
                {/each}
              </div>
            </div>
          {/if}

        </div>

        <!-- Coluna lateral (1/3 no desktop) -->
        <div class="space-y-4">

          <!-- Bloco 4: Histórico recente -->
          {#if stats.recent_matches.length > 0}
            <div class="card card-body">
              <div class="flex items-center justify-between mb-3">
                <h2 class="text-sm font-semibold text-gray-700 dark:text-gray-200">{$t('stats.section_history')}</h2>
                <span class="text-xs text-gray-400">{$t('stats.history_matches').replace('{n}', String(stats.recent_matches.length))}</span>
              </div>
              <div class="flex flex-wrap gap-1.5">
                {#each stats.recent_matches as m}
                  <div
                    class="w-3.5 h-3.5 rounded-full shrink-0 {m.status === 'confirmed' ? 'bg-green-500' : 'bg-red-400'}"
                    title="{m.match_date} — {m.group_name} — {m.status === 'confirmed' ? $t('stats.history_confirmed') : $t('stats.history_missed')}"
                  ></div>
                {/each}
              </div>
              <div class="flex items-center gap-4 mt-2.5">
                <span class="flex items-center gap-1.5 text-[10px] text-gray-400">
                  <span class="w-2.5 h-2.5 rounded-full bg-green-500 inline-block"></span> {$t('stats.history_confirmed')}
                </span>
                <span class="flex items-center gap-1.5 text-[10px] text-gray-400">
                  <span class="w-2.5 h-2.5 rounded-full bg-red-400 inline-block"></span> {$t('stats.history_missed')}
                </span>
              </div>
            </div>
          {/if}

          <!-- Bloco 7: Meus grupos -->
          {#if stats.groups.length > 0}
            <div class="card overflow-hidden">
              <div class="px-4 py-3 border-b border-gray-100 dark:border-gray-700">
                <h2 class="text-sm font-semibold text-gray-700 dark:text-gray-200">{$t('stats.section_my_groups')}</h2>
              </div>
              <ul class="divide-y divide-gray-100 dark:divide-gray-700">
                {#each stats.groups as g}
                  <li class="px-4 py-2.5 flex items-center gap-2.5">
                    <Shield size={14} class="text-primary-500 shrink-0" fill="currentColor" />
                    <span class="flex-1 text-sm font-medium text-gray-900 dark:text-gray-100 truncate">{g.group_name}</span>
                    <div class="flex items-center gap-2 shrink-0">
                      {#if g.is_goalkeeper}
                        <span class="text-[9px] px-1.5 py-0.5 rounded-full font-bold bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400">GK</span>
                      {/if}
                      {#if g.role === 'admin'}
                        <span class="text-[9px] px-1.5 py-0.5 rounded-full font-bold bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">Admin</span>
                      {/if}
                      <span class="text-xs text-gray-400">{$t('stats.group_matches').replace('{n}', String(g.matches_confirmed))}</span>
                      <span class="text-xs text-amber-500 tracking-tighter">{'★'.repeat(g.skill_stars)}</span>
                    </div>
                  </li>
                {/each}
              </ul>
            </div>
          {/if}

        </div>
      </div>

    {/if}
  </main>
</PageBackground>
