<script lang="ts">
  import { authStore, isAdmin, currentPlayer } from '$lib/stores/auth';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { Users, LogOut, Home, Trophy, BookOpen, UserCircle } from 'lucide-svelte';

  function logout() {
    authStore.logout();
    goto('/login');
  }

  const links = [
    { href: '/',        icon: Home,   label: 'Dashboard' },
    { href: '/groups',  icon: Trophy, label: 'Grupos' },
    { href: '/players',   icon: Users,     label: 'Jogadores',  adminOnly: true },
    { href: '/admin/faq', icon: BookOpen,  label: 'Guia Admin', adminOnly: true },
  ];
</script>

<nav class="bg-primary-700 text-white shadow-md">
  <div class="max-w-7xl mx-auto px-4 flex items-center justify-between h-14">
    <div class="flex items-center gap-1">
      <span class="text-xl font-bold mr-1">⚽</span>
      <span class="font-semibold text-sm mr-1 hidden sm:inline">rachao.app</span>
      <span class="text-xs font-semibold bg-yellow-400 text-yellow-900 px-1.5 py-0.5 rounded-full mr-2">Beta</span>
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

    <div class="flex items-center gap-2">
      <span class="text-sm text-primary-200 hidden sm:block">
        {$currentPlayer?.name}
      </span>
      <a href="/profile"
        class="btn-ghost btn-sm text-primary-100 hover:text-white hover:bg-primary-600 {$page.url.pathname === '/profile' ? 'bg-primary-900' : ''}"
        title="Minha conta">
        <UserCircle size={15} />
        <span class="hidden sm:inline">Conta</span>
      </a>
      <button onclick={logout} class="btn-ghost btn-sm text-primary-100 hover:text-white hover:bg-primary-600">
        <LogOut size={15} />
        <span class="hidden sm:inline">Sair</span>
      </button>
    </div>
  </div>
</nav>
