<script lang="ts">
  import { authStore, isAdmin, currentPlayer } from '$lib/stores/auth';
  import { themeStore } from '$lib/stores/theme';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { Users, LogOut, Home, Trophy, BookOpen, UserCircle, Menu, X, Sun, Moon, ChevronLeft, Star, HelpCircle, FileText, Shield, BarChart2, Calendar, CreditCard, Download, Compass, Globe, Award, ChevronDown, Shuffle, Gamepad2 } from 'lucide-svelte';
  import { billingEnabled } from '$lib/billing';
  import { pwaInstall } from '$lib/stores/pwaInstall';
  import PwaInstallButton from '$lib/components/PwaInstallButton.svelte';
  import LanguageSwitcher from '$lib/components/LanguageSwitcher.svelte';
  import { t, locale, setLocale, SUPPORTED_LOCALES, type Locale } from '$lib/i18n';

  const LANG_LABELS: Record<Locale, { full: string; flag: string }> = {
    'pt-BR': { full: 'Português (BR)', flag: '🇧🇷' },
    'en':    { full: 'English',        flag: '🇺🇸' },
    'es':    { full: 'Español',        flag: '🇪🇸' },
  };

  let showLangModal = $state(false);

  function logout() {
    authStore.logout();
    goto('/login');
  }

  // ── Itens principal (desktop + mobile) ──────────────────────────────────────
  const mainLinks = [
    { href: '/',         icon: Home,     labelKey: 'nav.dashboard' },
    { href: '/groups',   icon: Trophy,   labelKey: 'nav.groups' },
    { href: '/matches',  icon: Calendar, labelKey: 'nav.matches',  playerOnly: true },
    { href: '/discover', icon: Compass,  labelKey: 'nav.discover', playerOnly: true },
    // admin-only links aparecem no lugar dos playerOnly
    { href: '/players',            icon: Users,      labelKey: 'nav.players',       adminOnly: true },
    { href: '/admin/reviews',      icon: Star,       labelKey: 'nav.admin_reviews', adminOnly: true },
    { href: '/admin/subscriptions',icon: CreditCard, labelKey: 'nav.subscriptions', adminOnly: true },
    { href: '/admin/faq',          icon: BookOpen,   labelKey: 'nav.guide',         adminOnly: true },
  ];

  // ── Itens Explorar (dropdown desktop + seção mobile) ────────────────────────
  const exploreLinks = [
    { href: '/ranking',       icon: Award,    labelKey: 'nav.ranking' },
    { href: '/profile/stats', icon: BarChart2,labelKey: 'nav.score',     playerOnly: true },
    { href: '/simulator',     icon: Shuffle,  labelKey: 'nav.simulator' },
  ];

  let menuOpen           = $state(false);
  let showAccountDropdown = $state(false);
  let showExploreDropdown = $state(false);
  let accountDropdownPos  = $state({ top: 0, right: 0 });
  let exploreDropdownPos  = $state({ top: 0, right: 0 });

  function toggleAccountDropdown(e: MouseEvent) {
    if (!showAccountDropdown) {
      const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
      accountDropdownPos = { top: rect.bottom + 4, right: window.innerWidth - rect.right };
      showExploreDropdown = false;
    }
    showAccountDropdown = !showAccountDropdown;
  }

  function toggleExploreDropdown(e: MouseEvent) {
    if (!showExploreDropdown) {
      const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
      exploreDropdownPos = { top: rect.bottom + 4, right: window.innerWidth - rect.right };
      showAccountDropdown = false;
    }
    showExploreDropdown = !showExploreDropdown;
  }

  function closeMenu() { menuOpen = false; showLangModal = false; }

  // Fecha dropdowns e drawer ao navegar
  $effect(() => {
    $page.url.pathname;
    menuOpen = false;
    showAccountDropdown = false;
    showExploreDropdown = false;
  });

  // Trava scroll do body quando o drawer está aberto (iOS-safe: position fixed)
  $effect(() => {
    if (typeof document === 'undefined') return;
    const locked = menuOpen || showLangModal;
    if (locked) {
      const scrollY = window.scrollY;
      document.body.style.position = 'fixed';
      document.body.style.top = `-${scrollY}px`;
      document.body.style.width = '100%';
    } else {
      const top = document.body.style.top;
      document.body.style.position = '';
      document.body.style.top = '';
      document.body.style.width = '';
      if (top) window.scrollTo(0, -parseInt(top));
    }
    return () => {
      const top = document.body.style.top;
      document.body.style.position = '';
      document.body.style.top = '';
      document.body.style.width = '';
      if (top) window.scrollTo(0, -parseInt(top));
    };
  });

  function getBackHref(pathname: string): string | null {
    if (pathname.startsWith('/groups/')) return '/groups';
    if (pathname === '/groups')   return '/';
    if (pathname === '/ranking')  return '/';
    if (pathname === '/players')  return '/';
    if (pathname === '/profile')        return '/';
    if (pathname === '/matches')        return '/';
    if (pathname === '/discover')       return '/';
    if (pathname === '/profile/stats')  return '/';
    if (pathname === '/review')   return '/';
    if (pathname === '/plans')    return '/';
    if (pathname === '/simulator') return '/';
    if (pathname === '/tetris')    return '/';
    if (pathname.startsWith('/account/')) return '/profile';
    if (pathname === '/faq')      return '/';
    if (pathname === '/terms')    return '/';
    if (pathname === '/privacy')  return '/';
    if (pathname.startsWith('/admin/')) return '/';
    if (pathname.startsWith('/players/')) return '/ranking';
    // Toda a árvore /match/* usa history.back() para não criar entradas extras no histórico
    // (evita loop: match → teams → [link] → match → history.back() → teams → ...)
    if (/^\/match\//.test(pathname)) return 'BACK';
    return null;
  }

  let backHref = $derived(getBackHref($page.url.pathname));

  // Está na seção Explorar? (para highlight do botão desktop)
  let exploreActive = $derived(
    exploreLinks.some(l => $page.url.pathname === l.href)
  );
</script>

<nav class="bg-primary-700 text-white shadow-md relative z-40" style="padding-top: env(safe-area-inset-top);">
  <div class="max-w-7xl mx-auto px-4 flex items-center justify-between h-16 relative overflow-hidden">

    <!-- Esquerda: botão voltar (mobile) + logo desktop -->
    <div class="flex items-center gap-1 shrink-0">
      {#if backHref}
        {#if backHref === 'BACK'}
          <button
            onclick={() => { if (history.length > 1) history.back(); else goto('/'); }}
            class="min-[940px]:hidden p-1.5 -ml-1.5 rounded-lg hover:bg-primary-600 transition-colors"
            aria-label={$t('aria.back')}
          >
            <ChevronLeft size={22} />
          </button>
        {:else}
          <a
            href={backHref}
            class="min-[940px]:hidden p-1.5 -ml-1.5 rounded-lg hover:bg-primary-600 transition-colors"
            aria-label={$t('aria.back')}
          >
            <ChevronLeft size={22} />
          </a>
        {/if}
      {/if}
      <!-- Logo desktop -->
      <a href="/" class="hidden min-[940px]:flex -ml-16 self-stretch items-end">
        <img src="/logo.png" alt="rachao.app" class="h-24 w-auto flex-shrink-0 -translate-y-2" />
      </a>
    </div>

    <!-- Logo mobile: centralizado -->
    <a href="/" class="min-[940px]:hidden absolute left-1/2 -translate-x-1/2 top-0 bottom-0 flex items-center pointer-events-auto">
      <img src="/logo.png" alt="rachao.app" class="h-14 w-auto flex-shrink-0" />
    </a>

    <!-- Links — desktop -->
    <div class="hidden min-[940px]:flex items-center gap-1">
      <!-- Itens principais -->
      {#each mainLinks as l}
        {#if (!l.adminOnly || $isAdmin) && (!l.playerOnly || !$isAdmin)}
          <a
            href={l.href}
            class="flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors
              {$page.url.pathname === l.href ? 'bg-primary-900' : 'hover:bg-primary-600'}"
          >
            <l.icon size={15} />
            {$t(l.labelKey)}
          </a>
        {/if}
      {/each}

      <!-- Dropdown Explorar (não aparece para admin) -->
      {#if !$isAdmin}
        <button
          onclick={toggleExploreDropdown}
          class="flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors
            {exploreActive || showExploreDropdown ? 'bg-primary-900' : 'hover:bg-primary-600'}"
        >
          <Compass size={15} />
          {$t('nav.explore')}
          <ChevronDown size={12} class="transition-transform duration-200 {showExploreDropdown ? 'rotate-180' : ''}" />
        </button>
      {/if}
    </div>

    <!-- Direita — desktop -->
    <div class="hidden min-[940px]:flex items-center gap-2">
      {#if $pwaInstall.canInstall}
        <button onclick={() => pwaInstall.install()} class="btn-ghost btn-sm text-emerald-300 hover:text-emerald-100 hover:bg-primary-600" title={$t('nav.install')}>
          <Download size={15} />
          <span>{$t('nav.install')}</span>
        </button>
      {/if}
      <span class="text-sm text-primary-200">{$currentPlayer?.nickname || $currentPlayer?.name}</span>
      <LanguageSwitcher variant="bar" />
      <button
        onclick={toggleAccountDropdown}
        class="btn-ghost btn-sm text-primary-100 hover:text-white hover:bg-primary-600
          {$page.url.pathname === '/profile' || $page.url.pathname === '/review' ? 'bg-primary-900' : ''}"
        title={$t('nav.my_account')}
      >
        <UserCircle size={15} />
        <span>{$t('nav.account')}</span>
        <ChevronDown size={12} class="transition-transform duration-200 {showAccountDropdown ? 'rotate-180' : ''}" />
      </button>
      <button onclick={themeStore.toggle} class="btn-ghost btn-sm text-primary-100 hover:text-white hover:bg-primary-600" title={$t('aria.theme')}>
        {#if $themeStore === 'dark'}<Sun size={15} />{:else}<Moon size={15} />{/if}
      </button>
      <button onclick={logout} class="btn-ghost btn-sm text-primary-100 hover:text-white hover:bg-primary-600">
        <LogOut size={15} />
        <span>{$t('nav.logout')}</span>
      </button>
    </div>

    <!-- Hambúrguer — mobile -->
    <button
      class="min-[940px]:hidden p-2 rounded-lg hover:bg-primary-600 transition-colors"
      onclick={() => menuOpen = !menuOpen}
      aria-label={$t('aria.menu')}
    >
      <Menu size={22} />
    </button>
  </div>

  <!-- Backdrop para fechar dropdowns desktop -->
  {#if showExploreDropdown || showAccountDropdown}
    <button
      class="fixed inset-0 z-[39]"
      onclick={() => { showExploreDropdown = false; showAccountDropdown = false; }}
      tabindex="-1"
      aria-hidden="true"
    ></button>
  {/if}

  <!-- Dropdown "Explorar" — desktop -->
  {#if showExploreDropdown}
    <div
      class="hidden min-[940px]:block fixed z-[41] w-52 bg-primary-800 rounded-xl shadow-xl border border-primary-700 overflow-hidden"
      style="top: {exploreDropdownPos.top}px; right: {exploreDropdownPos.right}px;"
    >
      {#each exploreLinks as l, i}
        {#if !l.playerOnly || !$isAdmin}
          <a href={l.href} onclick={() => showExploreDropdown = false}
            class="flex items-center gap-2.5 px-4 py-3 text-sm font-medium transition-colors
              {i > 0 ? 'border-t border-primary-700/50' : ''}
              {$page.url.pathname === l.href ? 'bg-primary-900 text-white' : 'text-primary-100 hover:bg-primary-700'}">
            <l.icon size={15} /> {$t(l.labelKey)}
          </a>
        {/if}
      {/each}
    </div>
  {/if}

  <!-- Dropdown "Conta" — desktop -->
  {#if showAccountDropdown}
    <div
      class="hidden min-[940px]:block fixed z-[41] w-48 bg-primary-800 rounded-xl shadow-xl border border-primary-700 overflow-hidden"
      style="top: {accountDropdownPos.top}px; right: {accountDropdownPos.right}px;"
    >
      <a href="/profile" onclick={() => showAccountDropdown = false}
        class="flex items-center gap-2.5 px-4 py-3 text-sm font-medium transition-colors
          {$page.url.pathname === '/profile' ? 'bg-primary-900 text-white' : 'text-primary-100 hover:bg-primary-700'}">
        <UserCircle size={15} /> {$t('nav.my_account')}
      </a>
      {#if !$isAdmin}
        <a href="/review" onclick={() => showAccountDropdown = false}
          class="flex items-center gap-2.5 px-4 py-3 text-sm font-medium transition-colors border-t border-primary-700/50
            {$page.url.pathname === '/review' ? 'bg-primary-900 text-white' : 'text-primary-100 hover:bg-primary-700'}">
          <Star size={15} /> {$t('nav.review')}
        </a>
      {/if}
      <a href="/tetris" onclick={() => showAccountDropdown = false}
        class="flex items-center gap-2.5 px-4 py-3 text-sm font-medium transition-colors border-t border-primary-700/50
          {$page.url.pathname === '/tetris' ? 'bg-primary-900 text-white' : 'text-primary-100 hover:bg-primary-700'}">
        <Gamepad2 size={15} /> {$t('nav.tetris')}
      </a>
    </div>
  {/if}
</nav>

<!-- Drawer lateral mobile -->
{#if menuOpen}
  <!-- Backdrop -->
  <button
    class="min-[940px]:hidden fixed inset-0 z-40 bg-black/50"
    onclick={closeMenu}
    aria-label={$t('aria.close_menu')}
  ></button>

  <!-- Painel deslizante da direita -->
  <div class="min-[940px]:hidden fixed top-0 right-0 h-full w-72 max-w-[85vw] z-50 bg-primary-800 shadow-2xl flex flex-col"
    style="animation: slideInRight 0.22s ease-out;">

    <!-- Cabeçalho do drawer -->
    <div class="flex items-center justify-between px-4 h-16 border-b border-primary-700 shrink-0">
      <p class="text-sm font-medium text-primary-200 truncate">{$currentPlayer?.nickname || $currentPlayer?.name}</p>
      <button onclick={closeMenu} class="p-2 rounded-lg hover:bg-primary-700 transition-colors" aria-label={$t('aria.close')}>
        <X size={20} />
      </button>
    </div>

    <div class="flex-1 overflow-y-auto">

      <!-- Seção: Principal -->
      <div class="px-3 pt-3 pb-1">
        <p class="px-3 pb-1 text-[10px] font-semibold uppercase tracking-widest text-primary-400">
          {$t('nav.section.main')}
        </p>
        <div class="space-y-0.5">
          {#each mainLinks as l}
            {#if (!l.adminOnly || $isAdmin) && (!l.playerOnly || !$isAdmin)}
              <a
                href={l.href}
                onclick={closeMenu}
                class="flex items-center gap-3 px-3 py-3 rounded-xl text-sm font-medium transition-colors
                  {$page.url.pathname === l.href ? 'bg-primary-900 text-white' : 'text-primary-100 hover:bg-primary-700'}"
              >
                <l.icon size={18} />
                {$t(l.labelKey)}
              </a>
            {/if}
          {/each}
        </div>
      </div>

      <!-- Seção: Explorar (não aparece para admin) -->
      {#if !$isAdmin}
        <div class="px-3 pt-3 pb-1 border-t border-primary-700/60 mt-1">
          <p class="px-3 pb-1 text-[10px] font-semibold uppercase tracking-widest text-primary-400">
            {$t('nav.section.explore')}
          </p>
          <div class="space-y-0.5">
            {#each exploreLinks as l}
              <a
                href={l.href}
                onclick={closeMenu}
                class="flex items-center gap-3 px-3 py-3 rounded-xl text-sm font-medium transition-colors
                  {$page.url.pathname === l.href ? 'bg-primary-900 text-white' : 'text-primary-100 hover:bg-primary-700'}"
              >
                <l.icon size={18} />
                {$t(l.labelKey)}
              </a>
            {/each}
          </div>
        </div>
      {/if}

      <!-- Seção: Conta -->
      <div class="px-3 pt-3 pb-1 border-t border-primary-700/60 mt-1">
        <p class="px-3 pb-1 text-[10px] font-semibold uppercase tracking-widest text-primary-400">
          {$t('nav.section.account')}
        </p>
        <div class="space-y-0.5">
          {#if billingEnabled && !$isAdmin}
            <a href="/account/subscription" onclick={closeMenu}
              class="flex items-center gap-3 px-3 py-3 rounded-xl text-sm font-medium transition-colors
                {$page.url.pathname.startsWith('/account/') || $page.url.pathname === '/plans' ? 'bg-primary-900 text-white' : 'text-primary-100 hover:bg-primary-700'}">
              <CreditCard size={18} /> {$t('nav.plan')}
            </a>
          {/if}
          <a href="/profile" onclick={closeMenu}
            class="flex items-center gap-3 px-3 py-3 rounded-xl text-sm font-medium transition-colors
              {$page.url.pathname === '/profile' ? 'bg-primary-900 text-white' : 'text-primary-100 hover:bg-primary-700'}">
            <UserCircle size={18} /> {$t('nav.my_account')}
          </a>
          <a href="/faq" onclick={closeMenu}
            class="flex items-center gap-3 px-3 py-3 rounded-xl text-sm font-medium transition-colors
              {$page.url.pathname === '/faq' ? 'bg-primary-900 text-white' : 'text-primary-100 hover:bg-primary-700'}">
            <HelpCircle size={18} /> {$t('nav.faq')}
          </a>
          {#if !$isAdmin}
            <a href="/review" onclick={closeMenu}
              class="flex items-center gap-3 px-3 py-3 rounded-xl text-sm font-medium transition-colors
                {$page.url.pathname === '/review' ? 'bg-primary-900 text-white' : 'text-primary-100 hover:bg-primary-700'}">
              <Star size={18} /> {$t('nav.review')}
            </a>
          {/if}
          <a href="/tetris" onclick={closeMenu}
            class="flex items-center gap-3 px-3 py-3 rounded-xl text-sm font-medium transition-colors
              {$page.url.pathname === '/tetris' ? 'bg-primary-900 text-white' : 'text-primary-100 hover:bg-primary-700'}">
            <Gamepad2 size={18} /> {$t('nav.tetris')}
          </a>
          <button
            onclick={() => showLangModal = true}
            class="flex items-center gap-3 px-3 py-3 rounded-xl text-sm font-medium transition-colors w-full text-left text-primary-100 hover:bg-primary-700"
            aria-label="Language / Idioma"
          >
            <Globe size={18} />
            <span>{LANG_LABELS[$locale].flag} {LANG_LABELS[$locale].full}</span>
          </button>
          <button onclick={themeStore.toggle}
            class="w-full flex items-center gap-3 px-3 py-3 rounded-xl text-sm font-medium hover:bg-primary-700 text-left text-primary-100 transition-colors">
            {#if $themeStore === 'dark'}<Sun size={18} />{:else}<Moon size={18} />{/if}
            {$t('aria.theme')}
          </button>
          <PwaInstallButton />
          <button onclick={logout}
            class="w-full flex items-center gap-3 px-3 py-3 rounded-xl text-sm font-medium hover:bg-primary-700 text-left text-primary-100 transition-colors">
            <LogOut size={18} /> {$t('nav.logout')}
          </button>
        </div>
      </div>

      <!-- Links legais -->
      <div class="flex items-center gap-3 px-6 py-4 mt-1 border-t border-primary-700/40">
        <a href="/terms" onclick={closeMenu}
          class="text-xs text-primary-400 hover:text-primary-200 transition-colors whitespace-nowrap">
          {$t('footer.terms')}
        </a>
        <span class="text-primary-600 text-xs">·</span>
        <a href="/privacy" onclick={closeMenu}
          class="text-xs text-primary-400 hover:text-primary-200 transition-colors whitespace-nowrap">
          {$t('footer.privacy')}
        </a>
      </div>

    </div>
  </div>
{/if}

<!-- Bottom sheet de seleção de idioma (mobile) -->
{#if showLangModal}
  <button
    class="fixed inset-0 z-[60] bg-black/60"
    onclick={() => showLangModal = false}
    aria-label={$t('aria.close')}
  ></button>
  <div
    class="fixed bottom-0 left-0 right-0 z-[60] bg-primary-800 rounded-t-2xl shadow-2xl"
    style="animation: slideInUp 0.2s ease-out; padding-bottom: env(safe-area-inset-bottom);"
  >
    <div class="flex items-center justify-between px-4 py-4 border-b border-primary-700">
      <p class="text-sm font-semibold text-white flex items-center gap-2">
        <Globe size={16} class="text-primary-300" /> Language / Idioma
      </p>
      <button onclick={() => showLangModal = false} class="p-1.5 rounded-lg hover:bg-primary-700 transition-colors" aria-label={$t('aria.close')}>
        <X size={18} class="text-primary-200" />
      </button>
    </div>
    <div class="px-3 py-3 space-y-1">
      {#each SUPPORTED_LOCALES as l}
        <button
          onclick={() => { setLocale(l); showLangModal = false; }}
          class="w-full flex items-center gap-3 px-4 py-3.5 rounded-xl text-sm font-medium transition-colors
            {$locale === l
              ? 'bg-primary-900 text-white'
              : 'text-primary-100 hover:bg-primary-700'}"
        >
          <span class="text-xl">{LANG_LABELS[l].flag}</span>
          <span>{LANG_LABELS[l].full}</span>
          {#if $locale === l}
            <span class="ml-auto text-primary-400 text-base">✓</span>
          {/if}
        </button>
      {/each}
    </div>
  </div>
{/if}

<style>
  @keyframes slideInRight {
    from { transform: translateX(100%); }
    to   { transform: translateX(0); }
  }
  @keyframes slideInUp {
    from { transform: translateY(100%); }
    to   { transform: translateY(0); }
  }
</style>
