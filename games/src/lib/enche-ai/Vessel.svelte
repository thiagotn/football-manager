<script lang="ts">
  /**
   * Recipiente em SVG: vidro + camadas (cor + textura) + finalização quando
   * pronto. `growFrom` anima as camadas a partir desse índice (despejo).
   */
  import { CAPACITY, isComplete } from './engine';
  import { FLAVORS, VIEWBOX_HEIGHT, VIEWBOX_TOP, type Vessel } from './content';

  let {
    vessel,
    layers,
    growFrom = null,
    growKey = 0,
    topping = true,
  }: {
    vessel: Vessel;
    layers: string[];
    growFrom?: number | null;
    growKey?: number;
    topping?: boolean;
  } = $props();

  const h = $derived((vessel.bottom - vessel.top) / CAPACITY);
  const rows = $derived(
    layers.map((id, i) => ({
      id,
      i,
      flavor: FLAVORS[id],
      y: vessel.bottom - (i + 1) * h - 0.4,
    }))
  );
  const settled = $derived(growFrom == null ? rows : rows.filter((r) => r.i < growFrom));
  const grown = $derived(growFrom == null ? [] : rows.filter((r) => r.i >= growFrom));
  const done = $derived(isComplete(layers));
  const finish = $derived(done && topping ? FLAVORS[layers[0]] : null);
  const m = $derived(vessel.mouth);
  const cx = $derived(vessel.cx);
</script>

<svg viewBox="0 {VIEWBOX_TOP} 100 {VIEWBOX_HEIGHT}" aria-hidden="true" class="block h-auto w-full overflow-visible">
  <path d={vessel.outline} fill="rgba(200,220,255,.07)" />

  {#if finish?.topping === 'straw'}
    <g class="ea-fade">
      <line x1={cx + 8} y1={vessel.top + 26} x2={cx + 24} y2={m - 30} stroke="#fff" stroke-width="7" stroke-linecap="round" />
      <line x1={cx + 8} y1={vessel.top + 26} x2={cx + 24} y2={m - 30} stroke="#e8473b" stroke-width="7" stroke-dasharray="5 6" />
    </g>
  {/if}

  <g clip-path="url(#ea-clip-{vessel.id})">
    {#each settled as r (r.i)}
      <rect x="0" y={r.y} width="100" height={h + 0.8} fill={r.flavor.color} />
      <rect class="ea-pat" x="0" y={r.y} width="100" height={h + 0.8} fill="url(#ea-p-{r.flavor.pattern})" />
    {/each}
    {#if grown.length}
      {#key growKey}
        <g class="ea-grow">
          {#each grown as r (r.i)}
            <rect x="0" y={r.y} width="100" height={h + 0.8} fill={r.flavor.color} />
            <rect class="ea-pat" x="0" y={r.y} width="100" height={h + 0.8} fill="url(#ea-p-{r.flavor.pattern})" />
          {/each}
        </g>
      {/key}
    {/if}
    {#if layers.length}
      <rect x="0" y={vessel.bottom - layers.length * h - 0.4} width="100" height="2.2" fill="rgba(255,255,255,.3)" />
    {/if}
    <path
      d="M{cx - 26} {m + 14}V{Math.min(vessel.bottom - 20, m + 90)}"
      stroke="rgba(255,255,255,.22)"
      stroke-width="5"
      stroke-linecap="round"
      fill="none"
    />
  </g>

  <path d={vessel.outline} fill="none" stroke="rgba(228,238,255,.78)" stroke-width="3.5" stroke-linejoin="round" />
  {#each vessel.details as d, i (i)}
    <path d={d.d} fill="none" stroke="rgba(228,238,255,{d.opacity})" stroke-width={d.width} stroke-linecap="round" />
  {/each}

  {#if finish?.topping === 'steam'}
    <g class="ea-steam" fill="none" stroke="rgba(255,255,255,.75)" stroke-width="3.2" stroke-linecap="round">
      <path d="M{cx - 13} {m - 6}q-6 -7 0 -13t0 -13" />
      <path d="M{cx} {m - 8}q-6 -7 0 -13t0 -13" />
      <path d="M{cx + 13} {m - 6}q-6 -7 0 -13t0 -13" />
    </g>
  {:else if finish?.topping === 'straw'}
    <g class="ea-fade">
      <circle cx={cx - 30} cy={m + 2} r="11" fill="#fff" />
      <circle cx={cx - 30} cy={m + 2} r="8.5" fill={finish.color} />
      <path
        d="M{cx - 30} {m + 2}l0 -8.5M{cx - 30} {m + 2}l7.4 4.2M{cx - 30} {m + 2}l-7.4 4.2"
        stroke="rgba(255,255,255,.7)"
        stroke-width="1.2"
      />
    </g>
  {:else if finish?.topping === 'cream'}
    <g class="ea-fade">
      <g fill="#fff8ef" stroke="rgba(120,90,70,.25)" stroke-width="1">
        <circle cx={cx - 22} cy={m - 2} r="11" />
        <circle cx={cx + 22} cy={m - 2} r="11" />
        <circle cx={cx} cy={m - 8} r="14" />
        <circle cx={cx - 10} cy={m - 17} r="9" />
        <circle cx={cx + 10} cy={m - 17} r="9" />
      </g>
      <path d="M{cx + 2} {m - 30}q4 -10 12 -12" stroke="#3f6b2a" stroke-width="2" fill="none" stroke-linecap="round" />
      <circle cx={cx + 2} cy={m - 27} r="6.5" fill="#d7263d" />
      <circle cx={cx} cy={m - 29} r="1.8" fill="rgba(255,255,255,.7)" />
    </g>
  {/if}
</svg>

<style>
  .ea-grow {
    transform-box: fill-box;
    transform-origin: 50% 100%;
    animation: ea-grow 0.42s ease-out both;
  }
  .ea-fade {
    animation: ea-fade 0.35s ease-out both;
  }
  .ea-steam path {
    animation: ea-steam 2.4s ease-in-out infinite;
  }
  .ea-steam path:nth-child(2) {
    animation-delay: 0.5s;
  }
  .ea-steam path:nth-child(3) {
    animation-delay: 1s;
  }
  :global(.ea-no-pat) .ea-pat {
    display: none;
  }
  @keyframes ea-grow {
    from { transform: scaleY(0); }
    to { transform: scaleY(1); }
  }
  @keyframes ea-fade {
    from { opacity: 0; transform: translateY(-6px); }
    to { opacity: 1; transform: none; }
  }
  @keyframes ea-steam {
    0% { opacity: 0; transform: translateY(6px); }
    40% { opacity: 0.85; }
    100% { opacity: 0; transform: translateY(-10px); }
  }
</style>
