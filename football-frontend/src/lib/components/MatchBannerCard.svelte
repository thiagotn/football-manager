<script lang="ts">
  import { Clock, MapPin, Lock, LockOpen } from 'lucide-svelte';
  import { t } from '$lib/i18n';
  import { formatMatchTimeRange } from '$lib/timezoneUtils';
  import type { MatchDetail } from '$lib/api';

  import type { Snippet } from 'svelte';

  let {
    match,
    isGroupAdmin = false,
    togglingStatus = false,
    onToggleOpen = undefined,
    onAskClose = undefined,
    children = undefined,
  }: {
    match: MatchDetail;
    isGroupAdmin?: boolean;
    togglingStatus?: boolean;
    onToggleOpen?: () => void;
    onAskClose?: () => void;
    children?: Snippet;
  } = $props();

  const MONTHS_PT = ['jan','fev','mar','abr','mai','jun','jul','ago','set','out','nov','dez'];
  function fmtDate(d: string) {
    const dt = new Date(d + 'T12:00:00');
    const weekday = dt.toLocaleDateString('pt-BR', { weekday: 'long' });
    return `${weekday.charAt(0).toUpperCase() + weekday.slice(1)}, ${dt.getDate()} de ${MONTHS_PT[dt.getMonth()]}`;
  }

  const courtLabels: Record<string, string> = {
    society: 'Society', futsal: 'Futsal', campo: 'Campo',
    quadra: 'Quadra', beach: 'Beach Soccer', sintetico: 'Sintético',
  };

  function fmtPricingParts(perMatch: number | string | null, monthly: number | string | null): string[] {
    const parts: string[] = [];
    if (perMatch != null) parts.push(`R$ ${Number(perMatch).toFixed(2).replace('.', ',')} ${$t('group.label_per_match')}`);
    if (monthly != null) parts.push(`R$ ${Number(monthly).toFixed(2).replace('.', ',')} ${$t('group.label_monthly')}`);
    return parts;
  }

  const showAdminActions = $derived(isGroupAdmin && (!!onToggleOpen || !!onAskClose));
</script>

<div class="card mb-4 overflow-hidden">
  <div class="relative overflow-hidden px-4 py-3 sm:py-4 text-white" style="min-height:100px;">
    <picture>
      <source srcset="/banners/banner-{match.court_type ?? 'default'}.webp" type="image/webp" />
      <img
        src="/banners/banner-{match.court_type ?? 'default'}.jpg"
        alt=""
        aria-hidden="true"
        width="1920"
        height="600"
        class="absolute inset-0 w-full h-full object-cover object-center"
      />
    </picture>
    <div class="absolute inset-0 bg-primary-900/80"></div>
    <div class="relative flex items-stretch gap-3">

      <!-- Left: match details -->
      <div class="flex-1 min-w-0 flex flex-col justify-center gap-0.5 sm:gap-0">

        <!-- Row 1: group name + status + lock -->
        <div class="flex items-center gap-1.5 min-w-0">
          <p class="text-xs sm:text-sm font-bold text-white truncate">#{match.number} {match.group_name}</p>
          <div class="flex items-center gap-1 shrink-0">
            {#if match.status === 'in_progress'}
              <span class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full text-xs font-semibold bg-red-500/30 text-red-200 border border-red-400/40">
                <span class="w-1.5 h-1.5 rounded-full bg-red-400 animate-pulse"></span>
                {$t('match.live')}
              </span>
            {:else}
              <span class="badge {match.status === 'open' ? 'bg-green-400 text-green-900' : 'bg-gray-400 text-gray-900'}">
                {match.status === 'open' ? $t('match.open') : $t('match.closed')}
              </span>
            {/if}
            {#if showAdminActions}
              {#if match.status === 'closed'}
                <button onclick={onToggleOpen} disabled={togglingStatus}
                  class="p-0.5 rounded text-white/60 hover:text-white hover:bg-white/10 transition-colors"
                  title={$t('match.reopen_title')}><LockOpen size={13} /></button>
              {:else}
                <button onclick={onAskClose} disabled={togglingStatus}
                  class="p-0.5 rounded text-white/60 hover:text-white hover:bg-white/10 transition-colors"
                  title={$t('match.close_title')}><Lock size={13} /></button>
              {/if}
            {/if}
          </div>
        </div>

        <!-- Row 2: date -->
        <h1 class="text-sm sm:text-xl font-bold capitalize leading-tight">{fmtDate(match.match_date)}</h1>

        <!-- Row 3 mobile: time only -->
        <p class="flex sm:hidden items-center gap-1 text-xs text-primary-100 truncate">
          <Clock size={11} class="shrink-0" />
          <span class="truncate">{formatMatchTimeRange(match.start_time, match.end_time, match.group_timezone)}</span>
        </p>

        <!-- Row 3 desktop: time + location with icons -->
        <div class="hidden sm:flex flex-wrap gap-3 mt-2 text-primary-100 text-sm">
          <span class="flex items-center gap-1.5"><Clock size={14} />{formatMatchTimeRange(match.start_time, match.end_time, match.group_timezone)}</span>
          {#if match.address}
            <a href="https://maps.google.com/?q={encodeURIComponent(match.address)}" target="_blank" rel="noopener noreferrer"
              class="flex items-center gap-1.5 underline underline-offset-2 hover:text-white transition-colors">
              <MapPin size={14} />{match.location}
            </a>
          {:else}
            <span class="flex items-center gap-1.5"><MapPin size={14} />{match.location}</span>
          {/if}
        </div>

        <!-- Row 4: tags -->
        {#if match.court_type || match.players_per_team || match.group_per_match_amount != null || match.group_monthly_amount != null || match.location}
          <div class="flex flex-wrap gap-1 sm:gap-3 mt-0.5 sm:mt-2 text-primary-200 text-xs">
            {#if match.court_type}
              <span class="bg-primary-800/40 rounded px-1.5 sm:px-2 py-0.5">{courtLabels[match.court_type] ?? match.court_type}</span>
            {/if}
            {#if match.players_per_team}
              <span class="bg-primary-800/40 rounded px-1.5 sm:px-2 py-0.5">{$t('match.line_plus_goalkeeper').replace('{n}', String(match.players_per_team))}</span>
            {/if}
            {#if match.location}
              {#if match.address}
                <a href="https://maps.google.com/?q={encodeURIComponent(match.address)}" target="_blank" rel="noopener noreferrer"
                  class="sm:hidden bg-primary-800/40 rounded px-1.5 py-0.5 inline-flex items-center gap-1 hover:text-white transition-colors">
                  <MapPin size={10} class="shrink-0" />{match.location}
                </a>
              {:else}
                <span class="sm:hidden bg-primary-800/40 rounded px-1.5 py-0.5 inline-flex items-center gap-1">
                  <MapPin size={10} class="shrink-0" />{match.location}
                </span>
              {/if}
            {/if}
            {#each fmtPricingParts(match.group_per_match_amount, match.group_monthly_amount) as part}
              <span class="bg-amber-500/30 text-amber-200 rounded px-1.5 sm:px-2 py-0.5 font-medium">{part}</span>
            {/each}
          </div>
        {/if}

        {#if match.notes}
          <p class="text-xs sm:text-sm text-primary-200 mt-0.5 sm:mt-2 bg-primary-800/30 rounded-lg px-2 sm:px-3 py-0.5 sm:py-1.5">{match.notes}</p>
        {/if}

      </div><!-- /left column -->

    </div><!-- /flex row -->
  </div><!-- /banner -->
  {#if children}
    {@render children()}
  {/if}
</div>
