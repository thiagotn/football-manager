<script lang="ts">
  let openIndex = $state<number | null>(null);

  function toggle(i: number) {
    openIndex = openIndex === i ? null : i;
  }

  const faqs = [
    {
      q: 'O que é o rachao.app?',
      a: 'É uma plataforma para organizar partidas de futebol amador. Você entra em grupos, recebe convites para partidas e confirma (ou recusa) sua presença com um clique.',
    },
    {
      q: 'Como confirmar presença em uma partida?',
      a: 'Você receberá um link de partida via WhatsApp ou e-mail. Abra o link, faça login (se ainda não estiver logado) e clique em "Vou jogar!" para confirmar ou "Não posso" para recusar.',
    },
    {
      q: 'Como entrar em um grupo?',
      a: 'Peça ao organizador do grupo que gere um link de convite. Clique no link e crie sua conta (ou faça login). Você será adicionado automaticamente ao grupo.',
    },
    {
      q: 'Como ver a lista de confirmados?',
      a: 'Na página da partida (acessível pelo link que você recebeu) você verá as seções "Confirmados", "Recusaram" e "Aguardando" com o nome de cada jogador.',
    },
    {
      q: 'Como compartilhar uma partida com amigos?',
      a: 'Na página da partida, role até o final da lista de jogadores. Lá você encontra dois botões: "Compartilhar no WhatsApp" envia uma mensagem pronta com todos os detalhes (data, local, confirmados e link), e "Copiar link" copia o link da partida para você colar onde quiser.',
    },
    {
      q: 'Como criar minha conta?',
      a: 'A conta é criada automaticamente quando você acessa um link de convite de grupo. Basta preencher seu nome, apelido e uma senha. Não é necessário e-mail.',
    },
    {
      q: 'O que significa cada status de partida?',
      a: '"Aberta" significa que você ainda pode confirmar ou recusar presença. "Encerrada" significa que a partida já aconteceu ou foi fechada pelo organizador — as respostas não são mais aceitas.',
    },
    {
      q: 'E se eu não confirmar pelo link a tempo?',
      a: 'O organizador do grupo pode confirmar ou registrar sua falta diretamente pela plataforma, sem precisar que você acesse o link. Mas se possível, responda pelo link — isso ajuda o organizador a ter o controle em tempo real.',
    },
  ];
</script>

<svelte:head>
  <title>FAQ — rachao.app</title>
</svelte:head>

<div class="min-h-screen bg-gray-50">
  <!-- Beta banner -->
  <div class="bg-yellow-400 text-yellow-900 text-xs font-medium text-center py-2 px-4">
    Versão Beta — produto em desenvolvimento. Funcionalidades podem mudar e dados podem ser resetados sem aviso.
  </div>
  <!-- Header -->
  <div class="bg-primary-700 text-white py-8 px-4 text-center">
    <div class="flex items-center justify-center gap-2 mb-3">
      <span class="font-semibold text-lg tracking-tight">rachao.app</span>
      <span class="text-xs font-bold bg-yellow-400 text-yellow-900 px-1.5 py-0.5 rounded-full">Beta</span>
    </div>
    <span class="text-3xl">⚽</span>
    <h1 class="text-2xl font-bold mt-2">Perguntas Frequentes</h1>
    <p class="text-primary-200 mt-1 text-sm">Tudo o que você precisa saber para jogar</p>
  </div>

  <main class="max-w-2xl mx-auto px-4 py-8">
    <div class="space-y-2">
      {#each faqs as faq, i}
        <div class="card overflow-hidden">
          <button
            class="w-full flex items-center justify-between px-5 py-4 text-left gap-3 hover:bg-gray-50 transition-colors"
            onclick={() => toggle(i)}
          >
            <span class="font-medium text-gray-800 text-sm">{faq.q}</span>
            <span class="text-primary-600 text-lg shrink-0 transition-transform duration-200 {openIndex === i ? 'rotate-45' : ''}">+</span>
          </button>
          {#if openIndex === i}
            <div class="px-5 pb-4 text-sm text-gray-600 leading-relaxed border-t border-gray-100 pt-3">
              {faq.a}
            </div>
          {/if}
        </div>
      {/each}
    </div>

    <div class="mt-8 text-center text-sm text-gray-400">
      Ainda com dúvidas? Fale com o organizador do seu grupo.
    </div>
  </main>
</div>
