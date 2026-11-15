<script lang="ts">
  /**
   * A imagem da palavra: desenho Twemoji, foto da família (blob URL) ou, para
   * palavra da família sem foto, a primeira letra num cartão colorido.
   */
  import { imgUrl } from './art';
  import type { PlayWord } from './letters';

  let { word, photoUrl = null, size = 168 }: { word: PlayWord; photoUrl?: string | null; size?: number } = $props();
</script>

<div class="pic" style="--size:{size}px">
  {#if word.family && photoUrl}
    <img src={photoUrl} alt="" class="photo" draggable="false" />
  {:else if word.img}
    <img src={imgUrl(word.img)} alt="" draggable="false" />
  {:else}
    <span class="initial">{[...word.text][0]}</span>
  {/if}
</div>

<style>
  .pic {
    display: grid;
    place-items: center;
    width: min(var(--size), 46vw);
    aspect-ratio: 1;
  }
  img {
    width: 100%;
    height: 100%;
    object-fit: contain;
    user-select: none;
    -webkit-user-drag: none;
  }
  .photo {
    object-fit: cover;
    border-radius: 24px;
    border: 5px solid #fff;
    box-shadow: 0 6px 0 rgba(36, 52, 90, 0.15);
  }
  .initial {
    display: grid;
    place-items: center;
    width: 100%;
    height: 100%;
    border-radius: 28px;
    background: linear-gradient(160deg, #ff5d8f, #7b5cff);
    color: #fff;
    font-family: theme('fontFamily.display');
    font-size: calc(min(var(--size), 46vw) * 0.6);
    line-height: 1;
  }
</style>
