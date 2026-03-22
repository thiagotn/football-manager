<script lang="ts">
  import { onMount } from 'svelte';
  import { PUBLIC_LEGAL_CONTACT_EMAIL } from '$env/static/public';
  import { themeStore } from '$lib/stores/theme';
  import { Sun, Moon, Link2 } from 'lucide-svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import { t } from '$lib/i18n';

  let openIndex = $state<number | null>(null);
  let copiedId = $state<string | null>(null);

  // Stable IDs (Portuguese slugs — never change, used for URL hashes)
  const FAQ_IDS = [
    'o-que-e-o-rachao-app',
    'como-confirmar-presenca',
    'como-entrar-em-um-grupo',
    'o-que-e-um-grupo-publico',
    'como-entrar-na-lista-de-espera',
    'participar-sem-cadastro',
    'como-fico-sabendo-se-fui-aceito',
    'posso-ver-o-status-da-candidatura',
    'se-for-aceito-preciso-entrar-na-fila',
    'como-descobrir-rachoes',
    'como-ver-lista-de-confirmados',
    'como-compartilhar-partida',
    'como-criar-minha-conta',
    'o-que-significa-cada-status',
    'rachao-fora-do-ar',
    'se-nao-confirmar-a-tempo',
    'como-instalar-android',
    'como-instalar-iphone',
  ];

  const FAQ_STEPS: Record<string, number> = {
    'como-instalar-android': 4,
    'como-instalar-iphone': 4,
  };

  let faqs = $derived(
    FAQ_IDS.map((id, i) => {
      const n = i + 1;
      const stepsCount = FAQ_STEPS[id];
      return {
        id,
        q: $t(`faq.q${n}.q`),
        a: $t(`faq.q${n}.a`),
        steps: stepsCount
          ? Array.from({ length: stepsCount }, (_, j) => $t(`faq.q${n}.s${j + 1}`))
          : undefined,
      };
    })
  );

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
      const idx = FAQ_IDS.indexOf(hash);
      if (idx !== -1) {
        openIndex = idx;
        setTimeout(() => {
          document.getElementById(hash)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
        }, 100);
      }
    }
  });
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
      title={$t('aria.theme')}
    >
      {#if $themeStore === 'dark'}<Sun size={18} />{:else}<Moon size={18} />{/if}
    </button>
    <img src="/logo.png" alt="rachao.app" width="320" height="174" class="w-44 block mx-auto mb-3" />
    <h1 class="text-2xl font-bold text-white">{$t('faq.title')}</h1>
    <p class="text-gray-300 mt-1 text-sm">{$t('faq.subtitle')}</p>
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
              title={$t('faq.copy_link')}
              aria-label={$t('faq.copy_link')}
            >
              {#if copiedId === faq.id}
                <span class="text-xs text-green-500 font-normal whitespace-nowrap">{$t('faq.copied')}</span>
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
      {$t('faq.contact')}
      <a href="mailto:{PUBLIC_LEGAL_CONTACT_EMAIL}" class="text-primary-400 hover:underline">{PUBLIC_LEGAL_CONTACT_EMAIL}</a>.
    </div>
  </main>
</PageBackground>
