<script lang="ts">
  /**
   * Um desafio da trilha das Letras: imagem + casas da palavra + a forma de
   * responder do modo (`first` 3 letras, `build` bandeja, `type` teclado ABC).
   *
   * Erro não pune: a letra errada balança e a criança tenta de novo. Depois de
   * 2 erros na mesma casa a resposta certa brilha (conta como dica).
   */
  import { Lightbulb } from 'lucide-svelte';
  import Picture from './Picture.svelte';
  import { KEYBOARD, lettersOf, sameLetter, type LetterChallenge } from './letters';
  import { plain } from './words';
  import { t } from '$lib/i18n';

  let {
    challenge,
    photoUrl = null,
    sayLetter,
    sayHint,
    onresult,
  }: {
    challenge: LetterChallenge;
    photoUrl?: string | null;
    /** Fala uma letra ao ser colocada. */
    sayLetter: (letter: string) => void;
    /** Fala a palavra devagar (dica). */
    sayHint: () => void;
    onresult: (mistakes: number, hinted: boolean) => void;
  } = $props();

  const AUTO_HINT_AFTER = 2;

  // A página remonta este componente a cada desafio (`{#key}`), então o
  // estado nasce do desafio e não precisa ser zerado.
  const letters = $derived(lettersOf(challenge.word.text));
  let pos = $state(0);
  let used = $state<boolean[]>([]);
  let wrong = $state<string[]>([]);
  let mistakes = $state(0);
  let mistakesHere = $state(0);
  let hinted = $state(false);
  let glow = $state<string | number | null>(null);
  let shaking = $state<string | number | null>(null);
  let finished = $state(false);

  /** O que está certo agora: a letra (first/type) ou um índice livre da bandeja (build). */
  function target(): string | number | null {
    if (challenge.mode === 'first') return plain(letters[0]);
    const want = letters[pos];
    if (challenge.mode === 'build') return challenge.tray.findIndex((l, i) => !used[i] && sameLetter(want, l));
    return plain(want);
  }

  function shake(id: string | number) {
    shaking = null;
    queueMicrotask(() => (shaking = id));
    setTimeout(() => {
      if (shaking === id) shaking = null;
    }, 360);
  }

  function miss(id: string | number) {
    mistakes++;
    mistakesHere++;
    shake(id);
    if (mistakesHere >= AUTO_HINT_AFTER) {
      hinted = true;
      glow = target();
    }
  }

  function finish() {
    finished = true;
    glow = null;
    setTimeout(() => onresult(mistakes, hinted), 450);
  }

  function advance(letter: string) {
    sayLetter(letter);
    pos++;
    mistakesHere = 0;
    glow = null;
    if (pos >= letters.length) finish();
  }

  function pickFirst(letter: string) {
    if (finished || wrong.includes(letter)) return;
    if (sameLetter(letters[0], letter)) {
      pos = 1;
      sayLetter(letters[0]);
      finish();
    } else {
      wrong = [...wrong, letter];
      miss(letter);
    }
  }

  function pickTile(i: number) {
    if (finished || used[i]) return;
    if (sameLetter(letters[pos], challenge.tray[i])) {
      used[i] = true;
      advance(letters[pos]);
    } else miss(i);
  }

  function pickKey(key: string) {
    if (finished) return;
    if (sameLetter(letters[pos], key)) advance(letters[pos]);
    else miss(key);
  }

  function askHint() {
    if (finished) return;
    hinted = true;
    glow = target();
    sayHint();
  }

  function onKeydown(e: KeyboardEvent) {
    if (challenge.mode !== 'type' || e.ctrlKey || e.metaKey || e.altKey || e.key.length !== 1) return;
    const key = e.key.toLocaleUpperCase('pt-BR');
    if (!/\p{L}/u.test(key)) return;
    e.preventDefault();
    pickKey(key);
  }

  /** Casa `i` mostra a letra? (`first`: tudo menos a primeira até acertar.) */
  const shown = (i: number) => (challenge.mode === 'first' ? i > 0 || pos > 0 : i < pos);
</script>

<svelte:window onkeydown={onKeydown} />

<div class="challenge" data-mode={challenge.mode} data-answer={challenge.word.text}>
  <div class="picture-card">
    <Picture word={challenge.word} {photoUrl} />
    <button type="button" class="hint" onclick={askHint} aria-label={$t('be_a_ba.hint')} disabled={finished}>
      <Lightbulb size={22} strokeWidth={2.4} />
    </button>
  </div>

  <div class="slots" style="--n:{letters.length}" aria-label={$t('be_a_ba.slots')}>
    {#each letters as letter, i (i)}
      <span class="slot" class:filled={shown(i)} class:current={!finished && challenge.mode !== 'first' && i === pos} class:ask={challenge.mode === 'first' && i === 0 && pos === 0}>
        {shown(i) ? letter : challenge.mode === 'first' && i === 0 ? '?' : ''}
      </span>
    {/each}
  </div>

  {#if challenge.mode === 'first'}
    <div class="options">
      {#each challenge.options as letter (letter)}
        <button
          type="button"
          class="big-letter"
          class:wrong={wrong.includes(letter)}
          class:glow={glow === letter}
          class:shake={shaking === letter}
          class:right={finished && sameLetter(letters[0], letter)}
          onclick={() => pickFirst(letter)}
          aria-label={$t('be_a_ba.letter', { letter })}
        >{letter}</button>
      {/each}
    </div>
  {:else if challenge.mode === 'build'}
    <div class="tray">
      {#each challenge.tray as letter, i (i)}
        <button
          type="button"
          class="tile"
          class:used={used[i]}
          class:glow={glow === i}
          class:shake={shaking === i}
          onclick={() => pickTile(i)}
          disabled={used[i]}
          aria-label={$t('be_a_ba.letter', { letter })}
        >{letter}</button>
      {/each}
    </div>
  {:else}
    <div class="keyboard" role="group" aria-label={$t('be_a_ba.keyboard')}>
      {#each KEYBOARD as key (key)}
        <button
          type="button"
          class="key"
          class:glow={glow === key}
          class:shake={shaking === key}
          onclick={() => pickKey(key)}
          aria-label={$t('be_a_ba.letter', { letter: key })}
        >{key}</button>
      {/each}
    </div>
  {/if}
</div>

<style>
  .challenge {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
    width: 100%;
  }
  .picture-card {
    position: relative;
    padding: 14px;
    border-radius: 28px;
    background: #fff;
    box-shadow: 0 6px 0 rgba(36, 52, 90, 0.12);
  }
  .hint {
    position: absolute;
    right: -10px;
    top: -10px;
    display: grid;
    place-items: center;
    width: 44px;
    height: 44px;
    border-radius: 50%;
    background: theme('colors.crayon.sun');
    color: theme('colors.crayon.ink');
    box-shadow: 0 4px 0 rgba(36, 52, 90, 0.2);
  }
  .hint:disabled {
    opacity: 0.5;
  }

  .slots {
    display: grid;
    grid-template-columns: repeat(var(--n), minmax(0, 46px));
    gap: 5px;
    width: 100%;
    justify-content: center;
  }
  .slot {
    display: grid;
    place-items: center;
    aspect-ratio: 0.85;
    border-radius: 10px;
    border: 3px dashed rgba(36, 52, 90, 0.25);
    background: rgba(255, 255, 255, 0.55);
    color: theme('colors.crayon.ink');
    font-family: theme('fontFamily.display');
    font-size: clamp(20px, calc(70vw / var(--n)), 34px);
    line-height: 1;
  }
  .slot.filled {
    border: 3px solid theme('colors.crayon.leaf');
    background: #fff;
    animation: pop 0.25s ease-out;
  }
  .slot.current {
    border-color: theme('colors.crayon.grape');
    border-style: solid;
    animation: breathe 1.4s ease-in-out infinite;
  }
  .slot.ask {
    border-color: theme('colors.crayon.berry');
    border-style: solid;
    color: theme('colors.crayon.berry');
  }

  .options {
    display: flex;
    gap: 14px;
  }
  .big-letter {
    width: 84px;
    height: 84px;
    border-radius: 22px;
    background: #fff;
    color: theme('colors.crayon.ink');
    font-family: theme('fontFamily.display');
    font-size: 48px;
    box-shadow: 0 6px 0 rgba(36, 52, 90, 0.18);
    transition: transform 0.1s;
  }
  .big-letter:active,
  .tile:active,
  .key:active {
    transform: translateY(3px);
  }
  .big-letter.wrong {
    opacity: 0.35;
  }
  .big-letter.right {
    background: theme('colors.crayon.leaf');
    color: #fff;
  }

  .tray {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: 10px;
    max-width: 380px;
  }
  .tile {
    width: 58px;
    height: 62px;
    border-radius: 14px;
    background: theme('colors.crayon.sun');
    color: theme('colors.crayon.ink');
    font-family: theme('fontFamily.display');
    font-size: 34px;
    box-shadow: 0 5px 0 #d9a514;
  }
  .tile.used {
    visibility: hidden;
  }

  .keyboard {
    display: grid;
    grid-template-columns: repeat(7, minmax(0, 1fr));
    gap: 6px;
    width: 100%;
    max-width: 420px;
  }
  .key {
    height: 50px;
    border-radius: 12px;
    background: #fff;
    color: theme('colors.crayon.ink');
    font-family: theme('fontFamily.display');
    font-size: 26px;
    box-shadow: 0 4px 0 rgba(36, 52, 90, 0.18);
  }

  .glow {
    outline: 4px solid theme('colors.crayon.sun');
    outline-offset: 2px;
    /* Pulsa a luz, não o tamanho: o alvo não pode fugir do dedo. */
    animation: glow 0.9s ease-in-out infinite;
  }
  .tile.glow {
    outline-color: theme('colors.crayon.grape');
  }
  .shake {
    animation: shake 0.34s;
  }

  @keyframes shake {
    0%,
    100% {
      transform: translateX(0);
    }
    25% {
      transform: translateX(-7px) rotate(-4deg);
    }
    75% {
      transform: translateX(7px) rotate(4deg);
    }
  }
  @keyframes glow {
    50% {
      outline-offset: 6px;
      box-shadow: 0 0 0 10px rgba(255, 207, 63, 0.45);
    }
  }
  @keyframes breathe {
    50% {
      transform: scale(1.06);
    }
  }
  @keyframes pop {
    0% {
      transform: scale(0.7);
    }
    70% {
      transform: scale(1.12);
    }
    100% {
      transform: scale(1);
    }
  }
</style>
