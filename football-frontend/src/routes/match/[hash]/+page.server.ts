import { env } from '$env/dynamic/private';

const WEEKDAYS = ['domingo', 'segunda', 'terça', 'quarta', 'quinta', 'sexta', 'sábado'];
const MONTHS = ['jan', 'fev', 'mar', 'abr', 'mai', 'jun', 'jul', 'ago', 'set', 'out', 'nov', 'dez'];

function toDateStr(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

function fmtDate(d: string): string {
  const today = new Date();
  if (d === toDateStr(today)) return 'Hoje';
  const tomorrow = new Date(today); tomorrow.setDate(today.getDate() + 1);
  if (d === toDateStr(tomorrow)) return 'Amanhã';
  const yesterday = new Date(today); yesterday.setDate(today.getDate() - 1);
  if (d === toDateStr(yesterday)) return 'Ontem';
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
        title: `Rachão ${match.group_name} — ${dateLabel}`,
        description: `${match.location} · ${match.start_time?.slice(0, 5)}${match.end_time ? ` – ${match.end_time.slice(0, 5)}` : ''} · ${match.confirmed_count} confirmado(s). Confirme sua presença!`,
      },
    };
  } catch {
    return { og: null };
  }
}
