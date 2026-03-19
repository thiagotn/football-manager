import { writable } from 'svelte/store';

type PwaState = {
  canInstall: boolean;   // Android/Chrome: deferred prompt disponível
  isIos: boolean;        // iOS Safari não standalone
  isStandalone: boolean; // App já instalado/rodando em modo standalone
};

function createPwaStore() {
  const { subscribe, set, update } = writable<PwaState>({
    canInstall: false,
    isIos: false,
    isStandalone: false,
  });

  let deferredPrompt: any = null;

  return {
    subscribe,
    init() {
      if (typeof window === 'undefined') return;

      const isStandalone =
        window.matchMedia('(display-mode: standalone)').matches ||
        (window.navigator as any).standalone === true;

      const ua = navigator.userAgent.toLowerCase();
      const isIosDevice =
        /iphone|ipad|ipod/.test(ua) ||
        (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1);

      set({ canInstall: false, isIos: isIosDevice && !isStandalone, isStandalone });

      if (isStandalone) return;

      window.addEventListener('beforeinstallprompt', (e: Event) => {
        e.preventDefault();
        deferredPrompt = e;
        update(s => ({ ...s, canInstall: true }));
      });

      window.addEventListener('appinstalled', () => {
        deferredPrompt = null;
        set({ canInstall: false, isIos: false, isStandalone: true });
      });
    },
    async install() {
      if (!deferredPrompt) return;
      deferredPrompt.prompt();
      const { outcome } = await (deferredPrompt as any).userChoice;
      if (outcome === 'accepted') {
        deferredPrompt = null;
        update(s => ({ ...s, canInstall: false }));
      }
    },
  };
}

export const pwaInstall = createPwaStore();
