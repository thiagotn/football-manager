<script lang="ts">
  import { Lock } from 'lucide-svelte';

  interface Props {
    open: boolean;
    title?: string;
    message?: string;
  }

  let {
    open = $bindable(false),
    title = 'Limite atingido — Plano Free',
    message = 'Você atingiu o limite do seu plano atual.',
  }: Props = $props();
</script>

{#if open}
  <!-- Backdrop -->
  <button
    class="fixed inset-0 z-40 bg-black/50"
    onclick={() => open = false}
    aria-label="Fechar"
    type="button"
  ></button>

  <!-- Panel — bottom sheet mobile, modal centralizado desktop -->
  <div class="fixed z-50 left-0 right-0 bottom-0 sm:inset-0 sm:flex sm:items-center sm:justify-center pointer-events-none">
    <div class="bg-white dark:bg-gray-800 rounded-t-2xl sm:rounded-2xl shadow-2xl w-full sm:max-w-sm pointer-events-auto">

      <!-- Header -->
      <div class="flex items-center gap-3 px-5 pt-5 pb-3">
        <div class="w-9 h-9 rounded-full bg-amber-100 dark:bg-amber-900/30 flex items-center justify-center shrink-0">
          <Lock size={16} class="text-amber-600 dark:text-amber-400" />
        </div>
        <h2 class="font-semibold text-gray-900 dark:text-gray-100 text-base leading-snug">{title}</h2>
      </div>

      <!-- Body -->
      <div class="px-5 pb-2">
        <p class="text-sm text-gray-600 dark:text-gray-300 leading-relaxed">{message}</p>
        <p class="text-xs text-gray-400 dark:text-gray-500 mt-3">
          Planos com mais recursos estarão disponíveis em breve.
        </p>
      </div>

      <!-- Footer -->
      <div class="px-5 py-4">
        <button
          type="button"
          onclick={() => open = false}
          class="btn-primary w-full justify-center"
        >
          Entendido
        </button>
      </div>

    </div>
  </div>
{/if}
