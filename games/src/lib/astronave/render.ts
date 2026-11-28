/**
 * Desenho da Astronave em canvas 2D.
 *
 * Não conhece as regras: recebe o estado do engine e pinta. Tudo é vetorial e
 * desenhado em unidades do campo (`W × H`), com uma única escala aplicada ao
 * contexto — trocar o tamanho da tela não muda nada aqui.
 *
 * As estrelas e as faíscas das explosões são enfeite e moram só neste arquivo
 * (podem usar `Math.random`: não entram no estado do jogo).
 */

import { ENEMY_R, H, SHIP_Y, W, type Enemy, type Game, type GameEvent, type Meteor } from './engine';
import {
  SHIP_COCKPIT,
  SHIP_FLAME,
  SHIP_HULL,
  SHIP_SCALE,
  SHIP_WINGS,
  type Skin,
} from './skins';

const SPACE_TOP = '#070b1d';
const SPACE_BOTTOM = '#111a3d';
const ROCK = '#8a7767';
const ROCK_DARK = '#6b5b4e';
const ROCK_EDGE = 'rgba(255, 255, 255, 0.16)';
const BOLT = '#ff6b5c';
const BOLT_CORE = '#ffe0d6';
const SCOUT = '#2fc79a';
const SCOUT_DOME = '#b9fff0';
const GUNNER = '#e8473b';
const GUNNER_PLATE = '#ffb020';

const DROP_COLOR = { shield: '#5ee1ff', double: '#ffc93c', life: '#ff7aa8' } as const;

export type DrawOptions = {
  skin: Skin;
  /** Relógio para as animações (ms). */
  time: number;
  /** `false` com `prefers-reduced-motion`: sem estrelas correndo, sem faíscas. */
  motion: boolean;
};

export type Renderer = {
  /** Define quantos px de CSS vale uma unidade do campo. */
  resize: (scale: number) => void;
  draw: (game: Game, options: DrawOptions) => void;
  /** Eventos do engine viram faíscas. */
  fx: (events: GameEvent[]) => void;
  dispose: () => void;
};

type Star = { x: number; y: number; depth: number };
type Spark = { x: number; y: number; vx: number; vy: number; life: number; max: number; color: string; size: number };

/** Ruído estável por (id, i): o meteoro tem sempre a mesma cara. */
function hash(id: number, i: number): number {
  let h = Math.imul(id * 374761393 + i * 668265263, 1274126177);
  h ^= h >>> 15;
  return ((h >>> 0) % 1000) / 1000;
}

export function createRenderer(canvas: HTMLCanvasElement): Renderer {
  const context = canvas.getContext('2d');
  if (!context) throw new Error('canvas 2d indisponível');
  // Reatribuído para o tipo perder o `null`: as funções abaixo rodam depois.
  const ctx: CanvasRenderingContext2D = context;

  let scale = 3;
  let dpr = 1;
  let lastTime = 0;
  let sparks: Spark[] = [];

  const stars: Star[] = [];
  for (let i = 0; i < 70; i++) {
    stars.push({ x: Math.random() * W, y: Math.random() * H, depth: 0.25 + Math.random() * 0.75 });
  }

  const hull = new Path2D(SHIP_HULL);
  const wings = new Path2D(SHIP_WINGS);
  const flame = new Path2D(SHIP_FLAME);

  function drawSpace(time: number, motion: boolean): void {
    const g = ctx.createLinearGradient(0, 0, 0, H);
    g.addColorStop(0, SPACE_TOP);
    g.addColorStop(1, SPACE_BOTTOM);
    ctx.fillStyle = g;
    ctx.fillRect(0, 0, W, H);
    const drift = motion ? time / 1000 : 0;
    for (const s of stars) {
      const y = (s.y + drift * 14 * s.depth) % H;
      ctx.globalAlpha = 0.25 + s.depth * 0.6;
      ctx.fillStyle = '#ffffff';
      const size = 0.25 + s.depth * 0.45;
      ctx.fillRect(s.x, y, size, size);
    }
    ctx.globalAlpha = 1;
  }

  function drawMeteor(m: Meteor, time: number, motion: boolean): void {
    ctx.save();
    ctx.translate(m.x, m.y);
    if (motion) ctx.rotate((time / 1000) * (hash(m.id, 99) - 0.5) * 1.6);
    const sides = 9;
    ctx.beginPath();
    for (let i = 0; i < sides; i++) {
      const a = (i / sides) * Math.PI * 2;
      const r = m.r * (0.8 + hash(m.id, i) * 0.26);
      if (i === 0) ctx.moveTo(Math.cos(a) * r, Math.sin(a) * r);
      else ctx.lineTo(Math.cos(a) * r, Math.sin(a) * r);
    }
    ctx.closePath();
    ctx.fillStyle = ROCK;
    ctx.fill();
    ctx.lineWidth = 0.35;
    ctx.strokeStyle = ROCK_EDGE;
    ctx.stroke();
    // Crateras; o meteoro grande ganha uma por tiro que já levou.
    ctx.fillStyle = ROCK_DARK;
    const craters = m.big ? 2 + (3 - m.hp) : 1;
    for (let i = 0; i < craters; i++) {
      const a = hash(m.id, 20 + i) * Math.PI * 2;
      const d = m.r * 0.42 * hash(m.id, 30 + i);
      ctx.beginPath();
      ctx.arc(Math.cos(a) * d, Math.sin(a) * d, m.r * (0.16 + hash(m.id, 40 + i) * 0.12), 0, Math.PI * 2);
      ctx.fill();
    }
    ctx.restore();
  }

  function drawEnemy(e: Enemy, time: number): void {
    ctx.save();
    ctx.translate(e.x, e.y);
    if (e.kind === 'scout') {
      // Batedor: um disco com cúpula.
      ctx.fillStyle = SCOUT_DOME;
      ctx.beginPath();
      ctx.ellipse(0, -1.4, 2.4, 2.4, 0, 0, Math.PI * 2);
      ctx.fill();
      ctx.fillStyle = SCOUT;
      ctx.beginPath();
      ctx.ellipse(0, 0.6, ENEMY_R, 2.1, 0, 0, Math.PI * 2);
      ctx.fill();
      const lit = Math.floor(time / 220) % 3;
      for (let i = 0; i < 3; i++) {
        ctx.fillStyle = i === lit ? '#fff7c2' : '#0d3b34';
        ctx.beginPath();
        ctx.arc((i - 1) * 2.6, 0.9, 0.55, 0, Math.PI * 2);
        ctx.fill();
      }
    } else {
      // Artilheiro: casco anguloso com canhão; as placas somem a cada tiro levado.
      ctx.fillStyle = GUNNER;
      ctx.beginPath();
      ctx.moveTo(-ENEMY_R, -1.5);
      ctx.lineTo(-2.4, -3.6);
      ctx.lineTo(2.4, -3.6);
      ctx.lineTo(ENEMY_R, -1.5);
      ctx.lineTo(3, 2.6);
      ctx.lineTo(-3, 2.6);
      ctx.closePath();
      ctx.fill();
      ctx.fillStyle = '#3a0a06';
      ctx.fillRect(-0.8, 2, 1.6, 3);
      ctx.fillStyle = GUNNER_PLATE;
      for (let i = 0; i < e.hp; i++) ctx.fillRect(-2.7 + i * 2, -2.4, 1.4, 1.4);
    }
    ctx.restore();
  }

  function drawShip(game: Game, skin: Skin, time: number, motion: boolean): void {
    const ship = game.ship;
    // Pisca enquanto está invulnerável.
    if (ship.invuln > 0 && Math.floor(time / 90) % 2 === 0) return;
    ctx.save();
    ctx.translate(ship.x, SHIP_Y);
    ctx.scale(SHIP_SCALE, SHIP_SCALE);
    ctx.save();
    if (motion) ctx.scale(1, 0.75 + 0.35 * Math.abs(Math.sin(time / 60)));
    ctx.fillStyle = skin.flame;
    ctx.fill(flame);
    ctx.restore();
    ctx.fillStyle = skin.wing;
    ctx.fill(wings);
    ctx.fillStyle = skin.hull;
    ctx.fill(hull);
    ctx.fillStyle = skin.cockpit;
    ctx.beginPath();
    ctx.ellipse(SHIP_COCKPIT.cx, SHIP_COCKPIT.cy, SHIP_COCKPIT.rx, SHIP_COCKPIT.ry, 0, 0, Math.PI * 2);
    ctx.fill();
    ctx.restore();
    if (ship.shield) {
      ctx.lineWidth = 0.5;
      ctx.strokeStyle = DROP_COLOR.shield;
      ctx.globalAlpha = 0.55 + (motion ? 0.3 * Math.sin(time / 180) : 0);
      ctx.beginPath();
      ctx.arc(ship.x, SHIP_Y + 0.5, 8, 0, Math.PI * 2);
      ctx.stroke();
      ctx.globalAlpha = 1;
    }
  }

  function drawDrop(x: number, y: number, kind: keyof typeof DROP_COLOR, time: number, motion: boolean): void {
    const pulse = motion ? 1 + 0.12 * Math.sin(time / 160) : 1;
    ctx.save();
    ctx.translate(x, y);
    ctx.scale(pulse, pulse);
    ctx.fillStyle = 'rgba(7, 11, 29, 0.85)';
    ctx.strokeStyle = DROP_COLOR[kind];
    ctx.lineWidth = 0.55;
    ctx.beginPath();
    ctx.arc(0, 0, 3, 0, Math.PI * 2);
    ctx.fill();
    ctx.stroke();
    ctx.fillStyle = DROP_COLOR[kind];
    if (kind === 'shield') {
      ctx.beginPath();
      ctx.moveTo(0, -1.7);
      ctx.lineTo(1.5, -1);
      ctx.lineTo(1.2, 0.8);
      ctx.lineTo(0, 1.8);
      ctx.lineTo(-1.2, 0.8);
      ctx.lineTo(-1.5, -1);
      ctx.closePath();
      ctx.fill();
    } else if (kind === 'double') {
      ctx.fillRect(-1.3, -1.6, 0.9, 3.2);
      ctx.fillRect(0.4, -1.6, 0.9, 3.2);
    } else {
      // Coração: dois círculos e um triângulo.
      ctx.beginPath();
      ctx.arc(-0.75, -0.5, 0.9, 0, Math.PI * 2);
      ctx.arc(0.75, -0.5, 0.9, 0, Math.PI * 2);
      ctx.fill();
      ctx.beginPath();
      ctx.moveTo(-1.6, -0.2);
      ctx.lineTo(1.6, -0.2);
      ctx.lineTo(0, 1.9);
      ctx.closePath();
      ctx.fill();
    }
    ctx.restore();
  }

  function burst(x: number, y: number, count: number, speed: number, colors: string[], life = 0.5): void {
    for (let i = 0; i < count; i++) {
      const a = Math.random() * Math.PI * 2;
      const v = speed * (0.35 + Math.random() * 0.65);
      const max = life * (0.6 + Math.random() * 0.6);
      sparks.push({
        x,
        y,
        vx: Math.cos(a) * v,
        vy: Math.sin(a) * v,
        life: max,
        max,
        color: colors[i % colors.length],
        size: 0.5 + Math.random() * 0.8,
      });
    }
    // Teto: uma tela cheia de explosões não pode virar gargalo.
    if (sparks.length > 260) sparks = sparks.slice(-260);
  }

  function drawSparks(dt: number): void {
    for (const s of sparks) {
      s.life -= dt;
      s.x += s.vx * dt;
      s.y += s.vy * dt;
      if (s.life <= 0) continue;
      ctx.globalAlpha = Math.min(1, s.life / s.max);
      ctx.fillStyle = s.color;
      ctx.fillRect(s.x - s.size / 2, s.y - s.size / 2, s.size, s.size);
    }
    ctx.globalAlpha = 1;
    sparks = sparks.filter((s) => s.life > 0);
  }

  return {
    resize(nextScale) {
      scale = Math.max(1, nextScale);
      dpr = Math.min(window.devicePixelRatio || 1, 2);
      const w = Math.floor(W * scale);
      const h = Math.floor(H * scale);
      canvas.width = Math.round(w * dpr);
      canvas.height = Math.round(h * dpr);
      canvas.style.width = `${w}px`;
      canvas.style.height = `${h}px`;
    },

    draw(game, { skin, time, motion }) {
      const dt = Math.min(0.05, Math.max(0, (time - lastTime) / 1000));
      lastTime = time;
      ctx.setTransform(canvas.width / W, 0, 0, canvas.height / H, 0, 0);

      drawSpace(time, motion);
      for (const d of game.drops) drawDrop(d.x, d.y, d.kind, time, motion);
      for (const m of game.meteors) drawMeteor(m, time, motion);
      for (const e of game.enemies) drawEnemy(e, time);

      ctx.fillStyle = skin.shot;
      for (const s of game.shots) {
        ctx.beginPath();
        ctx.roundRect(s.x - 0.5, s.y - 2.2, 1, 4.4, 0.5);
        ctx.fill();
      }
      for (const b of game.bolts) {
        ctx.fillStyle = BOLT;
        ctx.beginPath();
        ctx.arc(b.x, b.y, 1.5, 0, Math.PI * 2);
        ctx.fill();
        ctx.fillStyle = BOLT_CORE;
        ctx.beginPath();
        ctx.arc(b.x, b.y, 0.6, 0, Math.PI * 2);
        ctx.fill();
      }

      if (!game.over) drawShip(game, skin, time, motion);
      if (motion) drawSparks(dt);
      else sparks = [];
    },

    fx(events) {
      for (const e of events) {
        if (e.type === 'boom') {
          const colors =
            e.kind === 'meteor'
              ? [ROCK, '#c9b8a8', '#ffd9a0']
              : e.kind === 'scout'
                ? [SCOUT, SCOUT_DOME, '#ffffff']
                : [GUNNER, GUNNER_PLATE, '#ffffff'];
          burst(e.x, e.y, 8 + Math.round(e.size * 1.6), 34, colors);
        } else if (e.type === 'hit') {
          burst(e.x, e.y, 14, 46, e.shielded ? [DROP_COLOR.shield, '#ffffff'] : [BOLT, '#ffffff', '#ffc93c']);
        } else if (e.type === 'died') {
          burst(e.x, e.y, 46, 70, ['#ffffff', '#ffc93c', BOLT, '#4f7dff'], 1.1);
        } else if (e.type === 'power') {
          burst(e.x, e.y, 10, 26, [DROP_COLOR[e.kind], '#ffffff'], 0.4);
        }
      }
    },

    dispose() {
      sparks = [];
      ctx.setTransform(1, 0, 0, 1, 0, 0);
      ctx.clearRect(0, 0, canvas.width, canvas.height);
    },
  };
}
