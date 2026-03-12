<script lang="ts">
  import { authStore, isAdmin, currentPlayer } from '$lib/stores/auth';
  import { themeStore } from '$lib/stores/theme';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { Users, LogOut, Home, Trophy, BookOpen, UserCircle, Menu, X, Sun, Moon, ChevronLeft, Star, HelpCircle, FileText, Shield, BarChart2 } from 'lucide-svelte';

  function logout() {
    authStore.logout();
    goto('/login');
  }

  const links = [
    { href: '/',               icon: Home,       label: 'Dashboard' },
    { href: '/groups',         icon: Trophy,     label: 'Grupos' },
    { href: '/profile/stats',  icon: BarChart2,  label: 'Rachão Score',  playerOnly: true },
    { href: '/review',         icon: Star,       label: 'Avaliar o App', playerOnly: true },
    { href: '/players',        icon: Users,      label: 'Jogadores',     adminOnly: true },
    { href: '/admin/reviews',  icon: Star,       label: 'Avaliações',    adminOnly: true },
    { href: '/admin/faq',      icon: BookOpen,   label: 'Guia Admin',    adminOnly: true },
  ];

  let menuOpen = $state(false);

  function closeMenu() { menuOpen = false; }

  // Fecha ao navegar
  $effect(() => {
    $page.url.pathname;
    menuOpen = false;
  });

  function getBackHref(pathname: string): string | null {
    if (pathname.startsWith('/groups/')) return '/groups';
    if (pathname === '/groups')   return '/';
    if (pathname === '/players')  return '/';
    if (pathname === '/profile')        return '/';
    if (pathname === '/profile/stats') return '/profile';
    if (pathname === '/review')   return '/';
    if (pathname === '/faq')      return '/';
    if (pathname === '/terms')    return '/';
    if (pathname === '/privacy')  return '/';
    if (pathname.startsWith('/admin/')) return '/';
    return null;
  }

  let backHref = $derived(getBackHref($page.url.pathname));
</script>

<nav class="bg-primary-700 text-white shadow-md relative z-40" style="padding-top: env(safe-area-inset-top);">
  <div class="max-w-7xl mx-auto px-4 flex items-center justify-between h-16 relative overflow-hidden">

    <!-- Esquerda: botão voltar (mobile) + logo desktop -->
    <div class="flex items-center gap-1 shrink-0">
      {#if backHref}
        <a
          href={backHref}
          class="min-[940px]:hidden p-1.5 -ml-1.5 rounded-lg hover:bg-primary-600 transition-colors"
          aria-label="Voltar"
        >
          <ChevronLeft size={22} />
        </a>
      {/if}
      <!-- Logo desktop: efeito sangramento à esquerda -->
      <a href="/" class="hidden min-[940px]:flex -ml-16 self-stretch items-end">
        <img src="/logo.png" alt="rachao.app" class="h-24 w-auto flex-shrink-0 -translate-y-2" />
      </a>
    </div>

    <!-- Logo mobile: centralizado na altura total (status bar + barra) -->
    <a href="/" class="min-[940px]:hidden absolute left-1/2 -translate-x-1/2 top-0 bottom-0 flex items-center pointer-events-auto">
      <img src="/logo.png" alt="rachao.app" class="h-14 w-auto flex-shrink-0" />
    </a>

    <!-- Links — desktop -->
    <div class="hidden min-[940px]:flex items-center gap-1">
      {#each links as l}
        {#if (!l.adminOnly || $isAdmin) && (!l.playerOnly || !$isAdmin)}
          <a
            href={l.href}
            class="flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors
              {$page.url.pathname === l.href ? 'bg-primary-900' : 'hover:bg-primary-600'}"
          >
            <l.icon size={15} />
            {l.label}
          </a>
        {/if}
      {/each}
    </div>

    <!-- Direita — desktop -->
    <div class="hidden min-[940px]:flex items-center gap-2">
      <span class="text-sm text-primary-200">{$currentPlayer?.name}</span>
      <a href="/profile"
        class="btn-ghost btn-sm text-primary-100 hover:text-white hover:bg-primary-600 {$page.url.pathname === '/profile' ? 'bg-primary-900' : ''}"
        title="Minha conta">
        <UserCircle size={15} />
        <span>Conta</span>
      </a>
      <button onclick={themeStore.toggle} class="btn-ghost btn-sm text-primary-100 hover:text-white hover:bg-primary-600" title="Alternar tema">
        {#if $themeStore === 'dark'}<Sun size={15} />{:else}<Moon size={15} />{/if}
      </button>
      <button onclick={logout} class="btn-ghost btn-sm text-primary-100 hover:text-white hover:bg-primary-600">
        <LogOut size={15} />
        <span>Sair</span>
      </button>
    </div>

    <!-- Hambúrguer — mobile -->
    <button
      class="min-[940px]:hidden p-2 rounded-lg hover:bg-primary-600 transition-colors"
      onclick={() => menuOpen = !menuOpen}
      aria-label="Menu"
    >
      <Menu size={22} />
    </button>
  </div>
</nav>

<!-- Drawer lateral mobile -->
{#if menuOpen}
  <!-- Backdrop -->
  <button
    class="min-[940px]:hidden fixed inset-0 z-40 bg-black/50"
    onclick={closeMenu}
    aria-label="Fechar menu"
  ></button>

  <!-- Painel deslizante da direita -->
  <div class="min-[940px]:hidden fixed top-0 right-0 h-full w-72 max-w-[85vw] z-50 bg-primary-800 shadow-2xl flex flex-col"
    style="animation: slideInRight 0.22s ease-out;">
    <!-- Cabeçalho do drawer -->
    <div class="flex items-center justify-between px-4 h-16 border-b border-primary-700 shrink-0">
      <p class="text-sm font-medium text-primary-200 truncate">{$currentPlayer?.name}</p>
      <button onclick={closeMenu} class="p-2 rounded-lg hover:bg-primary-700 transition-colors" aria-label="Fechar">
        <X size={20} />
      </button>
    </div>

    <!-- Links de navegação -->
    <div class="flex-1 overflow-y-auto px-3 py-3 space-y-1">
      {#each links as l}
        {#if (!l.adminOnly || $isAdmin) && (!l.playerOnly || !$isAdmin)}
          <a
            href={l.href}
            onclick={closeMenu}
            class="flex items-center gap-3 px-3 py-3 rounded-xl text-sm font-medium transition-colors
              {$page.url.pathname === l.href ? 'bg-primary-900 text-white' : 'text-primary-100 hover:bg-primary-700'}"
          >
            <l.icon size={18} />
            {l.label}
          </a>
        {/if}
      {/each}

      <div class="border-t border-primary-700/60 pt-1 mt-1 space-y-1">
        <a href="/faq" onclick={closeMenu}
          class="flex items-center gap-3 px-3 py-3 rounded-xl text-sm font-medium transition-colors
            {$page.url.pathname === '/faq' ? 'bg-primary-900 text-white' : 'text-primary-100 hover:bg-primary-700'}">
          <HelpCircle size={18} /> FAQ
        </a>
        <a href="/terms" onclick={closeMenu}
          class="flex items-center gap-3 px-3 py-3 rounded-xl text-sm font-medium transition-colors
            {$page.url.pathname === '/terms' ? 'bg-primary-900 text-white' : 'text-primary-100 hover:bg-primary-700'}">
          <FileText size={18} /> Termos de Uso
        </a>
        <a href="/privacy" onclick={closeMenu}
          class="flex items-center gap-3 px-3 py-3 rounded-xl text-sm font-medium transition-colors
            {$page.url.pathname === '/privacy' ? 'bg-primary-900 text-white' : 'text-primary-100 hover:bg-primary-700'}">
          <Shield size={18} /> Privacidade
        </a>
      </div>
    </div>

    <!-- Rodapé do drawer -->
    <div class="px-3 py-3 border-t border-primary-700 space-y-1 shrink-0">
      <a href="/profile" onclick={closeMenu}
        class="flex items-center gap-3 px-3 py-3 rounded-xl text-sm font-medium transition-colors text-primary-100 hover:bg-primary-700
          {$page.url.pathname === '/profile' ? 'bg-primary-900 text-white' : ''}">
        <UserCircle size={18} /> Minha Conta
      </a>
      <button onclick={themeStore.toggle}
        class="w-full flex items-center gap-3 px-3 py-3 rounded-xl text-sm font-medium hover:bg-primary-700 text-left text-primary-100 transition-colors">
        {#if $themeStore === 'dark'}<Sun size={18} /> Tema claro{:else}<Moon size={18} /> Tema escuro{/if}
      </button>
      <button onclick={logout}
        class="w-full flex items-center gap-3 px-3 py-3 rounded-xl text-sm font-medium hover:bg-primary-700 text-left text-primary-100 transition-colors">
        <LogOut size={18} /> Sair
      </button>
    </div>
  </div>
{/if}

<style>
  @keyframes slideInRight {
    from { transform: translateX(100%); }
    to   { transform: translateX(0); }
  }
</style>
