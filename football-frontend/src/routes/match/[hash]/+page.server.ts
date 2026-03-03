import { env } from '$env/dynamic/private';

const WEEKDAYS = ['domingo', 'segunda', 'terça', 'quarta', 'quinta', 'sexta', 'sábado'];
const MONTHS = ['jan', 'fev', 'mar', 'abr', 'mai', 'jun', 'jul', 'ago', 'set', 'out', 'nov', 'dez'];

function fmtDate(d: string): string {
  const [year, month, day] = d.split('-').map(Number);
  const date = new Date(year, month - 1, day);
  return `${WEEKDAYS[date.getDay()]}, ${day} de ${MONTHS[month - 1]}`;
}

export async function load({ params }) {
  const apiUrl = env.API_INTERNAL_URL ?? 'http://api:8000/api/v1';
  try {
    const res = await fetch(`${apiUrl}/matches/public/${params.hash}`);
    if (!res.ok) return { og: null };
    const match = await res.json();
    const dateLabel = fmtDate(match.match_date);
    return {
      og: {
        title: `Pelada ${match.group_name} — ${dateLabel}`,
        description: `${match.location} · ${match.start_time?.slice(0, 5)} · ${match.confirmed_count} confirmado(s). Confirme sua presença!`,
      },
    };
  } catch {
    return { og: null };
  }
}
