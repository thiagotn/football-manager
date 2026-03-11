<script lang="ts">
  import { PUBLIC_LEGAL_CONTROLLER_NAME, PUBLIC_LEGAL_CONTROLLER_DOC, PUBLIC_LEGAL_FORUM_CITY, PUBLIC_LEGAL_CONTACT_EMAIL } from '$env/static/public';
  import { isLoggedIn } from '$lib/stores/auth';

  let backHref = $derived($isLoggedIn ? '/' : '/lp');
</script>

<svelte:head><title>Termos de Uso — rachao.app</title></svelte:head>

<div class="min-h-screen bg-white dark:bg-gray-900 text-gray-800 dark:text-gray-200">
  <div class="max-w-3xl mx-auto px-6 py-10">

    <div class="mb-8">
      <a href={backHref} class="text-sm text-primary-600 hover:underline">← rachao.app</a>
    </div>

    <h1 class="text-2xl font-bold mb-1">Termos de Uso</h1>
    <p class="text-xs text-gray-400 mb-8">Última atualização: março de 2026 · Versão: 1.0</p>

    <div class="prose prose-sm dark:prose-invert max-w-none space-y-8">

      <section>
        <h2 class="text-base font-semibold mb-2">1. Aceitação</h2>
        <p>Ao criar uma conta ou usar o rachao.app, você declara ter lido e aceitar estes Termos de Uso. Se não concordar, não utilize a plataforma.</p>
        <p class="mt-2">O rachao.app é operado por <strong>{PUBLIC_LEGAL_CONTROLLER_NAME}</strong>, {PUBLIC_LEGAL_CONTROLLER_DOC} ("nós"), com contato em <strong>{PUBLIC_LEGAL_CONTACT_EMAIL}</strong>.</p>
        <p class="mt-2">A plataforma está em <strong>fase Beta</strong>: funcionalidades podem mudar, ser removidas ou ter dados resetados com aviso mínimo de 48 horas. Não há garantia de disponibilidade contínua (SLA) nesta fase.</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">2. Cadastro e conta</h2>
        <ul class="list-disc list-inside space-y-1">
          <li>Você deve informar dados verídicos no cadastro (nome, apelido, número de WhatsApp).</li>
          <li>Cada número de WhatsApp corresponde a uma única conta.</li>
          <li>Você é responsável pela segurança da sua senha e por toda atividade realizada com sua conta.</li>
          <li>Não é permitido criar contas em nome de terceiros sem autorização.</li>
          <li>Menores de 18 anos devem ter consentimento do responsável legal.</li>
        </ul>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">3. Uso aceito</h2>
        <p class="mb-2">Você pode usar o rachao.app para:</p>
        <ul class="list-disc list-inside space-y-1 mb-4">
          <li>Organizar grupos e partidas de futebol amador;</li>
          <li>Confirmar ou recusar presença em partidas;</li>
          <li>Votar nos melhores jogadores e acompanhar estatísticas do grupo.</li>
        </ul>
        <p class="mb-2">É expressamente <strong>proibido</strong>:</p>
        <ul class="list-disc list-inside space-y-1">
          <li>Usar a plataforma para fins ilegais ou que violem direitos de terceiros;</li>
          <li>Criar contas falsas, automatizar ações ou usar bots;</li>
          <li>Tentar acessar dados de outros usuários ou grupos sem autorização;</li>
          <li>Enviar conteúdo ofensivo, discriminatório ou abusivo para outros usuários.</li>
        </ul>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">4. Fase Beta</h2>
        <p class="mb-2">Durante o período Beta:</p>
        <ul class="list-disc list-inside space-y-1">
          <li>Funcionalidades podem ser alteradas, adicionadas ou removidas sem aviso prévio.</li>
          <li>Dados de toda a plataforma podem ser resetados com aviso mínimo de 48 horas pelo canal de comunicação do produto.</li>
          <li>Não garantimos disponibilidade contínua, tempo de resposta ou ausência de falhas.</li>
        </ul>
        <p class="mt-2">Ao aceitar estes termos, você reconhece e aceita essas condições específicas do período Beta.</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">5. Responsabilidades do usuário</h2>
        <p>Você é responsável por:</p>
        <ul class="list-disc list-inside space-y-1 mt-2">
          <li>Garantir que as informações fornecidas são verdadeiras e atualizadas;</li>
          <li>Usar a plataforma de acordo com estes Termos e a legislação vigente;</li>
          <li>Qualquer disputa ou conflito decorrente das partidas organizadas pela plataforma — o rachao.app é uma ferramenta de organização, não parte das partidas.</li>
        </ul>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">6. Limitação de responsabilidade</h2>
        <p class="mb-2">Na extensão permitida pela lei, o rachao.app não se responsabiliza por:</p>
        <ul class="list-disc list-inside space-y-1">
          <li>Danos indiretos, incidentais ou lucros cessantes;</li>
          <li>Perda de dados durante o período Beta;</li>
          <li>Falhas de disponibilidade ou interrupções do serviço;</li>
          <li>Conflitos entre usuários decorrentes das partidas organizadas.</li>
        </ul>
        <p class="mt-2">O serviço é fornecido "no estado em que se encontra" (as-is), sem garantias implícitas de adequação a uma finalidade específica.</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">7. Propriedade intelectual</h2>
        <p>A marca, logotipo, design e código-fonte do rachao.app são de propriedade exclusiva do controlador. O uso da plataforma não transfere nenhum direito de propriedade intelectual ao usuário.</p>
        <p class="mt-2">Os dados inseridos pelos usuários (nomes, resultados, estatísticas) permanecem de propriedade dos respectivos usuários. Ao usar a plataforma, você concede ao rachao.app licença não exclusiva para armazenar e exibir esses dados no contexto do serviço.</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">8. Cancelamento e encerramento</h2>
        <p>Você pode solicitar a exclusão da sua conta a qualquer momento pelo canal <strong>{PUBLIC_LEGAL_CONTACT_EMAIL}</strong>. A conta e seus dados serão removidos conforme os prazos descritos na <a href="/privacy" class="text-primary-600 hover:underline">Política de Privacidade</a>.</p>
        <p class="mt-2">Podemos suspender ou encerrar sua conta caso haja violação destes Termos, com ou sem aviso prévio dependendo da gravidade da infração.</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">9. Alterações nestes Termos</h2>
        <p>Quando publicarmos uma nova versão dos Termos, você será notificado na próxima vez que acessar a plataforma e precisará aceitar para continuar usando. A data no topo deste documento indica a versão vigente.</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">10. Lei aplicável e foro</h2>
        <p>Estes Termos são regidos pela legislação brasileira. Fica eleito o foro da comarca de <strong>{PUBLIC_LEGAL_FORUM_CITY}</strong> para dirimir quaisquer controvérsias, com renúncia a qualquer outro, por mais privilegiado que seja.</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">11. Contato</h2>
        <p><strong>{PUBLIC_LEGAL_CONTROLLER_NAME}</strong>, {PUBLIC_LEGAL_CONTROLLER_DOC}</p>
        <p>E-mail: <a href="mailto:{PUBLIC_LEGAL_CONTACT_EMAIL}" class="text-primary-600 hover:underline">{PUBLIC_LEGAL_CONTACT_EMAIL}</a></p>
        <p>Plataforma: <a href="https://rachao.app" class="text-primary-600 hover:underline">https://rachao.app</a></p>
      </section>

    </div>

    <div class="mt-12 pt-6 border-t border-gray-200 dark:border-gray-700 text-xs text-gray-400 flex flex-wrap gap-4">
      <a href="/privacy" class="hover:text-gray-600 dark:hover:text-gray-300">Política de Privacidade</a>
      <a href="/faq" class="hover:text-gray-600 dark:hover:text-gray-300">FAQ</a>
      <a href={backHref} class="hover:text-gray-600 dark:hover:text-gray-300">rachao.app</a>
    </div>

  </div>
</div>
