import { getTimezoneLabel } from '$lib/timezones';

/** Detecta se o timezone do grupo é o mesmo do dispositivo do usuário. */
export function isLocalTimezone(groupTimezone: string): boolean {
  try {
    return groupTimezone === Intl.DateTimeFormat().resolvedOptions().timeZone;
  } catch {
    return true;
  }
}

/**
 * Formata um horário de partida com indicação de fuso quando diferente do local.
 *
 * Retorna "20:00" se mesmo fuso, ou "20:00 · Berlim" se fuso diferente.
 * O horário exibido é SEMPRE o horário do evento — não convertido.
 */
export function formatMatchTime(
  timeStr: string,
  groupTimezone: string,
): string {
  const time = timeStr.slice(0, 5);
  if (isLocalTimezone(groupTimezone)) return time;
  return `${time} · ${getTimezoneLabel(groupTimezone)}`;
}

/**
 * Formata o intervalo de horário de uma partida com duração e indicação de fuso.
 *
 * Retorna "20:00 – 21:30 (1h30)" ou "20:00 – 21:30 (1h30) · Berlim".
 */
export function formatMatchTimeRange(
  start: string,
  end: string | null,
  groupTimezone: string,
): string {
  const s = start.slice(0, 5);
  if (!end) return formatMatchTime(start, groupTimezone);

  const e = end.slice(0, 5);
  const [sh, sm] = s.split(':').map(Number);
  const [eh, em] = e.split(':').map(Number);
  const mins = eh * 60 + em - (sh * 60 + sm);

  let range = `${s} – ${e}`;
  if (mins > 0) {
    const h = Math.floor(mins / 60);
    const m = mins % 60;
    const dur = h && m ? `${h}h${String(m).padStart(2, '0')}` : h ? `${h}h` : `${m}min`;
    range = `${range} (${dur})`;
  }

  if (!isLocalTimezone(groupTimezone)) {
    return `${range} · ${getTimezoneLabel(groupTimezone)}`;
  }
  return range;
}
