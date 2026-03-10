<script lang="ts">
  import type { Attendance } from '$lib/api';

  interface Props {
    eligiblePlayers: Attendance[];   // jogadores confirmados (exceto o próprio)
    onsubmit: (top5: { player_id: string; position: number }[], flop_player_id: string | null) => void;
    saving?: boolean;
  }

  let { eligiblePlayers, onsubmit, saving = false }: Props = $props();

  const POSITIONS_DEF = [
    { pos: 1, label: '🥇 1º lugar', pts: 10 },
    { pos: 2, label: '🥈 2º lugar', pts: 8  },
    { pos: 3, label: '🥉 3º lugar', pts: 6  },
    { pos: 4, label: '    4º lugar', pts: 4  },
    { pos: 5, label: '    5º lugar', pts: 2  },
  ];

  // Apenas posições onde há jogadores suficientes (todas obrigatórias se exibidas)
  let visiblePositions = $derived(POSITIONS_DEF.filter(p => p.pos <= eligiblePlayers.length));

  // Decepção: exibida e obrigatória somente quando sobra ao menos 1 jogador após o top5
  // min(eligible, 5) posições usadas → sobra se eligible > 5
  let flopRequired = $derived(eligiblePlayers.length > 5);

  let selections = $state<Record<number, string>>({});  // position -> player_id
  let flop = $state('');

  // IDs já selecionados no top5
  let usedInTop5 = $derived(new Set(Object.values(selections).filter(Boolean)));
  // IDs indisponíveis para cada dropdown: os demais top5 + o da decepção
  function unavailable(pos: number): Set<string> {
    const used = new Set<string>();
    for (const [p, id] of Object.entries(selections)) {
      if (Number(p) !== pos && id) used.add(id);
    }
    if (flop) used.add(flop);
    return used;
  }
  // IDs indisponíveis para decepção: qualquer top5
  let flopUnavailable = $derived(usedInTop5);

  function handleSelect(pos: number, value: string) {
    selections = { ...selections, [pos]: value };
  }
  function handleFlop(value: string) {
    flop = value;
  }

  function handleSubmit() {
    const top5 = visiblePositions
      .filter(p => selections[p.pos])
      .map(p => ({ player_id: selections[p.pos], position: p.pos }));
    onsubmit(top5, flop || null);
  }

  let allPositionsFilled = $derived(visiblePositions.every(p => !!selections[p.pos]));
  let canSubmit = $derived(allPositionsFilled && (!flopRequired || !!flop) && !saving);
</script>

<div class="space-y-4">
  <div>
    <p class="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-3">🏆 Top {visiblePositions.length} — Melhores da partida</p>
    <div class="space-y-2">
      {#each visiblePositions as { pos, label, pts }}
        <div class="flex items-center gap-2">
          <span class="text-sm w-28 shrink-0 text-gray-700 dark:text-gray-300 font-medium">{label}</span>
          <span class="text-xs text-gray-400 shrink-0 w-12">{pts} pts</span>
          <select
            class="input text-sm flex-1"
            value={selections[pos] ?? ''}
            onchange={(e) => handleSelect(pos, (e.target as HTMLSelectElement).value)}
            disabled={saving}
          >
            <option value="">Selecionar *</option>
            {#each eligiblePlayers as a}
              {#if !unavailable(pos).has(a.player.id)}
                <option value={a.player.id}>{a.player.nickname || a.player.name}</option>
              {/if}
            {/each}
          </select>
        </div>
      {/each}
    </div>
  </div>

  {#if flopRequired}
    <div>
      <p class="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">😬 Decepção do jogo *</p>
      <select
        class="input text-sm w-full"
        value={flop}
        onchange={(e) => handleFlop((e.target as HTMLSelectElement).value)}
        disabled={saving}
      >
        <option value="">Selecionar *</option>
        {#each eligiblePlayers as a}
          {#if !flopUnavailable.has(a.player.id)}
            <option value={a.player.id}>{a.player.nickname || a.player.name}</option>
          {/if}
        {/each}
      </select>
    </div>
  {/if}

  <button
    type="button"
    class="btn-primary w-full justify-center py-2.5"
    onclick={handleSubmit}
    disabled={!canSubmit}
  >
    {saving ? 'Enviando…' : 'Enviar voto'}
  </button>
  <p class="text-xs text-gray-400 text-center">* Todos os campos são obrigatórios.</p>
</div>
