<script lang="ts">
  import { PLAN_ORDER, PLANS, formatPrice } from '$lib/plans';
</script>

<svelte:head>
  <title>rachao.app — Organize suas partidas de futebol com facilidade</title>
  <meta name="description" content="Organize grupos de futebol, convide jogadores e controle presenças em um clique." />
  <meta property="og:url" content="https://rachao.app/lp" />
  <meta property="og:title" content="rachao.app — Organize suas partidas de futebol com facilidade" />
  <meta property="og:description" content="Organize grupos de futebol, convide jogadores e controle presenças em um clique." />
  <meta property="og:image" content="https://rachao.app/banner-lp.jpg" />
</svelte:head>

<!-- BETA BANNER -->
<div class="bg-yellow-400 text-yellow-900 text-xs font-medium text-center py-2 px-4">
  Versão Beta — produto em desenvolvimento. Funcionalidades podem mudar e dados podem ser resetados sem aviso.
</div>

<!-- HERO -->
<section class="relative overflow-hidden text-white" style="min-height:600px;">
  <picture>
    <source srcset="/banner-lp.webp" type="image/webp" />
    <img
      src="/banner-lp.jpg"
      alt=""
      aria-hidden="true"
      width="1920"
      height="600"
      class="absolute inset-0 w-full h-full object-cover object-center"
      fetchpriority="high"
    />
  </picture>
  <div class="absolute inset-0 bg-primary-900/75"></div>
  <div class="relative max-w-4xl mx-auto px-6 py-16 text-center">
    <img
      src="/logo.png"
      alt="rachao.app"
      width="320"
      height="174"
      class="w-72 sm:w-96 block mx-auto mb-6"
      fetchpriority="high"
    />
    <p class="text-xl text-primary-100 max-w-xl mx-auto mb-3">
      Organize suas partidas de futebol sem complicação.
    </p>
    <p class="text-primary-200 max-w-lg mx-auto mb-10 text-sm leading-relaxed">
      Convide jogadores, confirme presenças e compartilhe os detalhes da partida —
      tudo direto pelo celular, sem precisar instalar nada.
    </p>
    <div class="flex flex-col sm:flex-row gap-3 justify-center">
      <a href="/register"
        class="inline-flex items-center justify-center gap-2 px-8 py-3 bg-white text-primary-700 rounded-xl font-semibold hover:bg-primary-50 transition-colors shadow-md">
        Cadastrar grátis
      </a>
      <a href="/login"
        class="inline-flex items-center justify-center gap-2 px-8 py-3 bg-primary-900/40 text-white rounded-xl font-semibold hover:bg-primary-900/60 transition-colors border border-primary-500">
        Entrar na plataforma
      </a>
    </div>
  </div><!-- /relative content -->
</section>

<!-- COMO FUNCIONA — JOGADOR -->
<section class="bg-white py-16 px-6">
  <div class="max-w-4xl mx-auto">
    <div class="text-center mb-12">
      <span class="text-xs font-semibold text-primary-600 uppercase tracking-widest">Para jogadores</span>
      <h2 class="text-2xl sm:text-3xl font-bold text-gray-900 mt-2">Como funciona?</h2>
      <p class="text-gray-500 mt-2 text-sm">Em três passos simples você está dentro da partida.</p>
    </div>

    <div class="grid sm:grid-cols-3 gap-8">
      <div class="text-center">
        <div class="w-14 h-14 rounded-2xl bg-primary-100 flex items-center justify-center mx-auto mb-4 text-2xl">
          ✉️
        </div>
        <div class="text-xs font-bold text-primary-500 uppercase tracking-widest mb-1">Passo 1</div>
        <h3 class="font-semibold text-gray-900 mb-2">Receba o convite</h3>
        <p class="text-sm text-gray-500 leading-relaxed">
          O organizador envia um link via WhatsApp. Clique, crie sua conta em segundos e já está no grupo.
        </p>
      </div>

      <div class="text-center">
        <div class="w-14 h-14 rounded-2xl bg-green-100 flex items-center justify-center mx-auto mb-4 text-2xl">
          📋
        </div>
        <div class="text-xs font-bold text-green-600 uppercase tracking-widest mb-1">Passo 2</div>
        <h3 class="font-semibold text-gray-900 mb-2">Confirme sua presença</h3>
        <p class="text-sm text-gray-500 leading-relaxed">
          Quando a partida for criada, você recebe o link. Abre, clica em "Vou jogar!" ou "Não posso" — simples assim.
        </p>
      </div>

      <div class="text-center">
        <div class="w-14 h-14 rounded-2xl bg-yellow-100 flex items-center justify-center mx-auto mb-4 text-2xl">
          🏃
        </div>
        <div class="text-xs font-bold text-yellow-600 uppercase tracking-widest mb-1">Passo 3</div>
        <h3 class="font-semibold text-gray-900 mb-2">Apareça e jogue</h3>
        <p class="text-sm text-gray-500 leading-relaxed">
          Veja quem confirmou, o horário e o local — e vai jogar. Sem confusão, sem grupo de WhatsApp bagunçado.
        </p>
      </div>
    </div>
  </div>
</section>

<!-- DIVISOR -->
<div class="h-px bg-gray-100"></div>

<!-- PARA O ORGANIZADOR -->
<section class="bg-gray-50 py-16 px-6">
  <div class="max-w-4xl mx-auto">
    <div class="text-center mb-12">
      <span class="text-xs font-semibold text-indigo-600 uppercase tracking-widest">Para organizadores</span>
      <h2 class="text-2xl sm:text-3xl font-bold text-gray-900 mt-2">Tudo que você precisa para gerir suas partidas de futebol</h2>
      <p class="text-gray-500 mt-2 text-sm">Sem planilha, sem contar resposta por resposta no WhatsApp.</p>
    </div>

    <div class="grid sm:grid-cols-2 gap-5">
      <div class="bg-white rounded-2xl border border-gray-200 p-6 shadow-sm flex gap-4">
        <div class="text-3xl shrink-0">👥</div>
        <div>
          <h3 class="font-semibold text-gray-900 mb-1">Grupos organizados</h3>
          <p class="text-sm text-gray-500 leading-relaxed">
            Crie um grupo para cada turma de jogadores. Adicione membros via link de convite ou manualmente.
          </p>
        </div>
      </div>

      <div class="bg-white rounded-2xl border border-gray-200 p-6 shadow-sm flex gap-4">
        <div class="text-3xl shrink-0">📅</div>
        <div>
          <h3 class="font-semibold text-gray-900 mb-1">Partidas com um clique</h3>
          <p class="text-sm text-gray-500 leading-relaxed">
            Defina data, horário e local. A partida gera um link único que você compartilha com o grupo.
          </p>
        </div>
      </div>

      <div class="bg-white rounded-2xl border border-gray-200 p-6 shadow-sm flex gap-4">
        <div class="text-3xl shrink-0">✅</div>
        <div>
          <h3 class="font-semibold text-gray-900 mb-1">Controle de presenças em tempo real</h3>
          <p class="text-sm text-gray-500 leading-relaxed">
            Veja na hora quem confirmou, quem recusou e quem ainda não respondeu. Sem precisar ficar perguntando um por um.
          </p>
        </div>
      </div>

      <div class="bg-white rounded-2xl border border-gray-200 p-6 shadow-sm flex gap-4">
        <div class="text-3xl shrink-0">📲</div>
        <div>
          <h3 class="font-semibold text-gray-900 mb-1">Compartilhe pelo WhatsApp</h3>
          <p class="text-sm text-gray-500 leading-relaxed">
            Com um clique, envie um resumo completo da partida — data, local, lista de confirmados e link — direto para o grupo.
          </p>
        </div>
      </div>

      <div class="bg-white rounded-2xl border border-gray-200 p-6 shadow-sm flex gap-4">
        <div class="text-3xl shrink-0">🏆</div>
        <div>
          <h3 class="font-semibold text-gray-900 mb-1">Votação pós-partida</h3>
          <p class="text-sm text-gray-500 leading-relaxed">
            Após cada partida, os jogadores elegem o Top 5 da pelada e a decepção do jogo. Engajamento garantido mesmo depois do apito final.
          </p>
        </div>
      </div>
    </div>
  </div>
</section>

<!-- DIVISOR -->
<div class="h-px bg-gray-100"></div>

<!-- VOTAÇÃO PÓS-PARTIDA SPOTLIGHT -->
<section class="bg-white py-16 px-6">
  <div class="max-w-4xl mx-auto">
    <div class="text-center mb-10">
      <span class="text-xs font-semibold text-amber-600 uppercase tracking-widest">Novidade</span>
      <h2 class="text-2xl sm:text-3xl font-bold text-gray-900 mt-2">Votação pós-partida 🏆</h2>
      <p class="text-gray-500 mt-2 text-sm max-w-lg mx-auto">
        Depois de cada rachão, os jogadores elegem o melhor de campo — e a decepção do dia. Tudo automático, sem precisar organizar nada.
      </p>
    </div>

    <div class="grid sm:grid-cols-2 gap-8 items-center">
      <!-- Explicação -->
      <div class="space-y-5">
        <div class="flex gap-4">
          <div class="w-10 h-10 rounded-xl bg-amber-100 flex items-center justify-center text-xl shrink-0">⏱️</div>
          <div>
            <h3 class="font-semibold text-gray-900 text-sm">Abre automaticamente</h3>
            <p class="text-sm text-gray-500 mt-0.5">20 minutos após o horário de início da partida, a votação abre sozinha para todos que confirmaram presença.</p>
          </div>
        </div>
        <div class="flex gap-4">
          <div class="w-10 h-10 rounded-xl bg-green-100 flex items-center justify-center text-xl shrink-0">🗳️</div>
          <div>
            <h3 class="font-semibold text-gray-900 text-sm">Top 5 da pelada</h3>
            <p class="text-sm text-gray-500 mt-0.5">Cada jogador vota nos 5 melhores do jogo, atribuindo 5, 4, 3, 2 e 1 ponto por posição. Sem votar em si mesmo.</p>
          </div>
        </div>
        <div class="flex gap-4">
          <div class="w-10 h-10 rounded-xl bg-red-100 flex items-center justify-center text-xl shrink-0">😬</div>
          <div>
            <h3 class="font-semibold text-gray-900 text-sm">Decepção do jogo</h3>
            <p class="text-sm text-gray-500 mt-0.5">Quando há jogadores suficientes, cada um também aponta a maior decepção da partida. Levado na esportiva — faz parte!</p>
          </div>
        </div>
        <div class="flex gap-4">
          <div class="w-10 h-10 rounded-xl bg-indigo-100 flex items-center justify-center text-xl shrink-0">📊</div>
          <div>
            <h3 class="font-semibold text-gray-900 text-sm">Ranking acumulado no grupo</h3>
            <p class="text-sm text-gray-500 mt-0.5">Os pontos somam ao longo das partidas. A aba Estatísticas do grupo exibe o ranking completo com pontos, decepções e horas jogadas.</p>
          </div>
        </div>
      </div>

      <!-- Mock ranking preview -->
      <div class="bg-gray-50 rounded-2xl border border-gray-200 p-5 shadow-sm">
        <div class="flex items-center gap-2 mb-4">
          <span class="text-lg">🏆</span>
          <span class="font-bold text-gray-900 text-sm">Resultado · Rachão #42</span>
          <span class="ml-auto text-xs text-gray-400">18 votaram</span>
        </div>
        <div class="space-y-2 mb-4">
          {#each [
            { pos: '🥇', name: 'Marquinhos', pts: 67 },
            { pos: '🥈', name: 'Jorginho', pts: 51 },
            { pos: '🥉', name: 'Betão', pts: 44 },
            { pos: '4º', name: 'Junin', pts: 29 },
            { pos: '5º', name: 'Cabeludo', pts: 18 },
          ] as item}
            <div class="flex items-center gap-2 bg-white rounded-xl px-3 py-2 border border-gray-100">
              <span class="text-base w-6 text-center">{item.pos}</span>
              <span class="flex-1 text-sm font-medium text-gray-800">{item.name}</span>
              <span class="text-xs font-bold text-amber-600">{item.pts} pts</span>
            </div>
          {/each}
        </div>
        <div class="bg-red-50 rounded-xl px-3 py-2 border border-red-100 flex items-center gap-2">
          <span class="text-base">😬</span>
          <span class="text-sm text-gray-700 flex-1">Decepção: <strong>Tigrão</strong></span>
          <span class="text-xs text-red-500 font-semibold">9 votos</span>
        </div>
        <p class="text-xs text-gray-400 text-center mt-3">Exemplo ilustrativo</p>
      </div>
    </div>
  </div>
</section>

<!-- DIVISOR -->
<div class="h-px bg-gray-100"></div>

<!-- DESTAQUES RÁPIDOS -->
<section class="bg-white py-14 px-6">
  <div class="max-w-3xl mx-auto">
    <div class="grid sm:grid-cols-3 gap-8 text-center">
      <div>
        <p class="text-4xl font-extrabold text-primary-600 mb-1">100%</p>
        <p class="text-sm text-gray-500">pelo navegador, sem instalar app</p>
      </div>
      <div>
        <p class="text-4xl font-extrabold text-primary-600 mb-1">1 link</p>
        <p class="text-sm text-gray-500">para confirmar presença</p>
      </div>
      <div>
        <p class="text-4xl font-extrabold text-primary-600 mb-1">0</p>
        <p class="text-sm text-gray-500">planilhas ou contagens manuais</p>
      </div>
    </div>
  </div>
</section>

<!-- DIVISOR -->
<div class="h-px bg-gray-100"></div>

<!-- PLANOS -->
<section class="bg-gray-50 py-16 px-6">
  <div class="max-w-4xl mx-auto">
    <div class="text-center mb-12">
      <span class="text-xs font-semibold text-primary-600 uppercase tracking-widest">Planos</span>
      <h2 class="text-2xl sm:text-3xl font-bold text-gray-900 mt-2">Simples e transparente</h2>
      <p class="text-gray-500 mt-2 text-sm">Comece grátis. Faça upgrade quando precisar de mais.</p>
    </div>

    <div class="grid sm:grid-cols-3 gap-5">
      {#each PLAN_ORDER as key}
        {@const plan = PLANS[key]}
        <div class="bg-white rounded-2xl border {plan.key === 'free' ? 'border-primary-400 shadow-md ring-1 ring-primary-300' : 'border-gray-200 shadow-sm'} p-6 flex flex-col relative">
          {#if plan.key === 'free'}
            <span class="absolute -top-3 left-1/2 -translate-x-1/2 bg-primary-500 text-white text-xs font-bold px-3 py-1 rounded-full">Disponível agora</span>
          {:else if !plan.available}
            <span class="absolute -top-3 left-1/2 -translate-x-1/2 bg-gray-300 text-gray-600 text-xs font-semibold px-3 py-1 rounded-full">Em breve</span>
          {/if}

          <div class="mb-4">
            <h3 class="text-lg font-bold text-gray-900">{plan.name}</h3>
            <div class="mt-1">
              {#if !plan.available}
                <span class="text-sm text-gray-400 italic">Preço a definir</span>
              {:else if plan.price_monthly === null}
                <span class="text-3xl font-extrabold text-primary-600">Grátis</span>
              {:else}
                <span class="text-3xl font-extrabold text-gray-900">R$ {plan.price_monthly.toFixed(2).replace('.', ',')}</span>
                <span class="text-sm text-gray-400">/mês</span>
              {/if}
            </div>
          </div>

          <ul class="space-y-2 mb-6 flex-1">
            {#each plan.highlights as item}
              <li class="flex items-start gap-2 text-sm text-gray-600">
                <span class="text-primary-500 mt-0.5 shrink-0">✓</span>
                {item}
              </li>
            {/each}
          </ul>

          {#if plan.key === 'free'}
            <a href="/register"
              class="inline-flex items-center justify-center gap-2 px-4 py-2.5 bg-primary-600 text-white rounded-xl font-semibold hover:bg-primary-700 transition-colors text-sm text-center">
              Cadastrar grátis
            </a>
          {:else}
            <button disabled
              class="inline-flex items-center justify-center gap-2 px-4 py-2.5 bg-gray-100 text-gray-400 rounded-xl font-semibold text-sm cursor-not-allowed">
              Em breve
            </button>
          {/if}
        </div>
      {/each}
    </div>
  </div>
</section>

<!-- DIVISOR -->
<div class="h-px bg-gray-100"></div>

<!-- CTA FINAL -->
<section class="bg-gradient-to-br from-primary-700 to-primary-900 text-white py-16 px-6 text-center">
  <div class="max-w-xl mx-auto">
    <h2 class="text-2xl sm:text-3xl font-bold mb-3">Pronto para o rachão?</h2>
    <p class="text-primary-200 mb-8 text-sm">
      Peça ao organizador do seu grupo um link de convite, ou entre se já tiver conta.
    </p>
    <div class="flex flex-col sm:flex-row gap-3 justify-center">
      <a href="/login"
        class="inline-flex items-center justify-center gap-2 px-8 py-3 bg-white text-primary-700 rounded-xl font-semibold hover:bg-primary-50 transition-colors shadow-md">
        Entrar
      </a>
      <a href="/faq"
        class="inline-flex items-center justify-center gap-2 px-8 py-3 bg-transparent text-white rounded-xl font-semibold hover:bg-primary-800 transition-colors border border-primary-400">
        Dúvidas frequentes
      </a>
    </div>
  </div>
</section>

<!-- FOOTER -->
<footer class="bg-gray-900 text-gray-400 py-8 px-6 text-center text-xs">
  <div class="max-w-4xl mx-auto flex flex-col sm:flex-row items-center justify-between gap-3">
    <span class="text-white font-semibold">⚽ rachao.app</span>
    <span class="text-gray-500 text-xs">© 2026</span>
    <div class="flex gap-5">
      <a href="/faq" class="hover:text-white transition-colors">FAQ</a>
      <a href="/login" class="hover:text-white transition-colors">Entrar</a>
    </div>
  </div>
</footer>
