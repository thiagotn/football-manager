<script lang="ts">
  import { onMount } from 'svelte';
  import * as THREE from 'three';

  onMount(() => {
    'use strict';

    const G  = document.getElementById('g')!;
    const cv = document.getElementById('c') as HTMLCanvasElement;

    // ── RENDERER + SCENE ──────────────────────────────────────────────────────
    const renderer = new THREE.WebGLRenderer({ canvas: cv, antialias: true });
    renderer.setPixelRatio(Math.min(devicePixelRatio, 2));
    renderer.shadowMap.enabled = true;

    function resize() {
      renderer.setSize(G.clientWidth, G.clientHeight);
      cam.aspect = G.clientWidth / G.clientHeight;
      cam.updateProjectionMatrix();
    }

    const scene = new THREE.Scene();
    scene.background = new THREE.Color(0x010112);
    scene.fog = new THREE.Fog(0x010112, 38, 75);

    const cam = new THREE.PerspectiveCamera(62, 1, 0.1, 200);

    // Lights
    scene.add(new THREE.AmbientLight(0x334466, 1.0));
    const sun = new THREE.DirectionalLight(0xffffff, 1.4);
    sun.position.set(8, 25, 18);
    sun.castShadow = true;
    scene.add(sun);
    const pl = new THREE.PointLight(0x4466bb, 0.7, 45);
    pl.position.set(-6, 8, 10);
    scene.add(pl);

    // Starfield
    {
      const n = 800, pos = new Float32Array(n * 3);
      for (let i = 0; i < n * 3; i++) pos[i] = (Math.random() - .5) * 120;
      const geo = new THREE.BufferGeometry();
      geo.setAttribute('position', new THREE.BufferAttribute(pos, 3));
      scene.add(new THREE.Points(geo, new THREE.PointsMaterial({
        color: 0xffffff, size: 0.12, transparent: true, opacity: 0.5
      })));
    }

    // ── BOARD ─────────────────────────────────────────────────────────────────
    const BW = 10, BH = 20;
    const BG = new THREE.Group();
    BG.position.set(-BW / 2, -BH / 2, 0);
    scene.add(BG);

    // Grid lines
    {
      const pts: number[] = [];
      for (let x = 0; x <= BW; x++) { pts.push(x, 0, 0); pts.push(x, BH, 0); }
      for (let y = 0; y <= BH; y++) { pts.push(0, y, 0); pts.push(BW, y, 0); }
      const geo = new THREE.BufferGeometry();
      geo.setAttribute('position', new THREE.BufferAttribute(new Float32Array(pts), 3));
      BG.add(new THREE.LineSegments(geo,
        new THREE.LineBasicMaterial({ color: 0x0c1838, transparent: true, opacity: 0.55 })));
    }

    // Floor base
    {
      const m = new THREE.Mesh(
        new THREE.BoxGeometry(BW + .2, .1, .8),
        new THREE.MeshStandardMaterial({ color: 0x0a1530, metalness: .6, roughness: .7 })
      );
      m.position.set(BW / 2, -.05, 0);
      BG.add(m);
    }

    // Board frame edges
    {
      const geo = new THREE.EdgesGeometry(new THREE.BoxGeometry(BW, BH, .45));
      const e = new THREE.LineSegments(geo,
        new THREE.LineBasicMaterial({ color: 0x1a4090, transparent: true, opacity: 0.4 }));
      e.position.set(BW / 2, BH / 2, 0);
      BG.add(e);
    }

    // Side walls
    function addWall(x: number) {
      const m = new THREE.Mesh(
        new THREE.BoxGeometry(.13, BH, .38),
        new THREE.MeshStandardMaterial({ color: 0x1530b0, transparent: true, opacity: .1, metalness: .9 })
      );
      m.position.set(x, BH / 2, 0);
      BG.add(m);
    }
    addWall(-.07); addWall(BW + .07);

    // ── TETROMINOS ────────────────────────────────────────────────────────────
    const TT: Record<string, { color: number; shapes: number[][][] }> = {
      I: { color: 0x00e5f5, shapes: [[[0,0],[1,0],[2,0],[3,0]], [[0,0],[0,1],[0,2],[0,3]]] },
      O: { color: 0xffd000, shapes: [[[0,0],[1,0],[0,1],[1,1]]] },
      T: { color: 0xee00bb, shapes: [
        [[0,0],[1,0],[2,0],[1,1]], [[0,0],[0,1],[1,1],[0,2]],
        [[1,0],[0,1],[1,1],[2,1]], [[1,0],[0,1],[1,1],[1,2]]
      ]},
      S: { color: 0x00ee77, shapes: [[[1,0],[2,0],[0,1],[1,1]], [[0,0],[0,1],[1,1],[1,2]]] },
      Z: { color: 0xff2244, shapes: [[[0,0],[1,0],[1,1],[2,1]], [[1,0],[0,1],[1,1],[0,2]]] },
      J: { color: 0x3377ff, shapes: [
        [[0,0],[0,1],[1,1],[2,1]], [[0,0],[1,0],[0,1],[0,2]],
        [[0,0],[1,0],[2,0],[2,1]], [[1,0],[1,1],[0,2],[1,2]]
      ]},
      L: { color: 0xff7700, shapes: [
        [[2,0],[0,1],[1,1],[2,1]], [[0,0],[0,1],[0,2],[1,2]],
        [[0,0],[1,0],[2,0],[0,1]], [[0,0],[1,0],[1,1],[1,2]]
      ]}
    };
    const TYPES = Object.keys(TT);

    // ── CUBE HELPERS ──────────────────────────────────────────────────────────
    const CGEO = new THREE.BoxGeometry(.86, .86, .44);

    function mkC(color: number, ei = .4, op = 1) {
      return new THREE.Mesh(CGEO, new THREE.MeshStandardMaterial({
        color, emissive: color, emissiveIntensity: ei,
        metalness: .2, roughness: .55,
        transparent: op < 1, opacity: op
      }));
    }

    function b2w(c: number, r: number) {
      return { x: c + .5, y: (BH - 1 - r) + .5 };
    }

    // ── GAME STATE ────────────────────────────────────────────────────────────
    let board: (number | 0)[][];
    let lockedM: THREE.Mesh[] = [], pieceM: THREE.Mesh[] = [], ghostM: THREE.Mesh[] = [];
    let cur: { type: string; rot: number; col: number; row: number; color: number } | null = null;
    let nxt: string | null = null;
    let score = 0, level = 1, lns = 0;
    let running = false, paused = false, dropT = 0, dropMs = 820;

    function resetBoard() {
      board = Array.from({ length: BH }, () => new Array(BW).fill(0));
      lockedM.forEach(m => BG.remove(m));
      lockedM = [];
    }

    // ── PIECE LOGIC ───────────────────────────────────────────────────────────
    function shp(type: string, rot: number) {
      const s = TT[type].shapes;
      return s[(rot || 0) % s.length];
    }

    function valid(p: typeof cur, dc: number, dr: number, nr?: number): boolean {
      if (!p) return false;
      const blk = shp(p.type, nr !== undefined ? nr : p.rot);
      for (const [bc, br] of blk) {
        const c = p.col + bc + (dc || 0);
        const r = p.row + br + (dr || 0);
        if (c < 0 || c >= BW) return false;
        if (r >= BH) return false;
        if (r < 0) continue;
        if (board[r][c]) return false;
      }
      return true;
    }

    function ghostR(): number {
      if (!cur) return 0;
      let o = 0;
      while (valid(cur, 0, o + 1)) o++;
      return cur.row + o;
    }

    function clearPM() { pieceM.forEach(m => BG.remove(m)); pieceM = []; }
    function clearGM() { ghostM.forEach(m => BG.remove(m)); ghostM = []; }

    function renderPiece() {
      clearPM(); if (!cur) return;
      for (const [bc, br] of shp(cur.type, cur.rot)) {
        const m = mkC(cur.color, .62);
        const p = b2w(cur.col + bc, cur.row + br);
        m.position.set(p.x, p.y, .12);
        m.castShadow = true;
        BG.add(m); pieceM.push(m);
      }
    }

    function renderGhost() {
      clearGM(); if (!cur) return;
      const gr = ghostR();
      if (gr === cur.row) return;
      for (const [bc, br] of shp(cur.type, cur.rot)) {
        const r = gr + br;
        if (r < 0 || r >= BH) continue;
        const m = mkC(cur.color, 0, .17);
        const p = b2w(cur.col + bc, r);
        m.position.set(p.x, p.y, .04);
        BG.add(m); ghostM.push(m);
      }
    }

    function renderBoard() {
      lockedM.forEach(m => BG.remove(m)); lockedM = [];
      for (let r = 0; r < BH; r++) {
        for (let c = 0; c < BW; c++) {
          if (board[r][c]) {
            const m = mkC(board[r][c] as number, .19);
            const p = b2w(c, r);
            m.position.set(p.x, p.y, 0);
            BG.add(m); lockedM.push(m);
          }
        }
      }
    }

    // ── GAME ACTIONS ──────────────────────────────────────────────────────────
    function spCol(type: string): number {
      const mx = Math.max(...TT[type].shapes[0].map(([c]) => c));
      return Math.floor((BW - mx - 1) / 2);
    }

    function spawn() {
      const type = nxt || TYPES[Math.floor(Math.random() * TYPES.length)];
      nxt = TYPES[Math.floor(Math.random() * TYPES.length)];
      cur = { type, rot: 0, col: spCol(type), row: 0, color: TT[type].color };
      if (!valid(cur, 0, 0)) { endGame(); return; }
      renderPiece(); renderGhost(); drawNext();
    }

    function lockPiece() {
      if (!cur) return;
      for (const [bc, br] of shp(cur.type, cur.rot)) {
        const c = cur.col + bc, r = cur.row + br;
        if (r >= 0 && r < BH && c >= 0 && c < BW) board[r][c] = cur.color;
      }
      clearPM(); clearGM();
      clearLines(); renderBoard(); spawn();
    }

    function clearLines() {
      let cl = 0, r = 0;
      while (r < BH) {
        if (board[r].every(c => c !== 0)) {
          board.splice(r, 1);
          board.unshift(new Array(BW).fill(0));
          cl++;
        } else r++;
      }
      if (cl > 0) {
        score += [0, 100, 300, 500, 800][cl] * level;
        lns += cl;
        level = Math.floor(lns / 10) + 1;
        dropMs = Math.max(80, 820 - (level - 1) * 74);
        updHUD();
      }
    }

    function moveL() { if (!cur || !running || paused) return; if (valid(cur, -1, 0)) { cur.col--; renderPiece(); renderGhost(); } }
    function moveR() { if (!cur || !running || paused) return; if (valid(cur,  1, 0)) { cur.col++; renderPiece(); renderGhost(); } }

    function moveD(): boolean {
      if (!cur || !running || paused) return false;
      if (valid(cur, 0, 1)) { cur.row++; renderPiece(); renderGhost(); return true; }
      lockPiece(); return false;
    }

    function doRotate() {
      if (!cur || !running || paused) return;
      const nr = (cur.rot + 1) % TT[cur.type].shapes.length;
      for (const [kc, kr] of [[0,0],[1,0],[-1,0],[2,0],[-2,0]]) {
        if (valid(cur, kc, kr, nr)) {
          cur.col += kc; cur.row += kr; cur.rot = nr;
          renderPiece(); renderGhost(); return;
        }
      }
    }

    function hardDrop() {
      if (!cur || !running || paused) return;
      const gr = ghostR();
      score += (gr - cur.row) * 2;
      cur.row = gr;
      lockPiece(); updHUD();
    }

    // ── HUD ───────────────────────────────────────────────────────────────────
    function updHUD() {
      const hs = document.getElementById('hs');
      const hlv = document.getElementById('hlv');
      const hln = document.getElementById('hln');
      if (hs)  hs.textContent  = score >= 10000 ? (score / 1000).toFixed(1) + 'K' : String(score);
      if (hlv) hlv.textContent = String(level);
      if (hln) hln.textContent = String(lns);
    }

    function drawNext() {
      const nc = document.getElementById('nc') as HTMLCanvasElement | null;
      if (!nc || !nxt) return;
      const ctx = nc.getContext('2d')!;
      ctx.clearRect(0, 0, 64, 64);
      const blk = TT[nxt].shapes[0];
      const col = '#' + TT[nxt].color.toString(16).padStart(6, '0');
      const minc = Math.min(...blk.map(([c]) => c)), maxc = Math.max(...blk.map(([c]) => c));
      const minr = Math.min(...blk.map(([, r]) => r)), maxr = Math.max(...blk.map(([, r]) => r));
      const cell = Math.floor(Math.min(52 / (maxc - minc + 1), 52 / (maxr - minr + 1)));
      const ox = (64 - cell * (maxc - minc + 1)) / 2;
      const oy = (64 - cell * (maxr - minr + 1)) / 2;
      ctx.fillStyle = col;
      ctx.shadowColor = col;
      ctx.shadowBlur = 7;
      for (const [bc, br] of blk) {
        ctx.fillRect(ox + (bc - minc) * cell + 1, oy + (br - minr) * cell + 1, cell - 2, cell - 2);
      }
    }

    // ── CAMERA ORBIT ──────────────────────────────────────────────────────────
    const orb = { th: .28, ph: 1.08, r: 26, drag: false, moved: false, lx: 0, ly: 0 };

    function applyCam() {
      cam.position.set(
        orb.r * Math.sin(orb.ph) * Math.sin(orb.th),
        orb.r * Math.cos(orb.ph),
        orb.r * Math.sin(orb.ph) * Math.cos(orb.th)
      );
      cam.lookAt(0, 0, 0);
    }
    applyCam();

    cv.addEventListener('mousedown', e => { orb.drag = true; orb.moved = false; orb.lx = e.clientX; orb.ly = e.clientY; });

    const handleMouseMove = (e: MouseEvent) => {
      if (!orb.drag) return;
      const dx = e.clientX - orb.lx, dy = e.clientY - orb.ly;
      if (Math.abs(dx) > 2 || Math.abs(dy) > 2) orb.moved = true;
      orb.th -= dx * .0085;
      orb.ph = Math.max(.15, Math.min(Math.PI - .1, orb.ph + dy * .0085));
      orb.lx = e.clientX; orb.ly = e.clientY;
      applyCam();
    };
    window.addEventListener('mousemove', handleMouseMove);

    const handleMouseUp = (_e: MouseEvent) => { if (orb.drag && !orb.moved) doRotate(); orb.drag = false; };
    window.addEventListener('mouseup', handleMouseUp);

    let t1: { sx: number; sy: number; lx: number; ly: number; moved: boolean } | null = null;
    cv.addEventListener('touchstart', e => {
      if (e.touches.length === 1) {
        t1 = { sx: e.touches[0].clientX, sy: e.touches[0].clientY, lx: e.touches[0].clientX, ly: e.touches[0].clientY, moved: false };
      }
      e.preventDefault();
    }, { passive: false });

    cv.addEventListener('touchmove', e => {
      if (t1 && e.touches.length === 1) {
        const tx = e.touches[0].clientX, ty = e.touches[0].clientY;
        const dx = tx - t1.lx, dy = ty - t1.ly;
        if (Math.abs(tx - t1.sx) > 7 || Math.abs(ty - t1.sy) > 7) t1.moved = true;
        orb.th -= dx * .0085;
        orb.ph = Math.max(.15, Math.min(Math.PI - .1, orb.ph + dy * .0085));
        t1.lx = tx; t1.ly = ty;
        applyCam();
      }
      e.preventDefault();
    }, { passive: false });

    cv.addEventListener('touchend', e => { if (t1 && !t1.moved) doRotate(); t1 = null; e.preventDefault(); }, { passive: false });

    // ── KEYBOARD ──────────────────────────────────────────────────────────────
    let kdn = false, ffiv: ReturnType<typeof setInterval> | null = null;
    function startFF() { if (ffiv) return; ffiv = setInterval(() => { if (running && !paused) moveD(); }, 55); }
    function stopFF()  { if (ffiv) { clearInterval(ffiv); ffiv = null; } kdn = false; }

    const handleKeyDown = (e: KeyboardEvent) => {
      if (!running || paused) return;
      if (e.code === 'ArrowLeft')                { moveL(); }
      if (e.code === 'ArrowRight')               { moveR(); }
      if (e.code === 'ArrowDown'  && !kdn)       { kdn = true; startFF(); }
      if (e.code === 'ArrowUp')                  { doRotate(); e.preventDefault(); }
      if (e.code === 'Space')                    { hardDrop(); e.preventDefault(); }
      if (e.code === 'KeyP')                     { paused = !paused; }
    };
    const handleKeyUp = (e: KeyboardEvent) => { if (e.code === 'ArrowDown') stopFF(); };

    document.addEventListener('keydown', handleKeyDown);
    document.addEventListener('keyup', handleKeyUp);

    // ── MOBILE BUTTONS ────────────────────────────────────────────────────────
    function mbtn(id: string, fn: () => void, iv = 130) {
      const el = document.getElementById(id); if (!el) return;
      let t: ReturnType<typeof setInterval> | null = null;
      const s    = (e: Event) => { e.preventDefault(); e.stopPropagation(); fn(); t = setInterval(fn, iv); };
      const stop = (e: Event) => { e.preventDefault(); if (t) { clearInterval(t); t = null; } };
      el.addEventListener('touchstart', s,    { passive: false });
      el.addEventListener('touchend',   stop, { passive: false });
      el.addEventListener('mousedown',  s);
      el.addEventListener('mouseup',    stop);
      el.addEventListener('mouseleave', stop);
    }
    mbtn('bl', moveL);
    mbtn('br', moveR);
    mbtn('bd', moveD, 60);

    const brot = document.getElementById('brot');
    if (brot) {
      brot.addEventListener('touchstart', e => { e.preventDefault(); e.stopPropagation(); doRotate(); }, { passive: false });
      brot.addEventListener('click', doRotate);
    }

    const bdrp = document.getElementById('bdrp');
    if (bdrp) {
      bdrp.addEventListener('touchstart', e => { e.preventDefault(); e.stopPropagation(); hardDrop(); }, { passive: false });
      bdrp.addEventListener('click', hardDrop);
    }

    // ── GAME FLOW ─────────────────────────────────────────────────────────────
    function startGame() {
      resetBoard();
      score = 0; level = 1; lns = 0; dropMs = 820;
      running = true; paused = false; nxt = null; cur = null;
      updHUD();
      document.getElementById('ov')?.classList.add('h');
      spawn();
      dropT = performance.now();
    }

    function endGame() {
      running = false; cur = null;
      clearPM(); clearGM();
      const ovt = document.getElementById('ovt');
      const ovsc = document.getElementById('ovsc');
      const ov = document.getElementById('ov');
      if (ovt)  ovt.textContent  = 'GAME OVER';
      if (ovsc) ovsc.textContent = 'SCORE: ' + score.toLocaleString();
      if (ov)   ov.classList.remove('h');
    }

    document.getElementById('ovb')?.addEventListener('click', startGame);

    // ── RESIZE + LOOP ─────────────────────────────────────────────────────────
    window.addEventListener('resize', resize);
    resize();

    let animId: number;
    function loop(t: number) {
      animId = requestAnimationFrame(loop);
      if (running && !paused && cur && t - dropT > dropMs) {
        moveD(); dropT = t;
      }
      renderer.render(scene, cam);
    }
    animId = requestAnimationFrame(loop);

    return () => {
      cancelAnimationFrame(animId);
      stopFF();
      window.removeEventListener('resize', resize);
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
      document.removeEventListener('keydown', handleKeyDown);
      document.removeEventListener('keyup', handleKeyUp);
      renderer.dispose();
    };
  });
</script>

<svelte:head>
  <title>Tetris 3D — rachao.app</title>
</svelte:head>

<div id="g">
  <canvas id="c"></canvas>

  <div id="hud">
    <div class="hb"><div class="hl">SCORE</div><div class="hv" id="hs">0</div></div>
    <div class="hb"><div class="hl">LEVEL</div><div class="hv" id="hlv">1</div></div>
    <div class="hb"><div class="hl">LINES</div><div class="hv" id="hln">0</div></div>
  </div>

  <div id="nxt">
    <div class="nl">PRÓXIMA</div>
    <canvas id="nc" width="64" height="64"></canvas>
  </div>

  <div id="keys">
    <div><span>← →</span> mover</div>
    <div><span>↓</span> descer rápido</div>
    <div><span>↑</span> rotacionar</div>
    <div><span>SPACE</span> drop</div>
    <div><span>P</span> pausar</div>
    <div><span>ARRASTE</span> câmera 3D</div>
    <div><span>CLIQUE</span> rotacionar</div>
  </div>

  <div id="ctrl">
    <div id="mg">
      <div id="lrb">
        <button class="cb" id="bl">◀</button>
        <button class="cb" id="br">▶</button>
      </div>
      <button class="cb blue" id="bd">▼</button>
    </div>
    <div id="ag">
      <button class="cb gold lg" id="brot">↻</button>
      <button class="cb blue sm" id="bdrp">DROP</button>
    </div>
  </div>

  <div id="tip">ARRASTE → CÂMERA 3D &nbsp;|&nbsp; TOQUE/CLIQUE → ROTACIONA PEÇA</div>

  <div id="ov">
    <div id="ovt">TETRIS 3D</div>
    <div id="ovs">RACHÃO EDITION</div>
    <div id="ovsc"></div>
    <button id="ovb">▶ JOGAR</button>
  </div>

  <!-- Botão para sair do jogo -->
  <a href="/" id="back">ESC</a>
</div>

<style>
  #g {
    position: fixed;
    inset: 0;
    z-index: 50;
    overflow: hidden;
    background: #010112;
  }
  #c {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    touch-action: none;
    display: block;
  }

  /* HUD */
  #hud { position: absolute; top: 0; left: 0; right: 0; display: flex; justify-content: center; gap: 10px; padding: 12px; z-index: 5; pointer-events: none; }
  .hb { background: rgba(3,8,28,.88); border: 1px solid rgba(70,130,255,.18); border-radius: 8px; padding: 6px 14px; text-align: center; }
  .hl { font-size: 8px; letter-spacing: 2px; color: rgba(70,140,255,.6); margin-bottom: 2px; font-family: monospace; }
  .hv { font-size: 17px; font-weight: 600; color: #eee; font-family: 'Courier New', monospace; }

  /* Next piece */
  #nxt { position: absolute; top: 75px; right: 12px; background: rgba(3,8,28,.88); border: 1px solid rgba(70,130,255,.18); border-radius: 8px; padding: 10px; z-index: 5; pointer-events: none; }
  .nl { font-size: 7px; letter-spacing: 2px; color: rgba(70,140,255,.55); text-align: center; margin-bottom: 6px; font-family: monospace; }

  /* Controls */
  #ctrl { position: absolute; bottom: 0; left: 0; right: 0; height: 165px; display: flex; align-items: center; justify-content: space-between; padding: 12px 26px 22px; z-index: 10; pointer-events: none; }
  #mg { display: flex; flex-direction: column; align-items: center; gap: 7px; }
  #lrb { display: flex; gap: 9px; }
  #ag { display: flex; flex-direction: column; align-items: center; gap: 9px; }
  .cb { pointer-events: all; border-radius: 50%; background: rgba(255,255,255,.06); border: 1.5px solid rgba(255,255,255,.18); color: rgba(255,255,255,.88); display: flex; align-items: center; justify-content: center; cursor: pointer; font-size: 19px; width: 56px; height: 56px; touch-action: none; user-select: none; -webkit-user-select: none; transition: background .1s, transform .08s; font-family: monospace; }
  .cb:active { background: rgba(255,255,255,.22); transform: scale(.87); }
  .cb.gold { width: 66px; height: 66px; font-size: 27px; background: rgba(255,200,30,.08); border-color: rgba(255,200,30,.32); color: rgba(255,215,50,.95); }
  .cb.blue { background: rgba(50,130,255,.08); border-color: rgba(50,130,255,.3); color: rgba(90,165,255,.9); }
  .cb.sm { border-radius: 10px; width: 56px; height: 34px; font-size: 9px; letter-spacing: 1px; }

  /* Keyboard hint */
  #tip { position: absolute; bottom: 172px; left: 50%; transform: translateX(-50%); font-size: 8px; letter-spacing: 1.5px; color: rgba(255,255,255,.17); z-index: 5; pointer-events: none; white-space: nowrap; font-family: monospace; }

  /* Overlay */
  #ov { position: absolute; inset: 0; background: rgba(1,1,18,.94); display: flex; flex-direction: column; align-items: center; justify-content: center; z-index: 20; }
  #ov.h { display: none; }
  #ovt { font-size: clamp(26px, 6.5vw, 52px); font-weight: 700; letter-spacing: 5px; color: #eee; margin-bottom: 5px; font-family: 'Courier New', monospace; }
  #ovs { font-size: clamp(9px, 2.5vw, 12px); letter-spacing: 4px; color: rgba(70,140,255,.55); margin-bottom: 24px; font-family: monospace; }
  #ovsc { font-size: clamp(12px, 3vw, 17px); color: rgba(255,210,50,.88); margin-bottom: 36px; letter-spacing: 2px; font-family: monospace; min-height: 22px; }
  #ovb { padding: 13px 40px; background: transparent; border: 1.5px solid rgba(70,140,255,.5); color: rgba(90,165,255,.9); font-family: monospace; font-size: 13px; font-weight: 600; letter-spacing: 4px; cursor: pointer; border-radius: 4px; transition: all .2s; }
  #ovb:hover, #ovb:active { background: rgba(70,140,255,.14); }

  /* Keyboard hints panel */
  #keys { position: absolute; top: 75px; left: 12px; background: rgba(3,8,28,.88); border: 1px solid rgba(70,130,255,.18); border-radius: 8px; padding: 10px 12px; z-index: 5; pointer-events: none; }
  #keys div { font-family: monospace; font-size: 9px; letter-spacing: 1px; color: rgba(180,200,255,.45); line-height: 1.8; }
  #keys span { color: rgba(90,165,255,.75); }

  /* Back button */
  #back {
    position: absolute;
    top: 12px;
    right: 12px;
    z-index: 30;
    text-decoration: none;
    border-radius: 10px;
    background: rgba(255,255,255,.06);
    border: 1.5px solid rgba(255,255,255,.18);
    color: rgba(255,255,255,.75);
    display: flex;
    align-items: center;
    justify-content: center;
    width: 56px;
    height: 34px;
    font-size: 9px;
    letter-spacing: 1px;
    font-family: monospace;
    cursor: pointer;
    transition: background .1s;
  }
  #back:hover { background: rgba(255,255,255,.14); color: #fff; }
</style>
