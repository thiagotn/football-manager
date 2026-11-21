<script lang="ts">
  /**
   * Bê-á-Bá — letras e números para crianças em alfabetização (PRD 058, fase 1).
   *
   * Tela cheia (`fixed inset-0`), clara, com botões grandes. Abas: Início
   * (as duas trilhas) · Álbum (figurinhas) · Pais (protegida por pergunta).
   * A rodada (5 desafios) entra por cima das abas e termina sempre com prêmio.
   *
   * Regras em `$lib/be-a-ba/{letters,numbers,progress}.ts`, conteúdo em
   * `words.ts`/`stickers.ts`, voz em `speech.ts`, fotos em `photos.ts`
   * (IndexedDB) e o resto em `storage.ts` (localStorage `games_be_a_ba`).
   */
  import { onMount } from 'svelte';
  import { BookOpen, House, Lock, Trash2, Volume2, X, Camera } from 'lucide-svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import LanguageSwitcher from '$lib/components/LanguageSwitcher.svelte';
  import LetterChallengeView from '$lib/be-a-ba/LetterChallenge.svelte';
  import NumberChallengeView from '$lib/be-a-ba/NumberChallenge.svelte';
  import LittleChallengeView from '$lib/be-a-ba/LittleChallenge.svelte';
  import {
    LITTLE_LETTER_STAGES,
    LITTLE_NUMBER_STAGES,
    LITTLE_ROUND_SIZE,
    littleAnswer,
    littleItemId,
    littleLetterRound,
    littleNumberRound,
    type LittleChallenge,
  } from '$lib/be-a-ba/little';
  import { MASCOT, MEDAL, PARTY, STAR, imgUrl } from '$lib/be-a-ba/art';
  import { WORDS, plain } from '$lib/be-a-ba/words';
  import { cleanFamilyWord, type PlayWord } from '$lib/be-a-ba/letters';
  import { answerOf, type NumberKind } from '$lib/be-a-ba/numbers';
  import {
    LETTER_STAGES,
    NUMBER_STAGES,
    ROUND_SIZE,
    afterRound,
    letterRound,
    numberItemId,
    numberRound,
    outcomeOf,
    recordOutcome,
    type Challenge,
    type Outcome,
    type Trail,
  } from '$lib/be-a-ba/progress';
  import { NAME_STICKER, PAGE_SIZE, STICKERS, nextSticker, stickerById, type Sticker } from '$lib/be-a-ba/stickers';
  import { emptyState, loadState, saveState, type AgeMode, type FamilyWord, type SavedState } from '$lib/be-a-ba/storage';
  import type { TrailProgress } from '$lib/be-a-ba/progress';
  import { deletePhoto, loadPhoto, savePhoto, shrinkPhoto } from '$lib/be-a-ba/photos';
  import { speak, speakSequence, stopSpeaking } from '$lib/speech';
  import {
    EN_LANG,
    EN_NUMBERS,
    EN_PRAISE,
    EN_STAGES,
    EN_WORDS,
    enAnswerWord,
    enPrompt,
    enSlug,
    englishRound,
  } from '$lib/be-a-ba/english';
  import { mulberry32, dateKey } from '$lib/enche-ai/engine';
  import { locale, t } from '$lib/i18n';

  type Screen = 'home' | 'album' | 'parents';
  type AnyChallenge = Challenge | LittleChallenge;
  type Round = { trail: Trail; little: boolean; challenges: AnyChallenge[]; index: number; outcomes: Outcome[]; wroteName: boolean };
  type Reward = {
    trail: Trail;
    sticker: Sticker | null;
    nameSticker: boolean;
    stars: number;
    leveledUp: boolean;
  };

  /** As palavras da trilha das Letras são em português (PRD 058). */
  const LETTERS_LANG = 'pt-BR';
  const PRAISE_MS = 1300;
  const GATE_NUMBERS = [27, 42, 53, 68, 74, 89];
  const TABS = [
    { key: 'home', icon: House, labelKey: 'be_a_ba.tab_home' },
    { key: 'album', icon: BookOpen, labelKey: 'be_a_ba.tab_album' },
    { key: 'parents', icon: Lock, labelKey: 'be_a_ba.tab_parents' },
  ] as const;

  let saved = $state<SavedState>(emptyState());
  let ready = $state(false);
  let screen = $state<Screen>('home');
  let round = $state<Round | null>(null);
  let praise = $state<string | null>(null);
  /** Trilha Inglês: a palavra em inglês que aparece escrita debaixo do elogio. */
  let praiseWord = $state<string | null>(null);
  let reward = $state<Reward | null>(null);
  let photoUrls = $state<Record<string, string>>({});
  let toast = $state<string | null>(null);

  // Portão dos pais
  let gate = $state<{ target: number; options: number[] } | null>(null);
  let gateError = $state(false);
  let parentsOpen = $state(false);

  // Área dos pais
  let nameInput = $state('');
  let newWord = $state('');
  let newPhoto = $state<File | null>(null);
  let photoInput = $state<HTMLInputElement>();
  let familyError = $state<string | null>(null);
  let removing = $state<FamilyWord | null>(null);
  let confirmRemove = $state(false);
  let confirmReset = $state(false);

  let rnd = mulberry32(Date.now() >>> 0);
  let toastTimer: ReturnType<typeof setTimeout> | undefined;

  const lettersAvailable = $derived($locale === 'pt-BR');
  /** Modo Pequenos (3–4 anos): outras atividades, rodadas de 3 e estágios próprios. */
  const isLittle = $derived(saved.ageMode === 'little');
  const current = $derived(round ? round.challenges[round.index] : null);
  const familyWords = $derived<PlayWord[]>(
    saved.family.map((f) => ({ id: f.id, text: f.text, family: true, photo: f.photo ? f.id : undefined }))
  );
  const ownedSet = $derived(new Set(saved.stickers));
  const today = dateKey(new Date());

  // ───────── Persistência e voz ─────────

  function persist() {
    saveState($state.snapshot(saved));
  }

  /**
   * Progresso da trilha no modo atual (Pequenos e 5–7 anos são independentes).
   * O Inglês é um só para as duas idades — o conteúdo é o mesmo.
   */
  function progressOf(trail: Trail, little = isLittle): TrailProgress {
    if (trail === 'english') return saved.english;
    return little ? saved.little[trail] : saved[trail];
  }
  function setProgress(trail: Trail, p: TrailProgress, little = isLittle) {
    if (trail === 'english') saved.english = p;
    else if (little) saved.little[trail] = p;
    else saved[trail] = p;
  }
  const stageCountOf = (trail: Trail, little = isLittle) =>
    trail === 'english'
      ? EN_STAGES.length
      : little
        ? (trail === 'letters' ? LITTLE_LETTER_STAGES : LITTLE_NUMBER_STAGES).length
        : (trail === 'letters' ? LETTER_STAGES : NUMBER_STAGES).length;

  function chooseAge(mode: AgeMode) {
    saved.ageMode = mode;
    persist();
  }

  function say(text: string, lang: string = $locale, rate = saved.settings.rate) {
    speak(text, { lang, rate, enabled: saved.settings.voice });
  }

  function showToast(msg: string) {
    toast = msg;
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => (toast = null), 2200);
  }

  onMount(() => {
    saved = loadState();
    nameInput = saved.childName;
    ready = true;
    void loadPhotos();

    // Tempo jogado por dia (só conta com rodada aberta e a aba visível).
    const tick = setInterval(() => {
      if (round && document.visibilityState === 'visible') saved.playtime[today] = (saved.playtime[today] ?? 0) + 1;
    }, 1000);
    const flush = () => persist();
    window.addEventListener('pagehide', flush);
    return () => {
      clearInterval(tick);
      window.removeEventListener('pagehide', flush);
      stopSpeaking();
      for (const url of Object.values(photoUrls)) URL.revokeObjectURL(url);
      persist();
    };
  });

  async function loadPhotos() {
    for (const f of saved.family) {
      if (!f.photo || photoUrls[f.id]) continue;
      try {
        const blob = await loadPhoto(f.id);
        if (blob) photoUrls[f.id] = URL.createObjectURL(blob);
      } catch {
        // IndexedDB indisponível: a palavra aparece com a letra inicial.
      }
    }
  }

  // ───────── Instruções ─────────

  function instruction(c: AnyChallenge): { show: string; ask: string; lang: string } {
    if (round?.trail === 'english') {
      const ask = enPrompt(c as LittleChallenge);
      return { show: ask, ask, lang: EN_LANG };
    }
    switch (c.kind) {
      case 'hear':
        return {
          show: $t('be_a_ba.show_hear'),
          ask: $t(`be_a_ba.ask_hear_${c.article}`, { word: c.word.text.toLocaleLowerCase('pt-BR') }),
          lang: LETTERS_LANG,
        };
      case 'pick_letter':
        return { show: $t('be_a_ba.show_pick_letter'), ask: $t('be_a_ba.ask_pick_letter', { letter: c.letter.toLowerCase() }), lang: LETTERS_LANG };
      case 'tap':
        return { show: $t('be_a_ba.show_tap'), ask: $t('be_a_ba.ask_tap'), lang: $locale };
      case 'find_number':
        return { show: $t('be_a_ba.show_find_number'), ask: $t('be_a_ba.ask_find_number', { n: c.n }), lang: $locale };
      case 'where':
        return { show: $t('be_a_ba.show_where'), ask: $t('be_a_ba.ask_where', { n: c.n }), lang: $locale };
      case 'pick_color': {
        // Só existe na trilha Inglês, tratada acima.
        const ask = enPrompt(c);
        return { show: ask, ask, lang: EN_LANG };
      }
    }
    if (c.kind === 'letter') {
      const word = c.word.text.toLocaleLowerCase('pt-BR');
      return { show: $t(`be_a_ba.show_${c.mode}`), ask: $t(`be_a_ba.ask_${c.mode}`, { word }), lang: LETTERS_LANG };
    }
    const show = $t(`be_a_ba.show_${c.kind}`);
    switch (c.kind) {
      case 'count':
        return { show, ask: $t(`be_a_ba.ask_count.${c.item}`), lang: $locale };
      case 'compare':
        return { show, ask: $t('be_a_ba.ask_compare'), lang: $locale };
      case 'add':
        return { show, ask: $t('be_a_ba.ask_add', { a: c.a, b: c.b }), lang: $locale };
      case 'sub':
        return { show, ask: $t('be_a_ba.ask_sub', { a: c.a, b: c.b }), lang: $locale };
    }
  }

  const askCurrent = () => {
    if (!current) return;
    const ins = instruction(current);
    say(ins.ask, ins.lang);
  };

  // Cada desafio novo é anunciado em voz alta.
  $effect(() => {
    if (round && !praise && !reward) {
      void round.index;
      askCurrent();
    }
  });

  // ───────── Rodada ─────────

  function startRound(trail: Trail) {
    if (trail === 'letters' && !lettersAvailable) return;
    reward = null;
    praise = null;
    // O Inglês usa os desafios de toque do modo Pequenos nas duas idades.
    const little = isLittle || trail === 'english';
    const p = progressOf(trail, isLittle);
    const challenges: AnyChallenge[] =
      trail === 'english'
        ? englishRound(p, isLittle ? LITTLE_ROUND_SIZE : ROUND_SIZE, rnd)
        : little
      ? trail === 'letters'
        ? littleLetterRound(p, saved.childName, rnd)
        : littleNumberRound(p, rnd)
      : trail === 'letters'
        ? letterRound(p, familyWords, rnd)
        : numberRound(p, rnd);
    round = { trail, little, challenges, index: 0, outcomes: [], wroteName: false };
  }

  function quitRound() {
    stopSpeaking();
    round = null;
    praise = null;
    persist();
  }

  function onResult(mistakes: number, hinted: boolean) {
    if (!round || !current) return;
    const outcome = outcomeOf(mistakes, hinted);
    let spoken: string;
    let lang: string = $locale;
    if (round.trail === 'english') {
      const c = current as LittleChallenge;
      saved.english = { ...saved.english, boxes: recordOutcome(saved.english.boxes, littleItemId(c), outcome) };
      saved.stars++;
      round.outcomes = [...round.outcomes, outcome];
      // "Great job! Apple!" em inglês e a ponte "maçã" no idioma da interface.
      const word = enAnswerWord(c);
      const bridge =
        c.kind === 'hear'
          ? $t(`be_a_ba.en_word.${enSlug(c.word)}`)
          : c.kind === 'pick_color'
            ? $t(`be_a_ba.en_color.${c.color}`)
            : 'n' in c
              ? String(c.n)
              : '';
      praise = EN_PRAISE[Math.floor(rnd() * EN_PRAISE.length)];
      praiseWord = word;
      speakSequence(
        [
          { text: `${praise} ${word}!`, lang: EN_LANG },
          { text: $locale.startsWith('en') ? '' : bridge, lang: $locale },
        ],
        { rate: saved.settings.rate, enabled: saved.settings.voice }
      );
      persist();
      setTimeout(() => {
        praise = null;
        praiseWord = null;
        if (!round) return;
        if (round.index < round.challenges.length - 1) round.index++;
        else finishRound();
      }, PRAISE_MS + 500);
      return;
    }
    if (round.little) {
      const c = current as LittleChallenge;
      const p = progressOf(round.trail, true);
      setProgress(round.trail, { ...p, boxes: recordOutcome(p.boxes, littleItemId(c), outcome) }, true);
      if (c.kind === 'hear') {
        spoken = c.word.text.toLocaleLowerCase('pt-BR');
        lang = LETTERS_LANG;
      } else if (c.kind === 'pick_letter') {
        spoken = c.letter.toLowerCase();
        lang = LETTERS_LANG;
      } else spoken = c.kind === 'where' ? String(c.n) : String(littleAnswer(c));
    } else if (current.kind === 'letter') {
      lang = LETTERS_LANG;
      saved.letters.boxes = recordOutcome(saved.letters.boxes, current.word.id, outcome);
      if (saved.family.some((f) => f.isChildName && f.id === current.word.id)) round.wroteName = true;
      spoken = current.word.text.toLocaleLowerCase('pt-BR');
    } else {
      saved.numbers.boxes = recordOutcome(saved.numbers.boxes, numberItemId(current as Exclude<Challenge, { kind: 'letter' }>), outcome);
      const ans = answerOf(current as Exclude<Challenge, { kind: 'letter' }>);
      spoken = typeof ans === 'number' ? String(ans) : '';
    }
    saved.stars++;
    round.outcomes = [...round.outcomes, outcome];
    praise = $t(`be_a_ba.praise_${Math.floor(rnd() * 4)}`);
    say(`${praise} ${spoken}`, lang);
    persist();

    setTimeout(() => {
      praise = null;
      if (!round) return;
      if (round.index < round.challenges.length - 1) round.index++;
      else finishRound();
    }, PRAISE_MS);
  }

  function finishRound() {
    if (!round) return;
    const r = round;
    const clean = r.outcomes.filter((o) => o === 'clean').length;
    const { progress, leveledUp } = afterRound(
      progressOf(r.trail, r.little),
      stageCountOf(r.trail, r.little),
      clean,
      r.challenges.length
    );
    setProgress(r.trail, progress, r.little);
    saved.rounds++;

    let nameSticker = false;
    if (r.wroteName && !saved.stickers.includes(NAME_STICKER.id)) {
      saved.stickers = [...saved.stickers, NAME_STICKER.id];
      nameSticker = true;
    }
    const sticker = nextSticker(saved.stickers, rnd());
    if (sticker) saved.stickers = [...saved.stickers, sticker.id];
    persist();

    reward = { trail: r.trail, sticker, nameSticker, stars: r.outcomes.length, leveledUp };
    round = null;
    const parts = [$t('be_a_ba.round_done')];
    if (nameSticker) parts.push($t('be_a_ba.name_sticker'));
    parts.push(sticker ? `${$t('be_a_ba.got_sticker')} ${$t(`be_a_ba.sticker.${sticker.id}`)}!` : $t('be_a_ba.album_full'));
    if (leveledUp) parts.push($t('be_a_ba.level_up'));
    say(parts.join(' '));
  }

  // ───────── Portão e área dos pais ─────────

  function shuffle<T>(xs: T[]): T[] {
    const a = xs.slice();
    for (let i = a.length - 1; i > 0; i--) {
      const j = Math.floor(rnd() * (i + 1));
      [a[i], a[j]] = [a[j], a[i]];
    }
    return a;
  }

  function openGate() {
    const options = shuffle(GATE_NUMBERS).slice(0, 4);
    gate = { target: options[Math.floor(rnd() * options.length)], options };
    gateError = false;
  }

  function pickGate(n: number) {
    if (!gate) return;
    if (n === gate.target) {
      gate = null;
      parentsOpen = true;
      screen = 'parents';
    } else {
      gateError = true;
      const err = true;
      openGate();
      gateError = err;
    }
  }

  function go(key: Screen) {
    if (key === 'parents') {
      if (parentsOpen) screen = 'parents';
      else openGate();
      return;
    }
    // Sair da área dos pais tranca de novo.
    parentsOpen = false;
    screen = key;
  }

  function saveChildName() {
    const name = nameInput.trim().slice(0, 24);
    saved.childName = name;
    const text = cleanFamilyWord(name);
    const prev = saved.family.find((f) => f.isChildName);
    const others = saved.family.filter((f) => !f.isChildName);
    saved.family = text
      ? [{ id: prev?.id ?? 'child', text, photo: prev?.photo ?? false, isChildName: true }, ...others]
      : others;
    persist();
    showToast($t('be_a_ba.saved'));
  }

  async function storePhoto(id: string, file: File): Promise<boolean> {
    try {
      const blob = await shrinkPhoto(file);
      await savePhoto(id, blob);
      if (photoUrls[id]) URL.revokeObjectURL(photoUrls[id]);
      photoUrls[id] = URL.createObjectURL(blob);
      return true;
    } catch {
      familyError = $t('be_a_ba.family_photo_error');
      return false;
    }
  }

  async function addFamily(e: SubmitEvent) {
    e.preventDefault();
    familyError = null;
    const text = cleanFamilyWord(newWord);
    if (!text) {
      familyError = $t('be_a_ba.family_invalid');
      return;
    }
    if (saved.family.some((f) => plain(f.text) === plain(text))) {
      familyError = $t('be_a_ba.family_exists');
      return;
    }
    const id = 'fam-' + Date.now().toString(36);
    const photo = newPhoto ? await storePhoto(id, newPhoto) : false;
    if (newPhoto && !photo) return;
    saved.family = [...saved.family, { id, text, photo }];
    newWord = '';
    newPhoto = null;
    if (photoInput) photoInput.value = '';
    persist();
  }

  async function setPhoto(f: FamilyWord, file: File | undefined) {
    if (!file) return;
    familyError = null;
    if (await storePhoto(f.id, file)) {
      saved.family = saved.family.map((x) => (x.id === f.id ? { ...x, photo: true } : x));
      persist();
    }
  }

  async function removeFamily() {
    const f = removing;
    if (!f) return;
    saved.family = saved.family.filter((x) => x.id !== f.id);
    if (f.isChildName) {
      saved.childName = '';
      nameInput = '';
    }
    if (photoUrls[f.id]) {
      URL.revokeObjectURL(photoUrls[f.id]);
      delete photoUrls[f.id];
    }
    removing = null;
    // Grava antes de apagar a foto: a remoção não pode depender do IndexedDB
    // (assíncrono) — fechar o app no meio não pode trazer a palavra de volta.
    persist();
    try {
      await deletePhoto(f.id);
    } catch {
      // sem IndexedDB: nada a apagar
    }
  }

  function setStage(trail: Trail, value: string) {
    const p = progressOf(trail);
    setProgress(trail, value === 'auto' ? { ...p, locked: false } : { ...p, locked: true, stage: Number(value), goodRounds: 0 });
    persist();
  }

  function resetProgress() {
    const keep = { ageMode: saved.ageMode, childName: saved.childName, family: saved.family, settings: saved.settings };
    saved = { ...emptyState(), ...keep };
    persist();
  }

  /** Texto de um item do Leitner: palavra do banco, da família ou a letra (`letter:A`). */
  function wordText(key: string): string {
    const id = key.includes(':') ? key.slice(key.indexOf(':') + 1) : key;
    return WORDS.find((w) => w.id === id)?.text ?? saved.family.find((f) => f.id === id)?.text ?? id;
  }

  /** Palavras e cores do Inglês já dominadas (caixa ≥ 3), em inglês. */
  const englishKnown = $derived(
    Object.entries(saved.english.boxes)
      .filter(([k, b]) => b.box >= 3 && (k.startsWith('hear:') || k.startsWith('color:')))
      .map(([k]) => (k.startsWith('color:') ? k.slice(6) : (EN_WORDS.find((w) => `hear:${w.id}` === k)?.text ?? k)))
  );
  const letterBoxes = $derived(progressOf('letters').boxes);
  const numberBoxes = $derived(progressOf('numbers').boxes);
  const mastered = $derived(
    Object.entries(letterBoxes)
      .filter(([, b]) => b.box >= 3)
      .map(([id]) => wordText(id))
  );
  const struggling = $derived(
    Object.entries(letterBoxes)
      .filter(([, b]) => b.misses > 0 && b.box <= 2)
      .map(([id]) => wordText(id))
  );
  const KINDS = $derived<string[]>(isLittle ? ['tap', 'find_number', 'where'] : (['count', 'compare', 'add', 'sub'] satisfies NumberKind[]));

  // ───────── Teclado físico ─────────

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && gate) gate = null;
  }
</script>

<svelte:head>
  <title>{$t('be_a_ba.title')} — games.thiagotn.com</title>
  <meta name="description" content={$t('be_a_ba.desc')} />
  <meta name="theme-color" content="#8fd3ff" />
  <link rel="manifest" href="/be-a-ba/manifest.webmanifest" />
  <link rel="apple-touch-icon" href="/be-a-ba/apple-touch-icon.png" />
  <meta name="apple-mobile-web-app-title" content={$t('be_a_ba.title')} />
</svelte:head>

<svelte:window onkeydown={onKeydown} />

<div class="bab fixed inset-0 z-50 flex flex-col overflow-hidden text-crayon-ink" data-screen={round ? 'round' : reward ? 'reward' : screen}>
  {#if !ready}
    <!-- localStorage ainda não lido -->
  {:else if round && current}
    <!-- ── Rodada ─────────────────────────────────────────────────────── -->
    <header class="flex flex-none items-center gap-2 px-3 pb-2" style="padding-top:calc(env(safe-area-inset-top) + 8px)">
      <button type="button" class="icon-btn" onclick={quitRound} aria-label={$t('be_a_ba.quit')}>
        <X size={24} strokeWidth={2.6} />
      </button>
      <div class="flex flex-1 justify-center gap-1.5" aria-label={$t('be_a_ba.progress', { n: round.index + 1, total: ROUND_SIZE })}>
        {#each round.challenges as _, i (i)}
          <img src={imgUrl(STAR)} alt="" class="h-7 w-7 transition-[filter,opacity] {i < round.outcomes.length ? '' : 'opacity-40 grayscale'}" />
        {/each}
      </div>
      <button type="button" class="icon-btn" onclick={askCurrent} aria-label={$t('be_a_ba.repeat')}>
        <Volume2 size={24} strokeWidth={2.4} />
      </button>
    </header>

    <main class="flex min-h-0 flex-1 flex-col items-center gap-4 overflow-y-auto px-4 pb-6 pt-1" style="padding-bottom:calc(env(safe-area-inset-bottom) + 20px)">
      <button type="button" class="instruction" onclick={askCurrent}>
        <img src={imgUrl(MASCOT)} alt="" class="h-9 w-9" />
        {instruction(current).show}
      </button>

      {#key round.index}
        {#if round.little}
          {@const english = round.trail === 'english'}
          <LittleChallengeView
            challenge={current as LittleChallenge}
            sayNumber={(n) => (english ? say(EN_NUMBERS[n] ?? String(n), EN_LANG) : say(String(n)))}
            sayLetter={(l) => say(l.toLowerCase(), LETTERS_LANG)}
            sayWord={(w) => say(w, english ? EN_LANG : LETTERS_LANG)}
            onresult={onResult}
          />
        {:else if current.kind === 'letter'}
          <LetterChallengeView
            challenge={current}
            photoUrl={current.word.photo ? (photoUrls[current.word.photo] ?? null) : null}
            sayLetter={(l) => say(l, LETTERS_LANG)}
            sayHint={() => say(current.kind === 'letter' ? current.word.text : '', LETTERS_LANG, 0.55)}
            onresult={onResult}
          />
        {:else}
          <NumberChallengeView challenge={current as Exclude<Challenge, { kind: 'letter' }>} sayNumber={(n) => say(String(n))} sayHint={askCurrent} onresult={onResult} />
        {/if}
      {/key}
    </main>

    {#if praise}
      <div class="praise" role="status">
        <img src={imgUrl(STAR)} alt="" class="praise-star" />
        <p class="font-display text-[44px] leading-none text-white [text-shadow:0_4px_0_rgba(36,52,90,.35)]">{praise}</p>
        {#if praiseWord}
          <p class="praise-word" lang="en" data-praise-word>{praiseWord}</p>
        {/if}
      </div>
    {/if}
  {:else if reward}
    <!-- ── Prêmio da rodada ───────────────────────────────────────────── -->
    <div class="confetti" aria-hidden="true">
      {#each Array.from({ length: 36 }, (_, i) => i) as i (i)}
        <span style="--x:{(i * 37) % 100}%;--d:{(i % 9) * 0.18}s;--c:{['#ff5d8f', '#7b5cff', '#ffcf3f', '#43b85c', '#ff9b3d', '#2e9bff'][i % 6]};--r:{(i * 53) % 360}deg"></span>
      {/each}
    </div>
    <main class="relative flex flex-1 flex-col items-center justify-center gap-4 overflow-y-auto px-5 py-6 text-center">
      <h1 class="font-display text-[40px] leading-none">{$t('be_a_ba.round_done')}</h1>
      {#if reward.sticker}
        <div class="sticker-reveal">
          <img src={imgUrl(reward.sticker.img)} alt="" />
        </div>
        <p class="text-lg font-extrabold">{$t('be_a_ba.got_sticker')}</p>
        <p class="font-display text-[30px] leading-none text-crayon-grape" data-sticker={reward.sticker.id}>
          {$t(`be_a_ba.sticker.${reward.sticker.id}`)}
        </p>
      {:else}
        <img src={imgUrl(PARTY)} alt="" class="h-32 w-32" />
        <p class="text-lg font-extrabold">{$t('be_a_ba.album_full')}</p>
      {/if}
      {#if reward.nameSticker}
        <p class="flex items-center gap-2 rounded-2xl bg-white px-4 py-2 font-extrabold shadow-[0_4px_0_rgba(36,52,90,.12)]">
          <img src={imgUrl(NAME_STICKER.img)} alt="" class="h-8 w-8" /> {$t('be_a_ba.name_sticker')}
        </p>
      {/if}
      <p class="flex items-center gap-1.5 text-lg font-extrabold">
        <img src={imgUrl(STAR)} alt="" class="h-7 w-7" /> {$t('be_a_ba.stars_earned', { n: reward.stars })}
      </p>
      {#if reward.leveledUp}
        <p class="flex items-center gap-2 font-extrabold text-crayon-leaf">
          <img src={imgUrl(MEDAL)} alt="" class="h-8 w-8" /> {$t('be_a_ba.level_up')}
        </p>
      {/if}
      <div class="mt-2 flex w-full max-w-xs flex-col gap-2.5">
        <button type="button" class="big-btn bg-crayon-sun" onclick={() => reward && startRound(reward.trail)}>{$t('be_a_ba.play_again')}</button>
        <button type="button" class="big-btn bg-white" onclick={() => { reward = null; go('album'); }}>{$t('be_a_ba.see_album')}</button>
        <button type="button" class="big-btn bg-white" onclick={() => { reward = null; go('home'); }}>{$t('be_a_ba.tab_home')}</button>
      </div>
    </main>
  {:else}
    <!-- ── Abas ───────────────────────────────────────────────────────── -->
    <div class="min-h-0 flex-1 overflow-y-auto" style="padding-top:calc(env(safe-area-inset-top) + 12px)">
      {#if screen === 'home' && !saved.ageMode}
        <!-- Primeira abertura: para quem é? (dá para trocar na área dos pais) -->
        <main class="mx-auto flex max-w-md flex-col gap-5 px-4 pb-8" data-age-chooser>
          <div class="flex items-center gap-3">
            <img src={imgUrl(MASCOT)} alt="" class="mascot h-24 w-24 flex-none" />
            <span class="bubble">
              <span class="block font-display text-[26px] leading-tight">{$t('be_a_ba.hello')}</span>
              <span class="text-base font-bold text-crayon-muted">{$t('be_a_ba.age_question')}</span>
            </span>
          </div>
          <button type="button" class="trail bg-crayon-orange" onclick={() => chooseAge('little')}>
            <span class="trail-glyph text-[34px]">3–4</span>
            <span class="flex flex-col items-start">
              <span class="font-display text-[30px] leading-none">{$t('be_a_ba.age_little')}</span>
              <span class="text-sm font-extrabold opacity-90">{$t('be_a_ba.age_little_desc')}</span>
            </span>
          </button>
          <button type="button" class="trail bg-crayon-leaf" onclick={() => chooseAge('kids')}>
            <span class="trail-glyph text-[34px]">5–7</span>
            <span class="flex flex-col items-start">
              <span class="font-display text-[30px] leading-none">{$t('be_a_ba.age_kids')}</span>
              <span class="text-sm font-extrabold opacity-90">{$t('be_a_ba.age_kids_desc')}</span>
            </span>
          </button>
          <p class="text-center text-sm font-bold text-crayon-muted">{$t('be_a_ba.age_change_hint')}</p>
        </main>
      {:else if screen === 'home'}
        <main class="mx-auto flex max-w-md flex-col gap-5 px-4 pb-8" data-age={saved.ageMode}>
          <div class="flex items-center justify-between">
            <span class="pill"><img src={imgUrl(STAR)} alt="" class="h-6 w-6" /> <b data-stars>{saved.stars}</b></span>
            <button type="button" class="pill" onclick={() => go('album')}>
              <BookOpen size={18} /> <b>{saved.stickers.length}</b>
            </button>
          </div>

          <button
            type="button"
            class="flex items-center gap-3 text-left"
            onclick={() => say(`${saved.childName ? $t('be_a_ba.hello_name', { name: saved.childName }) : $t('be_a_ba.hello')} ${$t('be_a_ba.lets_play')}`)}
          >
            <img src={imgUrl(MASCOT)} alt="" class="mascot h-24 w-24 flex-none" />
            <span class="bubble">
              <span class="block font-display text-[26px] leading-tight">
                {saved.childName ? $t('be_a_ba.hello_name', { name: saved.childName }) : $t('be_a_ba.hello')}
              </span>
              <span class="text-base font-bold text-crayon-muted">{$t('be_a_ba.lets_play')}</span>
            </span>
          </button>

          <button type="button" class="trail bg-crayon-berry" onclick={() => startRound('letters')} disabled={!lettersAvailable}>
            <span class="trail-glyph">ABC</span>
            <span class="flex flex-col items-start">
              <span class="font-display text-[32px] leading-none">{$t('be_a_ba.trail_letters')}</span>
              <span class="text-sm font-extrabold opacity-90">
                {lettersAvailable ? $t('be_a_ba.stage', { n: progressOf('letters').stage + 1 }) : $t('be_a_ba.letters_soon')}
              </span>
            </span>
            <img src={imgUrl('1f347')} alt="" class="ml-auto h-14 w-14" />
          </button>

          <button type="button" class="trail bg-crayon-grape" onclick={() => startRound('numbers')}>
            <span class="trail-glyph">123</span>
            <span class="flex flex-col items-start">
              <span class="font-display text-[32px] leading-none">{$t('be_a_ba.trail_numbers')}</span>
              <span class="text-sm font-extrabold opacity-90">{$t('be_a_ba.stage', { n: progressOf('numbers').stage + 1 })}</span>
            </span>
            <img src={imgUrl('1f34e')} alt="" class="ml-auto h-14 w-14" />
          </button>

          <button type="button" class="trail bg-crayon-sea" onclick={() => startRound('english')}>
            <span class="trail-glyph" lang="en">Hi!</span>
            <span class="flex flex-col items-start">
              <span class="font-display text-[32px] leading-none">{$t('be_a_ba.trail_english')}</span>
              <span class="text-sm font-extrabold opacity-90">
                {$t('be_a_ba.stage', { n: progressOf('english').stage + 1 })} · {$t('be_a_ba.english_desc')}
              </span>
            </span>
            <img src={imgUrl('1f981')} alt="" class="ml-auto h-14 w-14" />
          </button>
        </main>
      {:else if screen === 'album'}
        <main class="mx-auto flex max-w-md flex-col gap-4 px-4 pb-8">
          <div>
            <h1 class="font-display text-[34px] leading-none">{$t('be_a_ba.album_title')}</h1>
            <p class="mt-1 font-bold text-crayon-muted">{$t('be_a_ba.album_count', { n: STICKERS.filter((s) => ownedSet.has(s.id)).length, total: STICKERS.length })}</p>
          </div>
          {#if !saved.stickers.length}
            <p class="rounded-2xl bg-white/70 p-4 font-bold text-crayon-muted">{$t('be_a_ba.album_empty')}</p>
          {/if}
          {#if ownedSet.has(NAME_STICKER.id)}
            <section class="album-page">
              <h2 class="page-title">{$t('be_a_ba.album_special')}</h2>
              <div class="grid grid-cols-3 gap-3">
                <figure class="slot owned">
                  <img src={imgUrl(NAME_STICKER.img)} alt="" />
                  <figcaption>{$t('be_a_ba.sticker.meu_nome')}</figcaption>
                </figure>
              </div>
            </section>
          {/if}
          {#each Array.from({ length: Math.ceil(STICKERS.length / PAGE_SIZE) }, (_, p) => p) as p (p)}
            <section class="album-page">
              <h2 class="page-title">{$t('be_a_ba.album_page', { n: p + 1 })}</h2>
              <div class="grid grid-cols-3 gap-3">
                {#each STICKERS.slice(p * PAGE_SIZE, (p + 1) * PAGE_SIZE) as s (s.id)}
                  {@const owned = ownedSet.has(s.id)}
                  <figure class="slot" class:owned data-sticker={s.id} data-owned={owned}>
                    <img src={imgUrl(s.img)} alt="" />
                    <figcaption>{owned ? $t(`be_a_ba.sticker.${s.id}`) : '?'}</figcaption>
                  </figure>
                {/each}
              </div>
            </section>
          {/each}
        </main>
      {:else if screen === 'parents'}
        <main class="mx-auto flex max-w-2xl flex-col gap-4 px-4 pb-8">
          <h1 class="font-display text-[34px] leading-none">{$t('be_a_ba.gate_title')}</h1>
          <p class="text-sm font-bold text-crayon-muted">{$t('be_a_ba.privacy')}</p>

          <div class="grid grid-cols-1 items-start gap-4 lg:grid-cols-2">
            <div class="flex flex-col gap-4">
              <section class="panel">
                <h2 class="panel-title">{$t('be_a_ba.child')}</h2>
                <label class="field-label" for="bab-child">{$t('be_a_ba.child_name')}</label>
                <div class="flex gap-2">
                  <input id="bab-child" class="field" bind:value={nameInput} maxlength="24" autocomplete="off" />
                  <button type="button" class="small-btn" onclick={saveChildName}>{$t('be_a_ba.save')}</button>
                </div>
                <p class="hint-text">{$t('be_a_ba.child_name_hint')}</p>
              </section>

              <section class="panel">
                <h2 class="panel-title">{$t('be_a_ba.family')}</h2>
                <p class="hint-text">{$t('be_a_ba.family_hint')}</p>
                {#if saved.family.length}
                  <ul class="mt-3 flex flex-col gap-2">
                    {#each saved.family as f (f.id)}
                      <li class="flex items-center gap-3 rounded-xl bg-crayon-cloud px-2.5 py-2" data-family={f.text}>
                        <label class="thumb" title={$t('be_a_ba.family_photo')}>
                          {#if photoUrls[f.id]}
                            <img src={photoUrls[f.id]} alt="" />
                          {:else}
                            <Camera size={20} />
                          {/if}
                          <input type="file" accept="image/*" class="sr-only" aria-label={`${$t('be_a_ba.family_photo')} ${f.text}`} onchange={(e) => setPhoto(f, e.currentTarget.files?.[0])} />
                        </label>
                        <span class="min-w-0 flex-1">
                          <b class="block truncate font-display text-xl tracking-wide">{f.text}</b>
                          {#if f.isChildName}<small class="font-bold text-crayon-muted">{$t('be_a_ba.family_child')}</small>{/if}
                        </span>
                        <button
                          type="button"
                          class="small-btn ghost"
                          onclick={() => { removing = f; confirmRemove = true; }}
                          aria-label={$t('be_a_ba.family_remove', { word: f.text })}
                        ><Trash2 size={16} /></button>
                      </li>
                    {/each}
                  </ul>
                {:else}
                  <p class="mt-2 font-bold text-crayon-muted">{$t('be_a_ba.family_empty')}</p>
                {/if}
                <form class="mt-3 flex flex-col gap-2" onsubmit={addFamily}>
                  <label class="field-label" for="bab-word">{$t('be_a_ba.family_word')}</label>
                  <input id="bab-word" class="field uppercase" bind:value={newWord} maxlength="24" autocomplete="off" />
                  <span class="field-label">{$t('be_a_ba.family_photo')}</span>
                  <label class="small-btn light self-start" for="bab-photo">
                    <Camera size={18} /> {newPhoto ? newPhoto.name : $t('be_a_ba.family_pick_photo')}
                  </label>
                  <input id="bab-photo" bind:this={photoInput} type="file" accept="image/*" class="sr-only" onchange={(e) => (newPhoto = e.currentTarget.files?.[0] ?? null)} />
                  {#if familyError}<p class="text-sm font-bold text-ketchup" role="alert">{familyError}</p>{/if}
                  <button type="submit" class="small-btn self-start">{$t('be_a_ba.family_add')}</button>
                </form>
              </section>
            </div>

            <div class="flex flex-col gap-4">
              <section class="panel">
                <h2 class="panel-title">{$t('be_a_ba.levels')}</h2>
                <label class="field-label mt-2" for="bab-age">{$t('be_a_ba.age')}</label>
                <select id="bab-age" class="field" value={saved.ageMode ?? 'kids'} onchange={(e) => chooseAge(e.currentTarget.value as AgeMode)}>
                  <option value="little">{$t('be_a_ba.age_little_option')}</option>
                  <option value="kids">{$t('be_a_ba.age_kids_option')}</option>
                </select>
                <p class="hint-text">{$t('be_a_ba.levels_hint')}</p>
                {#each [['letters', stageCountOf('letters')], ['numbers', stageCountOf('numbers')], ['english', stageCountOf('english')]] as [trail, count] (`${saved.ageMode}-${trail}`)}
                  {@const p = progressOf(trail as Trail)}
                  <label class="field-label mt-3" for="bab-stage-{trail}">{$t(trail === 'letters' ? 'be_a_ba.trail_letters' : 'be_a_ba.trail_numbers')}</label>
                  <select id="bab-stage-{trail}" class="field" value={p.locked ? String(p.stage) : 'auto'} onchange={(e) => setStage(trail as Trail, e.currentTarget.value)}>
                    <option value="auto">{$t('be_a_ba.level_auto', { n: p.stage + 1 })}</option>
                    {#each Array.from({ length: count as number }, (_, i) => i) as i (i)}
                      <option value={String(i)}>{$t('be_a_ba.level_fixed', { n: i + 1, name: $t(`be_a_ba.${isLittle && trail !== 'english' ? 'little_' : ''}stage_${trail}_${i}`) })}</option>
                    {/each}
                  </select>
                {/each}
              </section>

              <section class="panel">
                <h2 class="panel-title">{$t('be_a_ba.voice')}</h2>
                <label class="toggle" for="bab-voice">
                  {$t('be_a_ba.voice_on')}
                  <input id="bab-voice" type="checkbox" checked={saved.settings.voice} onchange={(e) => { saved.settings.voice = e.currentTarget.checked; persist(); }} />
                </label>
                <label class="field-label mt-2" for="bab-rate">{$t('be_a_ba.voice_rate')}</label>
                <div class="flex items-center gap-2 text-sm font-bold">
                  {$t('be_a_ba.voice_slow')}
                  <input id="bab-rate" type="range" min="0.6" max="1.1" step="0.05" class="flex-1" value={saved.settings.rate} onchange={(e) => { saved.settings.rate = Number(e.currentTarget.value); persist(); }} />
                  {$t('be_a_ba.voice_normal')}
                </div>
                <button type="button" class="small-btn mt-3" onclick={() => say($t('be_a_ba.voice_sample'))}>
                  <Volume2 size={16} /> {$t('be_a_ba.voice_test')}
                </button>
              </section>

              <section class="panel">
                <h2 class="panel-title">{$t('be_a_ba.report')}</h2>
                <dl class="grid grid-cols-4 gap-2 text-center">
                  <div><dt class="stat-label">{$t('be_a_ba.report_rounds')}</dt><dd class="stat">{saved.rounds}</dd></div>
                  <div><dt class="stat-label">{$t('be_a_ba.report_stars')}</dt><dd class="stat">{saved.stars}</dd></div>
                  <div><dt class="stat-label">{$t('be_a_ba.report_stickers')}</dt><dd class="stat">{saved.stickers.length}</dd></div>
                  <div><dt class="stat-label">{$t('be_a_ba.report_today')}</dt><dd class="stat">{$t('be_a_ba.minutes', { n: Math.round((saved.playtime[today] ?? 0) / 60) })}</dd></div>
                </dl>
                <h3 class="field-label mt-3">{$t('be_a_ba.mastered')}</h3>
                <p class="word-list">{mastered.length ? mastered.join(' · ') : $t('be_a_ba.none_yet')}</p>
                <h3 class="field-label mt-3">{$t('be_a_ba.struggling')}</h3>
                <p class="word-list">{struggling.length ? struggling.join(' · ') : $t('be_a_ba.none_yet')}</p>
                <h3 class="field-label mt-3">{$t('be_a_ba.numbers_report')}</h3>
                <ul class="text-sm font-bold">
                  {#each KINDS as kind (kind)}
                    {#if numberBoxes[kind]}
                      <li>{$t(`be_a_ba.kind.${kind}`)} — {$t('be_a_ba.box', { n: numberBoxes[kind].box })}</li>
                    {/if}
                  {/each}
                  {#if !KINDS.some((k) => numberBoxes[k])}<li class="text-crayon-muted">{$t('be_a_ba.none_yet')}</li>{/if}
                </ul>
                <h3 class="field-label mt-3">{$t('be_a_ba.english_report')}</h3>
                <p class="word-list" lang="en" data-english-known>{englishKnown.length ? englishKnown.join(' · ') : $t('be_a_ba.none_yet')}</p>
                <p class="hint-text">{$t('be_a_ba.english_hint')}</p>
              </section>

              <section class="panel">
                <div class="toggle">
                  <span>{$t('be_a_ba.language')}</span>
                  <LanguageSwitcher />
                </div>
                <p class="hint-text">{$t('be_a_ba.letters_language_hint')}</p>
                <button type="button" class="small-btn danger mt-3" onclick={() => (confirmReset = true)}>
                  <Trash2 size={16} /> {$t('be_a_ba.reset')}
                </button>
                <p class="hint-text mt-3">{$t('be_a_ba.credits')}</p>
              </section>
            </div>
          </div>
        </main>
      {/if}
    </div>

    <nav class="relative z-10 flex-none border-t-4 border-white/70 bg-crayon-cloud" style="padding-bottom:calc(env(safe-area-inset-bottom) + 6px)">
      <div class="mx-auto grid max-w-md grid-cols-3">
        {#each TABS as item (item.key)}
          <button
            type="button"
            onclick={() => go(item.key)}
            aria-current={screen === item.key ? 'page' : undefined}
            class="flex flex-col items-center gap-[3px] pb-1.5 pt-2.5 text-[12px] font-extrabold transition-colors
              {screen === item.key ? 'text-crayon-berry' : 'text-crayon-muted'}"
          >
            <item.icon size={24} strokeWidth={2.4} />
            {$t(item.labelKey)}
          </button>
        {/each}
      </div>
    </nav>
  {/if}

  {#if toast}
    <p class="toast" role="status">{toast}</p>
  {/if}

  {#if gate}
    <div class="overlay" role="presentation" onclick={(e) => { if (e.target === e.currentTarget) gate = null; }}>
      <div class="card" role="dialog" aria-modal="true" aria-labelledby="bab-gate-title">
        <h2 id="bab-gate-title" class="font-display text-[28px] leading-none">{$t('be_a_ba.gate_title')}</h2>
        <p class="mt-2 font-bold">{$t('be_a_ba.gate_ask', { n: $t(`be_a_ba.n.${gate.target}`) })}</p>
        <div class="mt-4 grid grid-cols-2 gap-2.5">
          {#each gate.options as n (n)}
            <button type="button" class="big-btn bg-white text-2xl" onclick={() => pickGate(n)}>{n}</button>
          {/each}
        </div>
        {#if gateError}<p class="mt-3 text-sm font-bold text-ketchup" role="alert">{$t('be_a_ba.gate_wrong')}</p>{/if}
        <button type="button" class="btn btn-ghost mt-4" onclick={() => (gate = null)}>{$t('be_a_ba.cancel')}</button>
      </div>
    </div>
  {/if}

  <ConfirmDialog
    bind:open={confirmRemove}
    message={$t('be_a_ba.family_remove_confirm', { word: removing?.text ?? '' })}
    confirmLabel={$t('be_a_ba.family_remove', { word: removing?.text ?? '' })}
    onConfirm={removeFamily}
  />
  <ConfirmDialog bind:open={confirmReset} message={$t('be_a_ba.reset_confirm')} confirmLabel={$t('be_a_ba.reset')} onConfirm={resetProgress} />
</div>

<style>
  .bab {
    background:
      radial-gradient(120% 60% at 85% -10%, rgba(255, 207, 63, 0.55), transparent 55%),
      linear-gradient(180deg, theme('colors.crayon.sky') 0%, theme('colors.crayon.cloud') 70%);
    font-synthesis: none;
    user-select: none;
    -webkit-user-select: none;
  }
  .bab :global(input),
  .bab :global(select) {
    user-select: text;
    -webkit-user-select: text;
  }

  .icon-btn {
    display: grid;
    place-items: center;
    width: 48px;
    height: 48px;
    flex: none;
    border-radius: 16px;
    background: rgba(255, 255, 255, 0.75);
    box-shadow: 0 4px 0 rgba(36, 52, 90, 0.15);
  }
  .instruction {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 14px 6px 8px;
    border-radius: 9999px;
    background: rgba(255, 255, 255, 0.8);
    font-size: 17px;
    font-weight: 800;
  }

  .pill {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 6px 14px 6px 8px;
    border-radius: 9999px;
    background: rgba(255, 255, 255, 0.8);
    font-size: 18px;
    box-shadow: 0 3px 0 rgba(36, 52, 90, 0.12);
  }
  .bubble {
    position: relative;
    padding: 12px 16px;
    border-radius: 22px;
    background: #fff;
    box-shadow: 0 5px 0 rgba(36, 52, 90, 0.12);
  }
  .bubble::before {
    content: '';
    position: absolute;
    left: -10px;
    top: 50%;
    border: 10px solid transparent;
    border-right-color: #fff;
    border-left: 0;
    transform: translateY(-50%);
  }
  .mascot {
    animation: bob 2.6s ease-in-out infinite;
  }
  .trail {
    display: flex;
    align-items: center;
    gap: 14px;
    min-height: 116px;
    padding: 16px 18px;
    border-radius: 28px;
    color: #fff;
    text-align: left;
    box-shadow: 0 8px 0 rgba(36, 52, 90, 0.22);
    transition: transform 0.1s;
  }
  .trail:active {
    transform: translateY(4px);
  }
  .trail:disabled {
    filter: grayscale(0.7);
    opacity: 0.7;
  }
  .trail-glyph {
    display: grid;
    place-items: center;
    width: 76px;
    height: 76px;
    flex: none;
    border-radius: 22px;
    background: rgba(255, 255, 255, 0.25);
    font-family: theme('fontFamily.display');
    font-size: 28px;
  }

  .big-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    height: 56px;
    border-radius: 18px;
    font-family: theme('fontFamily.display');
    font-size: 22px;
    color: theme('colors.crayon.ink');
    box-shadow: 0 5px 0 rgba(36, 52, 90, 0.18);
  }

  .praise {
    position: absolute;
    inset: 0;
    z-index: 20;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 10px;
    background: rgba(143, 211, 255, 0.55);
    animation: fade 0.2s ease-out;
  }
  .praise-word {
    padding: 4px 18px;
    border-radius: 9999px;
    background: #fff;
    color: theme('colors.crayon.ink');
    font-family: theme('fontFamily.display');
    font-size: 34px;
    line-height: 1.1;
    box-shadow: 0 4px 0 rgba(36, 52, 90, 0.18);
  }
  .praise-star {
    width: 140px;
    height: 140px;
    animation: star-pop 0.6s cubic-bezier(0.3, 1.6, 0.5, 1);
  }

  .sticker-reveal {
    display: grid;
    place-items: center;
    width: 180px;
    height: 180px;
    border-radius: 36px;
    background: #fff;
    border: 6px solid theme('colors.crayon.sun');
    box-shadow: 0 8px 0 rgba(36, 52, 90, 0.18);
    animation: reveal 0.8s cubic-bezier(0.3, 1.5, 0.5, 1);
  }
  .sticker-reveal img {
    width: 130px;
    height: 130px;
  }
  .confetti {
    position: absolute;
    inset: 0;
    overflow: hidden;
    pointer-events: none;
  }
  .confetti span {
    position: absolute;
    top: -20px;
    left: var(--x);
    width: 10px;
    height: 16px;
    border-radius: 3px;
    background: var(--c);
    transform: rotate(var(--r));
    animation: fall 2.6s ease-in var(--d) forwards;
  }

  .album-page {
    padding: 14px;
    border-radius: 24px;
    background: rgba(255, 255, 255, 0.7);
  }
  .page-title {
    margin-bottom: 10px;
    font-size: 12px;
    font-weight: 800;
    letter-spacing: 0.14em;
    text-transform: uppercase;
    color: theme('colors.crayon.muted');
  }
  .slot {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    padding: 10px 4px 8px;
    border-radius: 18px;
    border: 3px dashed rgba(36, 52, 90, 0.18);
  }
  .slot img {
    width: 58px;
    height: 58px;
    filter: brightness(0);
    opacity: 0.12;
  }
  .slot figcaption {
    font-size: 13px;
    font-weight: 800;
    color: theme('colors.crayon.muted');
  }
  .slot.owned {
    border: 3px solid #fff;
    background: #fff;
    box-shadow: 0 4px 0 rgba(36, 52, 90, 0.1);
  }
  .slot.owned img {
    filter: none;
    opacity: 1;
  }
  .slot.owned figcaption {
    color: theme('colors.crayon.ink');
  }

  .panel {
    padding: 16px;
    border-radius: 22px;
    background: #fff;
    box-shadow: 0 5px 0 rgba(36, 52, 90, 0.1);
  }
  .panel-title {
    margin-bottom: 6px;
    font-family: theme('fontFamily.display');
    font-size: 22px;
  }
  .field-label {
    display: block;
    margin-bottom: 4px;
    font-size: 12px;
    font-weight: 800;
    letter-spacing: 0.1em;
    text-transform: uppercase;
    color: theme('colors.crayon.muted');
  }
  .field {
    width: 100%;
    min-width: 0;
    height: 44px;
    padding: 0 12px;
    border-radius: 12px;
    border: 2px solid rgba(36, 52, 90, 0.18);
    background: #fff;
    font-size: 16px;
    font-weight: 700;
  }
  .field:focus {
    border-color: theme('colors.crayon.grape');
    outline: none;
  }
  .hint-text {
    margin-top: 6px;
    font-size: 13px;
    font-weight: 600;
    color: theme('colors.crayon.muted');
  }
  .small-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    height: 44px;
    padding: 0 14px;
    flex: none;
    border-radius: 12px;
    background: theme('colors.crayon.grape');
    color: #fff;
    font-weight: 800;
  }
  .small-btn.light {
    max-width: 100%;
    overflow: hidden;
    background: theme('colors.crayon.cloud');
    color: theme('colors.crayon.ink');
    cursor: pointer;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  .small-btn.ghost {
    width: 44px;
    padding: 0;
    background: transparent;
    color: theme('colors.ketchup');
  }
  .small-btn.danger {
    background: transparent;
    color: theme('colors.ketchup');
    box-shadow: inset 0 0 0 2px theme('colors.ketchup');
  }
  .thumb {
    display: grid;
    place-items: center;
    width: 48px;
    height: 48px;
    flex: none;
    overflow: hidden;
    border-radius: 12px;
    background: #fff;
    color: theme('colors.crayon.muted');
    cursor: pointer;
  }
  .thumb img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
  .word-list {
    font-size: 14px;
    font-weight: 700;
    line-height: 1.5;
  }
  .stat-label {
    color: theme('colors.crayon.muted');
    letter-spacing: 0.06em;
    font-size: 10px;
  }
  .stat {
    font-size: 22px;
  }

  @keyframes bob {
    50% {
      transform: translateY(-6px) rotate(-3deg);
    }
  }
  @keyframes fade {
    from {
      opacity: 0;
    }
  }
  @keyframes star-pop {
    0% {
      transform: scale(0.2) rotate(-40deg);
    }
    100% {
      transform: scale(1) rotate(0);
    }
  }
  @keyframes reveal {
    0% {
      transform: scale(0.1) rotate(-180deg);
      opacity: 0;
    }
    100% {
      transform: scale(1) rotate(0);
      opacity: 1;
    }
  }
  @keyframes fall {
    to {
      transform: translateY(110vh) rotate(calc(var(--r) + 540deg));
    }
  }
</style>
