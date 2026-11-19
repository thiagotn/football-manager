/**
 * Tetris — gestos de um dedo, sem botões na tela. Puro: recebe posições e
 * tempos, devolve ações; o DOM fica na rota.
 *
 * - **toque** (sem arrastar, curto) → girar
 * - **arrastar para o lado** → uma coluna a cada `cellPx` de arraste
 * - **arrastar para baixo devagar** → desce uma linha a cada `cellPx`
 * - **deslizar rápido para baixo** e soltar → solta a peça de uma vez
 *
 * O gesto **trava num eixo** assim que passa de `LOCK_PX`: quem arrasta para
 * o lado não desce a peça sem querer, e vice-versa. Arrastar para cima não
 * faz nada (no Tetris não há como subir a peça).
 */

export type GestureAction = 'left' | 'right' | 'down' | 'rotate' | 'drop';

/** Distância a partir da qual o toque vira arraste e escolhe o eixo. */
export const LOCK_PX = 10;
/** Toque mais longo que isso, sem arrastar, não gira (o dedo só descansou). */
export const TAP_MS = 280;
/** Velocidade para baixo (px/ms) no fim do gesto que conta como "jogar a peça". */
export const FLICK_PX_PER_MS = 0.9;
/** Janela usada para medir a velocidade final do gesto. */
const FLICK_WINDOW_MS = 90;
/**
 * Pausa máxima entre o último movimento e o dedo sair da tela para ainda
 * contar como flick. O `pointerup` chega um pouco depois do último
 * `pointermove`, no mesmo lugar — a velocidade se mede só nos movimentos.
 */
const FLICK_REST_MS = 120;

export type Gesture = {
  move: (x: number, y: number, t: number) => GestureAction[];
  end: (x: number, y: number, t: number) => GestureAction[];
};

export function createGesture(cellPx: number, x0: number, y0: number, t0: number): Gesture {
  const cell = Math.max(8, cellPx);
  let axis: 'x' | 'y' | null = null;
  let anchorX = x0;
  let anchorY = y0;
  let lastX = x0;
  let lastY = y0;
  const samples: { y: number; t: number }[] = [{ y: y0, t: t0 }];

  function track(y: number, t: number) {
    samples.push({ y, t });
    while (samples.length > 2 && t - samples[0].t > FLICK_WINDOW_MS) samples.shift();
  }

  function move(x: number, y: number, t: number): GestureAction[] {
    lastX = x;
    lastY = y;
    track(y, t);
    if (!axis) {
      const dx = x - x0;
      const dy = y - y0;
      if (Math.max(Math.abs(dx), Math.abs(dy)) < LOCK_PX) return [];
      axis = Math.abs(dx) > Math.abs(dy) ? 'x' : 'y';
    }
    const out: GestureAction[] = [];
    if (axis === 'x') {
      let steps = Math.trunc((x - anchorX) / cell);
      anchorX += steps * cell;
      const action = steps > 0 ? 'right' : 'left';
      for (steps = Math.abs(steps); steps > 0; steps--) out.push(action);
    } else {
      const steps = Math.trunc((y - anchorY) / cell);
      if (steps > 0) {
        anchorY += steps * cell;
        for (let i = 0; i < steps; i++) out.push('down');
      } else if (y < anchorY) {
        // Voltou para cima: a âncora acompanha, para não "devolver" linhas.
        anchorY = y;
      }
    }
    return out;
  }

  function end(x: number, y: number, t: number): GestureAction[] {
    // Soltar no mesmo lugar do último movimento não é movimento.
    const out = x !== lastX || y !== lastY ? move(x, y, t) : [];
    if (!axis) return t - t0 <= TAP_MS ? [...out, 'rotate'] : out;
    if (axis === 'y') {
      const last = samples[samples.length - 1];
      const first = samples[0];
      const speed = (last.y - first.y) / Math.max(1, last.t - first.t);
      const rested = t - last.t > FLICK_REST_MS;
      if (!rested && speed >= FLICK_PX_PER_MS && y - y0 >= cell) out.push('drop');
    }
    return out;
  }

  return { move, end };
}
