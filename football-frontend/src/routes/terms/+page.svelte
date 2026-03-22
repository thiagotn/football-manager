<script lang="ts">
  import { PUBLIC_LEGAL_CONTROLLER_NAME, PUBLIC_LEGAL_CONTROLLER_DOC, PUBLIC_LEGAL_FORUM_CITY, PUBLIC_LEGAL_CONTACT_EMAIL } from '$env/static/public';
  import { isLoggedIn } from '$lib/stores/auth';
  import { t } from '$lib/i18n';

  let backHref = $derived($isLoggedIn ? '/' : '/lp');

  let controllerFull = $derived(
    PUBLIC_LEGAL_CONTROLLER_DOC
      ? `${PUBLIC_LEGAL_CONTROLLER_NAME}, ${PUBLIC_LEGAL_CONTROLLER_DOC}`
      : PUBLIC_LEGAL_CONTROLLER_NAME
  );
</script>

<svelte:head><title>Termos de Uso — rachao.app</title></svelte:head>

<div class="min-h-screen bg-white dark:bg-gray-900 text-gray-800 dark:text-gray-200">
  <div class="max-w-3xl mx-auto px-6 py-10">

    <div class="mb-8">
      <a href={backHref} class="text-sm text-primary-600 hover:underline">{$t('terms.back')}</a>
    </div>

    <h1 class="text-2xl font-bold mb-1">{$t('terms.title')}</h1>
    <p class="text-xs text-gray-400 mb-8">{$t('terms.version')}</p>

    <div class="prose prose-sm dark:prose-invert max-w-none space-y-8">

      <section>
        <h2 class="text-base font-semibold mb-2">{$t('terms.s1.h')}</h2>
        <p>{$t('terms.s1.p1')}</p>
        <p class="mt-2">{@html $t('terms.s1.p2', { controller: controllerFull, email: PUBLIC_LEGAL_CONTACT_EMAIL })}</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">{$t('terms.s2.h')}</h2>
        <ul class="list-disc list-inside space-y-1">
          <li>{$t('terms.s2.l1')}</li>
          <li>{$t('terms.s2.l2')}</li>
          <li>{$t('terms.s2.l3')}</li>
          <li>{$t('terms.s2.l4')}</li>
          <li>{$t('terms.s2.l5')}</li>
        </ul>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">{$t('terms.s3.h')}</h2>
        <p class="mb-2">{$t('terms.s3.p1')}</p>
        <ul class="list-disc list-inside space-y-1 mb-4">
          <li>{$t('terms.s3.l1')}</li>
          <li>{$t('terms.s3.l2')}</li>
          <li>{$t('terms.s3.l3')}</li>
        </ul>
        <p class="mb-2">{@html $t('terms.s3.p2')}</p>
        <ul class="list-disc list-inside space-y-1">
          <li>{$t('terms.s3.l4')}</li>
          <li>{$t('terms.s3.l5')}</li>
          <li>{$t('terms.s3.l6')}</li>
          <li>{$t('terms.s3.l7')}</li>
        </ul>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">{$t('terms.s4.h')}</h2>
        <p>{$t('terms.s4.p1')}</p>
        <ul class="list-disc list-inside space-y-1 mt-2">
          <li>{$t('terms.s4.l1')}</li>
          <li>{$t('terms.s4.l2')}</li>
          <li>{$t('terms.s4.l3')}</li>
        </ul>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">{$t('terms.s4b.h')}</h2>
        <p>{@html $t('terms.s4b.p1')}</p>
        <p class="mt-2">{@html $t('terms.s4b.p2')}</p>
        <p class="mt-2">{@html $t('terms.s4b.p3')}</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">{$t('terms.s5.h')}</h2>
        <p class="mb-2">{$t('terms.s5.p1')}</p>
        <ul class="list-disc list-inside space-y-1">
          <li>{$t('terms.s5.l1')}</li>
          <li>{$t('terms.s5.l2')}</li>
          <li>{$t('terms.s5.l3')}</li>
        </ul>
        <p class="mt-2">{$t('terms.s5.p2')}</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">{$t('terms.s6.h')}</h2>
        <p>{$t('terms.s6.p1')}</p>
        <p class="mt-2">{$t('terms.s6.p2')}</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">{$t('terms.s7.h')}</h2>
        <p>{@html $t('terms.s7.p1', { email: PUBLIC_LEGAL_CONTACT_EMAIL })}</p>
        <p class="mt-2">{$t('terms.s7.p2')}</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">{$t('terms.s8.h')}</h2>
        <p>{$t('terms.s8.p1')}</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">{$t('terms.s9.h')}</h2>
        <p>{@html $t('terms.s9.p1', { city: PUBLIC_LEGAL_FORUM_CITY })}</p>
      </section>

      <section>
        <h2 class="text-base font-semibold mb-2">{$t('terms.s10.h')}</h2>
        <p><strong>{PUBLIC_LEGAL_CONTROLLER_NAME}</strong>{#if PUBLIC_LEGAL_CONTROLLER_DOC}, {PUBLIC_LEGAL_CONTROLLER_DOC}{/if}</p>
        <p>E-mail: <a href="mailto:{PUBLIC_LEGAL_CONTACT_EMAIL}" class="text-primary-600 hover:underline">{PUBLIC_LEGAL_CONTACT_EMAIL}</a></p>
        <p>Plataforma: <a href="https://rachao.app" class="text-primary-600 hover:underline">https://rachao.app</a></p>
      </section>

    </div>

    <div class="mt-12 pt-6 border-t border-gray-200 dark:border-gray-700 text-xs text-gray-400 flex flex-wrap gap-4">
      <a href="/privacy" class="hover:text-gray-600 dark:hover:text-gray-300">{$t('footer.privacy')}</a>
      <a href="/faq" class="hover:text-gray-600 dark:hover:text-gray-300">{$t('footer.faq')}</a>
      <a href={backHref} class="hover:text-gray-600 dark:hover:text-gray-300">rachao.app</a>
    </div>

  </div>
</div>
