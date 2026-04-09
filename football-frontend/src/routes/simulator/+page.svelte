<script lang="ts">
  import { browser } from '$app/environment';
  import { t } from '$lib/i18n';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import StarRating from '$lib/components/StarRating.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import LanguageSwitcher from '$lib/components/LanguageSwitcher.svelte';
  import { Shuffle, Trash2, UserPlus, RotateCcw, AlertTriangle, Check, Star } from 'lucide-svelte';
  import { authStore } from '$lib/stores/auth';

  let isLoggedIn = $derived(!!$authStore.player);
  import { buildTeams, POS_ABBR, POS_COLOR_CLASSES, sortPlayersByPosition, type DrawPlayer, type TeamResult, type Position } from '$lib/team-builder';
  import { seedWithIds, allianceSeedWithIds } from '$lib/draw-seed';
  import { shuffledNames } from '$lib/team-names';

  const STORAGE_KEY = 'draw_players_v1';

  function loadPlayers(): DrawPlayer[] {
    if (browser) {
      try {
        const saved = localStorage.getItem(STORAGE_KEY);
        if (saved) return JSON.parse(saved) as DrawPlayer[];
      } catch { /* fall through to empty */ }
    }
    return [];
  }

  const _initial = loadPlayers();
  let players        = $state<DrawPlayer[]>(_initial);
  let playersPerTeam = $state(5);
  let result         = $state<TeamResult | null>(null);
  let errors         = $state<string[]>([]);
  let warnings       = $state<string[]>([]);
  let resetOpen      = $state(false);
  let clearOpen      = $state(false);
  let nextId         = $state(_initial.length);

  // Persist to localStorage whenever the list changes
  $effect(() => {
    if (browser) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(players));
    }
  });

  let activePlayers = $derived(players.filter(p => p.active));
  let teamSize      = $derived(playersPerTeam + 1);
  let numTeams      = $derived(Math.floor(activePlayers.length / teamSize));
  let numReserves   = $derived(activePlayers.length - numTeams * teamSize);
  let canSort       = $derived(numTeams >= 2);

  function addPlayer() {
    players.push({
      id:       String(nextId++),
      name:     '',
      nickname: '',
      stars:    3,
      position: 'midfielder',
      active:   true,
    });
  }

  function removePlayer(i: number) {
    players.splice(i, 1);
  }

  function validate(): boolean {
    errors   = [];
    warnings = [];

    if (activePlayers.some(p => !p.nickname.trim() && !p.name.trim())) {
      errors = [$t('draw.error.no_name')];
      return false;
    }
    if (numTeams < 2) {
      errors = [$t('draw.error.min_players', { min: String(teamSize * 2), teams: '2' })];
      return false;
    }
    if (!activePlayers.some(p => p.position === 'goalkeeper')) {
      warnings = [$t('draw.warn.no_goalkeepers')];
    }
    return true;
  }

  function sort() {
    if (!validate()) return;
    result = buildTeams(activePlayers, playersPerTeam, numTeams, shuffledNames());
  }

  function generateSeed() {
    const seed = seedWithIds();
    players  = seed;
    nextId   = seed.length;
    result   = null;
    errors   = [];
    warnings = [];
  }

  function generateAllianceSeed() {
    const seed = allianceSeedWithIds();
    players  = seed;
    nextId   = seed.length;
    result   = null;
    errors   = [];
    warnings = [];
  }

  function resetToDefaults() {
    generateSeed();
    if (browser) localStorage.removeItem(STORAGE_KEY);
  }

  function clearList() {
    players  = [];
    nextId   = 0;
    result   = null;
    errors   = [];
    warnings = [];
    if (browser) localStorage.removeItem(STORAGE_KEY);
  }

  function displayName(p: { nickname: string; name: string }): string {
    return p.nickname.trim() || p.name.trim() || '—';
  }

  function starChars(n: number): string {
    return '★'.repeat(n) + '☆'.repeat(5 - n);
  }
</script>

<svelte:head>
  <title>Simulador de Sorteio — rachao.app</title>
  <meta name="description" content="Monte e teste seus times antes do rachão. Ferramenta gratuita de sorteio equilibrado por posição e habilidade." />
</svelte:head>

{#if !isLoggedIn}
  <div class="fixed top-0 left-0 right-0 z-50 flex items-center justify-end gap-2 px-4 py-2 bg-gray-950/80 backdrop-blur-sm border-b border-white/10">
    <LanguageSwitcher variant="bar" />
    <a
      href="/login"
      class="text-sm font-semibold text-white bg-white/10 hover:bg-white/20 border border-white/25 px-4 py-1.5 rounded-lg transition-colors"
    >{$t('lp.topbar_login')}</a>
  </div>
{/if}

<PageBackground>
  <main class="relative z-10 max-w-7xl mx-auto px-4 pb-24" class:pt-8={isLoggedIn} class:pt-16={!isLoggedIn}>

    <!-- Cabeçalho padrão -->
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-white flex items-center gap-2">
          <Shuffle size={24} class="text-primary-400" /> {$t('draw.title')}
        </h1>
        <p class="text-sm text-white/60 mt-0.5">{$t('draw.subtitle')}</p>
      </div>
      {#if players.length > 0}
        <div class="flex items-center gap-2 flex-wrap justify-end">
          <button
            type="button"
            onclick={() => (clearOpen = true)}
            class="btn btn-sm btn-ghost text-red-400/60 hover:text-red-400 flex items-center gap-1.5 text-xs"
          >
            <Trash2 size={13} /> {$t('draw.clear_list')}
          </button>
          <button
            type="button"
            onclick={() => (resetOpen = true)}
            class="btn btn-sm btn-ghost text-white/50 hover:text-white/80 flex items-center gap-1.5 text-xs"
          >
            <RotateCcw size={13} /> {$t('draw.restore_defaults')}
          </button>
        </div>
      {/if}
    </div>

    <!-- Estado vazio: nenhum jogador na lista -->
    {#if players.length === 0}
      <div class="flex flex-col items-center justify-center py-20 gap-6">
        <div class="text-center space-y-2">
          <p class="text-white/40 text-sm">{$t('draw.empty_hint')}</p>
        </div>
        <div class="flex flex-col sm:flex-row flex-wrap justify-center gap-3">
          <button
            type="button"
            onclick={generateSeed}
            class="btn btn-primary flex items-center gap-2 px-6 py-3 text-base font-semibold"
          >
            <Shuffle size={18} /> {$t('draw.generate_btn')}
          </button>
          <button
            type="button"
            onclick={generateAllianceSeed}
            class="btn btn-secondary flex items-center gap-2 px-6 py-3 text-base font-semibold"
          >
            <Shuffle size={18} /> {$t('draw.generate_alliance_btn')}
          </button>
          <button
            type="button"
            onclick={addPlayer}
            class="btn btn-ghost text-white/70 hover:text-white flex items-center gap-2 px-5 py-3"
          >
            <UserPlus size={16} /> {$t('draw.add_player')}
          </button>
        </div>
      </div>
    {:else}

    <!-- Layout principal: lista (esquerda) + config (direita no desktop) -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">

      <!-- Lista de jogadores -->
      <div class="lg:col-span-2 space-y-3">
        <div class="flex items-center justify-between">
          <span class="text-sm text-white/60">
            {$t('draw.active_count', {
              active: String(activePlayers.length),
              total:  String(players.length),
            })}
          </span>
          <button
            type="button"
            onclick={addPlayer}
            class="btn btn-sm btn-ghost text-primary-400 hover:text-primary-300 flex items-center gap-1.5 text-sm"
          >
            <UserPlus size={15} /> {$t('draw.add_player')}
          </button>
        </div>

        <!-- Scroll container -->
        <div class="overflow-y-auto max-h-[32rem] rounded-xl bg-gray-900/85 backdrop-blur-sm divide-y divide-white/10 border border-white/10">
          {#each players as player, i}
            <div class="flex items-center gap-2 px-3 py-2 transition-colors
              {players[i].active ? 'hover:bg-white/10' : 'opacity-40'}">

              <!-- Active toggle -->
              <button
                type="button"
                onclick={() => { players[i].active = !players[i].active; }}
                class="flex-shrink-0 w-5 h-5 rounded border-2 flex items-center justify-center transition-all
                  {players[i].active
                    ? 'bg-primary-500 border-primary-500'
                    : 'border-white/30 bg-transparent'}"
                aria-label={players[i].active ? 'Desativar jogador' : 'Ativar jogador'}
              >
                {#if players[i].active}
                  <Check size={10} class="text-white" />
                {/if}
              </button>

              <!-- Stars (inline, compact) -->
              <div class="flex-shrink-0">
                <StarRating bind:rating={players[i].stars} size={14} />
              </div>

              <!-- Nickname / Nome input -->
              <input
                type="text"
                placeholder={$t('draw.player_name_placeholder')}
                bind:value={players[i].nickname}
                class="flex-1 min-w-0 bg-transparent text-sm text-white placeholder-white/30
                       outline-none border-b border-transparent focus:border-white/40
                       transition-colors py-0.5 truncate"
              />

              <!-- Position select -->
              <select
                bind:value={players[i].position}
                class="flex-shrink-0 bg-gray-800 border border-white/20 text-white text-xs
                       rounded px-1 py-1 outline-none focus:border-primary-400 cursor-pointer"
                style="width: 54px"
              >
                <option value="goalkeeper">GK</option>
                <option value="defender">ZAG</option>
                <option value="fullback">LAT</option>
                <option value="midfielder">MEI</option>
                <option value="forward">ATA</option>
              </select>

              <!-- Remove -->
              <button
                type="button"
                onclick={() => removePlayer(i)}
                class="flex-shrink-0 text-white/30 hover:text-red-400 transition-colors p-0.5"
                aria-label={$t('draw.remove_player')}
              >
                <Trash2 size={13} />
              </button>
            </div>
          {/each}
        </div>
      </div>

      <!-- Painel de configuração (sticky no desktop) -->
      <div class="space-y-4">
        <div class="sticky top-4 space-y-4">

          <div class="bg-gray-900/85 backdrop-blur-sm rounded-xl p-4 space-y-4 border border-white/10">
            <h2 class="text-sm font-semibold text-white/80 uppercase tracking-wider">
              {$t('draw.config_title')}
            </h2>

            <!-- Jogadores por time -->
            <div>
              <p class="text-xs text-white/60 mb-1">
                {$t('draw.players_per_team')}
              </p>
              <div class="flex items-center gap-2">
                <button
                  type="button"
                  onclick={() => { if (playersPerTeam > 1) playersPerTeam--; }}
                  class="w-8 h-8 rounded-lg bg-white/20 hover:bg-white/30 text-white
                         flex items-center justify-center text-lg font-bold transition-colors"
                  disabled={playersPerTeam <= 1}
                >−</button>
                <span class="w-8 text-center text-white font-bold text-lg">{playersPerTeam}</span>
                <button
                  type="button"
                  onclick={() => { if (playersPerTeam < 10) playersPerTeam++; }}
                  class="w-8 h-8 rounded-lg bg-white/20 hover:bg-white/30 text-white
                         flex items-center justify-center text-lg font-bold transition-colors"
                  disabled={playersPerTeam >= 10}
                >+</button>
                <span class="text-xs text-white/50 ml-1">
                  + 1 GK = {teamSize} por time
                </span>
              </div>
            </div>

            <!-- Resumo do sorteio -->
            <div class="bg-black/40 rounded-lg p-3 text-sm">
              {#if activePlayers.length === 0}
                <p class="text-white/40 text-xs">Ative pelo menos {teamSize * 2} jogadores para sortear.</p>
              {:else if numTeams < 2}
                <p class="text-amber-400/80 text-xs">
                  {$t('draw.need_more_players', { n: String(teamSize * 2 - activePlayers.length) })}
                </p>
              {:else}
                <p class="text-white/80">
                  <span class="text-white font-semibold">{numTeams} times</span>
                  &nbsp;·&nbsp;{teamSize} por time
                </p>
                {#if numReserves > 0}
                  <p class="text-white/50 text-xs mt-0.5">
                    {numReserves} reserva{numReserves > 1 ? 's' : ''}
                  </p>
                {:else}
                  <p class="text-white/50 text-xs mt-0.5">Nenhuma reserva</p>
                {/if}
              {/if}
            </div>

            <!-- Avisos -->
            {#each warnings as w}
              <div class="flex items-start gap-2 bg-amber-500/10 border border-amber-500/20 rounded-lg px-3 py-2">
                <AlertTriangle size={14} class="text-amber-400 flex-shrink-0 mt-0.5" />
                <p class="text-xs text-amber-300">{w}</p>
              </div>
            {/each}

            <!-- Erros -->
            {#each errors as e}
              <div class="flex items-start gap-2 bg-red-500/10 border border-red-500/20 rounded-lg px-3 py-2">
                <AlertTriangle size={14} class="text-red-400 flex-shrink-0 mt-0.5" />
                <p class="text-xs text-red-300">{e}</p>
              </div>
            {/each}

            <!-- Botão sortear -->
            <button
              type="button"
              onclick={sort}
              disabled={!canSort}
              class="w-full btn btn-primary flex items-center justify-center gap-2 py-3 font-semibold
                     disabled:opacity-40 disabled:cursor-not-allowed"
            >
              <Shuffle size={16} />
              {result ? $t('draw.resort_btn') : $t('draw.sort_btn')}
            </button>
          </div>

        </div>
      </div>
    </div>

    <!-- Resultado do sorteio -->
    {#if result}
      <div class="mt-8 space-y-6">

        <div class="flex items-center justify-between">
          <h2 class="text-lg font-bold text-white flex items-center gap-2">
            <Star size={18} class="text-primary-400" /> {$t('draw.result_title')}
          </h2>
          <!-- Resumo de equilíbrio -->
          <div class="hidden sm:flex items-center gap-2 text-xs text-white/50 flex-wrap">
            {#each result.teams as team, i}
              <span class="flex items-center gap-1">
                <span class="inline-block w-2.5 h-2.5 rounded-full flex-shrink-0" style="background:{team.color}"></span>
                <span class="text-amber-400">★</span>{team.totalStars}
              </span>
            {/each}
          </div>
        </div>

        <!-- Cards dos times -->
        <div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
          {#each result.teams as team, ti}
            <div class="bg-gray-900/85 backdrop-blur-sm rounded-xl overflow-hidden border border-white/10">
              <!-- Header do time -->
              <div class="flex items-center justify-between px-4 py-3 border-b border-white/10"
                   style="border-left: 3px solid {team.color}">
                <div class="flex items-center gap-2 min-w-0">
                  <span class="font-bold text-white truncate">{team.name}</span>
                  {#if !team.hasGoalkeeper}
                    <span class="text-[10px] bg-amber-500/20 text-amber-300 border border-amber-500/30
                                 px-1.5 py-0.5 rounded font-medium flex-shrink-0">
                      {$t('draw.no_gk_badge')}
                    </span>
                  {/if}
                </div>
                <span class="text-amber-400 text-sm font-semibold flex-shrink-0 ml-2">
                  ★ {team.totalStars}
                </span>
              </div>

              <!-- Lista de jogadores -->
              <div class="divide-y divide-white/[0.06]">
                {#each sortPlayersByPosition(team.players) as p}
                  <div class="flex items-center gap-2 px-4 py-2">
                    <span class="text-[10px] font-bold px-1.5 py-0.5 rounded flex-shrink-0
                                 {POS_COLOR_CLASSES[p.position as Position]}">
                      {POS_ABBR[p.position as Position]}
                    </span>
                    <span class="flex-1 min-w-0 text-sm text-white truncate">
                      {displayName(p)}
                    </span>
                    <span class="text-amber-400 text-xs flex-shrink-0 tracking-tight">
                      {starChars(p.stars)}
                    </span>
                  </div>
                {/each}
              </div>
            </div>
          {/each}
        </div>

        <!-- Equilíbrio (mobile: inline abaixo dos times) -->
        <div class="sm:hidden flex flex-wrap gap-3 text-xs text-white/50">
          <span class="text-white/40">{$t('draw.balance_title')}:</span>
          {#each result.teams as team}
            <span class="flex items-center gap-1">
              <span class="inline-block w-2 h-2 rounded-full" style="background:{team.color}"></span>
              {team.name.split(' ').slice(0, 2).join(' ')}&nbsp;<span class="text-amber-400">★</span>{team.totalStars}
            </span>
          {/each}
        </div>

        <!-- Reservas -->
        {#if result.reserves.length > 0}
          <div class="bg-gray-900/85 backdrop-blur-sm rounded-xl p-4 border border-white/10">
            <h3 class="text-sm font-semibold text-white/70 mb-3">
              {$t('draw.reserves_title', { count: String(result.reserves.length) })}
            </h3>
            <div class="flex flex-wrap gap-2">
              {#each result.reserves as p}
                <div class="flex items-center gap-1.5 bg-white/15 rounded-lg px-3 py-1.5">
                  <span class="text-[10px] font-bold px-1 py-0.5 rounded
                               {POS_COLOR_CLASSES[p.position as Position]}">
                    {POS_ABBR[p.position as Position]}
                  </span>
                  <span class="text-sm text-white">{displayName(p)}</span>
                  <span class="text-amber-400 text-xs">{starChars(p.stars)}</span>
                </div>
              {/each}
            </div>
          </div>
        {/if}

      </div>
    {/if}

    {/if}<!-- end {#if players.length > 0} -->

  </main>
</PageBackground>

<ConfirmDialog
  bind:open={resetOpen}
  message={$t('draw.restore_confirm')}
  confirmLabel={$t('draw.restore_confirm_btn')}
  danger={false}
  onConfirm={resetToDefaults}
/>

<ConfirmDialog
  bind:open={clearOpen}
  message={$t('draw.clear_list_confirm')}
  confirmLabel={$t('draw.clear_list')}
  danger={true}
  onConfirm={clearList}
/>
