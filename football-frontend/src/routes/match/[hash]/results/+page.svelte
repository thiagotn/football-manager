<script lang="ts">
  import { page } from '$app/stores';
  import { votes as votesApi, matches as matchesApi, ApiError } from '$lib/api';
  import type { VoteResultsResponse, MatchDetail } from '$lib/api';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import JoinCTABanner from '$lib/components/JoinCTABanner.svelte';
  import { isLoggedIn } from '$lib/stores/auth';
  import { Trophy, Share2, Clock, MapPin } from 'lucide-svelte';
  import { relativeDate, playerDisplayName } from '$lib/utils.js';
  import { t, locale } from '$lib/i18n';

  const COURT_LABELS: Record<string, string> = { campo: 'Campo', sintetico: 'Sintético', terrao: 'Terrão', quadra: 'Quadra' };

  function fmtDate(d: string) {
    return relativeDate(d, { weekday: 'long', day: '2-digit', month: 'long', year: 'numeric' }, $locale, {
      today: $t('date.today'),
      tomorrow: $t('date.tomorrow'),
      yesterday: $t('date.yesterday'),
    });
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
    const dur = h && m ? `${h}h${String(m).padStart(2, '0')}` : h ? `${h}h` : `${m}min`;
    return `${s} – ${e} (${dur})`;
  }

  let { data } = $props();

  const matchHash = $page.params.hash;

  let results = $state<VoteResultsResponse | null>(null);
  let match = $state<MatchDetail | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);

  $effect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [r, m] = await Promise.all([
          votesApi.getPublicResults(matchHash),
          matchesApi.getByHash(matchHash),
        ]);
        if (!cancelled) {
          results = r;
          match = m;
        }
      } catch (e) {
        if (!cancelled) {
          if (e instanceof ApiError && e.status === 404) {
            error = $t('results.not_available_desc');
          } else {
            error = $t('results.load_error');
          }
        }
      } finally {
        if (!cancelled) loading = false;
      }
    })();
    return () => { cancelled = true; };
  });

  const MEDALS: Record<number, string> = { 1: '🥇', 2: '🥈', 3: '🥉' };
  const PODIUM_ORDER = [2, 1, 3]; // 2nd left, 1st center, 3rd right

  function podiumHeight(pos: number): string {
    if (pos === 1) return 'h-24';
    if (pos === 2) return 'h-16';
    return 'h-12';
  }

  function podiumBg(pos: number): string {
    if (pos === 1) return 'bg-amber-400 dark:bg-amber-500';
    if (pos === 2) return 'bg-gray-300 dark:bg-gray-500';
    return 'bg-amber-700 dark:bg-amber-800';
  }

  function podiumTextColor(pos: number): string {
    if (pos === 1) return 'text-amber-700 dark:text-amber-300';
    if (pos === 2) return 'text-gray-600 dark:text-gray-300';
    return 'text-amber-800 dark:text-amber-600';
  }

  let top3 = $derived(
    results ? PODIUM_ORDER.map(p => results!.top5.find(x => x.position === p)).filter(Boolean) as typeof results.top5 : []
  );
  let rest = $derived(results ? results.top5.filter(x => x.position > 3) : []);

  function shareResults() {
    if (!results || !match) return;
    const top3 = results.top5.filter(p => p.position <= 3).map(p => `${MEDALS[p.position]} ${p.name} (${p.points} pts)`).join('\n');
    const rest = results.top5.filter(p => p.position > 3).map(p => `#${p.position} ${p.name} (${p.points} pts)`).join('\n');
    const flop = results.flop.length > 0
      ? results.flop.map(p => `😬 ${p.name} (${p.votes} voto${p.votes !== 1 ? 's' : ''})`).join('\n')
      : null;
    const time = match.start_time.slice(0, 5) + (match.end_time ? ` – ${match.end_time.slice(0, 5)}` : '');
    const lines = [
      `🏆 Resultado do Rachão ${match.group_name}`,
      `📅 ${fmtDate(match.match_date)} · ${time}`,
      `📍 ${match.location}`,
      '',
      '*Melhores da Partida*',
      top3,
      ...(rest ? ['', rest] : []),
      ...(flop ? ['', '*Decepção do Jogo*', flop] : []),
      '',
      `${results.total_voters} de ${results.eligible_voters} jogadores votaram`,
      '',
      `https://rachao.app/match/${matchHash}/results`,
    ];
    const text = lines.join('\n');
    if (navigator.share) {
      navigator.share({ text }).catch(() => {});
    } else {
      navigator.clipboard.writeText(text);
    }
  }

  function maxPoints(): number {
    if (!results || results.top5.length === 0) return 1;
    return Math.max(...results.top5.map(x => x.points));
  }
</script>

<svelte:head>
  {#if data.og}
    <title>{data.og.title} — rachao.app</title>
    <meta property="og:title" content="{data.og.title} — rachao.app" />
    <meta property="og:description" content={data.og.description} />
    <meta property="og:image" content="https://rachao.app/banner-lp.jpg" />
    <meta property="og:url" content="https://rachao.app/match/{matchHash}/results" />
  {:else}
    <title>Resultado do Rachão — rachao.app</title>
  {/if}
</svelte:head>

<PageBackground>
  <main class="relative z-10 max-w-lg mx-auto px-4 pt-4 {$isLoggedIn ? 'pb-10' : 'pb-24'}">
    <button
      onclick={() => history.length > 1 ? history.back() : (window.location.href = `/match/${matchHash}`)}
      class="mb-4 flex items-center gap-1 text-sm font-medium text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 transition-colors">
      {$t('results.back')}
    </button>

    <!-- Match info card -->
    {#if match}
      <div class="card mb-4 overflow-hidden">
        <div class="relative px-4 py-4 text-white" style="min-height:80px;">
          <picture>
            <source srcset="/banners/banner-{match.court_type ?? 'default'}.webp" type="image/webp" />
            <img
              src="/banners/banner-{match.court_type ?? 'default'}.jpg"
              alt="" aria-hidden="true" width="1920" height="600"
              class="absolute inset-0 w-full h-full object-cover object-center"
            />
          </picture>
          <div class="absolute inset-0 bg-primary-900/80"></div>
          <div class="relative">
            <p class="text-xs font-bold text-white/70 mb-0.5">#{match.number} {match.group_name}</p>
            <h2 class="text-lg font-bold capitalize">{fmtDate(match.match_date)}</h2>
            <div class="flex flex-wrap gap-3 mt-1.5 text-primary-100 text-sm">
              <span class="flex items-center gap-1.5"><Clock size={13} />{fmtTimeRange(match.start_time, match.end_time)}</span>
              {#if match.address}
                <a href="https://maps.google.com/?q={encodeURIComponent(match.address)}" target="_blank" rel="noopener noreferrer"
                  class="flex items-center gap-1.5 underline underline-offset-2 hover:text-white transition-colors">
                  <MapPin size={13} />{match.location}
                </a>
              {:else}
                <span class="flex items-center gap-1.5"><MapPin size={13} />{match.location}</span>
              {/if}
            </div>
            {#if match.court_type || match.players_per_team}
              <div class="flex flex-wrap gap-2 mt-1.5 text-primary-200 text-xs">
                {#if match.court_type}<span class="bg-primary-800/40 rounded px-2 py-0.5">{COURT_LABELS[match.court_type]}</span>{/if}
                {#if match.players_per_team}<span class="bg-primary-800/40 rounded px-2 py-0.5">{$t('results.line_goalkeeper').replace('{n}', String(match.players_per_team))}</span>{/if}
              </div>
            {/if}
          </div>
        </div>
      </div>
    {/if}

    {#if loading}
      <div class="animate-pulse space-y-4">
        <div class="h-8 bg-gray-200 dark:bg-gray-700 rounded w-2/3"></div>
        <div class="h-40 bg-gray-100 dark:bg-gray-700 rounded"></div>
      </div>

    {:else if error}
      <div class="card p-10 text-center">
        <p class="text-4xl mb-3">⏳</p>
        <p class="text-base font-semibold text-gray-700 dark:text-gray-200 mb-1">{$t('results.unavailable_title')}</p>
        <p class="text-sm text-gray-400 dark:text-gray-500">{error}</p>
        <a href="/match/{matchHash}" class="mt-4 inline-block btn btn-secondary btn-sm">{$t('results.see_match')}</a>
      </div>

    {:else if results}
      <!-- Header -->
      <div class="text-center mb-6">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-gray-100 flex items-center justify-center gap-2">
          <Trophy size={24} class="text-amber-500" />
          {$t('results.title')}
        </h1>
        <p class="text-sm text-gray-600 dark:text-gray-300 mt-1">
          {$t('results.voters').replace('{voted}', String(results.total_voters)).replace('{eligible}', String(results.eligible_voters)).replace('{plural}', results.eligible_voters !== 1 ? 'es' : '')}
        </p>
      </div>

      {#if results.top5.length === 0}
        <div class="card p-8 text-center">
          <p class="text-gray-400 dark:text-gray-500 text-sm">{$t('results.no_votes')}</p>
        </div>
      {:else}
        <!-- Podium -->
        <div class="card mb-4 px-4 pt-6 pb-4 overflow-hidden">
          <p class="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide text-center mb-6">{$t('results.podium')}</p>
          <div class="flex items-end justify-center gap-3">
            {#each top3 as item}
              {@const pos = item.position}
              <div class="flex flex-col items-center gap-1 flex-1">
                <!-- Name + medal above podium block -->
                <p class="text-center text-xs font-semibold text-gray-800 dark:text-gray-100 leading-tight line-clamp-2 px-1">{playerDisplayName(item.name, item.nickname)}</p>
                <span class="text-xl">{MEDALS[pos]}</span>
                <span class="text-xs font-bold {podiumTextColor(pos)}">{item.points} pts</span>
                <!-- Podium block -->
                <div class="w-full {podiumHeight(pos)} {podiumBg(pos)} rounded-t-lg flex items-center justify-center">
                  <span class="text-white font-bold text-lg">{pos}º</span>
                </div>
              </div>
            {/each}
          </div>
        </div>

        <!-- 4th and 5th place (if any) -->
        {#if rest.length > 0}
          <div class="card mb-4 overflow-hidden">
            <div class="px-4 py-2 border-b border-gray-100 dark:border-gray-700">
              <p class="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide">{$t('results.also_listed')}</p>
            </div>
            <div class="divide-y divide-gray-100 dark:divide-gray-700">
              {#each rest as item}
                {@const pct = Math.round((item.points / maxPoints()) * 100)}
                <div class="px-4 py-3">
                  <div class="flex items-center gap-2 mb-1">
                    <span class="text-sm font-semibold text-gray-500 dark:text-gray-400 w-6 shrink-0">{item.position}º</span>
                    <span class="text-sm font-medium text-gray-800 dark:text-gray-100 flex-1 truncate">{playerDisplayName(item.name, item.nickname)}</span>
                    <span class="text-xs font-bold text-primary-600 dark:text-primary-400 shrink-0">{item.points} pts</span>
                  </div>
                  <div class="ml-8 h-1.5 bg-gray-100 dark:bg-gray-700 rounded-full overflow-hidden">
                    <div class="h-full bg-primary-500 rounded-full" style="width: {pct}%"></div>
                  </div>
                </div>
              {/each}
            </div>
          </div>
        {/if}

        <!-- Flop -->
        {#if results.flop.length > 0}
          <div class="card mb-4 overflow-hidden">
            <div class="px-4 py-2 border-b border-gray-100 dark:border-gray-700 bg-red-50 dark:bg-red-900/10">
              <p class="text-xs font-semibold text-red-600 dark:text-red-400 uppercase tracking-wide">{$t('results.flop')}</p>
            </div>
            <div class="divide-y divide-gray-100 dark:divide-gray-700">
              {#each results.flop as item}
                <div class="px-4 py-3 flex items-center gap-2">
                  <span class="flex-1 text-sm text-gray-700 dark:text-gray-300 truncate">{playerDisplayName(item.name, item.nickname)}</span>
                  <span class="text-xs text-red-600 dark:text-red-400 shrink-0">{item.votes} voto{item.votes !== 1 ? 's' : ''}</span>
                </div>
              {/each}
            </div>
          </div>
        {/if}

        <!-- Share button -->
        <button
          onclick={shareResults}
          class="w-full btn btn-secondary justify-center gap-2">
          <Share2 size={16} /> {$t('results.share')}
        </button>
      {/if}

      <div class="mt-6 text-center">
        <a href="https://rachao.app" target="_blank" rel="noopener noreferrer"
          class="text-xs text-gray-400 dark:text-gray-600 hover:text-gray-500 dark:hover:text-gray-400 transition-colors">
          rachao.app © 2026
        </a>
      </div>
    {/if}
  </main>
</PageBackground>

<JoinCTABanner />
