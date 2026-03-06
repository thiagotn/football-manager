<script lang="ts">
  import { authStore, isAdmin, currentPlayer } from '$lib/stores/auth';
  import { themeStore } from '$lib/stores/theme';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { Users, LogOut, Home, Trophy, BookOpen, UserCircle, Menu, X, Sun, Moon, ChevronLeft } from 'lucide-svelte';

  function logout() {
    authStore.logout();
    goto('/login');
  }

  const links = [
    { href: '/',          icon: Home,     label: 'Dashboard' },
    { href: '/groups',    icon: Trophy,   label: 'Grupos' },
    { href: '/players',   icon: Users,    label: 'Jogadores',  adminOnly: true },
    { href: '/admin/faq', icon: BookOpen, label: 'Guia Admin', adminOnly: true },
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
    if (pathname === '/profile')  return '/';
    if (pathname.startsWith('/admin/')) return '/';
    return null;
  }

  let backHref = $derived(getBackHref($page.url.pathname));
</script>

<nav class="bg-primary-700 text-white shadow-md relative z-40">
  <div class="max-w-7xl mx-auto px-4 flex items-center justify-between h-14">

    <!-- Esquerda: botão voltar (mobile, sub-páginas) + logo -->
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
      <div class="flex items-center gap-1.5">
        <span class="text-xl font-bold">⚽</span>
        <span class="font-semibold text-sm">rachao.app</span>
        <span class="text-xs font-semibold bg-yellow-400 text-yellow-900 px-1.5 py-0.5 rounded-full">Beta</span>
      </div>
    </div>

    <!-- Links — desktop -->
    <div class="hidden min-[940px]:flex items-center gap-1">
      {#each links as l}
        {#if !l.adminOnly || $isAdmin}
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
      {#if menuOpen}
        <X size={22} />
      {:else}
        <Menu size={22} />
      {/if}
    </button>
  </div>

  <!-- Menu mobile dropdown -->
  {#if menuOpen}
    <div class="min-[940px]:hidden bg-primary-800 border-t border-primary-600 px-4 pb-4 pt-2 space-y-1">
      <!-- Usuário -->
      <p class="text-xs text-primary-300 px-2 py-1 font-medium">{$currentPlayer?.name}</p>

      {#each links as l}
        {#if !l.adminOnly || $isAdmin}
          <a
            href={l.href}
            onclick={closeMenu}
            class="flex items-center gap-2.5 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors
              {$page.url.pathname === l.href ? 'bg-primary-900' : 'hover:bg-primary-600'}"
          >
            <l.icon size={16} />
            {l.label}
          </a>
        {/if}
      {/each}

      <div class="border-t border-primary-600 mt-2 pt-2 space-y-1">
        <a href="/profile" onclick={closeMenu}
          class="flex items-center gap-2.5 px-3 py-2.5 rounded-lg text-sm font-medium hover:bg-primary-600
            {$page.url.pathname === '/profile' ? 'bg-primary-900' : ''}">
          <UserCircle size={16} /> Minha Conta
        </a>
        <button onclick={themeStore.toggle}
          class="w-full flex items-center gap-2.5 px-3 py-2.5 rounded-lg text-sm font-medium hover:bg-primary-600 text-left text-primary-100">
          {#if $themeStore === 'dark'}<Sun size={16} /> Tema claro{:else}<Moon size={16} /> Tema escuro{/if}
        </button>
        <button onclick={logout}
          class="w-full flex items-center gap-2.5 px-3 py-2.5 rounded-lg text-sm font-medium hover:bg-primary-600 text-left text-primary-100">
          <LogOut size={16} /> Sair
        </button>
      </div>
    </div>
  {/if}
</nav>
