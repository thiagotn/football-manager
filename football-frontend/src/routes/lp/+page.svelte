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
  Versao Beta — produto em desenvolvimento. Funcionalidades podem mudar e dados podem ser resetados sem aviso.
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
  <div class="relative max-w-4xl mx-auto px-6 py-20 text-center">
    <div class="flex items-center justify-center gap-3 mb-4">
      <h1 class="text-4xl sm:text-5xl font-extrabold leading-tight">
        rachao.app
      </h1>
      <span class="text-xs font-bold bg-yellow-400 text-yellow-900 px-2 py-1 rounded-full self-center">Beta</span>
    </div>
    <p class="text-xl text-primary-100 max-w-xl mx-auto mb-3">
      Organize suas partidas de futebol sem complicação.
    </p>
    <p class="text-primary-200 max-w-lg mx-auto mb-10 text-sm leading-relaxed">
      Convide jogadores, confirme presenças e compartilhe os detalhes da partida —
      tudo direto pelo celular, sem precisar instalar nada.
    </p>
    <div class="flex flex-col sm:flex-row gap-3 justify-center">
      <a href="/login"
        class="inline-flex items-center justify-center gap-2 px-8 py-3 bg-white text-primary-700 rounded-xl font-semibold hover:bg-primary-50 transition-colors shadow-md">
        Entrar na plataforma
      </a>
      <a href="/faq"
        class="inline-flex items-center justify-center gap-2 px-8 py-3 bg-primary-900/40 text-white rounded-xl font-semibold hover:bg-primary-900/60 transition-colors border border-primary-500">
        Ver perguntas frequentes
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
    <div class="flex gap-5">
      <a href="/faq" class="hover:text-white transition-colors">FAQ</a>
      <a href="/login" class="hover:text-white transition-colors">Entrar</a>
    </div>
  </div>
</footer>
