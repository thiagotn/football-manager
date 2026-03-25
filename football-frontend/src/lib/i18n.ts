import { writable, derived } from 'svelte/store';
// Pre-import pt-BR synchronously so there's no flash on first render for the default locale
import ptBR from '../../messages/pt-BR.json';

export type Locale = 'pt-BR' | 'en' | 'es';

export const SUPPORTED_LOCALES: Locale[] = ['pt-BR', 'en', 'es'];
export const DEFAULT_LOCALE: Locale = 'pt-BR';

const LOCALE_KEY = 'rachao_locale';
const LOCALE_SOURCE_KEY = 'rachao_locale_source';
type LocaleSource = 'user' | 'auto';

type Messages = Record<string, string>;

const _messages = writable<Messages>(ptBR as Messages);
export const locale = writable<Locale>(DEFAULT_LOCALE);

/**
 * Reactive translation function.
 * Usage in templates: {$t('nav.groups')}
 * Usage with params:  {$t('match.players', { count: 10 })}
 */
export const t = derived(
  [locale, _messages],
  ([$locale, $messages]) =>
    (key: string, params?: Record<string, string | number>): string => {
      let msg = $messages[key] ?? key;
      if (params) {
        for (const [k, v] of Object.entries(params)) {
          msg = msg.replace(`{${k}}`, String(v));
        }
      }
      return msg;
    }
);

// Lazy-load map — Vite bundles each locale as a separate chunk
const LOADERS: Record<Locale, () => Promise<{ default: Messages }>> = {
  'pt-BR': () => import('../../messages/pt-BR.json'),
  'en':    () => import('../../messages/en.json'),
  'es':    () => import('../../messages/es.json'),
};

async function loadMessages(l: Locale): Promise<void> {
  if (l === 'pt-BR') {
    _messages.set(ptBR as Messages);
    return;
  }
  try {
    const mod = await LOADERS[l]();
    _messages.set(mod.default);
  } catch {
    // Fallback to pt-BR if load fails
    _messages.set(ptBR as Messages);
  }
}

/**
 * Switch locale, persist to localStorage, reload messages.
 * source='user'  → explicit choice via language switcher (won't be overridden by auto-detect)
 * source='auto'  → inferred from navigator.language or country selector (can be overridden)
 */
export async function setLocale(newLocale: Locale, source: LocaleSource = 'user'): Promise<void> {
  await loadMessages(newLocale);
  locale.set(newLocale);
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem(LOCALE_KEY, newLocale);
    localStorage.setItem(LOCALE_SOURCE_KEY, source);
  }
}

/**
 * Returns true if the user explicitly chose a locale via the language switcher.
 * When true, automatic locale changes (e.g. from country selector) should be skipped.
 */
export function isLocaleUserChosen(): boolean {
  if (typeof localStorage === 'undefined') return false;
  return localStorage.getItem(LOCALE_SOURCE_KEY) === 'user';
}

/**
 * Detect and initialize locale from localStorage → navigator.language → default (pt-BR).
 * Call once in the root layout's onMount.
 */
export async function initLocale(): Promise<void> {
  if (typeof window === 'undefined') return;

  const saved = localStorage.getItem(LOCALE_KEY) as Locale | null;

  if (saved && SUPPORTED_LOCALES.includes(saved)) {
    // Restore without changing the source — keep whatever was persisted
    await loadMessages(saved);
    locale.set(saved);
    return;
  }

  const nav = navigator.language;
  let detected: Locale = DEFAULT_LOCALE;
  if (nav.startsWith('pt')) detected = 'pt-BR';
  else if (nav.startsWith('es')) detected = 'es';
  else if (nav.startsWith('en')) detected = 'en';

  await setLocale(detected, 'auto');
}
