<script lang="ts">
  /**
   * Seletor PT/EN/ES. Nasceu dentro dos ajustes do Enche Aí!; com mais de um
   * jogo virou componente, usado também no hub.
   *
   * `tone="paper"` para fundo claro (cards de menu), `tone="dark"` para o hub.
   */
  import { SUPPORTED_LOCALES, locale, setLocale } from '$lib/i18n';

  let { tone = 'paper' }: { tone?: 'paper' | 'dark' } = $props();

  const LABEL: Record<string, string> = { 'pt-BR': 'PT', en: 'EN', es: 'ES' };
</script>

<span class="flex gap-1">
  {#each SUPPORTED_LOCALES as l (l)}
    <button
      class="h-8 rounded-lg px-2.5 text-xs font-extrabold {tone === 'paper'
        ? $locale === l
          ? 'bg-paper-ink text-paper'
          : 'bg-paper-ink/10'
        : $locale === l
          ? 'bg-mustard text-mustard-ink'
          : 'bg-white/10 text-white/70'}"
      onclick={() => setLocale(l)}
      aria-pressed={$locale === l}
      lang={l}
    >{LABEL[l]}</button>
  {/each}
</span>
