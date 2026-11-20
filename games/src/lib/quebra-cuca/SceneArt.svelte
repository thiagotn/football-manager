<script lang="ts">
  /**
   * O desenho de uma cena: céu em degradê, chão com morrinhos e os desenhos
   * Twemoji. Renderiza só o conteúdo (`<g>`) — quem usa coloca dentro de um
   * `<svg viewBox="0 0 400 300">` ou de um `<defs>` para reaproveitar com `<use>`.
   */
  import { SCENE_H, SCENE_W, type Scene } from './scenes';
  import { imgUrl } from './art';

  let { scene, uid = '' }: { scene: Scene; uid?: string } = $props();

  const sky = $derived(`qc-sky-${scene.id}${uid}`);
  const g = $derived(scene.ground);
  /** Horizonte com morrinhos suaves e uma faixa mais escura na frente. */
  const hills = $derived(
    g
      ? `M0 ${g.y} C 70 ${g.y - 16}, 150 ${g.y + 10}, 220 ${g.y - 4} S 340 ${g.y - 14}, ${SCENE_W} ${g.y + 2} V${SCENE_H} H0 Z`
      : ''
  );
</script>

<g>
  <defs>
    <linearGradient id={sky} x1="0" y1="0" x2="0" y2={SCENE_H} gradientUnits="userSpaceOnUse">
      <stop offset="0" stop-color={scene.sky[0]} />
      <stop offset="1" stop-color={scene.sky[1]} />
    </linearGradient>
  </defs>
  <rect width={SCENE_W} height={SCENE_H} fill="url(#{sky})" />
  {#if g}
    <path d={hills} fill={g.color} />
    <path d={hills} fill="rgba(0,0,0,.08)" transform="translate(0 {(SCENE_H - g.y) * 0.45})" />
  {/if}
  {#each scene.items as item, i (i)}
    <image
      href={imgUrl(item.img)}
      x={item.x - item.s / 2}
      y={item.y - item.s / 2}
      width={item.s}
      height={item.s}
      transform={item.flip ? `translate(${2 * item.x} 0) scale(-1 1)` : undefined}
    />
  {/each}
</g>
