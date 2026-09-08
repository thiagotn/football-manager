<script lang="ts">
  /**
   * AvatarLightbox — overlay fullscreen para ver a foto do jogador ampliada.
   *
   * O avatar é armazenado em 256×256, então a imagem é limitada a ~384px
   * (min(85vw, 384px)) para não ficar visivelmente borrada.
   * Fecha por: botão X, clique/toque no fundo escuro ou tecla Escape.
   *
   * Props:
   *   open      — bindable; controla exibição
   *   src       — URL da foto (null → não renderiza nada)
   *   name      — nome do jogador (alt + legenda)
   *   updatedAt — cache-buster opcional (mesmo comportamento do AvatarImage)
   */
  import { X } from 'lucide-svelte';
  import { fade, scale } from 'svelte/transition';
  import { t } from '$lib/i18n';

  let {
    open = $bindable(false),
    src,
    name,
    updatedAt,
  }: {
    open?: boolean;
    src: string | null | undefined;
    name: string;
    updatedAt?: string | null;
  } = $props();

  const imgSrc = $derived(
    src ? (updatedAt ? `${src}?v=${encodeURIComponent(updatedAt)}` : src) : null
  );

  function close() {
    open = false;
  }

  // Fase de captura + stopImmediatePropagation: o Esc fecha só o lightbox,
  // sem chegar ao listener do Modal que estiver aberto por baixo.
  function handleKeydown(e: KeyboardEvent) {
    if (open && e.key === 'Escape') {
      e.preventDefault();
      e.stopImmediatePropagation();
      close();
    }
  }
</script>

<svelte:window onkeydowncapture={handleKeydown} />

{#if open && imgSrc}
  <div class="fixed inset-0 z-[70] flex items-center justify-center p-4" data-testid="avatar-lightbox">
    <!-- Backdrop: clique/toque em qualquer lugar fora da foto fecha -->
    <button
      type="button"
      class="absolute inset-0 bg-black/90 backdrop-blur-sm"
      onclick={close}
      aria-label={$t('aria.close')}
      transition:fade={{ duration: 150 }}
    ></button>

    <!-- Botão fechar: canto superior direito, área de toque ≥ 44px -->
    <button
      type="button"
      class="absolute top-3 right-3 z-20 p-3 rounded-full text-white/90 hover:text-white hover:bg-white/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/70"
      onclick={close}
      aria-label={$t('aria.close')}
    >
      <X size={24} />
    </button>

    <!-- Foto + legenda -->
    <div class="relative z-10 flex flex-col items-center gap-3 pointer-events-none" transition:scale={{ duration: 180, start: 0.92 }}>
      <img
        src={imgSrc}
        alt={name}
        class="rounded-2xl shadow-2xl object-cover bg-gray-800"
        style="width: min(85vw, 384px); height: min(85vw, 384px);"
      />
      <p class="text-white/80 text-sm font-medium text-center max-w-[85vw] truncate">{name}</p>
    </div>
  </div>
{/if}
