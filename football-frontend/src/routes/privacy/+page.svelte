<script lang="ts">
  import { PUBLIC_LEGAL_CONTROLLER_NAME, PUBLIC_LEGAL_CONTROLLER_DOC, PUBLIC_LEGAL_CONTACT_EMAIL } from '$env/static/public';
  import { isLoggedIn } from '$lib/stores/auth';
  import { t } from '$lib/i18n';

  let backHref = $derived($isLoggedIn ? '/' : '/lp');

  let controllerFull = $derived(
    PUBLIC_LEGAL_CONTROLLER_DOC
      ? `${PUBLIC_LEGAL_CONTROLLER_NAME}, ${PUBLIC_LEGAL_CONTROLLER_DOC}`
      : PUBLIC_LEGAL_CONTROLLER_NAME
  );
</script>

<svelte:head><title>Política de Privacidade — rachao.app</title></svelte:head>

<div class="min-h-screen bg-white dark:bg-gray-900 text-gray-800 dark:text-gray-200">
  <div class="max-w-3xl mx-auto px-6 py-10">

    <div class="mb-8">
      <a href={backHref} class="text-sm text-primary-600 hover:underline">{$t('privacy.back')}</a>
    </div>

    <h1 class="text-2xl font-bold mb-1">{$t('privacy.title')}</h1>
    <p class="text-xs text-gray-400 mb-8">{$t('privacy.version')}</p>

    <div class="prose prose-sm dark:prose-invert max-w-none space-y-8">

      <section>
        <h2 class="text-base font-semibold mb-2">{$t('privacy.s1.h')}</h2>
        <p>{@html $t('privacy.s1.p1', { controller: controllerFull, email: PUBLIC_LEGAL_CONTACT_EMAIL })}</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">{$t('privacy.s2.h')}</h2>
        <div class="overflow-x-auto">
          <table class="w-full text-sm border-collapse">
            <thead>
              <tr class="border-b border-gray-200 dark:border-gray-700">
                <th class="text-left py-2 pr-4 font-semibold">{$t('privacy.table.data')}</th>
                <th class="text-left py-2 pr-4 font-semibold">{$t('privacy.table.purpose')}</th>
                <th class="text-left py-2 font-semibold">{$t('privacy.table.basis')}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-gray-800">
              <tr>
                <td class="py-2 pr-4 align-top">{$t('privacy.s2.r1.data')}</td>
                <td class="py-2 pr-4 align-top">{$t('privacy.s2.r1.purpose')}</td>
                <td class="py-2 align-top">{$t('privacy.s2.basis_contract')}</td>
              </tr>
              <tr>
                <td class="py-2 pr-4 align-top">{$t('privacy.s2.r2.data')}</td>
                <td class="py-2 pr-4 align-top">{$t('privacy.s2.r2.purpose')}</td>
                <td class="py-2 align-top">{$t('privacy.s2.basis_contract')}</td>
              </tr>
              <tr>
                <td class="py-2 pr-4 align-top">{$t('privacy.s2.r3.data')}</td>
                <td class="py-2 pr-4 align-top">{$t('privacy.s2.r3.purpose')}</td>
                <td class="py-2 align-top">{$t('privacy.s2.basis_contract')}</td>
              </tr>
              <tr>
                <td class="py-2 pr-4 align-top">{$t('privacy.s2.r4.data')}</td>
                <td class="py-2 pr-4 align-top">{$t('privacy.s2.r4.purpose')}</td>
                <td class="py-2 align-top">{$t('privacy.s2.basis_contract')}</td>
              </tr>
              <tr>
                <td class="py-2 pr-4 align-top">{$t('privacy.s2.r5.data')}</td>
                <td class="py-2 pr-4 align-top">{$t('privacy.s2.r5.purpose')}</td>
                <td class="py-2 align-top">{$t('privacy.s2.basis_contract')}</td>
              </tr>
              <tr>
                <td class="py-2 pr-4 align-top">{$t('privacy.s2.r6.data')}</td>
                <td class="py-2 pr-4 align-top">{$t('privacy.s2.r6.purpose')}</td>
                <td class="py-2 align-top">{$t('privacy.s2.basis_legitimate')}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p class="mt-3 text-sm">{$t('privacy.s2.note')}</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">{$t('privacy.s3.h')}</h2>
        <div class="overflow-x-auto">
          <table class="w-full text-sm border-collapse">
            <thead>
              <tr class="border-b border-gray-200 dark:border-gray-700">
                <th class="text-left py-2 pr-4 font-semibold">{$t('privacy.table.partner')}</th>
                <th class="text-left py-2 pr-4 font-semibold">{$t('privacy.table.shared_data')}</th>
                <th class="text-left py-2 font-semibold">{$t('privacy.table.reason')}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-gray-800">
              <tr>
                <td class="py-2 pr-4 align-top">{$t('privacy.s3.r1.partner')}</td>
                <td class="py-2 pr-4 align-top">{$t('privacy.s3.r1.data')}</td>
                <td class="py-2 align-top">{$t('privacy.s3.r1.reason')}</td>
              </tr>
              <tr>
                <td class="py-2 pr-4 align-top">{$t('privacy.s3.r2.partner')}</td>
                <td class="py-2 pr-4 align-top">{$t('privacy.s3.r2.data')}</td>
                <td class="py-2 align-top">{$t('privacy.s3.r2.reason')}</td>
              </tr>
              <tr>
                <td class="py-2 pr-4 align-top">{$t('privacy.s3.r3.partner')}</td>
                <td class="py-2 pr-4 align-top">{$t('privacy.s3.r3.data')}</td>
                <td class="py-2 align-top">{$t('privacy.s3.r3.reason')}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p class="mt-3 text-sm">{@html $t('privacy.s3.note')}</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">{$t('privacy.s4.h')}</h2>
        <div class="overflow-x-auto">
          <table class="w-full text-sm border-collapse">
            <thead>
              <tr class="border-b border-gray-200 dark:border-gray-700">
                <th class="text-left py-2 pr-4 font-semibold">{$t('privacy.table.data')}</th>
                <th class="text-left py-2 font-semibold">{$t('privacy.table.retention')}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-gray-800">
              <tr>
                <td class="py-2 pr-4 align-top">{$t('privacy.s4.r1.data')}</td>
                <td class="py-2 align-top">{$t('privacy.s4.r1.retention')}</td>
              </tr>
              <tr>
                <td class="py-2 pr-4 align-top">{$t('privacy.s4.r2.data')}</td>
                <td class="py-2 align-top">{$t('privacy.s4.r2.retention')}</td>
              </tr>
              <tr>
                <td class="py-2 pr-4 align-top">{$t('privacy.s4.r3.data')}</td>
                <td class="py-2 align-top">{$t('privacy.s4.r3.retention')}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">{$t('privacy.s5.h')}</h2>
        <p class="mb-2">{$t('privacy.s5.p1')}</p>
        <ul class="list-disc list-inside space-y-1">
          <li>{@html $t('privacy.s5.l1')}</li>
          <li>{@html $t('privacy.s5.l2')}</li>
          <li>{@html $t('privacy.s5.l3')}</li>
          <li>{@html $t('privacy.s5.l4')}</li>
          <li>{@html $t('privacy.s5.l5')}</li>
          <li>{@html $t('privacy.s5.l6')}</li>
          <li>{@html $t('privacy.s5.l7')}</li>
        </ul>
        <p class="mt-3 text-sm">{@html $t('privacy.s5.p2', { email: PUBLIC_LEGAL_CONTACT_EMAIL })}</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">{$t('privacy.s6.h')}</h2>
        <p class="mb-2">{$t('privacy.s6.p1')}</p>
        <ul class="list-disc list-inside space-y-1">
          <li>{$t('privacy.s6.l1')}</li>
          <li>{$t('privacy.s6.l2')}</li>
          <li>{$t('privacy.s6.l3')}</li>
          <li>{$t('privacy.s6.l4')}</li>
        </ul>
        <p class="mt-2">{$t('privacy.s6.p2')}</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">{$t('privacy.s7.h')}</h2>
        <p>{$t('privacy.s7.p1')}</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">{$t('privacy.s8.h')}</h2>
        <p>{$t('privacy.s8.p1')}</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">{$t('privacy.s9.h')}</h2>
        <p><strong>{PUBLIC_LEGAL_CONTROLLER_NAME}</strong>{#if PUBLIC_LEGAL_CONTROLLER_DOC}, {PUBLIC_LEGAL_CONTROLLER_DOC}{/if}</p>
        <p>E-mail: <a href="mailto:{PUBLIC_LEGAL_CONTACT_EMAIL}" class="text-primary-600 hover:underline">{PUBLIC_LEGAL_CONTACT_EMAIL}</a></p>
        <p>Plataforma: <a href="https://rachao.app" class="text-primary-600 hover:underline">https://rachao.app</a></p>
      </section>

    </div>

    <div class="mt-12 pt-6 border-t border-gray-200 dark:border-gray-700 text-xs text-gray-400 flex flex-wrap gap-4">
      <a href="/terms" class="hover:text-gray-600 dark:hover:text-gray-300">{$t('footer.terms')}</a>
      <a href="/faq" class="hover:text-gray-600 dark:hover:text-gray-300">{$t('footer.faq')}</a>
      <a href={backHref} class="hover:text-gray-600 dark:hover:text-gray-300">rachao.app</a>
    </div>

  </div>
</div>
