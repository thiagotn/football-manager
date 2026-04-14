<script lang="ts">
  // Este componente só é montado no cliente (via {#if browser} no layout).
  // Toda detecção pode ser feita sincronamente, sem onMount.
  import { X, Smartphone } from 'lucide-svelte';
  import { androidBeta } from '$lib/api';

  const DISMISSED_KEY = 'android_beta_dismissed';
  const SUBMITTED_KEY = 'android_beta_submitted';
  const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

  const ua = navigator.userAgent.toLowerCase();
  const isStandalone =
    window.matchMedia('(display-mode: standalone)').matches ||
    (navigator as any).standalone === true;
  const isAndroid = /android/.test(ua) && !isStandalone;

  let dismissed  = $state(!!localStorage.getItem(DISMISSED_KEY));
  let submitted  = $state(!!localStorage.getItem(SUBMITTED_KEY));
  let success    = $state(false);
  let email      = $state('');
  let emailError = $state('');
  let loading    = $state(false);

  function dismiss() {
    dismissed = true;
    success = false;
    localStorage.setItem(DISMISSED_KEY, '1');
  }

  async function submit() {
    emailError = '';
    const trimmed = email.trim();
    if (!EMAIL_RE.test(trimmed)) {
      emailError = 'Informe um email Google válido';
      return;
    }
    loading = true;
    try {
      await androidBeta.submit(trimmed);
      localStorage.setItem(SUBMITTED_KEY, '1');
      submitted = true;
      success = true;
    } catch {
      emailError = 'Erro ao enviar. Tente novamente.';
    } finally {
      loading = false;
    }
  }
</script>

{#if (isAndroid && !dismissed && !submitted) || success}
  <div
    class="fixed bottom-0 inset-x-0 z-50 bg-primary-900/95 backdrop-blur border-t border-primary-700/60 shadow-2xl"
    style="padding-bottom: env(safe-area-inset-bottom)"
  >
    <div class="max-w-lg mx-auto px-4 py-3">

      {#if success}
        <!-- Estado de sucesso -->
        <div class="flex items-center gap-3">
          <img src="/logo.png" alt="rachao.app" class="w-10 h-10 object-contain shrink-0 rounded-xl" />
          <p class="flex-1 text-sm font-semibold text-white leading-tight">
            Inscrição feita! Em breve você receberá o convite no email informado.
          </p>
          <button
            onclick={dismiss}
            class="shrink-0 p-1.5 rounded-lg text-primary-400 hover:text-white hover:bg-primary-700 transition-colors"
            aria-label="Fechar"
          >
            <X size={16} />
          </button>
        </div>

      {:else}
        <!-- Estado padrão -->
        <div class="flex items-center gap-2">
          <img src="/logo.png" alt="rachao.app" class="w-8 h-8 object-contain shrink-0 rounded-lg" />
          <div class="flex-1 min-w-0">
            <p class="text-sm font-semibold text-white leading-tight">App Android em beta!</p>
            <p class="text-xs text-primary-300">Teste o app nativo antes do lançamento</p>
          </div>
          <button
            onclick={dismiss}
            class="shrink-0 p-1.5 rounded-lg text-primary-400 hover:text-white hover:bg-primary-700 transition-colors"
            aria-label="Fechar"
          >
            <X size={16} />
          </button>
        </div>

        <!-- Campo de email -->
        <div class="mt-2.5 pt-2.5 border-t border-primary-700/60">
          <p class="text-xs text-primary-300 mb-2">Email Google para receber o convite do Play Store:</p>
          <div class="flex gap-2">
            <input
              type="email"
              bind:value={email}
              placeholder="seu@gmail.com"
              onkeydown={(e) => e.key === 'Enter' && submit()}
              class="flex-1 min-w-0 px-3 py-1.5 text-sm rounded-lg bg-primary-800 border border-primary-600
                     text-white placeholder-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-400"
            />
            <button
              onclick={submit}
              disabled={loading}
              class="shrink-0 inline-flex items-center gap-1.5 px-4 py-1.5 bg-primary-500
                     hover:bg-primary-400 active:bg-primary-600 text-white text-sm font-semibold
                     rounded-lg transition-colors disabled:opacity-50"
            >
              {loading ? '...' : 'Quero testar'}
            </button>
          </div>
          {#if emailError}
            <p class="text-xs text-red-400 mt-1">{emailError}</p>
          {/if}
        </div>
      {/if}

    </div>
  </div>
{/if}
