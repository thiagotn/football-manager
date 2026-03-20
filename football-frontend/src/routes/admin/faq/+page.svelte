<script lang="ts">
  import { isAdmin } from '$lib/stores/auth';
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';

  onMount(() => {
    if (!$isAdmin) goto('/');
  });

  let openIndex = $state<number | null>(null);

  function toggle(i: number) {
    openIndex = openIndex === i ? null : i;
  }

  const faqs = [
    {
      q: 'Como criar e gerenciar grupos?',
      a: 'Acesse "Grupos" no menu e clique em "Novo Grupo". Preencha o nome, descrição e role mínima. Após criar, você pode editar ou excluir o grupo na página de detalhes. Apenas admins podem criar grupos.',
    },
    {
      q: 'Como adicionar jogadores manualmente?',
      a: 'Vá em "Jogadores" no menu (visível apenas para admins). Clique em "Novo Jogador" e preencha nome, apelido e WhatsApp. O jogador receberá um link para definir sua senha no primeiro acesso.',
    },
    {
      q: 'Como gerar um link de convite?',
      a: 'Na página de detalhes do grupo, clique em "Gerar Link de Convite". O link gerado pode ser compartilhado com qualquer pessoa — ao acessar, ela criará uma conta e será adicionada automaticamente ao grupo.',
    },
    {
      q: 'Como criar e gerenciar partidas?',
      a: 'Dentro de um grupo, clique em "Nova Partida". Defina data, horário, local e observações. Após criar, você pode abrir ou encerrar a partida, além de editar os detalhes. Partidas encerradas não aceitam mais confirmações.',
    },
    {
      q: 'Como controlar as presenças?',
      a: 'Na página da partida (acesso pelo link ou pelo painel do grupo), você vê a lista completa de confirmados, recusados e pendentes. Como admin do grupo, você pode confirmar ou registrar falta de qualquer jogador diretamente pela lista — basta clicar no botão ao lado do nome. Isso é especialmente útil para jogadores que não acessam o link a tempo.',
    },
    {
      q: 'Posso confirmar a presença de um jogador que não respondeu?',
      a: 'Sim. Como admin do grupo, você pode confirmar ou recusar a presença de qualquer membro diretamente na página da partida, sem depender do jogador. Na lista de pendentes, use os botões ✓ e ✕ ao lado de cada nome. Nas listas de confirmados e recusados, também há botão para reverter o status caso precise corrigir.',
    },
    {
      q: 'Como remover membros de um grupo?',
      a: 'Na página de detalhes do grupo, localize o membro na lista e clique no ícone de remover (lixeira). A remoção é imediata. O jogador perderá acesso ao grupo, mas a conta dele permanece ativa.',
    },
    {
      q: 'Qual é a diferença entre perfil Admin e Jogador?',
      a: 'Jogadores podem confirmar presença e ver informações das partidas dos grupos em que participam. Admins têm acesso a todas as funcionalidades: criar grupos, adicionar jogadores, gerar convites, criar partidas e gerenciar presenças de qualquer membro.',
    },
    {
      q: 'Como tornar meu grupo público?',
      a: 'Na página do grupo, clique em "Editar". No modal de edição, você encontrará o toggle "Grupo público". Quando ativado, qualquer jogador com o link do grupo ou de uma partida poderá se candidatar a uma vaga via lista de espera. Você pode reverter para fechado a qualquer momento.',
    },
    {
      q: 'O que é a lista de espera e quando ela aparece?',
      a: 'A lista de espera permite que jogadores externos se candidatem a uma vaga no seu rachão. Ela fica disponível quando o grupo é público e há uma partida aberta com vagas disponíveis. Como admin, você verá o painel "Lista de espera" na aba Próximos do grupo sempre que houver candidatos pendentes.',
    },
    {
      q: 'Como revisar e aceitar candidatos na lista de espera?',
      a: 'Na página do grupo, acesse a aba "Próximos". Abaixo do próximo rachão aparece o painel "Lista de espera" com os candidatos pendentes. Para cada candidato você vê o nome, data/hora da candidatura e a apresentação que ele escreveu. Clique em "Aceitar" para aprovar ou "Rejeitar" para recusar.',
    },
    {
      q: 'O que acontece quando aceito um candidato?',
      a: 'O jogador é adicionado automaticamente ao grupo como membro e sua presença no rachão é marcada como confirmada. Ele recebe uma notificação push informando que foi aceito. Nos próximos rachões, ele participará normalmente como qualquer outro membro.',
    },
    {
      q: 'O que acontece quando rejeito um candidato?',
      a: 'O candidato recebe uma notificação informando que a candidatura não foi aprovada. Ele não é adicionado ao grupo. A rejeição fica registrada no histórico da lista de espera daquele rachão.',
    },
    {
      q: 'O rachão está lotado mas ainda há candidatos na fila — o que acontece?',
      a: 'Quando o número de confirmados atinge o limite máximo de jogadores configurado, o botão "Aceitar" fica bloqueado com a mensagem "Rachão já está lotado". Você precisa primeiro abrir uma vaga (registrando a falta de algum membro) para conseguir aceitar novos candidatos.',
    },
    {
      q: 'Jogadores na fila são notificados automaticamente quando o rachão encerra?',
      a: 'Não. Candidatos pendentes não recebem notificação quando o rachão é encerrado ou a data passa. A notificação de rejeição só é enviada se você manualmente rejeitar a candidatura. Portanto, se um candidato ficou na fila e o rachão encerrou sem aprovação, ele simplesmente não recebe mais retorno — recomendamos revisar a lista com antecedência.',
    },
  ];
</script>

<svelte:head>
  <title>Guia do Admin — rachao.app</title>
</svelte:head>

{#if $isAdmin}
<div class="min-h-screen bg-gray-50 dark:bg-gray-900">
  <!-- Header -->
  <div class="bg-primary-700 text-white py-8 px-4 text-center">
    <span class="text-3xl">🛡️</span>
    <h1 class="text-2xl font-bold mt-2">Guia do Administrador</h1>
    <p class="text-primary-200 mt-1 text-sm">Como gerenciar grupos, jogadores e partidas</p>
  </div>

  <main class="max-w-2xl mx-auto px-4 py-8">
    <div class="space-y-2">
      {#each faqs as faq, i}
        <div class="card overflow-hidden">
          <button
            class="w-full flex items-center justify-between px-5 py-4 text-left gap-3 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
            onclick={() => toggle(i)}
          >
            <span class="font-medium text-gray-800 dark:text-gray-100 text-sm">{faq.q}</span>
            <span class="text-primary-600 dark:text-primary-400 text-lg shrink-0 transition-transform duration-200 {openIndex === i ? 'rotate-45' : ''}">+</span>
          </button>
          {#if openIndex === i}
            <div class="px-5 pb-4 text-sm text-gray-600 dark:text-gray-300 leading-relaxed border-t border-gray-100 dark:border-gray-700 pt-3">
              {faq.a}
            </div>
          {/if}
        </div>
      {/each}
    </div>

    <div class="mt-8 card card-body bg-primary-50 dark:bg-primary-900/30 border border-primary-100 dark:border-primary-800">
      <p class="text-sm text-primary-800 dark:text-primary-300 font-medium mb-1">💡 Dica rápida</p>
      <p class="text-sm text-primary-700 dark:text-primary-400">
        Use o botão "WhatsApp" na página de cada partida para enviar um resumo completo
        (data, local, confirmados e link) diretamente para o grupo de WhatsApp.
      </p>
    </div>
  </main>
</div>
{/if}
