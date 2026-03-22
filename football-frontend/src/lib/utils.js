export function formatDate(dateStr) {
  if (!dateStr) return '';
  const d = new Date(dateStr + 'T00:00:00');
  return d.toLocaleDateString('pt-BR', { weekday: 'short', day: '2-digit', month: 'short', year: 'numeric' });
}

/**
 * Retorna 'Hoje', 'Amanhã' ou 'Ontem' quando aplicável,
 * caso contrário formata com as opções fornecidas.
 */
export function relativeDate(dateStr, options = { weekday: 'long', day: '2-digit', month: 'long' }) {
  if (!dateStr) return '';
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const d = new Date(dateStr + 'T00:00');
  const diffDays = Math.round((d.getTime() - today.getTime()) / 86400000);
  if (diffDays === 0) return 'Hoje';
  if (diffDays === 1) return 'Amanhã';
  if (diffDays === -1) return 'Ontem';
  return d.toLocaleDateString('pt-BR', options);
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
