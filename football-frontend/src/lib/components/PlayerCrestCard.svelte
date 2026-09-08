<script lang="ts">
  /**
   * PlayerCrestCard — cartinha em forma de escudo com os dados do jogador
   * (handoff design_handoff_player_crest_card, Componente A).
   *
   * Tamanho fixo 300×360 (clip-path em px). Abaixo de 340px de viewport
   * aplica scale(.92) para caber com o padding do modal.
   * O escudo é sempre escuro (verde/dourado), independente do tema do app.
   *
   * Props:
   *   name         — nome real do jogador
   *   nickname     — apelido global (fallback: name)
   *   avatarUrl    — foto; null → iniciais (padrão do AvatarImage)
   *   updatedAt    — cache-buster opcional da foto
   *   position     — código da API (gk/zag/lat/mei/ata) ou null → oculta bloco
   *   skillStars   — 1..5 ou null → oculta estrelas
   *   onPhotoClick — se informado, a foto vira botão (ex: abrir AvatarLightbox)
   *   photoLabel   — aria-label do botão da foto (já traduzido)
   */
  import '@fontsource/barlow-condensed/800.css';
  import AvatarImage from '$lib/components/AvatarImage.svelte';
  import { POS_ABBR } from '$lib/team-builder';
  import { API_TO_POS, POS_I18N_KEY } from '$lib/positions';
  import { t } from '$lib/i18n';

  let {
    name,
    nickname,
    avatarUrl,
    updatedAt,
    position,
    skillStars,
    onPhotoClick,
    photoLabel,
  }: {
    name: string;
    nickname?: string | null;
    avatarUrl?: string | null;
    updatedAt?: string | null;
    position?: string | null;
    skillStars?: number | null;
    onPhotoClick?: () => void;
    photoLabel?: string;
  } = $props();

  const STAR = 'M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01z';

  const pos = $derived(position ? API_TO_POS[position] : undefined);
  const displayNickname = $derived((nickname && nickname.trim()) || name);
</script>

<div class="max-[339px]:scale-[.92] max-[339px]:origin-top max-[339px]:-mb-7" data-testid="player-crest">
  <div class="crest-outer relative shrink-0 drop-shadow-[0_20px_40px_rgba(31,154,77,.35)]">
    <div class="crest-inner">

      <!-- A1. Habilidade (topo) -->
      {#if skillStars != null}
        <div class="absolute top-0 inset-x-0 px-4 pt-4 z-20 text-center">
          <div class="flex items-center justify-center gap-1.5" role="img" aria-label="{skillStars}/5">
            {#each [1, 2, 3, 4, 5] as i}
              {#if i <= skillStars}
                <svg width="19" height="19" viewBox="0 0 24 24" fill="#f5d585" aria-hidden="true"><path d={STAR} /></svg>
              {:else}
                <svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="#f5d585" stroke-opacity=".5" stroke-width="1.8" aria-hidden="true"><path d={STAR} /></svg>
              {/if}
            {/each}
          </div>
          <div class="text-[12px] font-bold uppercase tracking-[.16em] text-white mt-1">{$t('group.detail_skill')}</div>
        </div>
      {/if}

      <!-- A2. Foto (centro) -->
      <div class="absolute top-[62px] left-1/2 -translate-x-1/2 z-10">
        <div class="w-[144px] h-[144px] rounded-full p-[2px] bg-[linear-gradient(150deg,#f7e2ad,#8a6c2a_55%,#f5d585)] shadow-[0_10px_24px_rgba(0,0,0,.45)]">
          <AvatarImage
            {name}
            {avatarUrl}
            {updatedAt}
            size={140}
            onclick={onPhotoClick}
            clickLabel={photoLabel}
          />
        </div>
      </div>

      <!-- A3. Identidade (inferior) — nada abaixo do divisor inferior (limite y≈300) -->
      <div class="absolute top-[212px] inset-x-0 px-5 z-20 text-center">
        <div class="divider mb-2.5"></div>
        <div class="font-cond font-extrabold uppercase text-white text-[30px] leading-none tracking-[.01em] truncate">{displayNickname}</div>
        <div class="text-[12.5px] font-semibold text-white mt-1 truncate">{name}</div>
        {#if pos}
          <div class="mt-2 text-center">
            <div class="font-cond font-extrabold text-white text-[27px] leading-none">{POS_ABBR[pos]}</div>
            <div class="text-[12px] font-bold uppercase tracking-[.1em] text-white mt-0.5">{$t(POS_I18N_KEY[pos] as any)}</div>
          </div>
        {/if}
        <div class="divider mt-2.5"></div>
      </div>

    </div>
  </div>
</div>

<style>
  /* Moldura metálica (3px) */
  .crest-outer {
    width: 300px;
    height: 360px;
    clip-path: path('M14 0 H286 A14 14 0 0 1 300 14 V202 C300 272 228 328 150 360 C72 328 0 272 0 202 V14 A14 14 0 0 1 14 0 Z');
    background: linear-gradient(160deg, #f7e2ad 0%, #e0b95c 28%, #8a6c2a 55%, #f5d585 82%, #c9a34d 100%);
  }
  /* Campo verde */
  .crest-inner {
    position: absolute;
    top: 3px;
    left: 3px;
    width: 294px;
    height: 354px;
    overflow: hidden;
    clip-path: path('M13 0 H281 A13 13 0 0 1 294 13 V198 C294 266 224 322 147 354 C70 322 0 266 0 198 V13 A13 13 0 0 1 13 0 Z');
    background: linear-gradient(158deg, #15422a 0%, #1f9a4d 34%, #0d5028 64%, #07331a 100%);
  }
  /* brilho superior */
  .crest-inner::before {
    content: '';
    position: absolute;
    inset: 0;
    background: radial-gradient(115% 60% at 50% 0%, rgba(255, 255, 255, 0.2), transparent 62%);
    pointer-events: none;
  }
  /* textura diagonal */
  .crest-inner::after {
    content: '';
    position: absolute;
    inset: 0;
    background: repeating-linear-gradient(118deg, rgba(255, 255, 255, 0.055) 0 2px, transparent 2px 10px);
    mix-blend-mode: overlay;
    pointer-events: none;
  }
  .divider {
    height: 1px;
    background: linear-gradient(90deg, transparent, rgba(245, 213, 133, 0.6), transparent);
  }
</style>
