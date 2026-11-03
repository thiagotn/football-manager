/**
 * Cena 3D do Tetris: poço, luzes, estrelas e os cubos das peças.
 *
 * Não conhece as regras — recebe tabuleiro e peça e desenha. Os materiais são
 * criados uma única vez (7 cores × 3 estados) e os cubos saem de um pool, em
 * vez de um material novo por bloco a cada quadro como no jogo original. Tudo
 * o que é alocado aqui é liberado em `dispose()`, porque numa rota SPA a cena
 * monta e desmonta várias vezes na mesma aba.
 */
import {
  AmbientLight,
  BoxGeometry,
  BufferAttribute,
  BufferGeometry,
  Color,
  DirectionalLight,
  EdgesGeometry,
  Fog,
  Group,
  LineBasicMaterial,
  LineSegments,
  Mesh,
  MeshStandardMaterial,
  PerspectiveCamera,
  Points,
  PointsMaterial,
  Scene,
  WebGLRenderer,
} from 'three';

import { BOARD_H, BOARD_W, TETROMINOES, TYPES, shapeOf, type Board, type Piece } from './engine';

/** Estado do cubo, que define brilho e transparência do material. */
type CellKind = 'active' | 'locked' | 'ghost';

const KIND_STYLE: Record<CellKind, { emissive: number; opacity: number; z: number }> = {
  active: { emissive: 0.62, opacity: 1, z: 0.12 },
  locked: { emissive: 0.19, opacity: 1, z: 0 },
  ghost: { emissive: 0, opacity: 0.17, z: 0.04 },
};

export type TetrisScene = {
  render: () => void;
  resize: (width: number, height: number) => void;
  /** Gira a câmera em torno do poço (radianos acumulados). */
  orbit: (dTheta: number, dPhi: number) => void;
  draw: (board: Board, piece: Piece | null, ghost: number | null) => void;
  dispose: () => void;
};

export function createScene(canvas: HTMLCanvasElement): TetrisScene {
  const renderer = new WebGLRenderer({ canvas, antialias: true });
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));

  const scene = new Scene();
  scene.background = new Color(0x010112);
  scene.fog = new Fog(0x010112, 38, 75);

  const camera = new PerspectiveCamera(62, 1, 0.1, 200);

  scene.add(new AmbientLight(0x6682bb, 2.4));
  const sun = new DirectionalLight(0xffffff, 2.6);
  sun.position.set(8, 25, 18);
  scene.add(sun);

  // ── Estrelas ──────────────────────────────────────────────────────────
  const starGeo = new BufferGeometry();
  const stars = new Float32Array(800 * 3);
  for (let i = 0; i < stars.length; i++) stars[i] = (Math.random() - 0.5) * 120;
  starGeo.setAttribute('position', new BufferAttribute(stars, 3));
  const starMat = new PointsMaterial({ color: 0xffffff, size: 0.12, transparent: true, opacity: 0.5 });
  scene.add(new Points(starGeo, starMat));

  // ── Poço ──────────────────────────────────────────────────────────────
  const well = new Group();
  well.position.set(-BOARD_W / 2, -BOARD_H / 2, 0);
  scene.add(well);

  const gridPoints: number[] = [];
  for (let x = 0; x <= BOARD_W; x++) gridPoints.push(x, 0, 0, x, BOARD_H, 0);
  for (let y = 0; y <= BOARD_H; y++) gridPoints.push(0, y, 0, BOARD_W, y, 0);
  const gridGeo = new BufferGeometry();
  gridGeo.setAttribute('position', new BufferAttribute(new Float32Array(gridPoints), 3));
  const gridMat = new LineBasicMaterial({ color: 0x0c1838, transparent: true, opacity: 0.55 });
  well.add(new LineSegments(gridGeo, gridMat));

  const floorGeo = new BoxGeometry(BOARD_W + 0.2, 0.1, 0.8);
  const floorMat = new MeshStandardMaterial({ color: 0x0a1530, metalness: 0.6, roughness: 0.7 });
  const floor = new Mesh(floorGeo, floorMat);
  floor.position.set(BOARD_W / 2, -0.05, 0);
  well.add(floor);

  const frameBox = new BoxGeometry(BOARD_W, BOARD_H, 0.45);
  const frameGeo = new EdgesGeometry(frameBox);
  const frameMat = new LineBasicMaterial({ color: 0x1a4090, transparent: true, opacity: 0.4 });
  const frame = new LineSegments(frameGeo, frameMat);
  frame.position.set(BOARD_W / 2, BOARD_H / 2, 0);
  well.add(frame);
  frameBox.dispose();

  const wallGeo = new BoxGeometry(0.13, BOARD_H, 0.38);
  const wallMat = new MeshStandardMaterial({ color: 0x1530b0, transparent: true, opacity: 0.1, metalness: 0.9 });
  for (const x of [-0.07, BOARD_W + 0.07]) {
    const wall = new Mesh(wallGeo, wallMat);
    wall.position.set(x, BOARD_H / 2, 0);
    well.add(wall);
  }

  // ── Cubos: geometria única, 21 materiais, pool de meshes ──────────────
  const cubeGeo = new BoxGeometry(0.86, 0.86, 0.44);
  const materials = new Map<string, MeshStandardMaterial>();
  for (const type of TYPES) {
    const { color } = TETROMINOES[type];
    for (const kind of Object.keys(KIND_STYLE) as CellKind[]) {
      const { emissive, opacity } = KIND_STYLE[kind];
      materials.set(
        `${color}:${kind}`,
        new MeshStandardMaterial({
          color,
          emissive: color,
          emissiveIntensity: emissive,
          metalness: 0.2,
          roughness: 0.55,
          transparent: opacity < 1,
          opacity,
        })
      );
    }
  }
  const fallback = new MeshStandardMaterial({ color: 0xffffff });
  const materialFor = (color: number, kind: CellKind) => materials.get(`${color}:${kind}`) ?? fallback;

  const pool: Mesh[] = [];
  let used = 0;

  function cube(color: number, kind: CellKind, col: number, row: number) {
    let mesh = pool[used];
    if (!mesh) {
      mesh = new Mesh(cubeGeo, fallback);
      pool.push(mesh);
      well.add(mesh);
    }
    const { z } = KIND_STYLE[kind];
    mesh.material = materialFor(color, kind);
    mesh.position.set(col + 0.5, BOARD_H - 1 - row + 0.5, z);
    mesh.visible = true;
    used++;
  }

  // ── Câmera orbital ────────────────────────────────────────────────────
  const orb = { theta: 0.28, phi: 1.08, radius: 26 };
  function applyCamera() {
    camera.position.set(
      orb.radius * Math.sin(orb.phi) * Math.sin(orb.theta),
      orb.radius * Math.cos(orb.phi),
      orb.radius * Math.sin(orb.phi) * Math.cos(orb.theta)
    );
    camera.lookAt(0, 0, 0);
  }
  applyCamera();

  return {
    render: () => renderer.render(scene, camera),

    resize(width, height) {
      renderer.setSize(width, height);
      camera.aspect = width / Math.max(1, height);
      camera.updateProjectionMatrix();
    },

    orbit(dTheta, dPhi) {
      orb.theta -= dTheta;
      orb.phi = Math.max(0.15, Math.min(Math.PI - 0.1, orb.phi + dPhi));
      applyCamera();
    },

    draw(board, piece, ghost) {
      used = 0;
      for (let r = 0; r < BOARD_H; r++)
        for (let c = 0; c < BOARD_W; c++) if (board[r][c]) cube(board[r][c], 'locked', c, r);

      if (piece) {
        if (ghost !== null && ghost !== piece.row) {
          for (const [bc, br] of shapeOf(piece.type, piece.rot)) {
            const r = ghost + br;
            if (r >= 0 && r < BOARD_H) cube(piece.color, 'ghost', piece.col + bc, r);
          }
        }
        for (const [bc, br] of shapeOf(piece.type, piece.rot)) {
          const r = piece.row + br;
          if (r >= 0 && r < BOARD_H) cube(piece.color, 'active', piece.col + bc, r);
        }
      }

      for (let i = used; i < pool.length; i++) pool[i].visible = false;
    },

    dispose() {
      for (const mesh of pool) well.remove(mesh);
      pool.length = 0;
      cubeGeo.dispose();
      for (const material of materials.values()) material.dispose();
      fallback.dispose();
      starGeo.dispose();
      starMat.dispose();
      gridGeo.dispose();
      gridMat.dispose();
      floorGeo.dispose();
      floorMat.dispose();
      frameGeo.dispose();
      frameMat.dispose();
      wallGeo.dispose();
      wallMat.dispose();
      renderer.dispose();
    },
  };
}
