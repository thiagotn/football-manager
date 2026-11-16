<script lang="ts">
  /**
   * Um desafio do modo Pequenos (3–4 anos). Nada para ler: tudo é figura,
   * letra grande ou número com bolinhas, e cada toque é falado.
   *
   * Mesma regra das outras trilhas: erro não pune (balança e fica apagado) e,
   * depois de 2 erros, a resposta certa pulsa. No `tap` não existe erro — a
   * criança conta tocando e cada objeto ganha o seu número.
   */
  import Objects from './Objects.svelte';
  import { imgUrl } from './art';
  import { littleAnswer, type LittleChallenge } from './little';
  import { t } from '$lib/i18n';

  let {
    challenge,
    sayNumber,
    sayLetter,
    sayWord,
    onresult,
  }: {
    challenge: LittleChallenge;
    sayNumber: (n: number) => void;
    sayLetter: (l: string) => void;
    sayWord: (w: string) => void;
    onresult: (mistakes: number, hinted: boolean) => void;
  } = $props();

  const AUTO_HINT_AFTER = 2;
  const answer = $derived(littleAnswer(challenge));

  let mistakes = $state(0);
  let hinted = $state(false);
  let wrong = $state<(string | number)[]>([]);
  let glow = $state<string | number | null>(null);
  let shaking = $state<string | number | null>(null);
  let finished = $state(false);
  /** `tap`: ordem em que cada objeto foi tocado (índice → número dito). */
  let counted = $state<Record<number, number>>({});

  function finish() {
    finished = true;
    glow = null;
    setTimeout(() => onresult(mistakes, hinted), 500);
  }

  function pick(value: string | number) {
    if (finished || wrong.includes(value)) return;
    if (value === answer) return finish();
    mistakes++;
    wrong = [...wrong, value];
    shaking = value;
    setTimeout(() => {
      if (shaking === value) shaking = null;
    }, 360);
    if (mistakes >= AUTO_HINT_AFTER) {
      hinted = true;
      glow = answer;
    }
  }

  function tapObject(i: number) {
    if (finished || challenge.kind !== 'tap' || counted[i]) return;
    const n = Object.keys(counted).length + 1;
    counted[i] = n;
    sayNumber(n);
    if (n === challenge.n) finish();
  }

  const dots = (n: number) => Array.from({ length: n }, (_, i) => i);
</script>

<div class="little" data-little-kind={challenge.kind} data-answer={answer}>
  {#if challenge.kind === 'hear'}
    <div class="choices" style="--n:{challenge.options.length}">
      {#each challenge.options as option (option.id)}
        <button
          type="button"
          class="choice"
          class:wrong={wrong.includes(option.id)}
          class:glow={glow === option.id}
          class:shake={shaking === option.id}
          class:right={finished && option.id === answer}
          onclick={() => {
            sayWord(option.text);
            pick(option.id);
          }}
          aria-label={option.text}
          data-option={option.id}
        >
          <img src={imgUrl(option.img)} alt="" draggable="false" />
        </button>
      {/each}
    </div>
  {:else if challenge.kind === 'pick_letter'}
    <div class="target" aria-hidden="true">{challenge.letter}</div>
    <div class="choices" style="--n:{challenge.options.length}">
      {#each challenge.options as letter (letter)}
        <button
          type="button"
          class="choice letter"
          class:wrong={wrong.includes(letter)}
          class:glow={glow === letter}
          class:shake={shaking === letter}
          class:right={finished && letter === answer}
          onclick={() => {
            sayLetter(letter);
            pick(letter);
          }}
          aria-label={$t('be_a_ba.letter', { letter })}
        >{letter}</button>
      {/each}
    </div>
  {:else if challenge.kind === 'tap'}
    <div class="tap-field">
      {#each dots(challenge.n) as i (i)}
        <button
          type="button"
          class="tap-object"
          class:counted={!!counted[i]}
          onclick={() => tapObject(i)}
          aria-label={counted[i] ? String(counted[i]) : $t('be_a_ba.tap_object')}
        >
          <img src={imgUrl(challenge.item)} alt="" draggable="false" />
          {#if counted[i]}<span class="badge">{counted[i]}</span>{/if}
        </button>
      {/each}
    </div>
  {:else if challenge.kind === 'find_number'}
    <div class="choices" style="--n:{challenge.options.length}">
      {#each challenge.options as n (n)}
        <button
          type="button"
          class="choice number"
          class:wrong={wrong.includes(n)}
          class:glow={glow === n}
          class:shake={shaking === n}
          class:right={finished && n === answer}
          onclick={() => {
            sayNumber(n);
            pick(n);
          }}
          aria-label={String(n)}
        >
          <span class="numeral">{n}</span>
          <span class="dots">{#each dots(n) as d (d)}<i></i>{/each}</span>
        </button>
      {/each}
    </div>
  {:else}
    <div class="choices" style="--n:2">
      {#each challenge.groups as g, i (i)}
        <button
          type="button"
          class="choice group"
          class:wrong={wrong.includes(i)}
          class:glow={glow === i}
          class:shake={shaking === i}
          class:right={finished && i === answer}
          onclick={() => {
            sayNumber(g);
            pick(i);
          }}
          aria-label={$t(i === 0 ? 'be_a_ba.side_left' : 'be_a_ba.side_right')}
        >
          <Objects item={challenge.item} n={g} size={40} perRow={3} />
        </button>
      {/each}
    </div>
  {/if}
</div>

<style>
  .little {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 22px;
    width: 100%;
  }
  .choices {
    display: grid;
    grid-template-columns: repeat(var(--n), minmax(0, 1fr));
    gap: 14px;
    width: 100%;
    max-width: 440px;
  }
  .choice {
    display: grid;
    place-items: center;
    min-height: 150px;
    padding: 14px;
    border-radius: 30px;
    background: #fff;
    box-shadow: 0 7px 0 rgba(36, 52, 90, 0.16);
    border: 5px solid transparent;
    transition: transform 0.1s;
  }
  .choice img {
    width: min(120px, 100%);
    aspect-ratio: 1;
    user-select: none;
  }
  .choice:active {
    transform: translateY(4px);
  }
  .choice.wrong {
    opacity: 0.3;
  }
  .choice.right {
    border-color: theme('colors.crayon.leaf');
  }
  .letter {
    min-height: 120px;
    font-family: theme('fontFamily.display');
    font-size: 72px;
    line-height: 1;
    color: theme('colors.crayon.ink');
  }
  .target {
    display: grid;
    place-items: center;
    width: 150px;
    height: 150px;
    border-radius: 36px;
    background: theme('colors.crayon.berry');
    color: #fff;
    font-family: theme('fontFamily.display');
    font-size: 110px;
    line-height: 1;
    box-shadow: 0 8px 0 rgba(36, 52, 90, 0.2);
  }
  .number {
    gap: 10px;
    min-height: 160px;
  }
  .numeral {
    font-family: theme('fontFamily.display');
    font-size: 76px;
    line-height: 1;
    color: theme('colors.crayon.ink');
  }
  .dots {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: 5px;
    max-width: 80px;
  }
  .dots i {
    width: 12px;
    height: 12px;
    border-radius: 50%;
    background: theme('colors.crayon.grape');
  }
  .group {
    min-height: 170px;
    background: theme('colors.crayon.cloud');
  }

  .tap-field {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: 16px;
    max-width: 380px;
    padding: 20px;
    border-radius: 30px;
    background: #fff;
    box-shadow: 0 7px 0 rgba(36, 52, 90, 0.12);
  }
  .tap-object {
    position: relative;
    display: grid;
    place-items: center;
    width: 92px;
    height: 92px;
    border-radius: 24px;
    background: theme('colors.crayon.cloud');
    transition: transform 0.12s;
  }
  .tap-object img {
    width: 66px;
    height: 66px;
  }
  .tap-object:active {
    transform: scale(0.94);
  }
  .tap-object.counted {
    background: #dff5e3;
  }
  .badge {
    position: absolute;
    right: -8px;
    top: -8px;
    display: grid;
    place-items: center;
    width: 40px;
    height: 40px;
    border-radius: 50%;
    background: theme('colors.crayon.leaf');
    color: #fff;
    font-family: theme('fontFamily.display');
    font-size: 26px;
    box-shadow: 0 3px 0 rgba(36, 52, 90, 0.2);
    animation: pop 0.3s cubic-bezier(0.3, 1.6, 0.5, 1);
  }

  .glow {
    outline: 5px solid theme('colors.crayon.sun');
    outline-offset: 3px;
    /* Pulsa a luz, não o tamanho: o alvo não pode fugir do dedo. */
    animation: glow 0.9s ease-in-out infinite;
  }
  .shake {
    animation: shake 0.34s;
  }
  @keyframes glow {
    50% {
      outline-offset: 7px;
      box-shadow: 0 0 0 12px rgba(255, 207, 63, 0.45);
    }
  }
  @keyframes shake {
    0%,
    100% {
      transform: translateX(0);
    }
    25% {
      transform: translateX(-8px) rotate(-3deg);
    }
    75% {
      transform: translateX(8px) rotate(3deg);
    }
  }
  @keyframes pop {
    0% {
      transform: scale(0.2);
    }
    100% {
      transform: scale(1);
    }
  }
</style>
