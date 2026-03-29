<script lang="ts">
  import { PLAN_ORDER, PLANS, formatCents } from '$lib/plans';
  import PwaSmartBanner from '$lib/components/PwaSmartBanner.svelte';
  import LanguageSwitcher from '$lib/components/LanguageSwitcher.svelte';
  import { t } from '$lib/i18n';

  let howTab = $state<'organize' | 'play'>('organize');
  let priceCycle = $state<'monthly' | 'yearly'>('monthly');
</script>

<PwaSmartBanner />

<!-- Top bar: language switcher + login -->
<div class="fixed top-0 left-0 right-0 z-50 flex items-center justify-end gap-2 px-4 py-2 bg-gray-950/80 backdrop-blur-sm border-b border-white/10">
  <LanguageSwitcher variant="bar" />
  <a
    href="/login"
    class="text-sm font-semibold text-white bg-white/10 hover:bg-white/20 border border-white/25 px-4 py-1.5 rounded-lg transition-colors"
  >{$t('lp.topbar_login')}</a>
</div>

<svelte:head>
  <title>{$t('lp.page_title')}</title>
  <meta name="description" content={$t('lp.page_desc')} />
  <link rel="canonical" href="https://rachao.app/lp" />
  <meta property="og:url" content="https://rachao.app/lp" />
  <meta property="og:title" content={$t('lp.page_title')} />
  <meta property="og:description" content={$t('lp.page_desc')} />
  <meta property="og:image" content="https://rachao.app/background-login.png" />
  <meta property="og:type" content="website" />
  <script type="application/ld+json">{JSON.stringify({
    "@context": "https://schema.org",
    "@type": "WebApplication",
    "name": "rachao.app",
    "url": "https://rachao.app",
    "description": "Organize seu rachão ou encontre uma pelada aberta. Confirme presenças, sorteie times, vote no melhor. Grátis.",
    "applicationCategory": "SportsApplication",
    "operatingSystem": "Web",
    "inLanguage": "pt-BR",
    "offers": {
      "@type": "Offer",
      "price": "0",
      "priceCurrency": "BRL",
      "description": "Plano gratuito disponível"
    }
  })}</script>
  <script type="application/ld+json">{JSON.stringify({
    "@context": "https://schema.org",
    "@type": "SportsEvent",
    "name": "Rachão de futebol society",
    "sport": "Football",
    "description": "Partidas abertas de futebol society com vagas disponíveis — organizadas pelo rachao.app"
  })}</script>
</svelte:head>

<!-- ============================================================ -->
<!-- SECTION 1 — HERO DUAL                                        -->
<!-- ============================================================ -->
<section
  class="relative overflow-hidden text-white"
  style="min-height:620px; background-image: url('/background-login.png'); background-size: cover; background-position: center top;"
>
  <div class="absolute inset-0 bg-gradient-to-b from-gray-900/90 via-gray-900/80 to-gray-900/70"></div>
  <div class="relative max-w-4xl mx-auto px-6 pt-24 pb-16 text-center">
    <img
      src="/logo.png"
      alt="rachao.app"
      width="320"
      height="174"
      class="w-56 sm:w-72 block mx-auto mb-8"
    />
    <h1 class="text-3xl sm:text-4xl font-bold text-white max-w-xl mx-auto mb-4 leading-tight">
      {$t('lp.hero_title')}
    </h1>
    <p class="text-white/75 max-w-lg mx-auto mb-4 text-sm sm:text-base leading-relaxed">
      {$t('lp.hero_subtitle')}
    </p>
    <p class="text-white/50 text-xs mb-10 tracking-wide">{$t('lp.hero_badges')}</p>

    <!-- Dual cards -->
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 max-w-2xl mx-auto">
      <!-- Card Organizador -->
      <div class="bg-white/10 border border-white/20 backdrop-blur-sm rounded-2xl p-6 text-left flex flex-col gap-3">
        <h2 class="text-lg font-bold text-white">{$t('lp.hero_organizer_title')}</h2>
        <p class="text-white/70 text-sm leading-relaxed flex-1">{$t('lp.hero_organizer_desc')}</p>
        <a
          href="/register"
          class="inline-flex items-center justify-center px-5 py-2.5 bg-primary-500 hover:bg-primary-400 text-white rounded-xl font-semibold text-sm transition-colors"
        >
          {$t('lp.hero_organizer_cta')}
        </a>
      </div>
      <!-- Card Jogador -->
      <div class="bg-white/10 border border-white/20 backdrop-blur-sm rounded-2xl p-6 text-left flex flex-col gap-3">
        <h2 class="text-lg font-bold text-white">{$t('lp.hero_player_title')}</h2>
        <p class="text-white/70 text-sm leading-relaxed flex-1">{$t('lp.hero_player_desc')}</p>
        <a
          href="/discover"
          class="inline-flex items-center justify-center px-5 py-2.5 bg-white/15 hover:bg-white/25 border border-white/40 text-white rounded-xl font-semibold text-sm transition-colors"
        >
          {$t('lp.hero_player_cta')}
        </a>
      </div>
    </div>
  </div>
</section>

<!-- ============================================================ -->
<!-- SECTION 2 — PARA ORGANIZADORES                               -->
<!-- ============================================================ -->
<section
  id="organizar"
  class="relative py-16 px-6 overflow-hidden"
  style="background-image: url('/organizer-bg.jpg'); background-size: cover; background-position: center top;"
>
  <div class="absolute inset-0 bg-gray-900/85"></div>
  <div class="relative max-w-5xl mx-auto">
    <div class="text-center mb-12">
      <span class="text-xs font-semibold text-primary-400 uppercase tracking-widest">{$t('lp.organizer_badge')}</span>
      <h2 class="text-2xl sm:text-3xl font-bold text-white mt-2">{$t('lp.organizer_title')}</h2>
      <p class="text-white/60 mt-2 text-sm max-w-lg mx-auto">{$t('lp.organizer_subtitle')}</p>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-8 items-start">
      <!-- Feature grid 2x3 -->
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        {#each [
          { emoji: '👥', title: $t('lp.organizer_feat1_title'), desc: $t('lp.organizer_feat1_desc') },
          { emoji: '✅', title: $t('lp.organizer_feat2_title'), desc: $t('lp.organizer_feat2_desc') },
          { emoji: '🎲', title: $t('lp.organizer_feat3_title'), desc: $t('lp.organizer_feat3_desc') },
          { emoji: '📲', title: $t('lp.organizer_feat4_title'), desc: $t('lp.organizer_feat4_desc') },
          { emoji: '🏆', title: $t('lp.organizer_feat5_title'), desc: $t('lp.organizer_feat5_desc') },
          { emoji: '🔄', title: $t('lp.organizer_feat6_title'), desc: $t('lp.organizer_feat6_desc') },
        ] as feat}
          <div class="bg-gray-800/60 border border-gray-700/50 rounded-2xl p-5 flex gap-3">
            <span class="text-2xl shrink-0">{feat.emoji}</span>
            <div>
              <h3 class="font-semibold text-white text-sm mb-1">{feat.title}</h3>
              <p class="text-white/55 text-xs leading-relaxed">{feat.desc}</p>
            </div>
          </div>
        {/each}
      </div>

      <!-- Mock voting result card -->
      <div class="bg-gray-800 border border-gray-700 rounded-2xl p-5">
        <div class="flex items-center gap-2 mb-4">
          <span class="text-lg">🏆</span>
          <span class="font-bold text-white text-sm">{$t('lp.mock_result_title')}</span>
          <span class="ml-auto text-xs text-gray-400">{$t('lp.mock_result_voted')}</span>
        </div>
        <div class="space-y-2 mb-4">
          {#each [
            { pos: '🥇', name: 'Zidanilo', pts: 67 },
            { pos: '🥈', name: 'Lê', pts: 51 },
            { pos: '🥉', name: 'Dudu', pts: 44 },
            { pos: '4º', name: 'Claudinho30', pts: 29 },
            { pos: '5º', name: 'Kafa', pts: 18 },
          ] as item}
            <div class="flex items-center gap-2 bg-gray-700/50 rounded-xl px-3 py-2 border border-gray-600/40">
              <span class="text-base w-6 text-center text-gray-300 font-semibold">{item.pos}</span>
              <span class="flex-1 text-sm font-medium text-white">{item.name}</span>
              <span class="text-xs font-bold text-amber-400">{item.pts} pts</span>
            </div>
          {/each}
        </div>
        <div class="bg-red-900/40 rounded-xl px-3 py-2 border border-red-700/40 flex items-center gap-2">
          <span class="text-base">😬</span>
          <span class="text-sm text-gray-300 flex-1">{$t('lp.mock_disappointment_label')} <strong class="text-white">Jo Letra</strong></span>
          <span class="text-xs text-red-400 font-semibold">9 {$t('lp.mock_votes_label')}</span>
        </div>
        <p class="text-xs text-gray-500 text-center mt-3">{$t('lp.mock_illustrative')}</p>
      </div>
    </div>

    <div class="text-center mt-10">
      <a
        href="/register"
        class="inline-flex items-center justify-center px-8 py-3 bg-primary-500 hover:bg-primary-400 text-white rounded-xl font-semibold transition-colors shadow-md"
      >
        {$t('lp.organizer_cta')}
      </a>
    </div>
  </div>
</section>

<!-- ============================================================ -->
<!-- SECTION 3 — PARA JOGADORES                                   -->
<!-- ============================================================ -->
<section
  id="jogar"
  class="relative py-16 px-6 overflow-hidden"
  style="background-image: url('/drible.jpg'); background-size: cover; background-position: center;"
>
  <div class="absolute inset-0 bg-gray-900/80 backdrop-blur-[2px]"></div>
  <div class="relative max-w-5xl mx-auto">
    <div class="text-center mb-12">
      <span class="text-xs font-semibold text-emerald-400 uppercase tracking-widest">{$t('lp.player_badge')}</span>
      <h2 class="text-2xl sm:text-3xl font-bold text-white mt-2">{$t('lp.player_title')}</h2>
      <p class="text-white/60 mt-2 text-sm max-w-lg mx-auto">{$t('lp.player_subtitle')}</p>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
      <!-- A) Partidas abertas -->
      <div>
        <h3 class="text-sm font-semibold text-white/80 uppercase tracking-widest mb-4">{$t('lp.player_open_matches')}</h3>
        <div class="space-y-3">
          <!-- Mock card 1 -->
          <div class="bg-gray-900 border border-gray-700 rounded-2xl p-4">
            <div class="flex items-center gap-2 mb-2">
              <div class="w-8 h-8 rounded-lg bg-blue-900/60 flex items-center justify-center shrink-0">
                <span class="text-sm">⚽</span>
              </div>
              <div>
                <p class="text-sm font-bold text-white">Futebol GQC</p>
                <p class="text-xs text-primary-400 font-medium">{$t('lp.mock_match1_date')}</p>
              </div>
            </div>
            <p class="text-xs text-gray-400 mb-1">📍 BarraSoccer · Sintético</p>
            <div class="flex items-center justify-between">
              <p class="text-xs text-green-400 font-semibold">{$t('lp.mock_match1_confirmed')}</p>
              <span class="inline-block px-2.5 py-1 bg-primary-900/60 text-primary-400 text-xs font-semibold rounded-full border border-primary-700/50">{$t('jogar.join_card')}</span>
            </div>
          </div>
          <!-- Mock card 2 -->
          <div class="bg-gray-900 border border-gray-700 rounded-2xl p-4">
            <div class="flex items-center gap-2 mb-2">
              <div class="w-8 h-8 rounded-lg bg-blue-900/60 flex items-center justify-center shrink-0">
                <span class="text-sm">⚽</span>
              </div>
              <div>
                <p class="text-sm font-bold text-white">Rachão da Vila</p>
                <p class="text-xs text-primary-400 font-medium">{$t('lp.mock_match2_date')}</p>
              </div>
            </div>
            <p class="text-xs text-gray-400 mb-1">📍 Campo do Bosque · Campo</p>
            <div class="flex items-center justify-between">
              <p class="text-xs text-amber-400 font-semibold">{$t('lp.mock_match2_confirmed')}</p>
              <span class="inline-block px-2.5 py-1 bg-amber-900/50 text-amber-400 text-xs font-semibold rounded-full border border-amber-700/50">{$t('lp.mock_match2_spots')}</span>
            </div>
          </div>
          <!-- Mock card 3 -->
          <div class="bg-gray-900 border border-gray-700 rounded-2xl p-4">
            <div class="flex items-center gap-2 mb-2">
              <div class="w-8 h-8 rounded-lg bg-blue-900/60 flex items-center justify-center shrink-0">
                <span class="text-sm">⚽</span>
              </div>
              <div>
                <p class="text-sm font-bold text-white">Pelada dos Brothers</p>
                <p class="text-xs text-primary-400 font-medium">{$t('lp.mock_match3_date')}</p>
              </div>
            </div>
            <p class="text-xs text-gray-400 mb-1">📍 Arena Sport · Quadra</p>
            <div class="flex items-center justify-between">
              <p class="text-xs text-green-400 font-semibold">{$t('lp.mock_match3_confirmed')}</p>
              <span class="inline-block px-2.5 py-1 bg-primary-900/60 text-primary-400 text-xs font-semibold rounded-full border border-primary-700/50">{$t('jogar.join_card')}</span>
            </div>
          </div>
        </div>
        <div class="mt-4 text-center">
          <a href="/discover" class="text-primary-400 hover:text-primary-300 transition-colors text-sm font-semibold">
            {$t('jogar.discover_cta')}
          </a>
        </div>
      </div>

      <!-- B) Ranking mock -->
      <div>
        <h3 class="text-sm font-semibold text-white/80 uppercase tracking-widest mb-4">{$t('lp.player_ranking_label')}</h3>
        <div class="bg-gray-900 border border-gray-700 rounded-2xl p-5">
          <div class="flex items-end justify-center gap-3 mb-6">
            <!-- 2º -->
            <div class="flex-1 flex flex-col items-center">
              <img src="/avatar-dentinho.jpg" alt="Dentinho" class="w-11 h-11 rounded-full object-cover mb-1 ring-2 ring-gray-500" />
              <p class="text-xs font-semibold text-gray-300 truncate w-full text-center">Dentinho</p>
              <p class="text-[10px] text-gray-500">287 pts</p>
              <div class="w-full bg-gray-600 rounded-t-lg mt-1" style="height:55px;"></div>
              <div class="w-full bg-gray-600 py-1 text-center text-gray-200 text-xs font-bold">2</div>
            </div>
            <!-- 1º -->
            <div class="flex-1 flex flex-col items-center">
              <img src="/avatar-alemao.jpg" alt="Alemão" class="w-12 h-12 rounded-full object-cover mb-1 ring-2 ring-amber-400" />
              <p class="text-xs font-bold text-white truncate w-full text-center">Alemão</p>
              <p class="text-[10px] text-gray-400">312 pts</p>
              <div class="w-full bg-amber-500 rounded-t-lg mt-1" style="height:85px;"></div>
              <div class="w-full bg-amber-500 py-1 text-center text-white text-xs font-bold">🥇 1</div>
            </div>
            <!-- 3º -->
            <div class="flex-1 flex flex-col items-center">
              <img src="/avatar-fezinha.jpg" alt="Fezinha" class="w-11 h-11 rounded-full object-cover mb-1 ring-2 ring-orange-500" />
              <p class="text-xs font-semibold text-gray-300 truncate w-full text-center">Fezinha</p>
              <p class="text-[10px] text-gray-500">241 pts</p>
              <div class="w-full bg-orange-600/70 rounded-t-lg mt-1" style="height:40px;"></div>
              <div class="w-full bg-orange-600/70 py-1 text-center text-orange-100 text-xs font-bold">3</div>
            </div>
          </div>
          <div class="text-center">
            <a href="/ranking" class="text-primary-400 hover:text-primary-300 transition-colors text-sm font-semibold">
              {$t('jogar.ranking_cta')}
            </a>
          </div>
        </div>
      </div>

      <!-- C) Rachão Score -->
      <div>
        <h3 class="text-sm font-semibold text-white/80 uppercase tracking-widest mb-4">{$t('lp.player_score_label')}</h3>
        <div class="bg-gray-900 border border-gray-700 rounded-2xl overflow-hidden">
          <div
            class="relative px-5 py-5 text-white"
            style="background: linear-gradient(135deg, #166534 0%, #15803d 60%, #16a34a 100%);"
          >
            <div class="flex items-center gap-3 mb-3">
              <img src="/avatar-coruja.jpg" alt="Coruja" class="w-12 h-12 rounded-full object-cover ring-2 ring-white/30" />
              <div>
                <p class="font-bold text-base leading-tight">Coruja</p>
                <div class="flex gap-0.5 mt-0.5">
                  {#each Array.from({ length: 5 }) as _}
                    <span class="text-amber-300 text-sm">★</span>
                  {/each}
                </div>
              </div>
            </div>
            <div class="flex items-center gap-3 text-xs text-green-100 flex-wrap">
              <span>{$t('lp.mock_score_games')}</span>
              <span>{$t('lp.mock_score_attendance')}</span>
              <span>{$t('lp.mock_score_streak')}</span>
            </div>
          </div>
          <div class="bg-gray-800 px-5 py-3 space-y-1.5">
            <p class="text-sm text-gray-300">{$t('lp.mock_score_top')}</p>
            <p class="text-sm text-gray-500">{$t('lp.mock_score_disappoint')}</p>
          </div>
        </div>
        <p class="text-xs text-white/50 text-center mt-3 leading-relaxed">{$t('lp.player_score_highlight')}</p>
      </div>
    </div>

    <div class="text-center mt-10">
      <a
        href="/register"
        class="inline-flex items-center justify-center px-8 py-3 bg-primary-500 hover:bg-primary-400 text-white rounded-xl font-semibold transition-colors shadow-md"
      >
        {$t('lp.player_cta')}
      </a>
    </div>
  </div>
</section>

<!-- ============================================================ -->
<!-- SECTION 4 — POR QUE É GRÁTIS                                 -->
<!-- ============================================================ -->
<section class="relative bg-primary-900 py-16 px-6 overflow-hidden">
  <!-- Subtle dot pattern texture -->
  <svg class="absolute inset-0 w-full h-full pointer-events-none opacity-5" xmlns="http://www.w3.org/2000/svg">
    <defs>
      <pattern id="dots" x="0" y="0" width="20" height="20" patternUnits="userSpaceOnUse">
        <circle cx="2" cy="2" r="1.5" fill="white" />
      </pattern>
    </defs>
    <rect width="100%" height="100%" fill="url(#dots)" />
  </svg>
  <div class="relative max-w-4xl mx-auto">
    <div class="text-center mb-12">
      <h2 class="text-2xl sm:text-3xl font-bold text-white">{$t('lp.free_title')}</h2>
      <p class="text-white/60 mt-3 text-sm max-w-xl mx-auto">{$t('lp.free_subtitle')}</p>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 gap-6">
      <!-- Para jogadores -->
      <div class="bg-primary-800/50 border border-primary-700/50 rounded-2xl p-6">
        <h3 class="font-bold text-white mb-4 flex items-center gap-2">
          <span>⚽</span> {$t('lp.free_players_title')}
        </h3>
        <ul class="space-y-3">
          {#each [
            $t('lp.free_players_1'),
            $t('lp.free_players_2'),
            $t('lp.free_players_3'),
            $t('lp.free_players_4'),
            $t('lp.free_players_5'),
            $t('lp.free_players_6'),
          ] as item}
            <li class="flex items-center gap-2 text-sm text-white/80">
              <span class="text-emerald-400 shrink-0">✓</span>
              {item}
            </li>
          {/each}
        </ul>
      </div>
      <!-- Para organizadores (grátis) -->
      <div class="bg-primary-800/50 border border-primary-700/50 rounded-2xl p-6">
        <h3 class="font-bold text-white mb-4 flex items-center gap-2">
          <span>🏟️</span> {$t('lp.free_organizer_title')}
        </h3>
        <ul class="space-y-3">
          {#each [
            $t('lp.free_org_1'),
            $t('lp.free_org_2'),
            $t('lp.free_org_3'),
            $t('lp.free_org_4'),
            $t('lp.free_org_5'),
            $t('lp.free_org_6'),
          ] as item}
            <li class="flex items-center gap-2 text-sm text-white/80">
              <span class="text-emerald-400 shrink-0">✓</span>
              {item}
            </li>
          {/each}
        </ul>
      </div>
    </div>
  </div>
</section>

<!-- ============================================================ -->
<!-- SECTION 5 — COMO FUNCIONA                                    -->
<!-- ============================================================ -->
<section class="bg-gray-900 py-16 px-6">
  <div class="max-w-4xl mx-auto">
    <div class="text-center mb-10">
      <h2 class="text-2xl sm:text-3xl font-bold text-white">{$t('lp.how_title')}</h2>
    </div>

    <!-- Toggle pill -->
    <div class="flex justify-center mb-10">
      <div class="flex bg-gray-800 border border-gray-700 rounded-full p-1 gap-1">
        <button
          onclick={() => (howTab = 'organize')}
          class="px-5 py-2 rounded-full text-sm font-semibold transition-colors {howTab === 'organize' ? 'bg-primary-600 text-white' : 'text-gray-400 hover:text-white'}"
        >
          {$t('lp.how_tab_organize')}
        </button>
        <button
          onclick={() => (howTab = 'play')}
          class="px-5 py-2 rounded-full text-sm font-semibold transition-colors {howTab === 'play' ? 'bg-primary-600 text-white' : 'text-gray-400 hover:text-white'}"
        >
          {$t('lp.how_tab_play')}
        </button>
      </div>
    </div>

    {#if howTab === 'organize'}
      <div class="grid sm:grid-cols-3 gap-8">
        {#each [
          { num: '01', color: 'text-primary-400', bg: 'bg-primary-900/60', title: $t('lp.how_org_step1_title'), desc: $t('lp.how_org_step1_desc') },
          { num: '02', color: 'text-emerald-400', bg: 'bg-emerald-900/60', title: $t('lp.how_org_step2_title'), desc: $t('lp.how_org_step2_desc') },
          { num: '03', color: 'text-amber-400', bg: 'bg-amber-900/60', title: $t('lp.how_org_step3_title'), desc: $t('lp.how_org_step3_desc') },
        ] as step}
          <div class="text-center">
            <div class="w-14 h-14 rounded-2xl {step.bg} border border-gray-700/50 flex items-center justify-center mx-auto mb-4">
              <span class="text-xl font-bold {step.color}">{step.num}</span>
            </div>
            <h3 class="font-semibold text-white mb-2">{step.title}</h3>
            <p class="text-sm text-white/55 leading-relaxed">{step.desc}</p>
          </div>
        {/each}
      </div>
    {:else}
      <div class="grid sm:grid-cols-3 gap-8">
        {#each [
          { num: '01', color: 'text-primary-400', bg: 'bg-primary-900/60', title: $t('lp.how_play_step1_title'), desc: $t('lp.how_play_step1_desc') },
          { num: '02', color: 'text-emerald-400', bg: 'bg-emerald-900/60', title: $t('lp.how_play_step2_title'), desc: $t('lp.how_play_step2_desc') },
          { num: '03', color: 'text-amber-400', bg: 'bg-amber-900/60', title: $t('lp.how_play_step3_title'), desc: $t('lp.how_play_step3_desc') },
        ] as step}
          <div class="text-center">
            <div class="w-14 h-14 rounded-2xl {step.bg} border border-gray-700/50 flex items-center justify-center mx-auto mb-4">
              <span class="text-xl font-bold {step.color}">{step.num}</span>
            </div>
            <h3 class="font-semibold text-white mb-2">{step.title}</h3>
            <p class="text-sm text-white/55 leading-relaxed">{step.desc}</p>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</section>

<!-- ============================================================ -->
<!-- SECTION 6 — PLANOS                                           -->
<!-- ============================================================ -->
<section class="bg-gray-950 py-16 px-6">
  <div class="max-w-5xl mx-auto">
    <div class="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-4 mb-12">
      <div>
        <span class="text-xs font-semibold text-primary-400 uppercase tracking-widest">{$t('lp.plans_badge')}</span>
        <h2 class="text-2xl sm:text-3xl font-bold text-white mt-2">{$t('lp.plans_title')}</h2>
        <p class="text-white/60 mt-1 text-sm">{$t('lp.plans_subtitle')}</p>
      </div>
      <!-- Toggle mensal/anual -->
      <div class="flex bg-gray-800 border border-gray-700 rounded-full p-1 gap-1 self-start sm:self-auto">
        <button
          onclick={() => (priceCycle = 'monthly')}
          class="px-4 py-1.5 rounded-full text-xs font-semibold transition-colors {priceCycle === 'monthly' ? 'bg-primary-600 text-white' : 'text-gray-400 hover:text-white'}"
        >
          {$t('lp.plans_monthly')}
        </button>
        <button
          onclick={() => (priceCycle = 'yearly')}
          class="px-4 py-1.5 rounded-full text-xs font-semibold transition-colors {priceCycle === 'yearly' ? 'bg-primary-600 text-white' : 'text-gray-400 hover:text-white'}"
        >
          {$t('lp.plans_yearly')}
        </button>
      </div>
    </div>

    <!-- Bloco A — Price cards -->
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-5 mb-12">
      {#each PLAN_ORDER as key}
        {@const plan = PLANS[key]}
        {@const isBasic = plan.key === 'basic'}
        <div class="bg-gray-800 border rounded-2xl p-6 flex flex-col relative
          {isBasic ? 'border-primary-500 ring-1 ring-primary-500' : 'border-gray-700'}">
          {#if isBasic}
            <span class="absolute -top-3 left-1/2 -translate-x-1/2 bg-primary-500 text-white text-xs font-bold px-3 py-1 rounded-full whitespace-nowrap">
              {$t('lp.plans_popular')}
            </span>
          {/if}

          <div class="mb-4">
            <h3 class="text-lg font-bold text-white">{$t(plan.name)}</h3>
          </div>

          <div class="mb-5">
            {#if plan.price_monthly === null}
              <span class="text-3xl font-extrabold text-primary-400">{$t('plans.free')}</span>
            {:else}
              <span class="text-3xl font-extrabold text-white">
                {priceCycle === 'monthly'
                  ? formatCents(plan.price_monthly)
                  : formatCents(Math.round((plan.price_yearly ?? plan.price_monthly * 10) / 12))}
              </span>
              <span class="text-sm text-gray-400">{$t('plans.per_month')}</span>
              {#if priceCycle === 'yearly' && plan.price_yearly}
                <p class="text-xs text-emerald-400 mt-1">{formatCents(plan.price_yearly)}{$t('plans.per_year')}</p>
              {/if}
            {/if}
          </div>

          <ul class="space-y-2 mb-6 flex-1">
            {#each plan.highlights as item}
              <li class="flex items-start gap-2 text-sm text-gray-300">
                <span class="text-primary-400 mt-0.5 shrink-0">✓</span>
                {$t(item)}
              </li>
            {/each}
          </ul>

          {#if plan.key === 'free'}
            <a
              href="/register"
              class="inline-flex items-center justify-center px-4 py-2.5 bg-primary-600 hover:bg-primary-700 text-white rounded-xl font-semibold text-sm transition-colors text-center"
            >
              {$t('plans.free_btn')}
            </a>
          {:else}
            <a
              href="/register?plan={plan.key}"
              class="inline-flex items-center justify-center px-4 py-2.5 {isBasic ? 'bg-primary-600 hover:bg-primary-700' : 'bg-gray-700 hover:bg-gray-600'} text-white rounded-xl font-semibold text-sm transition-colors text-center"
            >
              {$t('plans.subscribe', { name: $t(plan.name) })}
            </a>
          {/if}
        </div>
      {/each}
    </div>

    <!-- Bloco B — Comparison table -->
    <div>
      <h3 class="text-lg font-bold text-white mb-6">{$t('lp.compare_title')}</h3>
      <div class="overflow-x-auto rounded-2xl border border-gray-800">
        <table class="w-full text-sm min-w-[540px]">
          <thead>
            <tr class="border-b border-gray-800">
              <th class="text-left py-3 px-4 text-gray-400 font-semibold w-2/5">{$t('lp.compare_feature')}</th>
              <th class="text-center py-3 px-4 text-white font-semibold">{$t('plan.free.name')}</th>
              <th class="text-center py-3 px-4 text-primary-400 font-semibold">{$t('plan.basic.name')}</th>
              <th class="text-center py-3 px-4 text-white font-semibold">{$t('plan.pro.name')}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-800">
            {#each [
              { label: $t('lp.compare_groups'), free: '1', basic: '3', pro: '10' },
              { label: $t('lp.compare_members'), free: '30', basic: '50', pro: $t('lp.compare_unlimited') },
              { label: $t('lp.compare_matches'), free: '3', basic: $t('lp.compare_unlimited_f'), pro: $t('lp.compare_unlimited_f') },
              { label: $t('lp.compare_history'), free: $t('lp.compare_history_free'), basic: $t('lp.compare_history_basic'), pro: $t('lp.compare_no_limit') },
              { label: $t('lp.compare_invites'), free: '✓', basic: '✓', pro: '✓' },
              { label: $t('lp.compare_attendance'), free: '✓', basic: '✓', pro: '✓' },
              { label: $t('lp.compare_draw'), free: '✓', basic: '✓', pro: '✓' },
              { label: $t('lp.compare_voting'), free: '✓', basic: '✓', pro: '✓' },
              { label: $t('lp.compare_stats'), free: '✓', basic: '✓', pro: '✓' },
              { label: $t('lp.compare_finance'), free: '—', basic: $t('lp.compare_basic_finance'), pro: $t('lp.compare_adv_finance') },
              { label: $t('lp.compare_support'), free: '—', basic: $t('lp.compare_priority'), pro: $t('lp.compare_priority') },
            ] as row}
              <tr class="hover:bg-gray-800/30 transition-colors">
                <td class="py-3 px-4 text-gray-300">{row.label}</td>
                <td class="py-3 px-4 text-center {row.free === '✓' ? 'text-primary-400' : row.free === '—' ? 'text-gray-600' : 'text-white font-semibold'}">{row.free}</td>
                <td class="py-3 px-4 text-center {row.basic === '✓' ? 'text-primary-400' : row.basic === '—' ? 'text-gray-600' : 'text-white font-semibold'}">{row.basic}</td>
                <td class="py-3 px-4 text-center {row.pro === '✓' ? 'text-primary-400' : row.pro === '—' ? 'text-gray-600' : 'text-white font-semibold'}">{row.pro}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
  </div>
</section>

<!-- ============================================================ -->
<!-- SECTION 7 — CTA FINAL                                        -->
<!-- ============================================================ -->
<section
  class="py-20 px-6 text-white text-center"
  style="background: linear-gradient(135deg, #166534 0%, #14532d 50%, #030712 100%);"
>
  <div class="max-w-lg mx-auto">
    <h2 class="text-3xl sm:text-4xl font-bold mb-4">{$t('lp.cta_title')}</h2>
    <p class="text-white/70 mb-10 text-sm leading-relaxed">{$t('lp.cta_subtitle')}</p>
    <div class="flex flex-col sm:flex-row gap-3 justify-center">
      <a
        href="/register"
        class="inline-flex items-center justify-center gap-2 px-8 py-3 bg-white text-primary-700 rounded-xl font-semibold hover:bg-primary-50 transition-colors shadow-md"
      >
        {$t('lp.cta_register')}
      </a>
      <a
        href="/discover"
        class="inline-flex items-center justify-center gap-2 px-8 py-3 bg-white/10 text-white rounded-xl font-semibold hover:bg-white/20 transition-colors border border-white/30"
      >
        {$t('lp.cta_discover')}
      </a>
    </div>
  </div>
</section>

<!-- ============================================================ -->
<!-- FOOTER                                                        -->
<!-- ============================================================ -->
<footer class="bg-gray-950 text-gray-400 py-8 px-6 text-center text-xs">
  <div class="max-w-4xl mx-auto flex flex-col sm:flex-row items-center justify-between gap-3">
    <span class="text-white font-semibold">⚽ rachao.app</span>
    <span class="text-gray-500 text-xs">© 2026</span>
    <div class="flex flex-wrap gap-5 justify-center">
      <a href="/terms" class="hover:text-white transition-colors">{$t('lp.footer_terms')}</a>
      <a href="/privacy" class="hover:text-white transition-colors">{$t('lp.footer_privacy')}</a>
      <a href="/faq" class="hover:text-white transition-colors">{$t('lp.footer_faq')}</a>
      <a href="https://status.rachao.app" target="_blank" rel="noopener noreferrer" class="hover:text-white transition-colors">{$t('lp.footer_status')}</a>
      <a href="/login" class="hover:text-white transition-colors">{$t('lp.topbar_login')}</a>
    </div>
  </div>
</footer>

<style>
  /* Mobile: anchor background image to top so players' upper bodies are visible */
  @media (max-width: 767px) {
    #jogar {
      background-position: 60% top !important;
    }
  }
</style>
