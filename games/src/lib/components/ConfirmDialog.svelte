<script lang="ts">
  /**
   * Confirmação destrutiva: overlay + card, no padrão dos diálogos do app.
   * Fecha pelo backdrop, pelo Esc ou pelo botão Cancelar.
   */
  import { t } from '$lib/i18n';

  let {
    open = $bindable(false),
    message = '',
    confirmLabel = '',
    onConfirm = () => {},
  }: {
    open?: boolean;
    message?: string;
    confirmLabel?: string;
    onConfirm?: () => void;
  } = $props();

  function confirm() {
    open = false;
    onConfirm();
  }
</script>

<svelte:window
  onkeydown={(e) => {
    if (open && e.key === 'Escape') open = false;
  }}
/>

{#if open}
  <div
    class="overlay"
    role="dialog"
    aria-modal="true"
    aria-label={message}
    onclick={(e) => {
      if (e.target === e.currentTarget) open = false;
    }}
    onkeydown={() => {}}
    tabindex="-1"
  >
    <div class="card flex flex-col gap-4">
      <p class="text-center text-base font-bold leading-relaxed">{message}</p>
      <button class="btn btn-danger" onclick={confirm}>{confirmLabel}</button>
      <button class="btn btn-ghost" onclick={() => (open = false)}>{$t('common.cancel')}</button>
    </div>
  </div>
{/if}
