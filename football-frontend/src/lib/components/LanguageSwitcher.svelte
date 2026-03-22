<script lang="ts">
  import { Globe } from 'lucide-svelte';
  import { locale, setLocale, SUPPORTED_LOCALES, type Locale } from '$lib/i18n';

  interface Props {
    /** 'bar' = navbar desktop style | 'drawer' = mobile drawer style */
    variant?: 'bar' | 'drawer';
  }

  let { variant = 'bar' }: Props = $props();

  const LABELS: Record<Locale, { short: string; full: string; flag: string }> = {
    'pt-BR': { short: 'PT', full: 'Português (BR)', flag: '🇧🇷' },
    'en':    { short: 'EN', full: 'English',        flag: '🇺🇸' },
    'es':    { short: 'ES', full: 'Español',        flag: '🇪🇸' },
  };

  let open = $state(false);
  let buttonEl = $state<HTMLButtonElement | null>(null);
  let dropdownStyle = $state('');

  function openDropdown() {
    if (variant === 'bar' && buttonEl) {
      const rect = buttonEl.getBoundingClientRect();
      dropdownStyle = `position:fixed;top:${rect.bottom + 4}px;right:${window.innerWidth - rect.right}px;z-index:9999;`;
    }
    open = true;
  }

  function select(l: Locale) {
    setLocale(l);
    open = false;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') open = false;
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="relative">
  {#if variant === 'bar'}
    <button
      bind:this={buttonEl}
      onclick={openDropdown}
      class="btn-ghost btn-sm text-primary-100 hover:text-white hover:bg-primary-600 gap-1"
      title="Language / Idioma"
      aria-label="Language / Idioma"
    >
      <Globe size={14} />
      <span class="text-xs font-medium">{LABELS[$locale].flag} {LABELS[$locale].short}</span>
    </button>

    {#if open}
      <!-- Backdrop -->
      <button
        class="fixed inset-0 z-40"
        onclick={() => open = false}
        aria-label="Fechar"
        tabindex="-1"
      ></button>

      <div
        class="w-44 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-xl shadow-lg py-1 overflow-hidden"
        style={dropdownStyle}
      >
        {#each SUPPORTED_LOCALES as l}
          <button
            onclick={() => select(l)}
            class="w-full text-left px-3 py-2.5 text-sm flex items-center gap-2 transition-colors
              {$locale === l
                ? 'bg-primary-50 dark:bg-primary-900/30 text-primary-700 dark:text-primary-300 font-semibold'
                : 'text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700'}"
          >
            <span>{LABELS[l].flag}</span>
            <span>{LABELS[l].full}</span>
            {#if $locale === l}
              <span class="ml-auto text-primary-500 text-xs">✓</span>
            {/if}
          </button>
        {/each}
      </div>
    {/if}
  {:else}
    <!-- Modo drawer: opções expandem ACIMA do botão (dentro da área scrollável) -->
    {#if open}
      <div class="pl-8 space-y-0.5 mb-1">
        {#each SUPPORTED_LOCALES as l}
          <button
            onclick={() => select(l)}
            class="w-full text-left px-3 py-2 rounded-xl text-sm flex items-center gap-2 transition-colors
              {$locale === l
                ? 'bg-primary-900/40 text-white font-semibold'
                : 'text-primary-200 hover:bg-primary-700'}"
          >
            <span>{LABELS[l].flag}</span>
            <span>{LABELS[l].full}</span>
            {#if $locale === l}
              <span class="ml-auto text-primary-400 text-xs">✓</span>
            {/if}
          </button>
        {/each}
      </div>
    {/if}

    <button
      onclick={() => open = !open}
      class="flex items-center gap-2 px-3 py-3 rounded-xl text-sm font-medium transition-colors w-full text-left text-primary-100 hover:bg-primary-700"
      aria-label="Language / Idioma"
    >
      <Globe size={18} />
      <span>{LABELS[$locale].flag} {LABELS[$locale].full}</span>
    </button>
  {/if}
</div>
