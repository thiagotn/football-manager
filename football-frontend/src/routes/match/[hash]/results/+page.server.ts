import { env } from '$env/dynamic/private';

const WEEKDAYS = ['domingo', 'segunda', 'terça', 'quarta', 'quinta', 'sexta', 'sábado'];
const MONTHS = ['jan', 'fev', 'mar', 'abr', 'mai', 'jun', 'jul', 'ago', 'set', 'out', 'nov', 'dez'];

function toDateStr(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

function fmtDate(d: string): string {
  const today = new Date();
  if (d === toDateStr(today)) return 'Hoje';
  const yesterday = new Date(today); yesterday.setDate(today.getDate() - 1);
  if (d === toDateStr(yesterday)) return 'Ontem';
  const [year, month, day] = d.split('-').map(Number);
  const date = new Date(year, month - 1, day);
  return `${WEEKDAYS[date.getDay()]}, ${day} de ${MONTHS[month - 1]}`;
}

export async function load({ params }) {
  const apiUrl = env.API_INTERNAL_URL ?? 'http://api:8000/api/v1';
  try {
    const [matchRes, resultsRes] = await Promise.all([
      fetch(`${apiUrl}/matches/public/${params.hash}`),
      fetch(`${apiUrl}/matches/public/${params.hash}/votes/results`),
    ]);
    if (!matchRes.ok) return { og: null };
    const match = await matchRes.json();
    const dateLabel = fmtDate(match.match_date);

    let description = `${match.group_name} · ${dateLabel}`;
    if (resultsRes.ok) {
      const results = await resultsRes.json();
      if (results.top5?.length > 0) {
        const top3 = results.top5.slice(0, 3).map((p: { name: string }) => p.name).join(', ');
        description = `🏆 Melhores: ${top3} · ${results.total_voters} voto(s)`;
      }
    }

    return {
      og: {
        title: `Resultado do Rachão ${match.group_name} — ${dateLabel}`,
        description,
      },
    };
  } catch {
    return { og: null };
  }
}
