<script lang="ts">
  import { onMount } from 'svelte';
  import { PUBLIC_LEGAL_CONTACT_EMAIL } from '$env/static/public';
  import { themeStore } from '$lib/stores/theme';
  import { Sun, Moon, Link2 } from 'lucide-svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';

  let openIndex = $state<number | null>(null);
  let copiedId = $state<string | null>(null);

  function slugify(text: string): string {
    return text
      .toLowerCase()
      .normalize('NFD')
      .replace(/[\u0300-\u036f]/g, '')
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-|-$/g, '');
  }

  function toggle(i: number, id: string) {
    if (openIndex === i) {
      openIndex = null;
      history.replaceState(null, '', location.pathname);
    } else {
      openIndex = i;
      history.replaceState(null, '', `#${id}`);
    }
  }

  async function copyLink(id: string, e: MouseEvent) {
    e.stopPropagation();
    const url = `${location.origin}${location.pathname}#${id}`;
    await navigator.clipboard.writeText(url);
    copiedId = id;
    setTimeout(() => { copiedId = null; }, 2000);
  }

  onMount(() => {
    const hash = location.hash.slice(1);
    if (hash) {
      const idx = faqs.findIndex(f => f.id === hash);
      if (idx !== -1) {
        openIndex = idx;
        setTimeout(() => {
          document.getElementById(hash)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
        }, 100);
      }
    }
  });

  type Faq = { id: string; q: string; a: string; steps?: string[] };

  const faqs: Faq[] = [
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
      a: 'Existem duas formas: (1) via convite — o organizador gera um link de convite, você clica e cria sua conta (ou faz login) para ser adicionado automaticamente; (2) via lista de espera — se o grupo for público e houver uma partida aberta, você pode se candidatar clicando em "Quero jogar!" e aguardar a aprovação do admin.',
    },
    {
      q: 'O que é um grupo público e como funciona?',
      a: 'Grupos públicos permitem que qualquer jogador logado se candidate a uma vaga em um rachão aberto, sem precisar de convite. Ao acessar o link do grupo ou da partida, você verá o botão "Quero jogar!" — basta clicar para enviar sua candidatura. O admin do grupo revisa e aprova ou rejeita.',
    },
    {
      q: 'Como entrar na lista de espera de um rachão?',
      a: 'Se o grupo for público e houver uma partida aberta com vagas disponíveis, o botão "Quero jogar!" aparece na página do grupo (aba Próximos) e na página da partida. Ao clicar, um modal exibe os detalhes do rachão (data, local, valores) e um campo para você se apresentar brevemente ao admin. Marque o checkbox de aceite e clique em "Enviar candidatura".',
    },
    {
      q: 'Posso participar de uma partida sem ter cadastro?',
      a: 'Sim! Se você receber o link de uma partida de um grupo público com vagas abertas, verá um card com o botão "Criar conta e participar". Clique, preencha seu cadastro (nome, WhatsApp, senha) e após o cadastro o app abre automaticamente o modal de candidatura para aquela partida.',
    },
    {
      q: 'Como fico sabendo se fui aceito ou recusado na lista de espera?',
      a: 'Você recebe uma notificação push assim que o admin tomar uma decisão. Se aceito, você já passa a ser membro do grupo e sua presença no rachão é confirmada automaticamente. Se recusado, você é notificado mas não é adicionado ao grupo.',
    },
    {
      q: 'Posso ver o status da minha candidatura?',
      a: 'Sim. Na página do grupo (aba Próximos) ou na página da partida, você verá o status atual: "Aguardando aprovação" enquanto a candidatura estiver pendente.',
    },
    {
      q: 'Se for aceito na lista de espera, preciso entrar na fila novamente nos próximos rachões?',
      a: 'Não. Ao ser aceito, você se torna membro permanente do grupo. Nos próximos rachões você participará normalmente, como qualquer outro membro — receberá convite automático se o grupo usar recorrência semanal.',
    },
    {
      q: 'Como descobrir rachões de outros grupos com vaga?',
      a: 'Na seção "Descobrir Rachões" (ícone de bússola no menu), você vê partidas abertas de grupos públicos onde ainda há vagas. Filtre por período, tipo de quadra ou dia da semana. Clique em "Quero jogar!" para enviar sua candidatura diretamente de lá. A home também exibe até 3 rachões com vaga em destaque.',
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
      q: 'O rachao.app está fora do ar?',
      a: 'Você pode verificar o status em tempo real em <a href="https://status.rachao.app" target="_blank" rel="noopener noreferrer" class="text-primary-600 dark:text-primary-400 hover:underline">status.rachao.app</a>. Se tudo estiver verde e o problema persistir, aguarde alguns minutos e tente novamente.',
    },
    {
      q: 'E se eu não confirmar pelo link a tempo?',
      a: 'O organizador do grupo pode confirmar ou registrar sua falta diretamente pela plataforma, sem precisar que você acesse o link. Mas se possível, responda pelo link — isso ajuda o organizador a ter o controle em tempo real.',
    },
    {
      q: 'Como instalar o app no Android?',
      a: 'O rachao.app pode ser instalado diretamente na tela inicial do seu celular, sem precisar da Play Store. No menu lateral do app, um botão "Instalar App" aparece automaticamente quando o seu navegador suporta a instalação. Você também pode instalar manualmente:',
      steps: [
        'Abra o rachao.app no Chrome ou Samsung Internet',
        'Toque no menu (⋮) no canto superior direito do navegador',
        'Toque em "Adicionar à tela inicial" ou "Instalar app"',
        'Confirme tocando em "Instalar"',
      ],
    },
    {
      q: 'Como instalar o app no iPhone ou iPad?',
      a: 'No iOS, o Safari permite salvar o app na tela inicial. O processo é feito manualmente — o botão "Instalar App" no menu do app também exibe esse passo a passo:',
      steps: [
        'Abra o rachao.app no Safari — obrigatório, não funciona em Chrome ou outros navegadores no iOS',
        'Toque no botão Compartilhar (ícone de caixa com seta apontando para cima, na barra inferior do Safari)',
        'Role a lista de opções para baixo e toque em "Adicionar à Tela de Início"',
        'Confirme o nome e toque em "Adicionar"',
      ],
    },
  ].map(f => ({ ...f, id: slugify(f.q) }));
</script>

<svelte:head>
  <title>FAQ — rachao.app</title>
</svelte:head>

<PageBackground>
  <!-- Header -->
  <div class="relative z-10 py-8 px-4 text-center">
    <button
      onclick={themeStore.toggle}
      class="absolute top-3 right-3 p-2 rounded-lg hover:bg-white/10 transition-colors text-white/80"
      title="Alternar tema"
    >
      {#if $themeStore === 'dark'}<Sun size={18} />{:else}<Moon size={18} />{/if}
    </button>
    <img src="/logo.png" alt="rachao.app" width="320" height="174" class="w-44 block mx-auto mb-3" />
    <h1 class="text-2xl font-bold text-white">Perguntas Frequentes</h1>
    <p class="text-gray-300 mt-1 text-sm">Tudo o que você precisa saber para jogar</p>
  </div>

  <main class="relative z-10 max-w-2xl mx-auto px-4 pb-8">
    <div class="space-y-2">
      {#each faqs as faq, i}
        <div id={faq.id} class="card overflow-hidden scroll-mt-4">
          <div class="flex items-center group hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors">
            <button
              class="flex-1 flex items-center justify-between px-5 py-4 text-left gap-3"
              onclick={() => toggle(i, faq.id)}
            >
              <span class="font-medium text-gray-800 dark:text-gray-100 text-sm">{faq.q}</span>
              <span class="text-primary-600 dark:text-primary-400 text-lg shrink-0 transition-transform duration-200 {openIndex === i ? 'rotate-45' : ''}">+</span>
            </button>
            <button
              type="button"
              onclick={(e) => copyLink(faq.id, e)}
              class="opacity-0 group-hover:opacity-100 transition-opacity text-gray-400 hover:text-primary-500 shrink-0 pr-4 py-4"
              title="Copiar link"
              aria-label="Copiar link para esta pergunta"
            >
              {#if copiedId === faq.id}
                <span class="text-xs text-green-500 font-normal whitespace-nowrap">copiado!</span>
              {:else}
                <Link2 size={13} />
              {/if}
            </button>
          </div>
          {#if openIndex === i}
            <div class="px-5 pb-4 text-sm text-gray-600 dark:text-gray-300 leading-relaxed border-t border-gray-100 dark:border-gray-700 pt-3">
              {@html faq.a}
              {#if faq.steps}
                <ol class="mt-2 space-y-1 list-decimal list-inside">
                  {#each faq.steps as step}
                    <li>{step}</li>
                  {/each}
                </ol>
              {/if}
            </div>
          {/if}
        </div>
      {/each}
    </div>

    <div class="mt-8 text-center text-sm text-gray-300">
      Ainda com dúvidas? Fale com o organizador do seu grupo ou entre em contato pelo e-mail
      <a href="mailto:{PUBLIC_LEGAL_CONTACT_EMAIL}" class="text-primary-400 hover:underline">{PUBLIC_LEGAL_CONTACT_EMAIL}</a>.
    </div>
  </main>
</PageBackground>
