/**
 * Voz dos jogos infantis (Bê-á-Bá, Quebra-Cuca): `speechSynthesis` com a voz
 * do próprio aparelho, no idioma pedido. Sem suporte (ou com a voz desligada nos ajustes) vira silêncio — o
 * jogo continua jogável só com imagem e animação.
 *
 * O último texto falado fica em `document.documentElement.dataset.spoken`
 * para os testes E2E (a fala em si não é observável no navegador sem som).
 */

export type SpeakOptions = { lang: string; rate?: number; enabled?: boolean };

let voices: SpeechSynthesisVoice[] = [];

function loadVoices() {
  if (typeof speechSynthesis === 'undefined') return;
  voices = speechSynthesis.getVoices();
}

if (typeof speechSynthesis !== 'undefined') {
  loadVoices();
  speechSynthesis.addEventListener?.('voiceschanged', loadVoices);
}

/** Voz do idioma exato (`pt-BR`), senão da língua (`pt`), senão a padrão. */
export function pickVoice(lang: string, list: readonly Pick<SpeechSynthesisVoice, 'lang'>[] = voices) {
  const norm = (l: string) => l.replace('_', '-').toLowerCase();
  const want = norm(lang);
  return (
    list.find((v) => norm(v.lang) === want) ?? list.find((v) => norm(v.lang).startsWith(want.slice(0, 2))) ?? null
  );
}

/**
 * Fala `text` (interrompe a fala anterior). Palavras vão em minúscula: em
 * CAIXA ALTA alguns motores soletram como sigla ("U-V-A").
 */
export function speak(text: string, { lang, rate = 0.85, enabled = true }: SpeakOptions): void {
  if (typeof document !== 'undefined') document.documentElement.dataset.spoken = text;
  if (!enabled || typeof speechSynthesis === 'undefined') return;
  try {
    speechSynthesis.cancel();
    const u = new SpeechSynthesisUtterance(text.toLocaleLowerCase(lang));
    u.lang = lang;
    const voice = pickVoice(lang) as SpeechSynthesisVoice | null;
    if (voice) u.voice = voice;
    u.rate = rate;
    speechSynthesis.speak(u);
  } catch {
    // Motor de voz indisponível: segue sem som.
  }
}

/**
 * Fala várias partes em sequência, cada uma no seu idioma — é a ponte da
 * trilha Inglês: "Great job! Apple!" (en-US) e depois "maçã" (pt-BR).
 */
export function speakSequence(parts: { text: string; lang: string }[], { rate = 0.85, enabled = true } = {}): void {
  const said = parts.filter((p) => p.text);
  if (typeof document !== 'undefined') document.documentElement.dataset.spoken = said.map((p) => p.text).join(' | ');
  if (!enabled || typeof speechSynthesis === 'undefined') return;
  try {
    speechSynthesis.cancel();
    for (const part of said) {
      const u = new SpeechSynthesisUtterance(part.text.toLocaleLowerCase(part.lang));
      u.lang = part.lang;
      const voice = pickVoice(part.lang) as SpeechSynthesisVoice | null;
      if (voice) u.voice = voice;
      u.rate = rate;
      speechSynthesis.speak(u);
    }
  } catch {
    // Motor de voz indisponível: segue sem som.
  }
}

export function stopSpeaking(): void {
  try {
    if (typeof speechSynthesis !== 'undefined') speechSynthesis.cancel();
  } catch {
    // nada
  }
}
