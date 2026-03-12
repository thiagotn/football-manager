<script lang="ts">
  import { page } from '$app/stores';
  import { isLoggedIn } from '$lib/stores/auth';
  import { Home, Trophy, BarChart2, AlertTriangle } from 'lucide-svelte';

  let is404 = $derived($page.status === 404);
  let title = $derived(is404 ? 'A bola saiu pela linha de fundo.' : 'O jogo parou inesperadamente.');
  let subtitle = $derived(is404
    ? 'Essa página foi para o vestiário mais cedo que o esperado.'
    : 'Algo deu errado no meio do campo. Tente novamente.'
  );
</script>

<svelte:head>
  <title>Erro {$page.status} — rachao.app</title>
</svelte:head>

<div class="min-h-screen relative bg-gray-900"
  style="background-image: url('/error-404.jpg'); background-size: cover; background-position: center;">
  <div class="absolute inset-0 bg-gradient-to-b from-gray-900/88 via-gray-900/60 to-gray-900/45 pointer-events-none"></div>

  <!-- Mini header — apenas quando não há Navbar (usuário não logado) -->
  {#if !$isLoggedIn}
    <div class="relative z-10 bg-primary-700/80 backdrop-blur-sm text-white py-3 px-4 flex items-center justify-center gap-2">
      <a href="/lp" class="flex items-center gap-2 font-semibold text-base tracking-tight hover:opacity-80 transition-opacity">
        <span class="text-lg">⚽</span>
        rachao.app
      </a>
    </div>
  {/if}

  <main class="relative z-10 max-w-7xl mx-auto px-4 py-8">

    <!-- Page heading -->
    <div class="mb-8">
      <h1 class="text-2xl font-bold text-white">
        {title}
      </h1>
      <p class="text-gray-300 text-sm mt-1">{subtitle}</p>
    </div>

    <!-- Stats row — igual ao dashboard -->
    <div class="grid gap-4 mb-8 grid-cols-3">
      <div class="card p-4 flex flex-col items-center text-center gap-1.5">
        <div class="w-8 h-8 rounded-full bg-red-100 dark:bg-red-900/30 flex items-center justify-center">
          <AlertTriangle size={16} class="text-red-600 dark:text-red-400" />
        </div>
        <p class="text-2xl font-bold text-gray-900 dark:text-gray-100 leading-none">{$page.status}</p>
        <p class="text-xs text-gray-500 dark:text-gray-400">
          <span class="hidden sm:inline">Código do erro</span>
          <span class="sm:hidden">Código</span>
        </p>
      </div>

      <div class="card p-4 flex flex-col items-center text-center gap-1.5">
        <div class="w-8 h-8 rounded-full bg-amber-100 dark:bg-amber-900/30 flex items-center justify-center text-lg leading-none">
          ⚽
        </div>
        <p class="text-2xl font-bold text-gray-900 dark:text-gray-100 leading-none">1</p>
        <p class="text-xs text-gray-500 dark:text-gray-400">
          <span class="hidden sm:inline">Bola perdida</span>
          <span class="sm:hidden">Perdida</span>
        </p>
      </div>

      <div class="card p-4 flex flex-col items-center text-center gap-1.5">
        <div class="w-8 h-8 rounded-full bg-green-100 dark:bg-green-900/30 flex items-center justify-center">
          <Home size={16} class="text-green-600 dark:text-green-400" />
        </div>
        <p class="text-2xl font-bold text-gray-900 dark:text-gray-100 leading-none">∞</p>
        <p class="text-xs text-gray-500 dark:text-gray-400">
          <span class="hidden sm:inline">Saídas disponíveis</span>
          <span class="sm:hidden">Saídas</span>
        </p>
      </div>
    </div>

    <!-- Main grid — 2 colunas no desktop, igual ao dashboard -->
    <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">

      <!-- Card com detalhe do erro e CTA -->
      <div class="card">
        <div class="card-header">
          <h2 class="card-title">
            {#if is404}🔍 Página não encontrada{:else}⚠️ Erro inesperado{/if}
          </h2>
        </div>
        <div class="card-body pt-0">
          {#if is404}
            <p class="text-gray-600 dark:text-gray-400 text-sm leading-relaxed mb-6">
              A página que você tentou acessar não existe ou foi removida do jogo.<br>
              Confira o endereço ou use os atalhos abaixo para voltar à pelada.
            </p>
          {:else}
            <p class="text-gray-600 dark:text-gray-400 text-sm leading-relaxed mb-6">
              {$page.error?.message ?? 'Ocorreu um erro inesperado. Tente recarregar a página ou voltar ao início.'}
            </p>
          {/if}

          <div class="flex flex-col sm:flex-row gap-3">
            <a href="/" class="btn btn-primary flex-1 justify-center gap-1.5">
              <Home size={15} /> Ir para o início
            </a>
            {#if !is404}
              <button onclick={() => window.location.reload()} class="btn btn-secondary flex-1 justify-center gap-1.5">
                Recarregar página
              </button>
            {:else}
              <a href="/groups" class="btn btn-secondary flex-1 justify-center gap-1.5">
                <Trophy size={15} /> Ver meus grupos
              </a>
            {/if}
          </div>
        </div>
      </div>

      <!-- Card de atalhos rápidos (como a lista de grupos no dashboard) -->
      <div class="card">
        <div class="card-header">
          <h2 class="card-title">Onde você quer ir?</h2>
        </div>
        <div class="divide-y divide-gray-100 dark:divide-gray-700">
          <a href="/"
            class="flex items-center gap-3 px-6 py-4 hover:bg-gray-50 dark:hover:bg-gray-700/40 transition-colors">
            <div class="w-9 h-9 rounded-full bg-primary-100 dark:bg-primary-900/30 flex items-center justify-center shrink-0">
              <Home size={18} class="text-primary-600 dark:text-primary-400" />
            </div>
            <div class="flex-1 min-w-0">
              <p class="text-sm font-semibold text-gray-900 dark:text-gray-100">Dashboard</p>
              <p class="text-xs text-gray-500 dark:text-gray-400">Seus próximos rachões e grupos</p>
            </div>
            <span class="text-gray-400 dark:text-gray-500 text-sm">→</span>
          </a>

          <a href="/groups"
            class="flex items-center gap-3 px-6 py-4 hover:bg-gray-50 dark:hover:bg-gray-700/40 transition-colors">
            <div class="w-9 h-9 rounded-full bg-amber-100 dark:bg-amber-900/30 flex items-center justify-center shrink-0">
              <Trophy size={18} class="text-amber-600 dark:text-amber-400" />
            </div>
            <div class="flex-1 min-w-0">
              <p class="text-sm font-semibold text-gray-900 dark:text-gray-100">Grupos</p>
              <p class="text-xs text-gray-500 dark:text-gray-400">Veja seus grupos e partidas</p>
            </div>
            <span class="text-gray-400 dark:text-gray-500 text-sm">→</span>
          </a>

          <a href="/profile/stats"
            class="flex items-center gap-3 px-6 py-4 hover:bg-gray-50 dark:hover:bg-gray-700/40 transition-colors">
            <div class="w-9 h-9 rounded-full bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center shrink-0">
              <BarChart2 size={18} class="text-blue-600 dark:text-blue-400" />
            </div>
            <div class="flex-1 min-w-0">
              <p class="text-sm font-semibold text-gray-900 dark:text-gray-100">Estatísticas</p>
              <p class="text-xs text-gray-500 dark:text-gray-400">Seu histórico e reputação</p>
            </div>
            <span class="text-gray-400 dark:text-gray-500 text-sm">→</span>
          </a>
        </div>
      </div>

    </div>
  </main>
</div>
