<script lang="ts">
  import { Clock } from 'lucide-svelte';

  interface Props {
    value: string;       // "HH:MM" or ""
    id?: string;
    required?: boolean;
    disabled?: boolean;
    placeholder?: string;
  }

  let { value = $bindable(''), id, required, disabled, placeholder = 'Selecionar hora' }: Props = $props();

  let open = $state(false);
  let tempHour   = $state('');
  let tempMinute = $state('');

  const HOURS   = Array.from({ length: 24 }, (_, i) => String(i).padStart(2, '0'));
  const MINUTES = ['00','05','10','15','20','25','30','35','40','45','50','55'];

  function openPicker() {
    if (disabled) return;
    const [h = '', m = ''] = value ? value.split(':') : [];
    tempHour   = h;
    // snap minute to nearest 5
    tempMinute = MINUTES.includes(m) ? m : (m ? MINUTES.reduce((a, b) => Math.abs(+b - +m) < Math.abs(+a - +m) ? b : a) : '');
    open = true;
  }

  function confirm() {
    if (tempHour && tempMinute) {
      value = `${tempHour}:${tempMinute}`;
    } else if (!required) {
      value = '';
    }
    open = false;
  }

  function cancel() {
    open = false;
  }

  function clear() {
    tempHour   = '';
    tempMinute = '';
  }

  let displayValue = $derived(value || '');
</script>

<!-- Trigger -->
<button
  type="button"
  {id}
  onclick={openPicker}
  class="input w-full text-left flex items-center justify-between gap-2 {!value ? 'text-gray-400 dark:text-gray-500' : ''}"
  {disabled}
>
  <span>{displayValue || placeholder}</span>
  <Clock size={16} class="shrink-0 text-gray-400 dark:text-gray-500" />
</button>

{#if open}
  <!-- Backdrop -->
  <button
    class="fixed inset-0 z-40 bg-black/50"
    onclick={cancel}
    aria-label="Fechar"
    type="button"
  ></button>

  <!-- Panel -->
  <div class="fixed z-50 left-0 right-0 bottom-0 sm:inset-0 sm:flex sm:items-center sm:justify-center pointer-events-none">
    <div class="bg-white dark:bg-gray-800 rounded-t-2xl sm:rounded-2xl shadow-2xl w-full sm:max-w-xs pointer-events-auto">

      <!-- Header -->
      <div class="px-4 pt-4 pb-3 border-b border-gray-100 dark:border-gray-700 flex items-center gap-2">
        <Clock size={16} class="text-primary-600" />
        <span class="font-semibold text-gray-900 dark:text-gray-100 text-sm">Selecionar hora</span>
        {#if tempHour && tempMinute}
          <span class="ml-auto text-2xl font-bold tabular-nums text-primary-600">{tempHour}:{tempMinute}</span>
        {/if}
      </div>

      <!-- Columns -->
      <div class="flex gap-3 px-4 pt-3 pb-1">
        <!-- Hours -->
        <div class="flex-1">
          <p class="text-xs font-medium text-gray-400 dark:text-gray-500 text-center mb-2">Hora</p>
          <div class="h-52 overflow-y-auto grid grid-cols-4 gap-1 pr-1"
               style="scrollbar-width:none">
            {#each HOURS as h}
              <button
                type="button"
                onclick={() => tempHour = h}
                class="py-2 rounded-lg text-sm font-medium transition-colors
                  {tempHour === h
                    ? 'bg-primary-600 text-white'
                    : 'text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700'}"
              >{h}</button>
            {/each}
          </div>
        </div>

        <div class="w-px bg-gray-100 dark:bg-gray-700 self-stretch"></div>

        <!-- Minutes -->
        <div class="flex-1">
          <p class="text-xs font-medium text-gray-400 dark:text-gray-500 text-center mb-2">Minuto</p>
          <div class="h-52 overflow-y-auto grid grid-cols-3 gap-1 pr-1"
               style="scrollbar-width:none">
            {#each MINUTES as m}
              <button
                type="button"
                onclick={() => tempMinute = m}
                class="py-2 rounded-lg text-sm font-medium transition-colors
                  {tempMinute === m
                    ? 'bg-primary-600 text-white'
                    : 'text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700'}"
              >{m}</button>
            {/each}
          </div>
        </div>
      </div>

      <!-- Footer -->
      <div class="flex items-center gap-2 px-4 py-3 border-t border-gray-100 dark:border-gray-700">
        <button
          type="button"
          onclick={clear}
          class="text-sm text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 px-2 py-1.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
        >
          Limpar
        </button>
        <div class="flex-1"></div>
        <button type="button" onclick={cancel} class="btn-secondary btn-sm">Cancelar</button>
        <button
          type="button"
          onclick={confirm}
          class="btn-primary btn-sm"
          disabled={required && (!tempHour || !tempMinute)}
        >
          Confirmar
        </button>
      </div>

    </div>
  </div>
{/if}
