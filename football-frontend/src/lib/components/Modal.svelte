<script lang="ts">
  import { X } from 'lucide-svelte';
  import type { Snippet } from 'svelte';

  let {
    open = $bindable(false),
    title = '',
    onClose = () => { open = false; },
    children,
  }: {
    open?: boolean;
    title?: string;
    onClose?: () => void;
    children?: Snippet;
  } = $props();
</script>

{#if open}
  <div class="fixed inset-0 z-40 flex items-end sm:items-center justify-center sm:p-4">
    <button class="absolute inset-0 bg-black/40" onclick={onClose} aria-label="Fechar modal" />
    <div class="relative bg-white dark:bg-gray-800 rounded-t-2xl sm:rounded-xl shadow-xl w-full sm:max-w-lg z-10 flex flex-col max-h-[92dvh] sm:max-h-[90dvh]">
      <div class="flex items-center justify-between px-6 py-4 border-b border-gray-100 dark:border-gray-700 shrink-0">
        <h3 class="font-semibold text-gray-900 dark:text-gray-100 text-lg">{title}</h3>
        <button onclick={onClose} class="btn-ghost btn-sm rounded-lg p-1.5">
          <X size={18} />
        </button>
      </div>
      <div class="p-6 overflow-y-auto">
        {@render children?.()}
      </div>
    </div>
  </div>
{/if}
