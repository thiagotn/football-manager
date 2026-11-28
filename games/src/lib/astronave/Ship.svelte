<script lang="ts">
  /**
   * A nave em SVG, com as cores de um visual. É o mesmo contorno que o canvas
   * desenha na partida — aqui serve à aba Naves e ao emblema.
   */
  import { SHIP_COCKPIT, SHIP_FLAME, SHIP_HULL, SHIP_WINGS, skinOf, type Skin } from './skins';

  let { skin = skinOf('pioneira'), size = 48 }: { skin?: Skin; size?: number } = $props();
</script>

<svg width={size} height={size} viewBox="-11 -11 22 24" aria-hidden="true">
  <path d={SHIP_FLAME} fill={skin.flame} />
  <path d={SHIP_WINGS} fill={skin.wing} />
  <path d={SHIP_HULL} fill={skin.hull} />
  <ellipse {...SHIP_COCKPIT} fill={skin.cockpit} />
</svg>
