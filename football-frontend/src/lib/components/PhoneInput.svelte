<script lang="ts">
  interface Country {
    code: string;
    name: string;
    dial: string;
  }

  const COUNTRIES: Country[] = [
    { code: 'BR', name: 'Brasil', dial: '55' },
    { code: 'US', name: 'United States', dial: '1' },
    { code: 'CA', name: 'Canadá', dial: '1' },
    { code: 'GB', name: 'United Kingdom', dial: '44' },
    { code: 'DE', name: 'Alemanha', dial: '49' },
    { code: 'FR', name: 'França', dial: '33' },
    { code: 'IT', name: 'Itália', dial: '39' },
    { code: 'ES', name: 'Espanha', dial: '34' },
    { code: 'PT', name: 'Portugal', dial: '351' },
    { code: 'AR', name: 'Argentina', dial: '54' },
    { code: 'MX', name: 'México', dial: '52' },
    { code: 'CL', name: 'Chile', dial: '56' },
    { code: 'CO', name: 'Colômbia', dial: '57' },
    { code: 'PE', name: 'Peru', dial: '51' },
    { code: 'UY', name: 'Uruguai', dial: '598' },
    { code: 'PY', name: 'Paraguai', dial: '595' },
    { code: 'BO', name: 'Bolívia', dial: '591' },
    { code: 'VE', name: 'Venezuela', dial: '58' },
    { code: 'EC', name: 'Equador', dial: '593' },
    { code: 'AU', name: 'Austrália', dial: '61' },
    { code: 'JP', name: 'Japão', dial: '81' },
    { code: 'CN', name: 'China', dial: '86' },
    { code: 'IN', name: 'Índia', dial: '91' },
    { code: 'NG', name: 'Nigéria', dial: '234' },
    { code: 'ZA', name: 'África do Sul', dial: '27' },
  ];

  function getFlag(code: string): string {
    return [...code.toUpperCase()].map(c =>
      String.fromCodePoint(0x1F1E6 + c.charCodeAt(0) - 65)
    ).join('');
  }

  interface Props {
    id?: string;
    value: string;
    placeholder?: string;
    required?: boolean;
    disabled?: boolean;
    oncountrychange?: (countryCode: string) => void;
  }

  let {
    id = 'phone',
    value = $bindable(''),
    placeholder = '',
    required = false,
    disabled = false,
    oncountrychange,
  }: Props = $props();

  let selectedDial = $state('55');
  let localNumber = $state('');

  let selectedCountry = $derived(COUNTRIES.find(c => c.dial === selectedDial) ?? COUNTRIES[0]);

  $effect(() => {
    const digits = localNumber.replace(/\D/g, '');
    value = digits ? '+' + selectedDial + digits : '';
  });

  function onNumberInput(e: Event) {
    const input = e.target as HTMLInputElement;
    localNumber = input.value.replace(/\D/g, '');
    input.value = localNumber;
  }
</script>

<div class="flex gap-2">
  <select
    bind:value={selectedDial}
    onchange={() => oncountrychange?.(selectedCountry.code)}
    {disabled}
    class="px-2 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500 transition-colors bg-white dark:bg-gray-800 dark:border-gray-600 dark:text-gray-100 shrink-0"
    style="max-width: 7rem;"
  >
    {#each COUNTRIES as country}
      <option value={country.dial}>
        {getFlag(country.code)} +{country.dial}
      </option>
    {/each}
  </select>

  <input
    {id}
    type="tel"
    inputmode="numeric"
    autocomplete="tel-national"
    value={localNumber}
    oninput={onNumberInput}
    {placeholder}
    {required}
    {disabled}
    class="input flex-1 min-w-0"
  />
</div>
