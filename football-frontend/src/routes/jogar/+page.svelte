<script lang="ts">
  import { t } from '$lib/i18n';
  import { UserPlus, Search, Footprints } from 'lucide-svelte';
</script>

<svelte:head>
  <title>{$t('jogar.page_title')}</title>
  <meta name="description" content={$t('jogar.page_desc')} />
  <link rel="canonical" href="https://rachao.app/jogar" />
  <meta property="og:url" content="https://rachao.app/jogar" />
  <meta property="og:title" content={$t('jogar.page_title')} />
  <meta property="og:description" content={$t('jogar.page_desc')} />
  <meta property="og:image" content="https://rachao.app/banner-lp.jpg" />
</svelte:head>

<!-- HERO -->
<section class="relative overflow-hidden text-white" style="min-height:540px; background-image: url('/background-login.png'); background-size: cover; background-position: center top;">
  <div class="absolute inset-0 bg-gradient-to-b from-gray-900/85 via-gray-900/70 to-gray-900/60"></div>
  <div class="relative max-w-4xl mx-auto px-6 py-16 text-center">
    <img
      src="/logo.png"
      alt="rachao.app"
      width="320"
      height="174"
      class="w-56 sm:w-72 block mx-auto mb-8"
    />
    <h1 class="text-2xl sm:text-3xl font-bold text-white max-w-xl mx-auto mb-4 leading-tight">
      {$t('jogar.hero_title')}
    </h1>
    <p class="text-white/85 max-w-lg mx-auto mb-6 text-sm leading-relaxed">
      {$t('jogar.hero_subtitle')}
    </p>
    <p class="text-xs font-semibold text-white/70 mb-8 tracking-wide">
      {$t('jogar.hero_badges')}
    </p>
    <div class="flex flex-col sm:flex-row gap-3 justify-center">
      <a href="/discover"
        class="inline-flex items-center justify-center gap-2 px-8 py-3 bg-primary-500 text-white rounded-xl font-semibold hover:bg-primary-400 transition-colors shadow-md">
        {$t('jogar.hero_cta_primary')}
      </a>
      <a href="/register"
        class="inline-flex items-center justify-center gap-2 px-8 py-3 bg-white text-primary-700 rounded-xl font-semibold hover:bg-primary-50 transition-colors shadow-md">
        {$t('jogar.hero_cta_secondary')}
      </a>
    </div>
  </div>
</section>

<!-- PARTIDAS ABERTAS -->
<section class="bg-white py-14 px-6">
  <div class="max-w-4xl mx-auto">
    <div class="text-center mb-10">
      <h2 class="text-2xl font-bold text-gray-900">{$t('jogar.discover_title')}</h2>
    </div>
    <div class="grid sm:grid-cols-3 gap-4 mb-8">
      <!-- Mock card 1 -->
      <div class="bg-white border border-gray-200 rounded-2xl shadow-sm p-4">
        <div class="flex items-center gap-2 mb-3">
          <div class="w-9 h-9 rounded-lg bg-blue-100 flex items-center justify-center shrink-0">
            <span class="text-lg">⚽</span>
          </div>
          <div>
            <p class="text-sm font-bold text-gray-900">Futebol GQC</p>
            <p class="text-xs text-primary-600 font-medium">Sex, 4 abr · 20:30</p>
          </div>
        </div>
        <p class="text-xs text-gray-500 mb-1">📍 BarraSoccer · Sintético</p>
        <p class="text-xs text-green-600 font-semibold mb-3">8/20 confirmados</p>
        <span class="inline-block px-3 py-1.5 bg-primary-100 text-primary-700 text-xs font-semibold rounded-full">{$t('jogar.join_card')}</span>
      </div>
      <!-- Mock card 2 -->
      <div class="bg-white border border-gray-200 rounded-2xl shadow-sm p-4">
        <div class="flex items-center gap-2 mb-3">
          <div class="w-9 h-9 rounded-lg bg-blue-100 flex items-center justify-center shrink-0">
            <span class="text-lg">⚽</span>
          </div>
          <div>
            <p class="text-sm font-bold text-gray-900">Rachão da Vila</p>
            <p class="text-xs text-primary-600 font-medium">Sáb, 5 abr · 10:00</p>
          </div>
        </div>
        <p class="text-xs text-gray-500 mb-1">📍 Campo do Bosque · Campo</p>
        <p class="text-xs text-amber-600 font-semibold mb-3">12/14 confirmados</p>
        <span class="inline-block px-3 py-1.5 bg-amber-100 text-amber-700 text-xs font-semibold rounded-full">2 vagas</span>
      </div>
      <!-- Mock card 3 -->
      <div class="bg-white border border-gray-200 rounded-2xl shadow-sm p-4">
        <div class="flex items-center gap-2 mb-3">
          <div class="w-9 h-9 rounded-lg bg-blue-100 flex items-center justify-center shrink-0">
            <span class="text-lg">⚽</span>
          </div>
          <div>
            <p class="text-sm font-bold text-gray-900">Pelada dos Brothers</p>
            <p class="text-xs text-primary-600 font-medium">Dom, 6 abr · 08:00</p>
          </div>
        </div>
        <p class="text-xs text-gray-500 mb-1">📍 Arena Sport · Quadra</p>
        <p class="text-xs text-green-600 font-semibold mb-3">5/22 confirmados</p>
        <span class="inline-block px-3 py-1.5 bg-primary-100 text-primary-700 text-xs font-semibold rounded-full">{$t('jogar.join_card')}</span>
      </div>
    </div>
    <div class="text-center">
      <a href="/discover" class="text-primary-600 font-semibold hover:text-primary-700 transition-colors text-sm">
        {$t('jogar.discover_cta')}
      </a>
    </div>
  </div>
</section>

<!-- RANKING -->
<section class="bg-gray-50 py-14 px-6">
  <div class="max-w-4xl mx-auto">
    <div class="text-center mb-10">
      <h2 class="text-2xl font-bold text-gray-900">{$t('jogar.ranking_title')}</h2>
    </div>
    <!-- Pódio mockado -->
    <div class="flex items-end justify-center gap-3 mb-8 max-w-xs mx-auto">
      <!-- 2º lugar -->
      <div class="flex-1 flex flex-col items-center">
        <div class="w-12 h-12 rounded-full bg-gray-300 flex items-center justify-center text-xl font-bold text-gray-700 mb-1">L</div>
        <p class="text-xs font-semibold text-gray-700 truncate w-full text-center">Lucas</p>
        <p class="text-[10px] text-gray-400">287 pts</p>
        <div class="w-full bg-gray-400 rounded-t-lg mt-1" style="height:60px;"></div>
        <div class="w-full bg-gray-400 py-1 text-center text-white text-xs font-bold">2</div>
      </div>
      <!-- 1º lugar -->
      <div class="flex-1 flex flex-col items-center">
        <div class="w-14 h-14 rounded-full bg-amber-400 flex items-center justify-center text-2xl font-bold text-amber-900 mb-1">T</div>
        <p class="text-xs font-bold text-gray-800 truncate w-full text-center">Thiagol</p>
        <p class="text-[10px] text-gray-500">312 pts</p>
        <div class="w-full bg-amber-500 rounded-t-lg mt-1" style="height:90px;"></div>
        <div class="w-full bg-amber-500 py-1 text-center text-white text-xs font-bold">🥇 1</div>
      </div>
      <!-- 3º lugar -->
      <div class="flex-1 flex flex-col items-center">
        <div class="w-12 h-12 rounded-full bg-orange-300 flex items-center justify-center text-xl font-bold text-orange-700 mb-1">R</div>
        <p class="text-xs font-semibold text-gray-700 truncate w-full text-center">Rafael</p>
        <p class="text-[10px] text-gray-400">241 pts</p>
        <div class="w-full bg-orange-400 rounded-t-lg mt-1" style="height:45px;"></div>
        <div class="w-full bg-orange-400 py-1 text-center text-white text-xs font-bold">3</div>
      </div>
    </div>
    <div class="text-center">
      <a href="/ranking" class="text-primary-600 font-semibold hover:text-primary-700 transition-colors text-sm">
        {$t('jogar.ranking_cta')}
      </a>
    </div>
  </div>
</section>

<!-- RACHÃO SCORE -->
<section class="bg-white py-14 px-6">
  <div class="max-w-4xl mx-auto">
    <div class="text-center mb-10">
      <h2 class="text-2xl font-bold text-gray-900">{$t('jogar.score_title')}</h2>
      <p class="text-gray-500 mt-2 text-sm max-w-md mx-auto">{$t('jogar.score_subtitle')}</p>
    </div>
    <!-- Mock stat card -->
    <div class="max-w-sm mx-auto">
      <div class="rounded-2xl overflow-hidden shadow-md border border-gray-200">
        <div class="relative px-5 py-5 text-white"
          style="background: linear-gradient(135deg, #166534 0%, #15803d 60%, #16a34a 100%);">
          <div class="flex items-center gap-3 mb-3">
            <div class="w-14 h-14 rounded-full bg-white/20 flex items-center justify-center text-2xl font-bold text-white">T</div>
            <div>
              <p class="font-bold text-lg leading-tight">Thiago (Thiagol)</p>
              <div class="flex gap-0.5 mt-0.5">
                {#each Array.from({ length: 5 }) as _}
                  <span class="text-amber-300 text-sm">★</span>
                {/each}
              </div>
            </div>
          </div>
          <div class="flex items-center gap-4 text-sm text-green-100">
            <span>⚽ 47 rachões</span>
            <span>89% presença</span>
            <span>🔥 12 seguidos</span>
          </div>
        </div>
        <div class="bg-white px-5 py-3 space-y-1.5">
          <p class="text-sm text-gray-700">🏆 Top 5 em 23 partidas · <span class="font-semibold text-amber-600">312 pontos</span></p>
          <p class="text-sm text-gray-400">😬 Decepção em 2</p>
        </div>
      </div>
    </div>
  </div>
</section>

<!-- POR QUE É GRÁTIS -->
<section class="bg-gray-50 py-14 px-6">
  <div class="max-w-4xl mx-auto">
    <div class="text-center mb-8">
      <h2 class="text-2xl font-bold text-gray-900">{$t('jogar.free_title')}</h2>
    </div>
    <div class="max-w-sm mx-auto space-y-3">
      {#each [
        $t('jogar.free_feature_1'),
        $t('jogar.free_feature_2'),
        $t('jogar.free_feature_3'),
        $t('jogar.free_feature_4'),
        $t('jogar.free_feature_5'),
        $t('jogar.free_feature_6'),
      ] as feature}
        <div class="flex items-center gap-3 bg-white rounded-xl border border-gray-200 px-4 py-3 shadow-sm">
          <span class="text-green-500 text-lg shrink-0">✓</span>
          <span class="text-sm text-gray-700">{feature}</span>
        </div>
      {/each}
    </div>
  </div>
</section>

<!-- COMO FUNCIONA -->
<section class="bg-white py-14 px-6">
  <div class="max-w-4xl mx-auto">
    <div class="text-center mb-12">
      <h2 class="text-2xl font-bold text-gray-900">{$t('jogar.how_title')}</h2>
    </div>
    <div class="grid sm:grid-cols-3 gap-8">
      <div class="text-center">
        <div class="w-14 h-14 rounded-2xl bg-primary-100 flex items-center justify-center mx-auto mb-4">
          <UserPlus size={26} class="text-primary-600" />
        </div>
        <div class="text-xs font-bold text-primary-500 uppercase tracking-widest mb-1">01</div>
        <h3 class="font-semibold text-gray-900 mb-2">{$t('jogar.step1_title')}</h3>
        <p class="text-sm text-gray-500 leading-relaxed">{$t('jogar.step1_desc')}</p>
      </div>
      <div class="text-center">
        <div class="w-14 h-14 rounded-2xl bg-green-100 flex items-center justify-center mx-auto mb-4">
          <Search size={26} class="text-green-600" />
        </div>
        <div class="text-xs font-bold text-green-600 uppercase tracking-widest mb-1">02</div>
        <h3 class="font-semibold text-gray-900 mb-2">{$t('jogar.step2_title')}</h3>
        <p class="text-sm text-gray-500 leading-relaxed">{$t('jogar.step2_desc')}</p>
      </div>
      <div class="text-center">
        <div class="w-14 h-14 rounded-2xl bg-amber-100 flex items-center justify-center mx-auto mb-4">
          <Footprints size={26} class="text-amber-600" />
        </div>
        <div class="text-xs font-bold text-amber-600 uppercase tracking-widest mb-1">03</div>
        <h3 class="font-semibold text-gray-900 mb-2">{$t('jogar.step3_title')}</h3>
        <p class="text-sm text-gray-500 leading-relaxed">{$t('jogar.step3_desc')}</p>
      </div>
    </div>
  </div>
</section>

<!-- CTA FINAL -->
<section class="py-16 px-6 text-white text-center"
  style="background: linear-gradient(135deg, #14532d 0%, #166534 50%, #15803d 100%);">
  <div class="max-w-lg mx-auto">
    <h2 class="text-2xl sm:text-3xl font-bold mb-3">{$t('jogar.cta_title')}</h2>
    <div class="flex flex-col sm:flex-row gap-3 justify-center mt-8">
      <a href="/register"
        class="inline-flex items-center justify-center gap-2 px-8 py-3 bg-white text-primary-700 rounded-xl font-semibold hover:bg-primary-50 transition-colors shadow-md">
        {$t('jogar.cta_register')}
      </a>
      <a href="/discover"
        class="inline-flex items-center justify-center gap-2 px-8 py-3 bg-white/15 text-white rounded-xl font-semibold hover:bg-white/25 transition-colors border border-white/50">
        {$t('jogar.cta_discover')}
      </a>
    </div>
  </div>
</section>

<!-- FOOTER -->
<footer class="bg-gray-900 text-gray-400 text-center py-6 px-6 text-xs">
  <div class="flex flex-wrap justify-center items-center gap-3">
    <span>{$t('jogar.footer')}</span>
    <span>·</span>
    <a href="/terms" class="hover:text-white transition-colors">Termos</a>
    <span>·</span>
    <a href="/privacy" class="hover:text-white transition-colors">Privacidade</a>
  </div>
</footer>
