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
  <div class="fixed inset-0 z-40 flex items-center justify-center p-4">
    <button class="absolute inset-0 bg-black/40" onclick={onClose} aria-label="Fechar modal" />
    <div class="relative bg-white rounded-xl shadow-xl w-full max-w-lg z-10">
      <div class="flex items-center justify-between px-6 py-4 border-b border-gray-100">
        <h3 class="font-semibold text-gray-900 text-lg">{title}</h3>
        <button onclick={onClose} class="btn-ghost btn-sm rounded-lg p-1.5">
          <X size={18} />
        </button>
      </div>
      <div class="p-6">
        {@render children?.()}
      </div>
    </div>
  </div>
{/if}
