<script lang="ts">
  /**
   * AvatarImage — exibe a foto do jogador ou iniciais com cor determinística como fallback.
   *
   * Props:
   *   name       — nome do jogador (para gerar iniciais e cor)
   *   avatarUrl  — URL da foto (null/undefined → mostra iniciais)
   *   updatedAt  — timestamp usado como cache-buster quando a foto muda (optional)
   *   size       — tamanho em px (default 40)
   *   class      — classes adicionais para o contêiner
   */
  interface Props {
    name: string;
    avatarUrl?: string | null;
    updatedAt?: string | null;
    size?: number;
    class?: string;
  }

  const { name, avatarUrl, updatedAt, size = 40, class: extraClass = '' }: Props = $props();

  const COLORS = [
    '#e53e3e', '#3b82f6', '#f59e0b', '#22c55e',
    '#f97316', '#a855f7', '#ec4899', '#06b6d4',
    '#84cc16', '#14b8a6',
  ];

  function avatarColor(n: string): string {
    let hash = 0;
    for (let i = 0; i < n.length; i++) {
      hash = (n.charCodeAt(i) + ((hash << 5) - hash)) | 0;
    }
    return COLORS[Math.abs(hash) % COLORS.length];
  }

  function initials(n: string): string {
    const parts = n.trim().split(/\s+/).filter(Boolean);
    if (parts.length >= 2) return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
    return n.slice(0, 2).toUpperCase();
  }

  const bg = $derived(avatarColor(name || '?'));
  const text = $derived(initials(name || '?'));
  const fontSize = $derived(Math.round(size * 0.38));

  // cache-buster: evita servir imagem stale após troca de avatar
  const imgSrc = $derived(
    avatarUrl
      ? (updatedAt ? `${avatarUrl}?v=${encodeURIComponent(updatedAt)}` : avatarUrl)
      : null
  );

  let imageLoaded = $state(false);
  let imageError = $state(false);

  // reset ao trocar de URL
  $effect(() => {
    imgSrc;
    imageLoaded = false;
    imageError = false;
  });
</script>

<div
  class="relative shrink-0 rounded-full overflow-hidden flex items-center justify-center select-none {extraClass}"
  style="width: {size}px; height: {size}px; background-color: {bg};"
>
  {#if imgSrc && !imageError}
    <!-- shimmer visível enquanto a imagem carrega -->
    {#if !imageLoaded}
      <div class="absolute inset-0 rounded-full overflow-hidden">
        <div class="absolute inset-0"
          style="background: linear-gradient(90deg, #374151 25%, #4b5563 50%, #374151 75%);
                 background-size: 200% 100%;
                 animation: shimmer 1.4s ease-in-out infinite;">
        </div>
      </div>
    {/if}
    <img
      src={imgSrc}
      alt={name}
      class="w-full h-full object-cover transition-opacity duration-300"
      style="opacity: {imageLoaded ? 1 : 0};"
      onload={() => { imageLoaded = true; }}
      onerror={() => { imageError = true; }}
    />
  {:else}
    <span
      class="font-bold text-white leading-none"
      style="font-size: {fontSize}px;"
    >{text}</span>
  {/if}
</div>

<style>
  @keyframes shimmer {
    0%   { background-position: 200% 0; }
    100% { background-position: -200% 0; }
  }
</style>
