<script lang="ts">
  let {
    open = $bindable(false),
    message = '',
    confirmLabel = 'Confirmar',
    danger = true,
    onConfirm = () => {},
  }: {
    open?: boolean;
    message?: string;
    confirmLabel?: string;
    danger?: boolean;
    onConfirm?: () => void;
  } = $props();

  function handleConfirm() {
    open = false;
    onConfirm();
  }

  function handleCancel() {
    open = false;
  }
</script>

{#if open}
  <button class="fixed inset-0 z-40 bg-black/40" onclick={handleCancel} aria-label="Cancelar" />

  <!-- Bottom sheet on mobile, centered modal on desktop -->
  <div class="fixed z-50 bottom-0 inset-x-0 sm:inset-0 sm:flex sm:items-center sm:justify-center sm:p-4 pointer-events-none">
    <div class="bg-white rounded-t-2xl sm:rounded-2xl shadow-xl w-full sm:max-w-sm p-6 pointer-events-auto">
      <p class="text-gray-800 font-medium text-center text-base mb-6 leading-relaxed">{message}</p>
      <div class="flex flex-col gap-2 sm:flex-row-reverse">
        <button
          class="btn justify-center {danger ? 'btn-danger' : 'btn-primary'} flex-1 py-3 sm:py-2"
          onclick={handleConfirm}>
          {confirmLabel}
        </button>
        <button class="btn btn-secondary justify-center flex-1 py-3 sm:py-2" onclick={handleCancel}>
          Cancelar
        </button>
      </div>
    </div>
  </div>
{/if}
