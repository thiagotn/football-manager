/**
 * Retorna o nome de exibição do jogador no formato "PrimeiroNome (Apelido)".
 * Se não houver apelido, retorna apenas o primeiro nome.
 *
 * Exemplos:
 *   playerDisplayName("Thiago Teixeira Nogueira", "Thiagol") → "Thiago (Thiagol)"
 *   playerDisplayName("Thiago Teixeira Nogueira")             → "Thiago"
 */
export function playerDisplayName(name, nickname) {
  const firstName = (name ?? '').trim().split(/\s+/)[0];
  return nickname ? `${firstName} (${nickname})` : firstName;
}

export function formatDate(dateStr, locale = 'pt-BR') {
  if (!dateStr) return '';
  const d = new Date(dateStr + 'T00:00:00');
  return d.toLocaleDateString(locale, { weekday: 'short', day: '2-digit', month: 'short', year: 'numeric' });
}

/**
 * Returns a relative label (Today/Tomorrow/Yesterday) when applicable,
 * otherwise formats with the given locale and options.
 *
 * @param {string} dateStr
 * @param {{ weekday?: string, day?: string, month?: string }} [options]
 * @param {string} [locale] - BCP 47 locale tag (e.g. 'pt-BR', 'en', 'es')
 * @param {{ today?: string, tomorrow?: string, yesterday?: string }} [labels] - localized relative labels
 */
export function relativeDate(
  dateStr,
  options = { weekday: 'long', day: '2-digit', month: 'long' },
  locale = 'pt-BR',
  labels = { today: 'Hoje', tomorrow: 'Amanhã', yesterday: 'Ontem' }
) {
  if (!dateStr) return '';
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const d = new Date(dateStr + 'T00:00');
  const diffDays = Math.round((d.getTime() - today.getTime()) / 86400000);
  if (diffDays === 0) return labels.today;
  if (diffDays === 1) return labels.tomorrow;
  if (diffDays === -1) return labels.yesterday;
  return d.toLocaleDateString(locale, options);
}

export function formatTime(timeStr) {
  if (!timeStr) return '';
  return timeStr.slice(0, 5);
}

export function formatWhatsapp(phone) {
  if (phone.startsWith('+')) {
    // E.164 — display in international format
    const digits = phone.replace(/\D/g, '');
    // Brazilian number: +55 DD XXXXX-XXXX (13 digits total)
    if (digits.startsWith('55') && digits.length === 13) {
      return `+55 (${digits.slice(2,4)}) ${digits.slice(4,9)}-${digits.slice(9)}`;
    }
    return phone;
  }
  // Legacy Brazilian format (digits only, 11 chars)
  const digits = phone.replace(/\D/g, '');
  if (digits.length === 11) {
    return `(${digits.slice(0,2)}) ${digits.slice(2,7)}-${digits.slice(7)}`;
  }
  return phone;
}

export function whatsappLink(phone, text = '') {
  if (phone.startsWith('+')) {
    // E.164 — use directly, stripping the +
    const num = phone.replace(/^\+/, '');
    return `https://wa.me/${num}${text ? `?text=${encodeURIComponent(text)}` : ''}`;
  }
  // Legacy: assume Brazilian
  const digits = phone.replace(/\D/g, '');
  const num = digits.startsWith('55') ? digits : `55${digits}`;
  return `https://wa.me/${num}${text ? `?text=${encodeURIComponent(text)}` : ''}`;
}

export function copyToClipboard(text) {
  return navigator.clipboard.writeText(text);
}

export function toastStore() {
  let toasts = $state([]);
  function show(message, type = 'success') {
    const id = Date.now();
    toasts = [...toasts, { id, message, type }];
    return id;
  }
  function remove(id) {
    toasts = toasts.filter(t => t.id !== id);
  }
  return { get toasts() { return toasts; }, show, remove };
}
