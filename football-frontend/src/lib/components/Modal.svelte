<script module lang="ts">
  let modalCounter = 0;
</script>

<script lang="ts">
  /**
   * Modal genérico.
   *
   * Props:
   *   open          — bindable
   *   title         — texto do header
   *   size          — 'md' (default, max-w-lg) | 'wide' (860px)
   *   layout        — 'sheet' (default: bottom sheet no mobile, centralizado no desktop)
   *                 | 'centered' (sempre centralizado, padding 16/32px, raio 24px)
   *   closeOnEscape — default true. Passar false enquanto um overlay filho
   *                   (ex: AvatarLightbox) estiver aberto, pra Esc fechar só ele.
   *   titleIcon     — snippet renderizado à esquerda do título
   *   onClose       — default: open = false
   *
   * Acessibilidade: role=dialog + aria-modal + aria-labelledby, foco vai pro
   * painel ao abrir, volta pro elemento de origem ao fechar, Tab fica preso.
   */
  import { X } from 'lucide-svelte';
  import { tick, type Snippet } from 'svelte';
  import { fade, fly } from 'svelte/transition';
  import { t } from '$lib/i18n';

  let {
    open = $bindable(false),
    title = '',
    size = 'md',
    layout = 'sheet',
    closeOnEscape = true,
    onClose = () => { open = false; },
    titleIcon,
    children,
  }: {
    open?: boolean;
    title?: string;
    size?: 'md' | 'wide';
    layout?: 'sheet' | 'centered';
    closeOnEscape?: boolean;
    onClose?: () => void;
    titleIcon?: Snippet;
    children?: Snippet;
  } = $props();

  const titleId = `modal-title-${++modalCounter}`;

  let panel = $state<HTMLDivElement | undefined>();
  let restoreFocusTo: HTMLElement | null = null;

  // Foco: guarda o elemento de origem ao abrir e devolve ao fechar.
  $effect(() => {
    if (open) {
      restoreFocusTo = (document.activeElement as HTMLElement | null) ?? null;
      tick().then(() => panel?.focus({ preventScroll: true }));
    } else if (restoreFocusTo) {
      const el = restoreFocusTo;
      restoreFocusTo = null;
      if (el.isConnected) el.focus({ preventScroll: true });
    }
  });

  const FOCUSABLE = 'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

  function handleKeydown(e: KeyboardEvent) {
    if (!open) return;
    if (e.key === 'Escape') {
      if (!closeOnEscape) return;
      e.preventDefault();
      onClose();
      return;
    }
    if (e.key === 'Tab' && panel) {
      const els = Array.from(panel.querySelectorAll<HTMLElement>(FOCUSABLE)).filter((el) => el.offsetParent !== null);
      if (els.length === 0) { e.preventDefault(); return; }
      const first = els[0];
      const last = els[els.length - 1];
      const active = document.activeElement;
      if (e.shiftKey && (active === first || active === panel)) {
        e.preventDefault(); last.focus();
      } else if (!e.shiftKey && active === last) {
        e.preventDefault(); first.focus();
      } else if (!panel.contains(active)) {
        e.preventDefault(); first.focus();
      }
    }
  }

  const centered = $derived(layout === 'centered');
  const containerClass = $derived(
    centered
      ? 'fixed inset-0 z-40 flex items-center justify-center p-4 sm:p-8'
      : 'fixed inset-0 z-40 flex items-end sm:items-center justify-center sm:p-4'
  );
  const panelClass = $derived(
    [
      'relative z-10 flex flex-col w-full outline-none',
      size === 'wide' ? 'sm:max-w-[860px]' : 'sm:max-w-lg',
      centered
        ? 'bg-white dark:bg-slate-900/95 rounded-3xl border border-gray-200 dark:border-white/10 shadow-[0_30px_80px_-20px_rgba(0,0,0,.9)] max-h-[92dvh]'
        : 'bg-white dark:bg-gray-800 rounded-t-2xl sm:rounded-xl shadow-xl max-h-[92dvh] sm:max-h-[90dvh]',
    ].join(' ')
  );
  const headerClass = $derived(
    centered
      ? 'flex items-center justify-between px-5 sm:px-6 py-4 border-b border-gray-100 dark:border-white/10 shrink-0'
      : 'flex items-center justify-between px-6 py-4 border-b border-gray-100 dark:border-gray-700 shrink-0'
  );
  const bodyClass = $derived(centered ? 'p-5 sm:p-7 overflow-y-auto' : 'p-6 overflow-y-auto');
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
  <div class={containerClass}>
    <button
      type="button"
      class="absolute inset-0 bg-black/40"
      onclick={onClose}
      aria-label={$t('aria.close')}
      transition:fade={{ duration: 150 }}
    ></button>
    <div
      bind:this={panel}
      class={panelClass}
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
      tabindex="-1"
      transition:fly={{ y: 8, duration: 200 }}
    >
      <div class={headerClass}>
        <div class="flex items-center gap-2.5 min-w-0">
          {#if titleIcon}
            {@render titleIcon()}
          {/if}
          <h3 id={titleId} class="font-semibold text-gray-900 dark:text-gray-100 text-lg truncate">{title}</h3>
        </div>
        <button type="button" onclick={onClose} class="btn-ghost btn-sm rounded-lg p-1.5 shrink-0" aria-label={$t('aria.close')}>
          <X size={18} />
        </button>
      </div>
      <div class={bodyClass}>
        {@render children?.()}
      </div>
    </div>
  </div>
{/if}
