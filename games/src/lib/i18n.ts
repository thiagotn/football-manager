/**
 * i18n mínimo — mesmo contrato do football-frontend (`$t('chave', { param })`),
 * com pt-BR embutido e en/es carregados sob demanda.
 */
import { derived, writable } from 'svelte/store';
import ptBR from '../../messages/pt-BR.json';

export type Locale = 'pt-BR' | 'en' | 'es';
export const SUPPORTED_LOCALES: Locale[] = ['pt-BR', 'en', 'es'];
export const DEFAULT_LOCALE: Locale = 'pt-BR';

const LOCALE_KEY = 'games_locale';

type Messages = Record<string, string>;

const messages = writable<Messages>(ptBR as Messages);
export const locale = writable<Locale>(DEFAULT_LOCALE);

export const t = derived(messages, ($messages) => (key: string, params?: Record<string, string | number>): string => {
  let msg = $messages[key] ?? (ptBR as Messages)[key] ?? key;
  if (params) for (const [k, v] of Object.entries(params)) msg = msg.replaceAll(`{${k}}`, String(v));
  return msg;
});

const LOADERS: Record<Locale, () => Promise<{ default: Messages }>> = {
  'pt-BR': () => import('../../messages/pt-BR.json'),
  en: () => import('../../messages/en.json'),
  es: () => import('../../messages/es.json'),
};

export async function setLocale(l: Locale, persist = true): Promise<void> {
  try {
    messages.set(l === 'pt-BR' ? (ptBR as Messages) : (await LOADERS[l]()).default);
  } catch {
    messages.set(ptBR as Messages);
    l = 'pt-BR';
  }
  locale.set(l);
  if (typeof document !== 'undefined') document.documentElement.lang = l;
  if (persist) {
    try {
      localStorage.setItem(LOCALE_KEY, l);
    } catch {
      // modo privado
    }
  }
}

/** Idioma salvo; senão o do navegador (pt → pt-BR, es → es, resto → en). */
export function initLocale(): void {
  let saved: string | null = null;
  try {
    saved = localStorage.getItem(LOCALE_KEY);
  } catch {
    // modo privado
  }
  if (saved && SUPPORTED_LOCALES.includes(saved as Locale)) {
    void setLocale(saved as Locale, false);
    return;
  }
  const nav = (navigator.language || '').toLowerCase();
  const detected: Locale = nav.startsWith('pt') ? 'pt-BR' : nav.startsWith('es') ? 'es' : nav ? 'en' : DEFAULT_LOCALE;
  if (detected !== DEFAULT_LOCALE) void setLocale(detected, false);
}
