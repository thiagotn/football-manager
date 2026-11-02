<script lang="ts">
  /**
   * Definições SVG compartilhadas: texturas dos sabores (`ea-p-<pattern>`) e
   * áreas internas dos recipientes (`ea-clip-<vessel>`). Renderizar uma vez por
   * página, antes de qualquer `<Vessel>`.
   */
  import { VESSELS } from './content';
</script>

<svg width="0" height="0" style="position:absolute" aria-hidden="true">
  <defs>
    <pattern id="ea-p-beans" width="16" height="14" patternUnits="userSpaceOnUse"><ellipse cx="4" cy="4" rx="2.6" ry="1.6" transform="rotate(-30 4 4)" fill="rgba(255,226,190,.32)" /><ellipse cx="12" cy="11" rx="2.6" ry="1.6" transform="rotate(25 12 11)" fill="rgba(255,226,190,.32)" /></pattern>
    <pattern id="ea-p-plain" width="10" height="10" patternUnits="userSpaceOnUse"><rect width="10" height="10" fill="none" /></pattern>
    <pattern id="ea-p-pulp" width="14" height="12" patternUnits="userSpaceOnUse"><path d="M2 3h5M9 9h4" stroke="rgba(255,244,214,.55)" stroke-width="1.6" stroke-linecap="round" /></pattern>
    <pattern id="ea-p-granola" width="14" height="14" patternUnits="userSpaceOnUse"><rect x="2" y="3" width="3.4" height="2.2" rx=".6" transform="rotate(20 3.7 4.1)" fill="rgba(255,214,150,.45)" /><rect x="9" y="9" width="3" height="2" rx=".6" transform="rotate(-25 10.5 10)" fill="rgba(255,214,150,.45)" /></pattern>
    <pattern id="ea-p-rings" width="16" height="16" patternUnits="userSpaceOnUse"><circle cx="5" cy="5" r="2.8" fill="none" stroke="rgba(255,255,255,.55)" stroke-width="1.2" /><circle cx="13" cy="13" r="2" fill="none" stroke="rgba(255,255,255,.45)" stroke-width="1.1" /></pattern>
    <pattern id="ea-p-seeds" width="12" height="12" patternUnits="userSpaceOnUse"><path d="M3 2.5l1 2-1 .6-1-.6z" fill="rgba(255,240,170,.9)" /><path d="M9 8.5l1 2-1 .6-1-.6z" fill="rgba(255,240,170,.9)" /></pattern>
    <pattern id="ea-p-pits" width="14" height="14" patternUnits="userSpaceOnUse"><ellipse cx="4" cy="4" rx="1.4" ry="2.2" fill="rgba(25,6,8,.7)" /><ellipse cx="11" cy="11" rx="1.4" ry="2.2" fill="rgba(25,6,8,.7)" /></pattern>
    <pattern id="ea-p-waves" width="20" height="10" patternUnits="userSpaceOnUse"><path d="M0 5q5-4 10 0t10 0" fill="none" stroke="rgba(255,236,214,.3)" stroke-width="1.4" /></pattern>
    <pattern id="ea-p-stripes" width="10" height="10" patternUnits="userSpaceOnUse"><path d="M-2 12L12 -2" stroke="rgba(255,255,255,.3)" stroke-width="1.5" /></pattern>
    <pattern id="ea-p-bubbles" width="18" height="18" patternUnits="userSpaceOnUse"><circle cx="5" cy="5" r="2.6" fill="none" stroke="rgba(255,255,255,.55)" stroke-width="1.1" /><circle cx="13" cy="12" r="1.6" fill="none" stroke="rgba(255,255,255,.5)" stroke-width="1" /><circle cx="7" cy="15" r="1" fill="rgba(255,255,255,.5)" /></pattern>
    <pattern id="ea-p-specks" width="10" height="10" patternUnits="userSpaceOnUse"><circle cx="3" cy="3" r="1.5" fill="rgba(48,24,0,.6)" /><circle cx="8" cy="8" r="1.5" fill="rgba(48,24,0,.6)" /></pattern>
    <pattern id="ea-p-ice" width="14" height="14" patternUnits="userSpaceOnUse"><path d="M4 1.5l2.5 2.5L4 6.5 1.5 4z" fill="rgba(255,255,255,.55)" /><path d="M11 8.5l2 2-2 2-2-2z" fill="rgba(255,255,255,.4)" /></pattern>
    {#each Object.values(VESSELS) as v (v.id)}
      <clipPath id="ea-clip-{v.id}"><path d={v.clip} /></clipPath>
    {/each}
    <clipPath id="ea-clip-dot"><circle cx="8" cy="8" r="7" /></clipPath>
  </defs>
</svg>
