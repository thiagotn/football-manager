<script lang="ts">
  import { toasts } from '$lib/stores/toast';
  import { CheckCircle, XCircle, Info, X } from 'lucide-svelte';
</script>

<div class="fixed top-4 right-4 z-50 flex flex-col gap-2 pointer-events-none">
  {#each $toasts as toast (toast.id)}
    <div
      class="pointer-events-auto flex items-start gap-3 px-4 py-3 rounded-lg shadow-lg max-w-sm text-sm font-medium
        {toast.type === 'success' ? 'bg-green-600 text-white' :
         toast.type === 'error'   ? 'bg-red-600 text-white' :
                                    'bg-blue-600 text-white'}"
      style="animation: slideIn .2s ease"
    >
      {#if toast.type === 'success'}<CheckCircle size={16} class="shrink-0 mt-0.5" />
      {:else if toast.type === 'error'}<XCircle size={16} class="shrink-0 mt-0.5" />
      {:else}<Info size={16} class="shrink-0 mt-0.5" />{/if}
      <span class="flex-1">{toast.message}</span>
    </div>
  {/each}
</div>

<style>
  @keyframes slideIn {
    from { transform: translateX(110%); opacity: 0; }
    to   { transform: translateX(0);    opacity: 1; }
  }
</style>
