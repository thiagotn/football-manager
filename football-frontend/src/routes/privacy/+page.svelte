<script lang="ts">
  import { PUBLIC_LEGAL_CONTROLLER_NAME, PUBLIC_LEGAL_CONTROLLER_DOC, PUBLIC_LEGAL_CONTACT_EMAIL } from '$env/static/public';
  import { isLoggedIn } from '$lib/stores/auth';

  let backHref = $derived($isLoggedIn ? '/' : '/lp');
</script>

<svelte:head><title>Política de Privacidade — rachao.app</title></svelte:head>

<div class="min-h-screen bg-white dark:bg-gray-900 text-gray-800 dark:text-gray-200">
  <div class="max-w-3xl mx-auto px-6 py-10">

    <div class="mb-8">
      <a href={backHref} class="text-sm text-primary-600 hover:underline">← rachao.app</a>
    </div>

    <h1 class="text-2xl font-bold mb-1">Política de Privacidade</h1>
    <p class="text-xs text-gray-400 mb-8">Última atualização: março de 2026 · Versão: 1.2</p>

    <div class="prose prose-sm dark:prose-invert max-w-none space-y-8">

      <section>
        <h2 class="text-base font-semibold mb-2">1. Quem somos</h2>
        <p>O rachao.app é uma plataforma de organização de grupos e partidas de futebol amador, operada por <strong>{PUBLIC_LEGAL_CONTROLLER_NAME}</strong>{#if PUBLIC_LEGAL_CONTROLLER_DOC}, {PUBLIC_LEGAL_CONTROLLER_DOC}{/if} ("nós", "nosso" ou "controlador"), com canal de contato em <strong>{PUBLIC_LEGAL_CONTACT_EMAIL}</strong>.</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">2. Quais dados coletamos e para quê</h2>
        <div class="overflow-x-auto">
          <table class="w-full text-sm border-collapse">
            <thead>
              <tr class="border-b border-gray-200 dark:border-gray-700">
                <th class="text-left py-2 pr-4 font-semibold">Dado</th>
                <th class="text-left py-2 pr-4 font-semibold">Finalidade</th>
                <th class="text-left py-2 font-semibold">Base legal (LGPD, Art. 7º)</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-gray-800">
              <tr>
                <td class="py-2 pr-4 align-top">Número de WhatsApp</td>
                <td class="py-2 pr-4 align-top">Identificação da conta, envio de código de verificação no cadastro</td>
                <td class="py-2 align-top">Execução de contrato</td>
              </tr>
              <tr>
                <td class="py-2 pr-4 align-top">Nome e apelido</td>
                <td class="py-2 pr-4 align-top">Identificação em grupos e partidas</td>
                <td class="py-2 align-top">Execução de contrato</td>
              </tr>
              <tr>
                <td class="py-2 pr-4 align-top">Senha (hash bcrypt — nunca em texto claro)</td>
                <td class="py-2 pr-4 align-top">Autenticação</td>
                <td class="py-2 align-top">Execução de contrato</td>
              </tr>
              <tr>
                <td class="py-2 pr-4 align-top">Confirmações de presença em partidas</td>
                <td class="py-2 pr-4 align-top">Gestão de partidas e listas de presença</td>
                <td class="py-2 align-top">Execução de contrato</td>
              </tr>
              <tr>
                <td class="py-2 pr-4 align-top">Votos pós-partida e estatísticas</td>
                <td class="py-2 pr-4 align-top">Ranking e histórico do grupo</td>
                <td class="py-2 align-top">Execução de contrato</td>
              </tr>
              <tr>
                <td class="py-2 pr-4 align-top">IP e logs de acesso</td>
                <td class="py-2 pr-4 align-top">Segurança, diagnóstico de falhas e prevenção de abusos</td>
                <td class="py-2 align-top">Legítimo interesse</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p class="mt-3 text-sm">Não coletamos dados sensíveis (saúde, biometria, opinião política, etnia, religião). Não vendemos dados a terceiros. Não utilizamos os dados para publicidade.</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">3. Com quem compartilhamos os dados</h2>
        <div class="overflow-x-auto">
          <table class="w-full text-sm border-collapse">
            <thead>
              <tr class="border-b border-gray-200 dark:border-gray-700">
                <th class="text-left py-2 pr-4 font-semibold">Parceiro</th>
                <th class="text-left py-2 pr-4 font-semibold">Dado compartilhado</th>
                <th class="text-left py-2 font-semibold">Motivo</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-gray-800">
              <tr>
                <td class="py-2 pr-4 align-top">Provedor de banco de dados (Supabase)</td>
                <td class="py-2 pr-4 align-top">Todos os dados da plataforma</td>
                <td class="py-2 align-top">Armazenamento e operação do serviço</td>
              </tr>
              <tr>
                <td class="py-2 pr-4 align-top">Gateway de pagamento (Stripe)</td>
                <td class="py-2 pr-4 align-top">Dados de cobrança e assinatura</td>
                <td class="py-2 align-top">Processamento de pagamentos dos planos pagos</td>
              </tr>
              <tr>
                <td class="py-2 pr-4 align-top">Provedor de SMS (Twilio)</td>
                <td class="py-2 pr-4 align-top">Número de WhatsApp</td>
                <td class="py-2 align-top">Envio de código de verificação no cadastro</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p class="mt-3 text-sm">Os provedores operam como <strong>operadores de dados</strong> sob as nossas instruções e possuem políticas de privacidade e segurança próprias. Servidores podem estar localizados fora do Brasil; ao usar a plataforma, você consente com essa transferência internacional nos termos do Art. 33 da LGPD.</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">4. Por quanto tempo guardamos seus dados</h2>
        <div class="overflow-x-auto">
          <table class="w-full text-sm border-collapse">
            <thead>
              <tr class="border-b border-gray-200 dark:border-gray-700">
                <th class="text-left py-2 pr-4 font-semibold">Dado</th>
                <th class="text-left py-2 font-semibold">Prazo de retenção</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-gray-800">
              <tr>
                <td class="py-2 pr-4 align-top">Dados de conta (nome, WhatsApp, senha)</td>
                <td class="py-2 align-top">Enquanto a conta estiver ativa; excluídos em até 30 dias após solicitação</td>
              </tr>
              <tr>
                <td class="py-2 pr-4 align-top">Histórico de partidas, presenças e votos</td>
                <td class="py-2 align-top">Enquanto o grupo ou a conta existir</td>
              </tr>
              <tr>
                <td class="py-2 pr-4 align-top">Logs de acesso e IP</td>
                <td class="py-2 align-top">90 dias</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">5. Seus direitos como titular</h2>
        <p class="mb-2">De acordo com o Art. 18 da LGPD, você tem direito a:</p>
        <ul class="list-disc list-inside space-y-1">
          <li><strong>Confirmar</strong> se tratamos seus dados</li>
          <li><strong>Acessar</strong> os dados que temos sobre você</li>
          <li><strong>Corrigir</strong> dados incompletos, inexatos ou desatualizados</li>
          <li><strong>Solicitar a exclusão</strong> dos seus dados (sujeito a obrigações legais de retenção)</li>
          <li><strong>Revogar o consentimento</strong> a qualquer momento, sem prejuízo da licitude dos tratamentos realizados antes</li>
          <li><strong>Portabilidade</strong> dos seus dados em formato estruturado (a implementar)</li>
          <li><strong>Reclamar</strong> à Autoridade Nacional de Proteção de Dados (ANPD) caso considere que seus direitos foram violados</li>
        </ul>
        <p class="mt-3 text-sm">Para exercer qualquer direito, envie um e-mail para <strong>{PUBLIC_LEGAL_CONTACT_EMAIL}</strong> com assunto "Direitos LGPD" e seu número de WhatsApp cadastrado para identificação. Respondemos em até <strong>15 dias úteis</strong>.</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">6. Segurança</h2>
        <p class="mb-2">Adotamos as seguintes medidas de segurança:</p>
        <ul class="list-disc list-inside space-y-1">
          <li>Senhas armazenadas exclusivamente como hash bcrypt (nunca em texto claro)</li>
          <li>Comunicação via HTTPS em todo o tráfego</li>
          <li>Acesso ao banco de dados restrito por variáveis de ambiente e credenciais de serviço</li>
          <li>Tokens JWT com expiração configurada</li>
        </ul>
        <p class="mt-2">Nenhum sistema é 100% seguro. Em caso de incidente de segurança que afete seus dados, notificaremos os usuários afetados e a ANPD conforme exigido pela lei.</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">7. Cookies e armazenamento local</h2>
        <p>O rachao.app não utiliza cookies de rastreamento ou publicidade. Utilizamos <strong>localStorage</strong> do navegador exclusivamente para armazenar o token de autenticação da sessão. Esses dados ficam apenas no seu dispositivo e são removidos ao fazer logout.</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">8. Alterações nesta política</h2>
        <p>Quando realizarmos alterações relevantes nesta política, atualizaremos a data no topo deste documento. Para mudanças que ampliem significativamente o uso dos seus dados, notificaremos os usuários ativos pelo canal de comunicação da plataforma com antecedência mínima de 15 dias.</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">9. Contato</h2>
        <p><strong>{PUBLIC_LEGAL_CONTROLLER_NAME}</strong>{#if PUBLIC_LEGAL_CONTROLLER_DOC}, {PUBLIC_LEGAL_CONTROLLER_DOC}{/if}</p>
        <p>E-mail: <a href="mailto:{PUBLIC_LEGAL_CONTACT_EMAIL}" class="text-primary-600 hover:underline">{PUBLIC_LEGAL_CONTACT_EMAIL}</a></p>
        <p>Plataforma: <a href="https://rachao.app" class="text-primary-600 hover:underline">https://rachao.app</a></p>
      </section>

    </div>

    <div class="mt-12 pt-6 border-t border-gray-200 dark:border-gray-700 text-xs text-gray-400 flex flex-wrap gap-4">
      <a href="/terms" class="hover:text-gray-600 dark:hover:text-gray-300">Termos de Uso</a>
      <a href="/faq" class="hover:text-gray-600 dark:hover:text-gray-300">FAQ</a>
      <a href={backHref} class="hover:text-gray-600 dark:hover:text-gray-300">rachao.app</a>
    </div>

  </div>
</div>
