<script lang="ts">
  import { auth as authApi, players as playersApi, push as pushApi, subscriptions as subsApi, ApiError } from '$lib/api';
  import { billingEnabled } from '$lib/billing';
  import type { SubscriptionInfo } from '$lib/api';
  import { authStore, currentPlayer, isAdmin } from '$lib/stores/auth';
  import { toastSuccess, toastError } from '$lib/stores/toast';
  import { goto } from '$app/navigation';
  import { Eye, EyeOff, KeyRound, Pencil, Bell, BellOff, BarChart2, User, CreditCard, ShieldCheck, ChevronDown } from 'lucide-svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import AvatarImage from '$lib/components/AvatarImage.svelte';
  import { t } from '$lib/i18n';

  // Plan
  let sub: SubscriptionInfo | null = $state(null);

  $effect(() => {
    if ($isAdmin) return;
    subsApi.me().then(data => { sub = data; }).catch(() => {});
  });

  function usageColor(used: number, limit: number | null): string {
    if (!limit) return 'bg-primary-500';
    const pct = used / limit;
    if (pct >= 1) return 'bg-red-500';
    if (pct >= 0.8) return 'bg-amber-400';
    return 'bg-primary-500';
  }

  function usageTextColor(used: number, limit: number | null): string {
    if (!limit) return 'text-primary-600 dark:text-primary-400';
    const pct = used / limit;
    if (pct >= 1) return 'text-red-600 dark:text-red-400';
    if (pct >= 0.8) return 'text-amber-600 dark:text-amber-400';
    return 'text-primary-600 dark:text-primary-400';
  }

  // Avatar
  let avatarUploading = $state(false);
  let avatarRemoving = $state(false);
  let avatarFileInput: HTMLInputElement | undefined = $state();

  async function onAvatarFileSelected(e: Event) {
    const file = (e.target as HTMLInputElement).files?.[0];
    if (!file) return;
    avatarUploading = true;
    try {
      const updated = await playersApi.uploadAvatar(file);
      authStore.updatePlayer(updated);
      toastSuccess($t('profile.avatar_upload_success'));
    } catch (err) {
      toastError(err instanceof ApiError ? err.message : $t('profile.avatar_upload_success'));
    } finally {
      avatarUploading = false;
      if (avatarFileInput) avatarFileInput.value = '';
    }
  }

  async function removeAvatar() {
    avatarRemoving = true;
    try {
      const updated = await playersApi.removeAvatar();
      authStore.updatePlayer(updated);
      toastSuccess($t('profile.avatar_remove_success'));
    } catch (err) {
      toastError(err instanceof ApiError ? err.message : 'Erro ao remover foto');
    } finally {
      avatarRemoving = false;
    }
  }

  // Nickname
  let nickname = $state($currentPlayer?.nickname ?? '');
  let editingNickname = $state(false);
  let savingNickname = $state(false);

  async function saveNickname() {
    if (!$currentPlayer) return;
    savingNickname = true;
    try {
      const updated = await playersApi.update($currentPlayer.id, { nickname: nickname.trim() || null });
      authStore.updatePlayer(updated);
      editingNickname = false;
      toastSuccess($t('profile.nickname_updated'));
    } catch (e) {
      toastError(e instanceof ApiError ? e.message : 'Erro ao salvar apelido');
    }
    savingNickname = false;
  }

  // Push notifications
  let pushSupported = $state(false);
  let pushPermission = $state<NotificationPermission>('default');
  let pushSubscribed = $state(false);
  let pushLoading = $state(false);

  $effect(() => {
    if (typeof window === 'undefined') return;
    pushSupported = 'serviceWorker' in navigator && 'PushManager' in window && 'Notification' in window;
    if (pushSupported) pushPermission = Notification.permission;

    (async () => {
      if (!pushSupported) return;
      try {
        const reg = await navigator.serviceWorker.ready;
        const sub = await reg.pushManager.getSubscription();
        pushSubscribed = !!sub;
      } catch { /* ignore */ }
    })();
  });

  async function enablePush() {
    pushLoading = true;
    try {
      const permission = await Notification.requestPermission();
      pushPermission = permission;
      if (permission !== 'granted') {
        toastError('Permissão de notificação negada.');
        return;
      }
      const { public_key } = await pushApi.getVapidPublicKey();
      if (!public_key) { toastError('Servidor não configurado para notificações.'); return; }

      const timeout = new Promise<never>((_, reject) =>
        setTimeout(() => reject(new Error('Timeout: serviço de notificações não respondeu.')), 15000)
      );
      console.log('[push] aguardando serviceWorker.ready...');
      const reg = await Promise.race([navigator.serviceWorker.ready, timeout]);
      console.log('[push] serviceWorker pronto. iniciando subscribe...');
      const sub = await Promise.race([
        reg.pushManager.subscribe({
          userVisibleOnly: true,
          applicationServerKey: urlBase64ToUint8Array(public_key),
        }),
        timeout,
      ]);
      console.log('[push] subscrito:', sub.endpoint);
      await pushApi.subscribe(sub.toJSON() as PushSubscriptionJSON, navigator.userAgent);
      pushSubscribed = true;
      toastSuccess($t('profile.push_enabled'));
    } catch (e) {
      console.error('[push] enablePush error:', e);
      toastError(e instanceof Error ? e.message : 'Erro ao ativar notificações');
    } finally {
      pushLoading = false;
    }
  }

  async function disablePush() {
    pushLoading = true;
    try {
      const reg = await navigator.serviceWorker.ready;
      const sub = await reg.pushManager.getSubscription();
      if (sub) await sub.unsubscribe();
      await pushApi.unsubscribe();
      pushSubscribed = false;
      toastSuccess($t('profile.push_disabled'));
    } catch (e) {
      toastError(e instanceof Error ? e.message : 'Erro ao desativar notificações');
    } finally {
      pushLoading = false;
    }
  }

  function urlBase64ToUint8Array(base64String: string): Uint8Array {
    const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
    const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
    const raw = atob(base64);
    return Uint8Array.from([...raw].map((c) => c.charCodeAt(0)));
  }

  // Password
  type PwMode = 'normal' | 'otp-pending' | 'otp-verified';
  let pwMode = $state<PwMode>('normal');
  let pwOpen = $state(false);

  $effect(() => {
    if ($currentPlayer?.must_change_password) pwOpen = true;
  });
  let currentPw = $state('');
  let newPw = $state('');
  let confirmPw = $state('');
  let showCurrent = $state(false);
  let showNew = $state(false);
  let saving = $state(false);

  // OTP for password reset
  let otpCode = $state('');
  let otpToken = $state('');
  let sendingOtp = $state(false);
  let verifyingOtp = $state(false);
  let otpError = $state('');
  let otpCountdown = $state(0);
  let otpCountdownTimer: ReturnType<typeof setInterval> | null = null;

  function startOtpCountdown() {
    otpCountdown = 60;
    if (otpCountdownTimer) clearInterval(otpCountdownTimer);
    otpCountdownTimer = setInterval(() => {
      otpCountdown--;
      if (otpCountdown <= 0 && otpCountdownTimer) clearInterval(otpCountdownTimer);
    }, 1000);
  }

  async function sendOtp() {
    sendingOtp = true;
    otpError = '';
    try {
      await authApi.sendOtpMe();
      pwMode = 'otp-pending';
      startOtpCountdown();
    } catch (e) {
      otpError = e instanceof ApiError ? e.message : 'Erro ao enviar código. Tente novamente.';
    }
    sendingOtp = false;
  }

  async function verifyOtp() {
    if (otpCode.length !== 6) return;
    verifyingOtp = true;
    otpError = '';
    try {
      const res = await authApi.verifyOtpMe(otpCode);
      otpToken = res.otp_token;
      pwMode = 'otp-verified';
    } catch {
      otpError = 'Código inválido ou expirado. Verifique e tente novamente.';
    }
    verifyingOtp = false;
  }

  function cancelOtp() {
    pwMode = 'normal';
    otpCode = '';
    otpToken = '';
    otpError = '';
    if (otpCountdownTimer) clearInterval(otpCountdownTimer);
  }

  let pwServerError = $state('');

  let validationError = $derived(
    newPw && confirmPw && newPw !== confirmPw ? $t('profile.pw_mismatch') :
    newPw && newPw.length < 6 ? $t('profile.pw_too_short') :
    null
  );

  $effect(() => {
    // Limpa erro de servidor quando usuário altera a nova senha
    newPw;
    pwServerError = '';
  });

  async function submit() {
    if (validationError || !newPw) return;
    if (pwMode === 'normal' && !currentPw) return;
    if (pwMode === 'otp-verified' && !otpToken) return;
    saving = true;
    pwServerError = '';
    try {
      if (pwMode === 'otp-verified') {
        await authApi.changePassword(newPw, { otp_token: otpToken });
      } else {
        await authApi.changePassword(newPw, { current_password: currentPw });
      }
      authStore.setMustChangePassword(false);
      toastSuccess($t('profile.pw_changed'));
      goto('/');
    } catch (e) {
      if (e instanceof ApiError && e.message === 'SAME_PASSWORD') {
        pwServerError = $t('auth.same_password_error');
      } else {
        toastError(e instanceof ApiError ? e.message : 'Erro ao alterar senha');
      }
    }
    saving = false;
  }
</script>

<svelte:head>
  <title>Minha Conta — rachao.app</title>
</svelte:head>

<PageBackground>
<main class="relative z-10 max-w-4xl mx-auto px-4 py-8">

  <!-- Cabeçalho padrão de página -->
  <div class="flex items-center justify-between mb-6">
    <div>
      <h1 class="text-2xl font-bold text-white flex items-center gap-2">
        <User size={24} class="text-primary-400" /> {$t('profile.title')}
      </h1>
      <p class="text-sm text-white/60 mt-0.5">{$t('profile.subtitle')}</p>
    </div>
  </div>

  {#if $currentPlayer?.must_change_password}
    <div class="bg-amber-50 border border-amber-300 text-amber-800 rounded-lg px-4 py-3 mb-6 text-sm font-medium">
      {$t('profile.must_change_pw')}
    </div>
  {/if}

  <!-- Layout em duas colunas no desktop -->
  <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 items-start">

    <!-- Coluna esquerda: stats, perfil, plano -->
    <div class="space-y-6">

      <!-- Atalho para estatísticas -->
      {#if !$isAdmin}
        <a href="/profile/stats" class="card card-body flex items-center gap-3 hover:bg-gray-50 dark:hover:bg-gray-700/60 transition-colors group">
          <div class="w-10 h-10 rounded-xl bg-primary-100 dark:bg-primary-900/30 flex items-center justify-center shrink-0">
            <BarChart2 size={20} class="text-primary-600 dark:text-primary-400" />
          </div>
          <div class="flex-1 min-w-0">
            <p class="text-sm font-semibold text-gray-900 dark:text-gray-100">{$t('profile.stats_link')}</p>
            <p class="text-xs text-gray-500 dark:text-gray-400">{$t('profile.stats_link_desc')}</p>
          </div>
          <span class="text-xs text-primary-600 dark:text-primary-400 font-medium shrink-0 group-hover:translate-x-0.5 transition-transform">→</span>
        </a>
      {/if}

      <!-- Avatar -->
      <div class="card card-body">
        <h2 class="font-semibold text-gray-800 dark:text-gray-200 mb-3">{$t('profile.avatar_section')}</h2>
        <div class="flex items-center gap-4">
          <AvatarImage
            name={$currentPlayer?.name ?? ''}
            avatarUrl={$currentPlayer?.avatar_url}
            updatedAt={$currentPlayer?.updated_at}
            size={64}
          />
          <div class="flex-1 min-w-0">
            <div class="flex flex-wrap gap-2 mb-2">
              <button
                type="button"
                onclick={() => avatarFileInput?.click()}
                disabled={avatarUploading}
                class="btn-secondary btn-sm"
              >
                {avatarUploading ? $t('profile.avatar_uploading') : $t('profile.avatar_upload')}
              </button>
              {#if $currentPlayer?.avatar_url}
                <button
                  type="button"
                  onclick={removeAvatar}
                  disabled={avatarRemoving}
                  class="btn-sm btn-ghost text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 border border-red-200 dark:border-red-800"
                >
                  {avatarRemoving ? $t('profile.avatar_removing') : $t('profile.avatar_remove')}
                </button>
              {/if}
            </div>
            <p class="text-[11px] text-gray-400 dark:text-gray-500 leading-snug">{$t('profile.avatar_format')}</p>
          </div>
        </div>
        <p class="text-[11px] text-gray-400 dark:text-gray-500 mt-3 leading-snug">{$t('profile.avatar_policy')}</p>
        <input
          bind:this={avatarFileInput}
          type="file"
          accept="image/jpeg,image/png,image/webp"
          class="hidden"
          onchange={onAvatarFileSelected}
        />
      </div>

      <!-- Dados do perfil (somente leitura) -->
      <div class="card card-body">
        <h2 class="font-semibold text-gray-800 dark:text-gray-200 mb-4">{$t('profile.section_profile')}</h2>
        <dl class="space-y-3 text-sm">
          <div class="flex justify-between">
            <dt class="text-gray-500 dark:text-gray-400">{$t('profile.name')}</dt>
            <dd class="font-medium text-gray-900 dark:text-gray-100">{$currentPlayer?.name ?? '—'}</dd>
          </div>
          <div class="flex justify-between items-center">
            <dt class="text-gray-500 dark:text-gray-400">{$t('profile.nickname')}</dt>
            {#if !editingNickname}
              <dd class="flex items-center gap-2">
                <span class="font-medium text-gray-900 dark:text-gray-100">{$currentPlayer?.nickname || '—'}</span>
                <button type="button" onclick={() => { nickname = $currentPlayer?.nickname ?? ''; editingNickname = true; }}
                  class="text-gray-400 hover:text-primary-600" title="Editar apelido">
                  <Pencil size={14} />
                </button>
              </dd>
            {/if}
          </div>
          {#if editingNickname}
            <form onsubmit={(e) => { e.preventDefault(); saveNickname(); }} class="flex gap-2 -mt-1">
              <input
                class="input text-sm flex-1 min-w-0"
                bind:value={nickname}
                placeholder={$t('profile.nickname_placeholder')}
                maxlength="50"
                disabled={savingNickname}
                autofocus />
              <button type="submit" class="btn-primary btn-sm shrink-0" disabled={savingNickname}>
                {savingNickname ? $t('profile.nickname_saving') : $t('profile.nickname_save')}
              </button>
              <button type="button" class="btn-secondary btn-sm shrink-0" onclick={() => { editingNickname = false; nickname = $currentPlayer?.nickname ?? ''; }}>
                {$t('profile.nickname_cancel')}
              </button>
            </form>
          {/if}
          <div class="flex justify-between">
            <dt class="text-gray-500 dark:text-gray-400">{$t('profile.whatsapp')}</dt>
            <dd class="font-mono text-gray-700 dark:text-gray-300">{$currentPlayer?.whatsapp ?? '—'}</dd>
          </div>
          <div class="flex justify-between">
            <dt class="text-gray-500 dark:text-gray-400">{$t('profile.role')}</dt>
            <dd>
              <span class="badge {$currentPlayer?.role === 'admin' ? 'badge-blue' : 'badge-gray'}">
                {$currentPlayer?.role === 'admin' ? $t('profile.role_admin') : $t('profile.role_player')}
              </span>
            </dd>
          </div>
        </dl>
      </div>

      <!-- Plano atual (apenas não-admins) -->
      {#if !$isAdmin && sub}
        <div class="card card-body">
          <h2 class="font-semibold text-gray-800 dark:text-gray-200 mb-4">{$t('profile.plan_section')}</h2>

          <div class="flex items-center gap-2 mb-4">
            <span class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-sm font-semibold bg-primary-100 dark:bg-primary-900/30 text-primary-700 dark:text-primary-300">
              <span class="w-2 h-2 rounded-full bg-primary-500"></span>
              {sub.plan.charAt(0).toUpperCase() + sub.plan.slice(1)}
            </span>
          </div>

          <div class="space-y-4">
            {#if sub.groups_limit !== null}
              {@const pct = Math.min(100, Math.round((sub.groups_used / sub.groups_limit) * 100))}
              <div>
                <div class="flex items-center justify-between mb-1.5">
                  <span class="text-sm text-gray-600 dark:text-gray-400">{$t('profile.groups_label')}</span>
                  <span class="text-sm font-medium {usageTextColor(sub.groups_used, sub.groups_limit)}">
                    {sub.groups_used} de {sub.groups_limit}
                  </span>
                </div>
                <div class="h-2 bg-gray-100 dark:bg-gray-700 rounded-full overflow-hidden">
                  <div
                    class="h-full rounded-full transition-all {usageColor(sub.groups_used, sub.groups_limit)}"
                    style="width: {pct}%"
                  ></div>
                </div>
              </div>
            {/if}

            {#if sub.members_limit !== null}
              <div class="flex items-center justify-between">
                <span class="text-sm text-gray-600 dark:text-gray-400">{$t('profile.members_label')}</span>
                <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{$t('profile.members_up_to').replace('{n}', String(sub.members_limit))}</span>
              </div>
            {/if}
          </div>

          {#if billingEnabled && sub.plan === 'free'}
            <div class="mt-4 pt-3 border-t border-gray-100 dark:border-gray-700 flex flex-col gap-2">
              <a href="/plans" class="btn-primary btn-sm w-full justify-center">
                <CreditCard size={14} /> {$t('profile.see_plans')}
              </a>
              <a href="/account/subscription" class="btn-secondary btn-sm w-full justify-center text-xs">
                {$t('profile.manage_subscription')}
              </a>
            </div>
          {:else if billingEnabled && sub.plan !== 'free'}
            <div class="mt-4 pt-3 border-t border-gray-100 dark:border-gray-700">
              <a href="/account/subscription" class="btn-secondary btn-sm w-full justify-center">
                <CreditCard size={14} /> {$t('profile.manage_subscription')}
              </a>
            </div>
          {/if}
        </div>
      {/if}

    </div><!-- /coluna esquerda -->

    <!-- Coluna direita: notificações, senha -->
    <div class="space-y-6">

      <!-- Notificações push -->
      {#if pushSupported}
        <div class="card card-body">
          <h2 class="font-semibold text-gray-800 dark:text-gray-200 mb-1 flex items-center gap-2">
            <Bell size={16} class="text-primary-600" /> {$t('profile.notifications_title')}
          </h2>
          <p class="text-xs text-gray-400 dark:text-gray-500 mb-4">
            {$t('profile.notifications_desc')}
          </p>
          {#if pushPermission === 'denied'}
            <p class="text-xs text-red-500 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg px-3 py-2">
              {$t('profile.notifications_blocked')}
            </p>
          {:else if pushSubscribed}
            <div class="flex items-center justify-between">
              <span class="text-sm text-green-600 dark:text-green-400 flex items-center gap-1.5">
                <Bell size={14} /> {$t('profile.notifications_active')}
              </span>
              <button class="btn-sm btn-secondary flex items-center gap-1" onclick={disablePush} disabled={pushLoading}>
                <BellOff size={14} /> {pushLoading ? $t('profile.notifications_disable_loading') : $t('profile.notifications_disable')}
              </button>
            </div>
          {:else}
            <button class="btn-primary w-full justify-center" onclick={enablePush} disabled={pushLoading}>
              <Bell size={15} /> {pushLoading ? $t('profile.notifications_enable_loading') : $t('profile.notifications_enable')}
            </button>
          {/if}
        </div>
      {/if}

      <!-- Troca de senha -->
      <div class="card">
        <button
          type="button"
          class="w-full card-body flex items-center justify-between gap-2 text-left"
          onclick={() => { if (pwOpen) { pwMode = 'normal'; otpCode = ''; otpToken = ''; otpError = ''; if (otpCountdownTimer) clearInterval(otpCountdownTimer); } pwOpen = !pwOpen; }}>
          <span class="font-semibold text-gray-800 dark:text-gray-200 flex items-center gap-2">
            <KeyRound size={16} class="text-primary-600" /> {$t('profile.change_password')}
          </span>
          <ChevronDown size={16} class="text-gray-400 shrink-0 transition-transform duration-200 {pwOpen ? 'rotate-180' : ''}" />
        </button>

        {#if pwOpen}
        <div class="px-4 pb-4 sm:px-6 sm:pb-6 space-y-0 border-t border-gray-100 dark:border-gray-700 pt-4">

        <!-- Modo normal: senha atual conhecida -->
        {#if pwMode === 'normal'}
          <form onsubmit={(e) => { e.preventDefault(); submit(); }} class="space-y-4">
            <div class="form-group">
              <label class="label" for="current-pw">{$t('profile.current_password')}</label>
              <div class="relative">
                <input id="current-pw" class="input pr-10"
                  type={showCurrent ? 'text' : 'password'}
                  bind:value={currentPw} placeholder="••••••" required autocomplete="current-password"
                  disabled={saving} />
                <button type="button" onclick={() => showCurrent = !showCurrent}
                  class="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600">
                  {#if showCurrent}<EyeOff size={15} />{:else}<Eye size={15} />{/if}
                </button>
              </div>
              <button type="button" onclick={sendOtp} disabled={sendingOtp}
                class="mt-1.5 text-xs text-primary-600 dark:text-primary-400 hover:underline disabled:opacity-50">
                {sendingOtp ? $t('profile.otp_sending') : $t('profile.forgot_password')}
              </button>
              {#if otpError}<p class="text-xs text-red-500 mt-1">{otpError}</p>{/if}
            </div>

            <div class="form-group">
              <label class="label" for="new-pw">{$t('profile.new_password')}</label>
              <div class="relative">
                <input id="new-pw" class="input pr-10"
                  type={showNew ? 'text' : 'password'}
                  bind:value={newPw} placeholder={$t('profile.new_password_placeholder')} required minlength="6" autocomplete="new-password"
                  disabled={saving} />
                <button type="button" onclick={() => showNew = !showNew}
                  class="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600">
                  {#if showNew}<EyeOff size={15} />{:else}<Eye size={15} />{/if}
                </button>
              </div>
            </div>

            <div class="form-group">
              <label class="label" for="confirm-pw">{$t('profile.confirm_password')}</label>
              <input id="confirm-pw" class="input"
                type="password" bind:value={confirmPw}
                placeholder={$t('profile.confirm_password_placeholder')} required autocomplete="new-password"
                disabled={saving} />
            </div>

            {#if validationError}
              <div class="alert-error text-sm">{validationError}</div>
            {/if}
            {#if pwServerError}
              <div class="alert-error text-sm">{pwServerError}</div>
            {/if}

            <button type="submit" class="btn-primary w-full justify-center py-2.5"
              disabled={saving || !!validationError || !currentPw || !newPw || !confirmPw}>
              {saving ? $t('profile.saving') : $t('profile.change_pw_btn')}
            </button>
          </form>

        <!-- Modo OTP: aguardando código SMS -->
        {:else if pwMode === 'otp-pending'}
          <div class="space-y-4">
            <div class="bg-primary-50 dark:bg-primary-900/20 border border-primary-200 dark:border-primary-800 rounded-lg px-4 py-3 text-sm text-primary-800 dark:text-primary-300">
              {$t('profile.otp_sent_to')}
              <strong class="block mt-0.5">
                {'•'.repeat(($currentPlayer?.whatsapp?.length ?? 4) - 4)}{$currentPlayer?.whatsapp?.slice(-4)}
              </strong>
            </div>

            <div class="form-group">
              <label class="label" for="otp-code">{$t('profile.otp_label')}</label>
              <input id="otp-code" class="input text-center text-xl tracking-widest font-mono"
                type="text" inputmode="numeric" pattern="[0-9]*" maxlength="6"
                bind:value={otpCode} placeholder="000000"
                disabled={verifyingOtp} />
            </div>

            {#if otpError}<div class="alert-error text-sm">{otpError}</div>{/if}

            <button onclick={verifyOtp} disabled={verifyingOtp || otpCode.length !== 6}
              class="btn-primary w-full justify-center py-2.5">
              {verifyingOtp ? $t('profile.otp_verify_loading') : $t('profile.otp_verify')}
            </button>

            <div class="flex items-center justify-between text-xs text-gray-400">
              <button onclick={cancelOtp} class="hover:text-gray-600 dark:hover:text-gray-200">{$t('profile.otp_back')}</button>
              {#if otpCountdown > 0}
                <span>{$t('profile.otp_resend_countdown').replace('{s}', String(otpCountdown))}</span>
              {:else}
                <button onclick={sendOtp} disabled={sendingOtp} class="text-primary-600 dark:text-primary-400 hover:underline disabled:opacity-50">
                  {sendingOtp ? $t('profile.otp_resend_loading') : $t('profile.otp_resend')}
                </button>
              {/if}
            </div>
          </div>

        <!-- Modo OTP verificado: definir nova senha -->
        {:else if pwMode === 'otp-verified'}
          <form onsubmit={(e) => { e.preventDefault(); submit(); }} class="space-y-4">
            <div class="flex items-center gap-2 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg px-4 py-2.5 text-sm text-green-800 dark:text-green-300">
              <ShieldCheck size={16} class="shrink-0" />
              {$t('profile.identity_verified')}
            </div>

            <div class="form-group">
              <label class="label" for="new-pw-otp">{$t('profile.new_password')}</label>
              <div class="relative">
                <input id="new-pw-otp" class="input pr-10"
                  type={showNew ? 'text' : 'password'}
                  bind:value={newPw} placeholder={$t('profile.new_password_placeholder')} required minlength="6" autocomplete="new-password"
                  disabled={saving} />
                <button type="button" onclick={() => showNew = !showNew}
                  class="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600">
                  {#if showNew}<EyeOff size={15} />{:else}<Eye size={15} />{/if}
                </button>
              </div>
            </div>

            <div class="form-group">
              <label class="label" for="confirm-pw-otp">{$t('profile.confirm_password')}</label>
              <input id="confirm-pw-otp" class="input"
                type="password" bind:value={confirmPw}
                placeholder={$t('profile.confirm_password_placeholder')} required autocomplete="new-password"
                disabled={saving} />
            </div>

            {#if validationError}
              <div class="alert-error text-sm">{validationError}</div>
            {/if}
            {#if pwServerError}
              <div class="alert-error text-sm">{pwServerError}</div>
            {/if}

            <button type="submit" class="btn-primary w-full justify-center py-2.5"
              disabled={saving || !!validationError || !newPw || !confirmPw}>
              {saving ? $t('profile.saving') : $t('profile.set_new_password')}
            </button>

            <button type="button" onclick={cancelOtp} class="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 w-full text-center">
              {$t('profile.otp_cancel')}
            </button>
          </form>
        {/if}

        </div><!-- /pwOpen content -->
        {/if}
      </div>

    </div><!-- /coluna direita -->

  </div><!-- /grid -->
</main>
</PageBackground>
