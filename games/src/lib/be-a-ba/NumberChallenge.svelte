<script lang="ts">
  /**
   * Um desafio da trilha dos Números: objetos desenhados + 3 respostas grandes
   * (ou os dois lados, no "quem tem mais?"). Mesma regra das Letras: erro não
   * pune, e depois de 2 erros a resposta certa brilha.
   */
  import { Lightbulb } from 'lucide-svelte';
  import Objects from './Objects.svelte';
  import { answerOf, type NumberChallenge } from './numbers';
  import { t } from '$lib/i18n';

  let {
    challenge,
    sayNumber,
    sayHint,
    onresult,
  }: {
    challenge: NumberChallenge;
    sayNumber: (n: number) => void;
    sayHint: () => void;
    onresult: (mistakes: number, hinted: boolean) => void;
  } = $props();

  const AUTO_HINT_AFTER = 2;
  const answer = $derived(answerOf(challenge));

  let mistakes = $state(0);
  let hinted = $state(false);
  let wrong = $state<(number | string)[]>([]);
  let glow = $state<number | string | null>(null);
  let shaking = $state<number | string | null>(null);
  let finished = $state(false);

  function pick(value: number | 'left' | 'right') {
    if (finished || wrong.includes(value)) return;
    if (typeof value === 'number') sayNumber(value);
    if (value === answer) {
      finished = true;
      glow = null;
      setTimeout(() => onresult(mistakes, hinted), 450);
      return;
    }
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

  function askHint() {
    if (finished) return;
    hinted = true;
    glow = answer;
    sayHint();
  }

  /** Tamanho do objeto: cabe mais quando a conta tem mais coisas. */
  const sizeFor = (n: number) => (n > 10 ? 28 : n > 5 ? 34 : 42);
</script>

<div class="challenge" data-kind={challenge.kind} data-answer={answer}>
  <div class="scene">
    <button type="button" class="hint" onclick={askHint} aria-label={$t('be_a_ba.hint')} disabled={finished}>
      <Lightbulb size={22} strokeWidth={2.4} />
    </button>

    {#if challenge.kind === 'count'}
      <Objects item={challenge.item} n={challenge.n} size={sizeFor(challenge.n)} />
    {:else if challenge.kind === 'compare'}
      <div class="sides">
        {#each [['left', challenge.left], ['right', challenge.right]] as [side, n] (side)}
          <button
            type="button"
            class="side"
            class:wrong={wrong.includes(side)}
            class:glow={glow === side}
            class:shake={shaking === side}
            class:right={finished && answer === side}
            onclick={() => pick(side as 'left' | 'right')}
            aria-label={$t(side === 'left' ? 'be_a_ba.side_left' : 'be_a_ba.side_right')}
          >
            <Objects item={challenge.item} n={n as number} size={sizeFor(Math.max(challenge.left, challenge.right))} />
          </button>
        {/each}
      </div>
    {:else if challenge.kind === 'add'}
      <div class="sum">
        <Objects item={challenge.item} n={challenge.a} size={sizeFor(challenge.a + challenge.b)} />
        <span class="op">+</span>
        <Objects item={challenge.item} n={challenge.b} size={sizeFor(challenge.a + challenge.b)} />
      </div>
      <p class="equation">{challenge.a} + {challenge.b} = ?</p>
    {:else}
      <Objects item={challenge.item} n={challenge.a} leaving={challenge.b} size={sizeFor(challenge.a)} />
      <p class="equation">{challenge.a} − {challenge.b} = ?</p>
    {/if}
  </div>

  {#if challenge.kind !== 'compare'}
    <div class="options">
      {#each challenge.options as n (n)}
        <button
          type="button"
          class="number"
          class:wrong={wrong.includes(n)}
          class:glow={glow === n}
          class:shake={shaking === n}
          class:right={finished && answer === n}
          onclick={() => pick(n)}
          aria-label={String(n)}
        >{n}</button>
      {/each}
    </div>
  {/if}
</div>

<style>
  .challenge {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 18px;
    width: 100%;
  }
  .scene {
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
    width: 100%;
    max-width: 420px;
    min-height: 180px;
    justify-content: center;
    padding: 18px 14px;
    border-radius: 28px;
    background: #fff;
    box-shadow: 0 6px 0 rgba(36, 52, 90, 0.12);
  }
  .hint {
    position: absolute;
    right: -8px;
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
  .sides {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
    width: 100%;
  }
  .side {
    display: grid;
    place-items: center;
    min-height: 150px;
    padding: 10px 4px;
    border-radius: 20px;
    background: theme('colors.crayon.cloud');
    border: 4px solid transparent;
  }
  .side.right {
    border-color: theme('colors.crayon.leaf');
  }
  .side.wrong {
    opacity: 0.45;
  }
  .sum {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: center;
    gap: 10px;
  }
  .op {
    font-family: theme('fontFamily.display');
    font-size: 40px;
    color: theme('colors.crayon.grape');
  }
  .equation {
    font-family: theme('fontFamily.display');
    font-size: 30px;
    color: theme('colors.crayon.ink');
    letter-spacing: 0.04em;
  }
  .options {
    display: flex;
    gap: 14px;
  }
  .number {
    width: 84px;
    height: 84px;
    border-radius: 22px;
    background: #fff;
    color: theme('colors.crayon.ink');
    font-family: theme('fontFamily.display');
    font-size: 46px;
    box-shadow: 0 6px 0 rgba(36, 52, 90, 0.18);
  }
  .number:active,
  .side:active {
    transform: translateY(3px);
  }
  .number.wrong {
    opacity: 0.35;
  }
  .number.right {
    background: theme('colors.crayon.leaf');
    color: #fff;
  }
  .glow {
    outline: 4px solid theme('colors.crayon.sun');
    outline-offset: 2px;
    /* Pulsa a luz, não o tamanho: o alvo não pode fugir do dedo. */
    animation: glow 0.9s ease-in-out infinite;
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
      transform: translateX(-7px) rotate(-3deg);
    }
    75% {
      transform: translateX(7px) rotate(3deg);
    }
  }
  @keyframes glow {
    50% {
      outline-offset: 6px;
      box-shadow: 0 0 0 10px rgba(255, 207, 63, 0.45);
    }
  }
</style>
