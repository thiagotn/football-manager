<script lang="ts">
  import { players as playersApi, ApiError } from '$lib/api';
  import type { PlayerMatchItem } from '$lib/api';
  import { isLoggedIn } from '$lib/stores/auth';
  import { goto } from '$app/navigation';
  import { Calendar, Clock, MapPin, ChevronRight } from 'lucide-svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import { relativeDate } from '$lib/utils.js';

  const COURT_LABELS: Record<string, string> = { campo: 'Campo', sintetico: 'Sintético', terrao: 'Terrão', quadra: 'Quadra' };
  const ATTENDANCE_LABEL: Record<string, { label: string; cls: string }> = {
    confirmed: { label: 'Confirmado', cls: 'badge-green' },
    declined:  { label: 'Recusado',   cls: 'badge-red' },
    pending:   { label: 'Pendente',   cls: 'badge-gray' },
  };

  let allMatches = $state<PlayerMatchItem[]>([]);
  let loading = $state(true);
  let error = $state('');
  let tab = $state<'upcoming' | 'past'>('upcoming');

  $effect(() => {
    if (!$isLoggedIn) { goto('/login'); return; }
    let cancelled = false;
    (async () => {
      try {
        const data = await playersApi.myMatches();
        if (!cancelled) allMatches = data;
      } catch (e) {
        if (!cancelled) error = e instanceof ApiError ? e.message : 'Erro ao carregar rachões';
      }
      if (!cancelled) loading = false;
    })();
    return () => { cancelled = true; };
  });

  let upcoming = $derived(
    allMatches
      .filter(m => m.status === 'open' || m.status === 'in_progress')
      .sort((a, b) => `${a.match_date}T${a.start_time}`.localeCompare(`${b.match_date}T${b.start_time}`))
  );
  let past = $derived(
    allMatches
      .filter(m => m.status === 'closed')
      .sort((a, b) => `${b.match_date}T${b.start_time}`.localeCompare(`${a.match_date}T${a.start_time}`))
  );

  function fmtDate(d: string) {
    const s = relativeDate(d, { weekday: 'long', day: '2-digit', month: 'long' });
    return s.charAt(0).toUpperCase() + s.slice(1);
  }

  function fmtTimeRange(start: string, end: string | null): string {
    const s = start.slice(0, 5);
    if (!end) return s;
    const e = end.slice(0, 5);
    const [sh, sm] = s.split(':').map(Number);
    const [eh, em] = e.split(':').map(Number);
    const mins = (eh * 60 + em) - (sh * 60 + sm);
    if (mins <= 0) return `${s} – ${e}`;
    const h = Math.floor(mins / 60), m = mins % 60;
    return `${s} – ${e} (${h > 0 ? `${h}h` : ''}${m > 0 ? `${m}min` : ''})`;
  }
</script>

<svelte:head><title>Rachões — rachao.app</title></svelte:head>

<PageBackground>
  <main class="relative z-10 max-w-7xl mx-auto px-4 py-8">

    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-white flex items-center gap-2">
          <Calendar size={24} class="text-primary-400" /> Rachões
        </h1>
        <p class="text-sm text-white/60 mt-0.5">Todas as partidas dos seus grupos.</p>
      </div>
    </div>

    <!-- Tabs -->
    <div class="flex border-b border-white/20 mb-4 overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
      <button
        class="px-3 py-2 text-sm font-medium border-b-2 transition-colors whitespace-nowrap {tab === 'upcoming' ? 'border-primary-400 text-primary-400' : 'border-transparent text-gray-300 hover:text-white'}"
        onclick={() => tab = 'upcoming'}>
        Próximos {#if !loading}({upcoming.length}){/if}
      </button>
      <button
        class="px-3 py-2 text-sm font-medium border-b-2 transition-colors whitespace-nowrap {tab === 'past' ? 'border-primary-400 text-primary-400' : 'border-transparent text-gray-300 hover:text-white'}"
        onclick={() => tab = 'past'}>
        Últimos {#if !loading}({past.length}){/if}
      </button>
    </div>

    {#if loading}
      <div class="space-y-3">
        {#each [1, 2, 3] as _}
          <div class="card animate-pulse h-24 bg-gray-100 dark:bg-gray-800"></div>
        {/each}
      </div>

    {:else if error}
      <div class="card card-body text-center text-red-500">{error}</div>

    {:else}
      {@const list = tab === 'upcoming' ? upcoming : past}

      {#if list.length === 0}
        <div class="card p-12 text-center">
          <Calendar size={40} class="text-gray-300 mx-auto mb-3" />
          <p class="text-gray-500">{tab === 'upcoming' ? 'Nenhum rachão agendado.' : 'Nenhum rachão encerrado ainda.'}</p>
        </div>
      {:else}
        <div class="space-y-3">
          {#each list as m}
            <div class="card hover:shadow-md transition-shadow">
              <div class="card-body">

                <!-- Date + status -->
                <div class="flex items-start gap-2 mb-1">
                  <div class="flex-1 min-w-0">
                    <p class="font-semibold text-gray-900 dark:text-gray-100 leading-snug">
                      <span class="text-primary-600 dark:text-primary-400 font-bold text-base mr-1">#{m.number}</span>{fmtDate(m.match_date)}
                    </p>
                    <p class="text-xs text-gray-400 dark:text-gray-500 mt-0.5">{m.group_name}</p>
                  </div>
                  {#if m.status === 'in_progress'}
                    <span class="shrink-0 inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold bg-red-500/20 text-red-400 border border-red-500/30">
                      <span class="w-1.5 h-1.5 rounded-full bg-red-400 animate-pulse"></span>
                      Bola rolando
                    </span>
                  {:else}
                    <span class="badge {m.status === 'open' ? 'badge-green' : 'badge-gray'} shrink-0">
                      {m.status === 'open' ? 'Aberta' : 'Encerrada'}
                    </span>
                  {/if}
                </div>

                <!-- Time + Location -->
                <div class="flex flex-wrap items-center gap-x-3 gap-y-0.5 mt-1 text-sm text-gray-500 dark:text-gray-400">
                  <span class="flex items-center gap-1 whitespace-nowrap"><Clock size={12} />{fmtTimeRange(m.start_time, m.end_time)}</span>
                  <span class="flex items-center gap-1 min-w-0"><MapPin size={12} /><span class="truncate">{m.location}</span></span>
                </div>

                <!-- Court / players -->
                {#if m.court_type || m.players_per_team || m.max_players}
                  <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">
                    {[
                      m.court_type ? COURT_LABELS[m.court_type] : null,
                      m.players_per_team ? `${m.players_per_team} na linha + gol` : null,
                      m.max_players ? `máx. ${m.max_players}` : null,
                    ].filter(Boolean).join(' · ')}
                  </p>
                {/if}

                <!-- Actions -->
                <div class="flex items-center gap-2 mt-3 pt-3 border-t border-gray-100 dark:border-gray-700">
                  <a href="/match/{m.hash}" class="btn-sm btn-secondary shrink-0">
                    Detalhes <ChevronRight size={14} />
                  </a>
                  {#if m.my_attendance}
                    <span class="badge {ATTENDANCE_LABEL[m.my_attendance].cls} shrink-0">
                      {ATTENDANCE_LABEL[m.my_attendance].label}
                    </span>
                  {/if}
                </div>

              </div>
            </div>
          {/each}
        </div>
      {/if}
    {/if}

  </main>
</PageBackground>
