export type TimezoneOption = {
  value: string;
  label: string;
  offset: string;
  group: string;
};

export const TIMEZONE_OPTIONS: TimezoneOption[] = [
  // América
  { value: 'America/Sao_Paulo',               label: 'Brasília / São Paulo',    offset: 'UTC-3',    group: 'América' },
  { value: 'America/Fortaleza',               label: 'Fortaleza / Belém',       offset: 'UTC-3',    group: 'América' },
  { value: 'America/Manaus',                  label: 'Manaus',                  offset: 'UTC-4',    group: 'América' },
  { value: 'America/Cuiaba',                  label: 'Cuiabá / Porto Velho',    offset: 'UTC-4',    group: 'América' },
  { value: 'America/Rio_Branco',              label: 'Rio Branco',              offset: 'UTC-5',    group: 'América' },
  { value: 'America/Noronha',                 label: 'Fernando de Noronha',     offset: 'UTC-2',    group: 'América' },
  { value: 'America/Argentina/Buenos_Aires',  label: 'Buenos Aires',            offset: 'UTC-3',    group: 'América' },
  { value: 'America/Santiago',                label: 'Santiago',                offset: 'UTC-4',    group: 'América' },
  { value: 'America/Bogota',                  label: 'Bogotá',                  offset: 'UTC-5',    group: 'América' },
  { value: 'America/Lima',                    label: 'Lima',                    offset: 'UTC-5',    group: 'América' },
  { value: 'America/Caracas',                 label: 'Caracas',                 offset: 'UTC-4',    group: 'América' },
  { value: 'America/Mexico_City',             label: 'Cidade do México',        offset: 'UTC-6',    group: 'América' },
  { value: 'America/New_York',                label: 'Nova York / Miami',       offset: 'UTC-5',    group: 'América' },
  { value: 'America/Chicago',                 label: 'Chicago / Houston',       offset: 'UTC-6',    group: 'América' },
  { value: 'America/Denver',                  label: 'Denver / Phoenix',        offset: 'UTC-7',    group: 'América' },
  { value: 'America/Los_Angeles',             label: 'Los Angeles / Seattle',   offset: 'UTC-8',    group: 'América' },
  { value: 'America/Toronto',                 label: 'Toronto',                 offset: 'UTC-5',    group: 'América' },
  { value: 'America/Vancouver',               label: 'Vancouver',               offset: 'UTC-8',    group: 'América' },
  { value: 'America/Anchorage',               label: 'Anchorage',               offset: 'UTC-9',    group: 'América' },
  // Europa
  { value: 'Europe/Lisbon',                   label: 'Lisboa',                  offset: 'UTC+0',    group: 'Europa' },
  { value: 'Europe/London',                   label: 'Londres',                 offset: 'UTC+0',    group: 'Europa' },
  { value: 'Europe/Paris',                    label: 'Paris',                   offset: 'UTC+1',    group: 'Europa' },
  { value: 'Europe/Berlin',                   label: 'Berlim',                  offset: 'UTC+1',    group: 'Europa' },
  { value: 'Europe/Madrid',                   label: 'Madrid',                  offset: 'UTC+1',    group: 'Europa' },
  { value: 'Europe/Rome',                     label: 'Roma',                    offset: 'UTC+1',    group: 'Europa' },
  { value: 'Europe/Amsterdam',                label: 'Amsterdã',                offset: 'UTC+1',    group: 'Europa' },
  { value: 'Europe/Warsaw',                   label: 'Varsóvia',                offset: 'UTC+1',    group: 'Europa' },
  { value: 'Europe/Athens',                   label: 'Atenas',                  offset: 'UTC+2',    group: 'Europa' },
  { value: 'Europe/Helsinki',                 label: 'Helsinque',               offset: 'UTC+2',    group: 'Europa' },
  { value: 'Europe/Moscow',                   label: 'Moscou',                  offset: 'UTC+3',    group: 'Europa' },
  // África
  { value: 'Africa/Luanda',                   label: 'Luanda',                  offset: 'UTC+1',    group: 'África' },
  { value: 'Africa/Maputo',                   label: 'Maputo',                  offset: 'UTC+2',    group: 'África' },
  { value: 'Africa/Nairobi',                  label: 'Nairóbi',                 offset: 'UTC+3',    group: 'África' },
  { value: 'Africa/Lagos',                    label: 'Lagos',                   offset: 'UTC+1',    group: 'África' },
  { value: 'Africa/Johannesburg',             label: 'Joanesburgo',             offset: 'UTC+2',    group: 'África' },
  // Ásia
  { value: 'Asia/Riyadh',                     label: 'Riad / Kuwait',           offset: 'UTC+3',    group: 'Ásia' },
  { value: 'Asia/Dubai',                      label: 'Dubai / Abu Dhabi',       offset: 'UTC+4',    group: 'Ásia' },
  { value: 'Asia/Kolkata',                    label: 'Mumbai / Calcutá',        offset: 'UTC+5:30', group: 'Ásia' },
  { value: 'Asia/Dhaka',                      label: 'Dacca',                   offset: 'UTC+6',    group: 'Ásia' },
  { value: 'Asia/Bangkok',                    label: 'Bangkok / Jacarta',       offset: 'UTC+7',    group: 'Ásia' },
  { value: 'Asia/Singapore',                  label: 'Singapura / Kuala Lumpur',offset: 'UTC+8',    group: 'Ásia' },
  { value: 'Asia/Shanghai',                   label: 'Xangai / Pequim',         offset: 'UTC+8',    group: 'Ásia' },
  { value: 'Asia/Tokyo',                      label: 'Tóquio',                  offset: 'UTC+9',    group: 'Ásia' },
  { value: 'Asia/Seoul',                      label: 'Seul',                    offset: 'UTC+9',    group: 'Ásia' },
  // Oceania
  { value: 'Australia/Perth',                 label: 'Perth',                   offset: 'UTC+8',    group: 'Oceania' },
  { value: 'Australia/Adelaide',              label: 'Adelaide',                offset: 'UTC+9:30', group: 'Oceania' },
  { value: 'Australia/Sydney',                label: 'Sydney / Melbourne',      offset: 'UTC+10',   group: 'Oceania' },
  { value: 'Pacific/Auckland',                label: 'Auckland',                offset: 'UTC+12',   group: 'Oceania' },
];

export const TIMEZONE_GROUPS = [...new Set(TIMEZONE_OPTIONS.map(t => t.group))];

/** Retorna o label amigável para um valor IANA, ou o próprio valor se não encontrado. */
export function getTimezoneLabel(value: string): string {
  return TIMEZONE_OPTIONS.find(t => t.value === value)?.label ?? value;
}
