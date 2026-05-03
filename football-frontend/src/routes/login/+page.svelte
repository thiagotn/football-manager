<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { auth, ApiError } from '$lib/api';
  import { authStore } from '$lib/stores/auth';
  import { goto } from '$app/navigation';
  import { toastError, toastSuccess } from '$lib/stores/toast';
  import { Eye, EyeOff, LogIn, ShieldCheck } from 'lucide-svelte';
  import PwaSmartBanner from '$lib/components/PwaSmartBanner.svelte';
  import PhoneInput from '$lib/components/PhoneInput.svelte';
  import { t, setLocale, isLocaleUserChosen, type Locale } from '$lib/i18n';

  // ── Auto locale by country ──────────────────────────────────
  const SPANISH_COUNTRIES = new Set(['ES','AR','MX','CL','CO','PE','UY','PY','BO','VE','EC']);

  function handleCountryChange(countryCode: string) {
    if (isLocaleUserChosen()) return;
    let newLocale: Locale;
    if (countryCode === 'BR') newLocale = 'pt-BR';
    else if (SPANISH_COUNTRIES.has(countryCode)) newLocale = 'es';
    else newLocale = 'en';
    setLocale(newLocale, 'auto');
  }

  // ── Login ──────────────────────────────────────────────────
  let whatsapp = $state('');
  let password = $state('');
  let loading = $state(false);
  let showPw = $state(false);
  let error = $state('');

  // Banner reativo ao parâmetro de URL — sem sessionStorage
  let sessionExpired = $derived($page.url.searchParams.get('expired') === '1');

  onMount(() => {
    // Limpa o token stale quando chegamos via expiração de sessão
    if ($page.url.searchParams.get('expired') === '1') {
      authStore.logout();
    }
  });

  async function handleLogin() {
    error = '';
    loading = true;
    try {
      const res = await auth.login(whatsapp, password);
      authStore.login(res.access_token, res.refresh_token ?? null, res);
      if (res.must_change_password) {
        goto('/profile');
      } else {
        const nextUrl = $page.url.searchParams.get('next');
        const joinWaitlist = $page.url.searchParams.get('join_waitlist');
        if (nextUrl) {
          const dest = joinWaitlist ? `${nextUrl}?join_waitlist=${joinWaitlist}` : nextUrl;
          goto(dest, { replaceState: true });
        } else {
          goto('/');
        }
      }
    } catch (e) {
      error = e instanceof ApiError ? e.message : $t('login.connect_error');
    } finally {
      loading = false;
    }
  }

  // ── Forgot password ────────────────────────────────────────
  type ForgotMode = 'idle' | 'whatsapp' | 'otp-pending' | 'otp-verified';
  let forgotMode = $state<ForgotMode>('idle');
  let forgotWhatsapp = $state('');
  let forgotOtpCode = $state('');
  let forgotOtpToken = $state('');
  let forgotNewPw = $state('');
  let forgotConfirmPw = $state('');
  let forgotShowPw = $state(false);
  let forgotLoading = $state(false);
  let forgotError = $state('');
  let otpCountdown = $state(0);
  let otpTimer: ReturnType<typeof setInterval> | null = null;


  function startCountdown() {
    otpCountdown = 60;
    if (otpTimer) clearInterval(otpTimer);
    otpTimer = setInterval(() => {
      otpCountdown--;
      if (otpCountdown <= 0 && otpTimer) clearInterval(otpTimer);
    }, 1000);
  }

  function cancelForgot() {
    forgotMode = 'idle';
    forgotWhatsapp = '';
    forgotOtpCode = '';
    forgotOtpToken = '';
    forgotNewPw = '';
    forgotConfirmPw = '';
    forgotError = '';
    if (otpTimer) clearInterval(otpTimer);
  }

  async function sendForgotOtp() {
    if (!forgotWhatsapp) return;
    forgotLoading = true;
    forgotError = '';
    try {
      await auth.forgotPasswordSendOtp(forgotWhatsapp);
      forgotMode = 'otp-pending';
      startCountdown();
    } catch (e) {
      forgotError = e instanceof ApiError ? e.message : $t('login.connect_error');
    }
    forgotLoading = false;
  }

  async function verifyForgotOtp() {
    if (forgotOtpCode.length !== 6) return;
    forgotLoading = true;
    forgotError = '';
    try {
      const res = await auth.forgotPasswordVerifyOtp(forgotWhatsapp, forgotOtpCode);
      forgotOtpToken = res.otp_token;
      forgotMode = 'otp-verified';
    } catch {
      forgotError = $t('login.pw_mismatch');
    }
    forgotLoading = false;
  }

  let forgotPwError = $derived(
    forgotNewPw && forgotConfirmPw && forgotNewPw !== forgotConfirmPw ? $t('login.pw_mismatch') :
    forgotNewPw && forgotNewPw.length < 6 ? $t('login.pw_too_short') :
    null
  );

  async function resetPassword() {
    if (forgotPwError || !forgotNewPw || !forgotConfirmPw) return;
    forgotLoading = true;
    forgotError = '';
    try {
      await auth.forgotPasswordReset(forgotWhatsapp, forgotOtpToken, forgotNewPw);
      toastSuccess($t('login.reset_success'));
      cancelForgot();
    } catch (e) {
      if (e instanceof ApiError && e.message === 'SAME_PASSWORD') {
        forgotError = $t('auth.same_password_error');
      } else {
        forgotError = e instanceof ApiError ? e.message : $t('login.connect_error');
      }
    }
    forgotLoading = false;
  }
</script>

<svelte:head><title>{$t('login.page_title')}</title></svelte:head>

<PwaSmartBanner />

<div class="min-h-screen flex items-center justify-center p-4 relative bg-primary-900"
  style="background-image: url('/background-login.png'); background-size: cover; background-position: center;">
  <div class="absolute inset-0 bg-primary-900/65"></div>
  <div class="relative z-10 bg-white dark:bg-gray-800 rounded-2xl shadow-2xl w-full max-w-sm p-8">

    <div class="text-center mb-8">
      <img src="/logo.png" alt="rachao.app" width="320" height="174" class="w-56 block mx-auto mb-1" />
      <p class="text-sm text-gray-500 dark:text-gray-400">{$t('login.subtitle')}</p>
    </div>

    {#if forgotMode === 'idle'}
      <!-- ── Login normal ── -->
      {#if sessionExpired}
        <div class="mb-4 px-4 py-3 rounded-lg bg-amber-500/10 border border-amber-500/30 text-amber-700 dark:text-amber-300 text-sm" data-testid="session-expired-banner">
          {$t('login.session_expired')}
        </div>
      {/if}
      {#if error}
        <div class="alert-error mb-4">{error}</div>
      {/if}

      <form onsubmit={(e) => { e.preventDefault(); handleLogin(); }} class="space-y-4">
        <div class="form-group">
          <label class="label" for="whatsapp">WhatsApp</label>
          <PhoneInput id="whatsapp" bind:value={whatsapp} placeholder="11999990000" required oncountrychange={handleCountryChange} />
        </div>

        <div class="form-group">
          <label class="label" for="password">{$t('login.password_label')}</label>
          <div class="relative">
            <input id="password" class="input pr-10"
              type={showPw ? 'text' : 'password'}
              bind:value={password} placeholder="••••••" required
              autocomplete="current-password" />
            <button type="button" onclick={() => showPw = !showPw}
              class="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600">
              {#if showPw}<EyeOff size={16} />{:else}<Eye size={16} />{/if}
            </button>
          </div>
          <button type="button"
            onclick={() => { forgotWhatsapp = whatsapp; forgotError = ''; forgotMode = 'whatsapp'; }}
            class="mt-1.5 text-xs text-primary-600 dark:text-primary-400 hover:underline">
            {$t('login.forgot_password')}
          </button>
        </div>

        <button type="submit" class="btn-primary w-full justify-center py-2.5" disabled={loading}>
          <LogIn size={16} />
          {loading ? $t('login.loading') : $t('login.submit')}
        </button>
      </form>

      <p class="text-xs text-gray-400 dark:text-gray-500 text-center mt-6">
        {#if $page.url.searchParams.get('next')}
          {$t('login.no_account')} <a href="/register?next={$page.url.searchParams.get('next')}{$page.url.searchParams.get('join_waitlist') ? '&join_waitlist=' + $page.url.searchParams.get('join_waitlist') : ''}" class="text-primary-600 hover:underline">{$t('login.register_free')}</a>
        {:else}
          {$t('login.no_account')} <a href="/register" class="text-primary-600 hover:underline">{$t('login.register_free')}</a>
        {/if}
      </p>

    {:else if forgotMode === 'whatsapp'}
      <!-- ── Confirmar número ── -->
      <div class="space-y-4">
        <div>
          <h2 class="font-semibold text-gray-800 dark:text-gray-200 mb-1">{$t('login.forgot_title')}</h2>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {$t('login.forgot_subtitle')}
          </p>
        </div>

        <div class="form-group">
          <label class="label" for="forgot-whatsapp">{$t('login.phone_label')}</label>
          <PhoneInput id="forgot-whatsapp" bind:value={forgotWhatsapp} placeholder="11999990000" disabled={forgotLoading} oncountrychange={handleCountryChange} />
          <p class="text-xs text-gray-400 mt-1">{$t('login.phone_hint')}</p>
        </div>

        {#if forgotError}<div class="alert-error text-sm">{forgotError}</div>{/if}

        <button onclick={sendForgotOtp}
          disabled={forgotLoading || !forgotWhatsapp}
          class="btn-primary w-full justify-center py-2.5">
          {forgotLoading ? $t('login.send_code_loading') : $t('login.send_code')}
        </button>

        <button type="button" onclick={cancelForgot}
          class="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 w-full text-center">
          {$t('login.back_to_login')}
        </button>
      </div>

    {:else if forgotMode === 'otp-pending'}
      <!-- ── Aguardando código SMS ── -->
      <div class="space-y-4">
        <div>
          <h2 class="font-semibold text-gray-800 dark:text-gray-200 mb-1">{$t('login.verify_title')}</h2>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {$t('login.verify_subtitle')} <strong>{forgotWhatsapp}</strong>
          </p>
        </div>

        <div class="form-group">
          <label class="label" for="otp-code">{$t('login.otp_label')}</label>
          <input id="otp-code" class="input text-center text-xl tracking-widest font-mono"
            type="text" inputmode="numeric" pattern="[0-9]*" maxlength="6"
            autocomplete="one-time-code"
            bind:value={forgotOtpCode} placeholder="000000"
            disabled={forgotLoading} />
        </div>

        {#if forgotError}<div class="alert-error text-sm">{forgotError}</div>{/if}

        <button onclick={verifyForgotOtp}
          disabled={forgotLoading || forgotOtpCode.length !== 6}
          class="btn-primary w-full justify-center py-2.5">
          {forgotLoading ? $t('login.verify_loading') : $t('login.verify_submit')}
        </button>

        <div class="flex items-center justify-between text-xs text-gray-400">
          <button onclick={cancelForgot} class="hover:text-gray-600 dark:hover:text-gray-200">{$t('login.back')}</button>
          {#if otpCountdown > 0}
            <span>{$t('login.resend_countdown').replace('{s}', String(otpCountdown))}</span>
          {:else}
            <button onclick={sendForgotOtp} disabled={forgotLoading}
              class="text-primary-600 dark:text-primary-400 hover:underline disabled:opacity-50">
              {forgotLoading ? $t('login.resend_loading') : $t('login.resend')}
            </button>
          {/if}
        </div>
      </div>

    {:else if forgotMode === 'otp-verified'}
      <!-- ── Definir nova senha ── -->
      <div class="space-y-4">
        <div class="flex items-center gap-2 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg px-4 py-2.5 text-sm text-green-800 dark:text-green-300">
          <ShieldCheck size={16} class="shrink-0" />
          {$t('login.identity_verified')}
        </div>

        <div class="form-group">
          <label class="label" for="new-pw">{$t('login.new_password_label')}</label>
          <div class="relative">
            <input id="new-pw" class="input pr-10"
              type={forgotShowPw ? 'text' : 'password'}
              bind:value={forgotNewPw} placeholder={$t('login.new_password_placeholder')}
              required minlength="6" autocomplete="new-password"
              disabled={forgotLoading} />
            <button type="button" onclick={() => forgotShowPw = !forgotShowPw}
              class="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600">
              {#if forgotShowPw}<EyeOff size={15} />{:else}<Eye size={15} />{/if}
            </button>
          </div>
        </div>

        <div class="form-group">
          <label class="label" for="confirm-pw">{$t('login.confirm_password_label')}</label>
          <input id="confirm-pw" class="input"
            type="password" bind:value={forgotConfirmPw}
            placeholder={$t('login.confirm_password_placeholder')} required autocomplete="new-password"
            disabled={forgotLoading} />
        </div>

        {#if forgotPwError}
          <div class="alert-error text-sm">{forgotPwError}</div>
        {/if}
        {#if forgotError}
          <div class="alert-error text-sm">{forgotError}</div>
        {/if}

        <button onclick={resetPassword}
          disabled={forgotLoading || !!forgotPwError || !forgotNewPw || !forgotConfirmPw}
          class="btn-primary w-full justify-center py-2.5">
          {forgotLoading ? $t('login.reset_loading') : $t('login.reset_submit')}
        </button>

        <button type="button" onclick={cancelForgot}
          class="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 w-full text-center">
          {$t('login.cancel')}
        </button>
      </div>
    {/if}

    <div class="flex justify-center gap-4 mt-6 text-xs text-gray-300 dark:text-gray-600">
      <a href="/terms" class="hover:text-gray-500 transition-colors">{$t('login.terms')}</a>
      <a href="/privacy" class="hover:text-gray-500 transition-colors">{$t('login.privacy')}</a>
    </div>

  </div>
</div>
