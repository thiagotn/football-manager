<script lang="ts">
  import { ChevronLeft, ChevronRight, Calendar } from 'lucide-svelte';

  interface Props {
    value: string;        // "YYYY-MM-DD" or ""
    id?: string;
    required?: boolean;
    disabled?: boolean;
    placeholder?: string;
  }

  let { value = $bindable(''), id, required, disabled, placeholder = 'Selecionar data' }: Props = $props();

  let open = $state(false);
  let temp = $state('');       // working copy while picker is open
  let viewYear = $state(new Date().getFullYear());
  let viewMonth = $state(new Date().getMonth()); // 0-indexed

  const MONTHS = ['Janeiro','Fevereiro','Março','Abril','Maio','Junho','Julho','Agosto','Setembro','Outubro','Novembro','Dezembro'];
  const DAYS   = ['D','S','T','Q','Q','S','S'];

  function openPicker() {
    if (disabled) return;
    temp = value;
    if (value) {
      const d = new Date(value + 'T00:00');
      viewYear  = d.getFullYear();
      viewMonth = d.getMonth();
    } else {
      const now = new Date();
      viewYear  = now.getFullYear();
      viewMonth = now.getMonth();
    }
    open = true;
  }

  function confirm() {
    value = temp;
    open = false;
  }

  function cancel() {
    open = false;
  }

  function clear() {
    temp = '';
  }

  function prevMonth() {
    if (viewMonth === 0) { viewMonth = 11; viewYear--; }
    else viewMonth--;
  }

  function nextMonth() {
    if (viewMonth === 11) { viewMonth = 0; viewYear++; }
    else viewMonth++;
  }

  function selectDay(day: number) {
    const mm = String(viewMonth + 1).padStart(2, '0');
    const dd = String(day).padStart(2, '0');
    temp = `${viewYear}-${mm}-${dd}`;
  }

  // Build calendar grid (cells: null = empty, number = day)
  let cells = $derived.by(() => {
    const firstDow = new Date(viewYear, viewMonth, 1).getDay(); // 0=Sun
    const daysInMonth = new Date(viewYear, viewMonth + 1, 0).getDate();
    const arr: (number | null)[] = Array(firstDow).fill(null);
    for (let d = 1; d <= daysInMonth; d++) arr.push(d);
    while (arr.length % 7 !== 0) arr.push(null);
    return arr;
  });

  function isSelected(day: number | null): boolean {
    if (!day || !temp) return false;
    const mm = String(viewMonth + 1).padStart(2, '0');
    const dd = String(day).padStart(2, '0');
    return temp === `${viewYear}-${mm}-${dd}`;
  }

  function isToday(day: number | null): boolean {
    if (!day) return false;
    const now = new Date();
    return now.getFullYear() === viewYear && now.getMonth() === viewMonth && now.getDate() === day;
  }

  // Format display value
  let displayValue = $derived.by(() => {
    if (!value) return '';
    const d = new Date(value + 'T00:00');
    return d.toLocaleDateString('pt-BR', { weekday: 'short', day: '2-digit', month: 'short', year: 'numeric' });
  });
</script>

<!-- Trigger input -->
<button
  type="button"
  {id}
  onclick={openPicker}
  class="input w-full text-left flex items-center justify-between gap-2 {!value ? 'text-gray-400 dark:text-gray-500' : ''}"
  {disabled}
>
  <span class="truncate">{displayValue || placeholder}</span>
  <Calendar size={16} class="shrink-0 text-gray-400 dark:text-gray-500" />
</button>

<!-- Backdrop + bottom-sheet / centered modal -->
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
    <div class="bg-white dark:bg-gray-800 rounded-t-2xl sm:rounded-2xl shadow-2xl w-full sm:max-w-sm pointer-events-auto">

      <!-- Month navigation -->
      <div class="flex items-center justify-between px-4 pt-4 pb-2">
        <button type="button" onclick={prevMonth} class="p-1.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors">
          <ChevronLeft size={18} class="text-gray-600 dark:text-gray-300" />
        </button>
        <span class="font-semibold text-gray-900 dark:text-gray-100 text-sm">
          {MONTHS[viewMonth]} {viewYear}
        </span>
        <button type="button" onclick={nextMonth} class="p-1.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors">
          <ChevronRight size={18} class="text-gray-600 dark:text-gray-300" />
        </button>
      </div>

      <!-- Day headers -->
      <div class="grid grid-cols-7 px-3 mb-1">
        {#each DAYS as d}
          <div class="text-center text-xs font-medium text-gray-400 dark:text-gray-500 py-1">{d}</div>
        {/each}
      </div>

      <!-- Day cells -->
      <div class="grid grid-cols-7 px-3 pb-2">
        {#each cells as cell}
          {#if cell === null}
            <div></div>
          {:else}
            <button
              type="button"
              onclick={() => selectDay(cell)}
              class="aspect-square flex items-center justify-center rounded-full text-sm font-medium transition-colors mx-auto w-9
                {isSelected(cell)
                  ? 'bg-primary-600 text-white'
                  : isToday(cell)
                  ? 'border border-primary-500 text-primary-600 dark:text-primary-400 hover:bg-primary-50 dark:hover:bg-primary-900/20'
                  : 'text-gray-800 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700'}"
            >
              {cell}
            </button>
          {/if}
        {/each}
      </div>

      <!-- Footer actions -->
      <div class="flex items-center gap-2 px-4 py-3 border-t border-gray-100 dark:border-gray-700">
        <button
          type="button"
          onclick={clear}
          class="text-sm text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 px-2 py-1.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
        >
          Limpar
        </button>
        <div class="flex-1"></div>
        <button
          type="button"
          onclick={cancel}
          class="btn-secondary btn-sm"
        >
          Cancelar
        </button>
        <button
          type="button"
          onclick={confirm}
          class="btn-primary btn-sm"
        >
          Confirmar
        </button>
      </div>

    </div>
  </div>
{/if}
