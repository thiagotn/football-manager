<script lang="ts">
  import { page } from '$app/stores';
  import { claims, ApiError } from '$lib/api';
  import { authStore } from '$lib/stores/auth';
  import { goto } from '$app/navigation';
  import { CheckCircle, Eye, EyeOff, ArrowLeft } from 'lucide-svelte';
  import PhoneInput from '$lib/components/PhoneInput.svelte';
  import { t } from '$lib/i18n';

  const token = $page.params.token ?? '';

  type Step = 'whatsapp' | 'otp' | 'password';

  let info: { player_first_name: string; group_name: string; expires_at: string } | null = $state(null);
  let errorReason = $state<'expired' | 'used' | 'already_claimed' | 'not_found' | null>(null);
  let loading = $state(true);

  let step = $state<Step>('whatsapp');
  let whatsapp = $state('');
  let otpCode = $state('');
  let otpToken = $state('');
  let password = $state('');
  let passwordConfirm = $state('');
  let showPw = $state(false);

  let sending = $state(false);
  let verifying = $state(false);
  let submitting = $state(false);
  let done = $state(false);
  let error = $state('');

  $effect(() => {
    let cancelled = false;
    (async () => {
      try {
        const i = await claims.getInfo(token);
        if (!cancelled) info = i;
      } catch (e) {
        if (!cancelled) {
          if (e instanceof ApiError) {
            if (e.message === 'INVITE_EXPIRED') errorReason = 'expired';
            else if (e.message === 'INVITE_USED') errorReason = 'used';
            else if (e.message === 'ALREADY_CLAIMED') errorReason = 'already_claimed';
            else errorReason = 'not_found';
          } else {
            errorReason = 'not_found';
          }
        }
      }
      if (!cancelled) loading = false;
    })();
    return () => { cancelled = true; };
  });

  function mapError(e: unknown, fallback: string): string {
    if (e instanceof ApiError) {
      if (e.message === 'WHATSAPP_TAKEN') return $t('claim.whatsapp_taken');
      if (e.message === 'OTP_INVALID') return $t('claim.otp_invalid');
      if (e.message === 'OTP_TOKEN_INVALID') return $t('claim.otp_invalid');
      return e.message;
    }
    return fallback;
  }

  async function sendOtp() {
    error = '';
    sending = true;
    try {
      await claims.sendOtp(token, whatsapp);
      otpCode = '';
      step = 'otp';
    } catch (e) {
      error = mapError(e, $t('claim.send_error'));
    }
    sending = false;
  }

  async function verifyOtp() {
    error = '';
    verifying = true;
    try {
      const res = await claims.verifyOtp(token, whatsapp, otpCode);
      otpToken = res.otp_token;
      step = 'password';
    } catch (e) {
      error = mapError(e, $t('claim.verify_error'));
    }
    verifying = false;
  }

  async function complete() {
    error = '';
    if (password !== passwordConfirm) {
      error = $t('claim.password_mismatch');
      return;
    }
    submitting = true;
    try {
      const res = await claims.complete(token, { whatsapp, otp_token: otpToken, password });
      authStore.login(res.access_token, res.refresh_token ?? null, res);
      done = true;
      setTimeout(() => goto('/'), 2000);
    } catch (e) {
      error = mapError(e, $t('claim.finish_error'));
    }
    submitting = false;
  }

  function backToWhatsapp() {
    error = '';
    otpCode = '';
    step = 'whatsapp';
  }

  function fmtExpiry(s: string) {
    return new Date(s).toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' });
  }
</script>

<svelte:head><title>{$t('claim.title')} — rachao.app</title></svelte:head>

<div class="min-h-screen bg-gradient-to-br from-primary-700 to-primary-900 flex items-center justify-center p-4">
  <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-2xl w-full max-w-sm p-8">
    <div class="text-center mb-6">
      <div class="text-4xl mb-2">⚽</div>
      <h1 class="text-xl font-bold text-gray-900 dark:text-gray-100">{$t('claim.title')}</h1>
    </div>

    {#if loading}
      <div class="animate-pulse space-y-3">
        <div class="h-4 bg-gray-200 rounded"></div>
        <div class="h-4 bg-gray-200 rounded w-2/3"></div>
      </div>

    {:else if errorReason === 'expired'}
      <div class="alert-error text-center">
        <p class="font-semibold">{$t('claim.expired_title')}</p>
        <p class="mt-1 text-xs">{$t('claim.expired_desc')}</p>
      </div>

    {:else if errorReason === 'used'}
      <div class="alert-error text-center">
        <p class="font-semibold">{$t('claim.used_title')}</p>
        <p class="mt-1 text-xs">{$t('claim.used_desc')}</p>
      </div>

    {:else if errorReason === 'already_claimed'}
      <div class="alert-info text-center">
        <p class="font-semibold">{$t('claim.already_claimed_title')}</p>
        <p class="mt-1 text-xs">{$t('claim.already_claimed_desc')}</p>
      </div>

    {:else if errorReason === 'not_found'}
      <div class="alert-error text-center">
        <p class="font-semibold">{$t('claim.invalid_title')}</p>
        <p class="mt-1 text-xs">{$t('claim.invalid_desc')}</p>
      </div>

    {:else if done}
      <div class="text-center py-4">
        <CheckCircle size={48} class="text-green-500 mx-auto mb-3" />
        <p class="font-semibold text-gray-900 dark:text-gray-100">{$t('claim.success')}</p>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">{$t('claim.redirecting')}</p>
      </div>

    {:else if info}
      <div class="alert-info mb-5 text-center">
        <p class="font-semibold">{$t('claim.greeting').replace('{name}', info.player_first_name)}</p>
        <p class="text-sm mt-1">{$t('claim.intro').replace('{group}', info.group_name)}</p>
        <p class="text-xs mt-1 text-blue-600">{$t('claim.expires_at').replace('{date}', fmtExpiry(info.expires_at))}</p>
      </div>

      {#if error}
        <div class="alert-error mb-4">{error}</div>
      {/if}

      <!-- ETAPA 1: WhatsApp real -->
      {#if step === 'whatsapp'}
        <form onsubmit={(e) => { e.preventDefault(); sendOtp(); }} class="space-y-4">
          <div class="form-group">
            <label class="label" for="wa">{$t('claim.whatsapp_label')}</label>
            <PhoneInput id="wa" bind:value={whatsapp} placeholder="11999990000" required />
            <p class="text-xs text-gray-400 mt-1">{$t('claim.whatsapp_hint')}</p>
          </div>
          <button type="submit" class="btn-primary w-full justify-center py-2.5" disabled={sending || !whatsapp}>
            {sending ? $t('claim.sending') : $t('claim.send_code')}
          </button>
        </form>

      <!-- ETAPA 2: Código OTP -->
      {:else if step === 'otp'}
        <form onsubmit={(e) => { e.preventDefault(); verifyOtp(); }} class="space-y-4">
          <p class="text-sm text-gray-500 dark:text-gray-400">{$t('claim.otp_sent').replace('{phone}', whatsapp)}</p>
          <div class="form-group">
            <label class="label" for="otp">{$t('claim.otp_label')}</label>
            <input
              id="otp" class="input text-center text-xl tracking-[0.5em] font-mono" type="text"
              inputmode="numeric" autocomplete="one-time-code" maxlength="6" minlength="6"
              pattern="[0-9]*" bind:value={otpCode} required
            />
          </div>
          <button type="submit" class="btn-primary w-full justify-center py-2.5" disabled={verifying || otpCode.length !== 6}>
            {verifying ? $t('claim.verifying') : $t('claim.verify')}
          </button>
          <div class="flex items-center justify-between text-xs">
            <button type="button" onclick={backToWhatsapp}
              class="flex items-center gap-1 text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-400">
              <ArrowLeft size={13} /> {$t('claim.change_number')}
            </button>
            <button type="button" onclick={sendOtp} disabled={sending}
              class="text-primary-500 hover:text-primary-600 disabled:opacity-50">
              {sending ? '...' : $t('claim.resend')}
            </button>
          </div>
        </form>

      <!-- ETAPA 3: Senha -->
      {:else if step === 'password'}
        <form onsubmit={(e) => { e.preventDefault(); complete(); }} class="space-y-4">
          <p class="text-sm text-gray-500 dark:text-gray-400">{$t('claim.password_intro')}</p>
          <div class="form-group">
            <label class="label" for="pw">{$t('claim.password_label')}</label>
            <div class="relative">
              <input id="pw" class="input pr-10" type={showPw ? 'text' : 'password'}
                bind:value={password} placeholder={$t('claim.password_placeholder')} required minlength="6" />
              <button type="button" onclick={() => showPw = !showPw}
                class="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600">
                {#if showPw}<EyeOff size={16} />{:else}<Eye size={16} />{/if}
              </button>
            </div>
          </div>
          <div class="form-group">
            <label class="label" for="pw-confirm">{$t('claim.password_confirm_label')}</label>
            <input id="pw-confirm" class="input" type={showPw ? 'text' : 'password'}
              bind:value={passwordConfirm} required minlength="6" />
          </div>
          <button type="submit" class="btn-primary w-full justify-center py-2.5" disabled={submitting || password.length < 6}>
            {submitting ? $t('claim.finishing') : $t('claim.finish')}
          </button>
        </form>
      {/if}
    {/if}
  </div>
</div>
