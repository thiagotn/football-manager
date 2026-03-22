<script lang="ts">
  import { auth, ApiError } from '$lib/api';
  import { authStore } from '$lib/stores/auth';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { Eye, EyeOff, UserPlus, MessageCircle, RotateCcw, ShieldCheck } from 'lucide-svelte';
  import { getPlan, formatCents } from '$lib/plans';
  import PhoneInput from '$lib/components/PhoneInput.svelte';

  const planKey = $derived($page.url.searchParams.get('plan') ?? 'free');
  const plan = $derived(getPlan(planKey));

  // ── Step 1 — WhatsApp ──────────────────────────────────────
  let whatsapp = $state('');

  // ── Step 2 — OTP ───────────────────────────────────────────
  let digits = $state(['', '', '', '', '', '']);
  let inputRefs: HTMLInputElement[] = [];
  let countdown = $state(0);
  let countdownTimer: ReturnType<typeof setInterval> | null = null;
  let otpToken = $state('');

  // ── Step 3 — Form ──────────────────────────────────────────
  let name = $state('');
  let nickname = $state('');
  let password = $state('');
  let confirmPassword = $state('');
  let showPw = $state(false);
  let termsAccepted = $state(false);

  // ── Shared ─────────────────────────────────────────────────
  let step = $state<'whatsapp' | 'otp' | 'form'>('whatsapp');
  let loading = $state(false);
  let error = $state('');

  const maskedWhatsapp = $derived(
    // Show country code + masked local number: +55 (11) ••••• 0000
    whatsapp.replace(/(\+\d{1,3})(\d{2})(\d+)(\d{4})$/, '$1 ($2) ••••• $4') || whatsapp
  );
  const otpCode = $derived(digits.join(''));

  function startCountdown() {
    countdown = 60;
    if (countdownTimer) clearInterval(countdownTimer);
    countdownTimer = setInterval(() => {
      countdown--;
      if (countdown <= 0) { clearInterval(countdownTimer!); countdownTimer = null; }
    }, 1000);
  }

  async function handleSendOtp() {
    error = '';
    loading = true;
    try {
      await auth.sendOtp(whatsapp);
      step = 'otp';
      startCountdown();
      setTimeout(() => inputRefs[0]?.focus(), 50);
    } catch (e) {
      if (e instanceof ApiError) {
        error = e.status === 409 ? 'Este WhatsApp já está cadastrado.'
          : e.status === 429 ? 'Muitas tentativas. Aguarde antes de solicitar um novo código.'
          : e.message;
      } else { error = 'Erro ao conectar'; }
    } finally { loading = false; }
  }

  async function handleResend() {
    if (countdown > 0) return;
    error = '';
    loading = true;
    try {
      await auth.sendOtp(whatsapp);
      digits = ['', '', '', '', '', ''];
      startCountdown();
      setTimeout(() => inputRefs[0]?.focus(), 50);
    } catch (e) {
      error = e instanceof ApiError && e.status === 429
        ? 'Muitas tentativas. Tente novamente mais tarde.'
        : 'Erro ao reenviar código';
    } finally { loading = false; }
  }

  async function handleVerifyOtp() {
    error = '';
    if (otpCode.length < 6) { error = 'Digite o código completo de 6 dígitos'; return; }
    loading = true;
    try {
      const res = await auth.verifyOtp(whatsapp, otpCode);
      otpToken = res.otp_token;
      step = 'form';
    } catch (e) {
      if (e instanceof ApiError) {
        if (e.message === 'OTP_INVALID' || e.message === 'OTP_EXPIRED') {
          error = 'Código inválido ou expirado. Verifique e tente novamente.';
        } else if (e.message === 'OTP_MAX_ATTEMPTS') {
          error = 'Tentativas esgotadas. Solicite um novo código.';
          digits = ['', '', '', '', '', ''];
        } else { error = e.message; }
      } else { error = 'Erro ao conectar'; }
    } finally { loading = false; }
  }

  async function handleRegister() {
    error = '';
    if (password !== confirmPassword) { error = 'As senhas não coincidem'; return; }
    loading = true;
    try {
      const res = await auth.register({ name, whatsapp, password, nickname: nickname || undefined, otp_token: otpToken });
      authStore.login(res.access_token, res);
      const nextUrl = $page.url.searchParams.get('next');
      const joinWaitlist = $page.url.searchParams.get('join_waitlist');
      if (nextUrl) {
        const dest = joinWaitlist ? `${nextUrl}?join_waitlist=${joinWaitlist}` : nextUrl;
        goto(dest);
      } else {
        goto('/');
      }
    } catch (e) {
      if (e instanceof ApiError) {
        if (e.status === 409) {
          // User already exists — redirect to login preserving params
          const nextUrl = $page.url.searchParams.get('next');
          const joinWaitlist = $page.url.searchParams.get('join_waitlist');
          let loginUrl = '/login';
          if (nextUrl) {
            loginUrl += `?next=${nextUrl}`;
            if (joinWaitlist) loginUrl += `&join_waitlist=${joinWaitlist}`;
          }
          error = 'Você já tem uma conta. Faça login para continuar.';
          // Brief delay so user can read the error
          setTimeout(() => goto(loginUrl), 2000);
        } else {
          error = e.message === 'OTP_TOKEN_INVALID'
            ? 'Sessão de verificação expirada. Recomece o cadastro.'
            : e.message;
        }
      } else { error = 'Erro ao conectar'; }
    } finally { loading = false; }
  }

  function handleDigitInput(i: number, e: Event) {
    const val = (e.target as HTMLInputElement).value.replace(/\D/g, '');
    digits[i] = val.slice(-1);
    if (val && i < 5) inputRefs[i + 1]?.focus();
  }

  function handleDigitKeydown(i: number, e: KeyboardEvent) {
    if (e.key === 'Backspace' && !digits[i] && i > 0) inputRefs[i - 1]?.focus();
  }

  function handlePaste(e: ClipboardEvent) {
    const text = e.clipboardData?.getData('text').replace(/\D/g, '') ?? '';
    if (text.length >= 6) {
      digits = text.slice(0, 6).split('');
      inputRefs[5]?.focus();
      e.preventDefault();
    }
  }
</script>

<svelte:head><title>Cadastro gratuito — rachao.app</title></svelte:head>

<div class="min-h-screen flex items-center justify-center p-4 relative bg-primary-900"
  style="background-image: url('/background-login.png'); background-size: cover; background-position: center;">
  <div class="absolute inset-0 bg-primary-900/65"></div>

  <div class="relative z-10 bg-white dark:bg-gray-800 rounded-2xl shadow-2xl w-full max-w-sm md:max-w-3xl">
    <!-- Logo centered at column divider — desktop only -->
    <div class="hidden md:block absolute left-1/2 -translate-x-1/2 top-6 z-20">
      <img src="/logo.png" alt="rachao.app" width="320" height="174" class="w-52 drop-shadow-xl" />
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2">

      <!-- ── Coluna esquerda: branding + info ───────────────── -->
      <div class="bg-primary-900/95 rounded-t-2xl md:rounded-l-2xl md:rounded-tr-none p-5 md:p-8 flex flex-col justify-between gap-4 md:gap-6 md:pt-36">
        <div>
          {#if step === 'whatsapp' || step === 'form'}
            <!-- Plan info: mobile = logo pequeno ao lado; desktop = sem logo (absoluto) -->
            <div class="flex items-start gap-4">
              <!-- Logo mobile only -->
              <img src="/logo.png" alt="rachao.app" width="320" height="174" class="w-20 shrink-0 md:hidden" />

              <div class="flex-1 space-y-2 md:space-y-3">
                <div class="flex items-center justify-between">
                  <span class="text-xs font-semibold text-primary-300 uppercase tracking-wide">Plano selecionado</span>
                  <span class="text-sm font-bold text-primary-300">
                    {plan.price_monthly === null ? 'R$ 0/mês' : `${formatCents(plan.price_monthly)}/mês`}
                  </span>
                </div>
                <p class="text-base font-semibold text-white">{plan.name}</p>
                <ul class="space-y-1">
                  {#each plan.highlights as item}
                    <li class="text-xs text-primary-200/80 flex items-start gap-2">
                      <span class="text-primary-400 shrink-0 mt-0.5">✓</span>{item}
                    </li>
                  {/each}
                </ul>
              </div>
            </div>
          {:else}
            <!-- OTP step info -->
            <div class="flex items-start gap-4">
              <!-- Logo mobile only -->
              <img src="/logo.png" alt="rachao.app" width="320" height="174" class="w-20 shrink-0 md:hidden" />

              <div class="flex-1 flex flex-col gap-3">
                <div class="bg-green-500/20 border border-green-500/30 rounded-xl p-4">
                  <div class="flex items-center gap-3 mb-2">
                    <MessageCircle size={20} class="text-green-400 shrink-0" />
                    <span class="text-sm font-semibold text-white">Código enviado</span>
                  </div>
                  <p class="text-xs text-primary-200/80">
                    Enviamos um código SMS para<br />
                    <span class="font-semibold text-white">{maskedWhatsapp}</span>
                  </p>
                </div>
                <p class="text-xs text-primary-300/70">⏱ O código é válido por 10 minutos</p>
              </div>
            </div>
          {/if}
        </div>

        <!-- Step indicator -->
        <div class="hidden md:flex items-center gap-2">
          {#each [['whatsapp', '1'], ['otp', '2'], ['form', '3']] as [s, n]}
            <div class="flex items-center gap-2">
              <div class="w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold
                {step === s ? 'bg-primary-400 text-white' :
                 (step === 'otp' && s === 'whatsapp') || (step === 'form' && s !== 'form') ? 'bg-primary-600 text-primary-300' :
                 'bg-primary-800 text-primary-500'}">
                {n}
              </div>
              <span class="text-xs {step === s ? 'text-white font-medium' : 'text-primary-400'}">
                {s === 'whatsapp' ? 'WhatsApp' : s === 'otp' ? 'Verificação' : 'Cadastro'}
              </span>
            </div>
            {#if n !== '3'}<div class="flex-1 h-px bg-primary-700 mx-1"></div>{/if}
          {/each}
        </div>
      </div>

      <!-- ── Coluna direita: formulário ────────────────────── -->
      <div class="p-5 md:p-8 flex flex-col justify-center md:pt-36">
        {#if error}
          <div class="alert-error mb-4">{error}</div>
        {/if}

        <!-- Step 1: WhatsApp -->
        {#if step === 'whatsapp'}
          <div class="mb-6">
            <h2 class="text-xl font-bold text-gray-800 dark:text-gray-100">Crie sua conta</h2>
            <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">Digite seu número de celular</p>
          </div>

          <form onsubmit={(e) => { e.preventDefault(); handleSendOtp(); }} class="space-y-5">
            <div class="form-group">
              <label class="label" for="whatsapp">Celular *</label>
              <PhoneInput id="whatsapp" bind:value={whatsapp} placeholder="11999990000" required />
              <p class="text-xs text-gray-400 mt-1">Selecione o país e digite o número. Você receberá um código por SMS ou WhatsApp.</p>
            </div>

            <button type="submit" class="btn-primary w-full justify-center py-2.5" disabled={loading || !whatsapp}>
              <MessageCircle size={16} />
              {loading ? 'Enviando código…' : 'Enviar código'}
            </button>
          </form>

          <p class="text-xs text-gray-400 dark:text-gray-500 text-center mt-6">
            {#if $page.url.searchParams.get('next')}
              Já tem conta? <a href="/login?next={$page.url.searchParams.get('next')}{$page.url.searchParams.get('join_waitlist') ? '&join_waitlist=' + $page.url.searchParams.get('join_waitlist') : ''}" class="text-primary-600 hover:underline">Entrar</a>
            {:else}
              Já tem conta? <a href="/login" class="text-primary-600 hover:underline">Entrar</a>
            {/if}
          </p>

        <!-- Step 2: OTP -->
        {:else if step === 'otp'}
          <div class="mb-6">
            <h2 class="text-xl font-bold text-gray-800 dark:text-gray-100">Digite o código</h2>
            <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">Insira o código de 6 dígitos recebido</p>
          </div>

          <form onsubmit={(e) => { e.preventDefault(); handleVerifyOtp(); }} class="space-y-6">
            <div class="flex justify-center gap-2" onpaste={handlePaste}>
              {#each digits as digit, i}
                <input
                  bind:this={inputRefs[i]}
                  type="text"
                  inputmode="numeric"
                  maxlength="1"
                  value={digit}
                  oninput={(e) => handleDigitInput(i, e)}
                  onkeydown={(e) => handleDigitKeydown(i, e)}
                  class="w-11 h-12 text-center text-xl font-bold border-2 rounded-lg
                         bg-white dark:bg-gray-700 text-gray-800 dark:text-gray-100
                         border-gray-300 dark:border-gray-600
                         focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500/20
                         transition-colors"
                  autocomplete={i === 0 ? 'one-time-code' : 'off'}
                />
              {/each}
            </div>

            <button type="submit" class="btn-primary w-full justify-center py-2.5"
              disabled={loading || otpCode.length < 6}>
              <ShieldCheck size={16} />
              {loading ? 'Verificando…' : 'Verificar código'}
            </button>

            <div class="text-center space-y-3">
              {#if countdown > 0}
                <p class="text-xs text-gray-400">
                  Não recebeu? Reenviar em <span class="font-semibold tabular-nums">{countdown}s</span>
                </p>
              {:else}
                <button type="button" onclick={handleResend} disabled={loading}
                  class="text-xs text-primary-600 hover:underline flex items-center gap-1 mx-auto disabled:opacity-50">
                  <RotateCcw size={12} /> Reenviar código
                </button>
              {/if}
              <button type="button"
                onclick={() => { step = 'whatsapp'; error = ''; digits = ['', '', '', '', '', '']; }}
                class="text-xs text-gray-400 hover:text-gray-600 block w-full">
                ← Alterar número
              </button>
            </div>
          </form>

        <!-- Step 3: Form -->
        {:else}
          <div class="mb-5">
            <div class="flex items-center gap-2 mb-1">
              <ShieldCheck size={16} class="text-green-500" />
              <h2 class="text-xl font-bold text-gray-800 dark:text-gray-100">Complete seu cadastro</h2>
            </div>
            <p class="text-sm text-gray-500 dark:text-gray-400">Número verificado — só falta criar sua conta</p>
          </div>

          <form onsubmit={(e) => { e.preventDefault(); handleRegister(); }} class="space-y-4">
            <div class="form-group">
              <label class="label" for="name">Nome completo *</label>
              <input id="name" class="input" type="text" bind:value={name}
                placeholder="Ex: João Silva" required autocomplete="name" />
            </div>

            <div class="form-group">
              <label class="label" for="nickname">Apelido <span class="text-gray-400 font-normal">(opcional)</span></label>
              <input id="nickname" class="input" type="text" bind:value={nickname}
                placeholder="Como te chamam no rachão" autocomplete="off" />
            </div>

            <div class="grid grid-cols-2 gap-3">
              <div class="form-group">
                <label class="label" for="password">Senha *</label>
                <div class="relative">
                  <input id="password" class="input pr-9"
                    type={showPw ? 'text' : 'password'}
                    bind:value={password} placeholder="Mín. 6 chars" required
                    minlength="6" autocomplete="new-password" />
                  <button type="button" onclick={() => showPw = !showPw}
                    class="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600">
                    {#if showPw}<EyeOff size={15} />{:else}<Eye size={15} />{/if}
                  </button>
                </div>
              </div>
              <div class="form-group">
                <label class="label" for="confirm-password">Confirmar *</label>
                <input id="confirm-password" class="input"
                  type={showPw ? 'text' : 'password'}
                  bind:value={confirmPassword} placeholder="Repita" required
                  autocomplete="new-password" />
              </div>
            </div>

            <label class="flex items-start gap-2 cursor-pointer">
              <input type="checkbox" bind:checked={termsAccepted} class="mt-0.5 shrink-0 accent-primary-600" />
              <span class="text-xs text-gray-500 dark:text-gray-400">
                Li e aceito os
                <a href="/terms" target="_blank" class="text-primary-600 hover:underline">Termos de Uso</a>
                e a
                <a href="/privacy" target="_blank" class="text-primary-600 hover:underline">Política de Privacidade</a>
              </span>
            </label>

            <button type="submit" class="btn-primary w-full justify-center py-2.5"
              disabled={loading || !termsAccepted}>
              <UserPlus size={16} />
              {loading ? 'Criando conta…' : 'Criar conta grátis'}
            </button>
          </form>
        {/if}
      </div>
    </div>
  </div>
</div>
