<script lang="ts">
  import { page } from '$app/stores';
  import { isLoggedIn } from '$lib/stores/auth';
  import { Home, Trophy, RefreshCw } from 'lucide-svelte';

  let is404 = $derived($page.status === 404);
  let title = $derived(is404 ? 'A bola saiu pela linha.' : 'O jogo parou.');
  let subtitle = $derived(is404
    ? 'Essa página foi para o vestiário.'
    : ($page.error?.message ?? 'Algo deu errado. Tente novamente.')
  );
</script>

<svelte:head>
  <title>Erro {$page.status} — rachao.app</title>
</svelte:head>

<div class="min-h-screen relative flex flex-col"
  style="background-image: url('/error-bg.webp'); background-size: cover; background-position: center top;">

  <!-- Overlay escuro na metade inferior para legibilidade do texto -->
  <div class="absolute inset-0 bg-gradient-to-t from-gray-950/95 via-gray-950/60 to-transparent pointer-events-none"></div>

  <!-- Mini header quando deslogado -->
  {#if !$isLoggedIn}
    <div class="relative z-10 py-4 px-4 flex items-center justify-center">
      <a href="/lp" class="flex items-center gap-2 font-semibold text-white text-base tracking-tight hover:opacity-80 transition-opacity drop-shadow">
        <span class="text-lg">⚽</span>
        rachao.app
      </a>
    </div>
  {/if}

  <!-- Conteúdo centralizado na parte inferior -->
  <div class="relative z-10 mt-auto px-6 pb-16 pt-8 flex flex-col items-center text-center max-w-sm mx-auto w-full">

    <p class="text-white/50 text-xs font-mono tracking-widest uppercase mb-2">{$page.status}</p>
    <h1 class="text-3xl font-bold text-white mb-2 drop-shadow-lg">{title}</h1>
    <p class="text-white/70 text-sm mb-8">{subtitle}</p>

    <div class="flex gap-3 w-full">
      <a href="/" class="btn btn-primary flex-1 justify-center gap-1.5">
        <Home size={15} /> Início
      </a>
      {#if !is404}
        <button onclick={() => window.location.reload()} class="btn btn-secondary flex-1 justify-center gap-1.5">
          <RefreshCw size={15} /> Recarregar
        </button>
      {:else}
        <a href="/groups" class="btn btn-secondary flex-1 justify-center gap-1.5">
          <Trophy size={15} /> Grupos
        </a>
      {/if}
    </div>

  </div>
</div>
