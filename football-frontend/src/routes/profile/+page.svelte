<script lang="ts">
  import { auth as authApi, players as playersApi, push as pushApi, subscriptions as subsApi, ApiError } from '$lib/api';
  import type { SubscriptionInfo } from '$lib/api';
  import { authStore, currentPlayer, isAdmin } from '$lib/stores/auth';
  import { toastSuccess, toastError } from '$lib/stores/toast';
  import { goto } from '$app/navigation';
  import { Eye, EyeOff, KeyRound, Pencil, Bell, BellOff, BarChart2 } from 'lucide-svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';

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
      toastSuccess('Apelido atualizado!');
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
      toastSuccess('Notificações ativadas!');
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
      toastSuccess('Notificações desativadas.');
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
  let currentPw = $state('');
  let newPw = $state('');
  let confirmPw = $state('');
  let showCurrent = $state(false);
  let showNew = $state(false);
  let saving = $state(false);

  let validationError = $derived(
    newPw && confirmPw && newPw !== confirmPw ? 'As senhas não coincidem.' :
    newPw && newPw.length < 6 ? 'A nova senha deve ter ao menos 6 caracteres.' :
    null
  );

  async function submit() {
    if (validationError || !currentPw || !newPw) return;
    saving = true;
    try {
      await authApi.changePassword(currentPw, newPw);
      authStore.setMustChangePassword(false);
      toastSuccess('Senha alterada com sucesso!');
      goto('/');
    } catch (e) {
      toastError(e instanceof ApiError ? e.message : 'Erro ao alterar senha');
    }
    saving = false;
  }
</script>

<svelte:head>
  <title>Minha Conta — rachao.app</title>
</svelte:head>

<PageBackground>
<main class="relative z-10 max-w-lg mx-auto px-4 py-8">
  <div class="mb-6">
    <h1 class="text-2xl font-bold text-white">Minha Conta</h1>
    <p class="text-sm text-gray-300 mt-0.5">Informações do seu perfil e segurança</p>
  </div>

  {#if $currentPlayer?.must_change_password}
    <div class="bg-amber-50 border border-amber-300 text-amber-800 rounded-lg px-4 py-3 mb-6 text-sm font-medium">
      ⚠️ Sua senha foi redefinida pelo administrador. Por favor, defina uma nova senha abaixo antes de continuar.
    </div>
  {/if}

  <!-- Atalho para estatísticas -->
  {#if !$isAdmin}
    <a href="/profile/stats" class="card card-body mb-4 flex items-center gap-3 hover:bg-gray-50 dark:hover:bg-gray-700/60 transition-colors group">
      <div class="w-10 h-10 rounded-xl bg-primary-100 dark:bg-primary-900/30 flex items-center justify-center shrink-0">
        <BarChart2 size={20} class="text-primary-600 dark:text-primary-400" />
      </div>
      <div class="flex-1 min-w-0">
        <p class="text-sm font-semibold text-gray-900 dark:text-gray-100">Minhas Estatísticas</p>
        <p class="text-xs text-gray-500 dark:text-gray-400">Partidas, presença, reputação e mais</p>
      </div>
      <span class="text-xs text-primary-600 dark:text-primary-400 font-medium shrink-0 group-hover:translate-x-0.5 transition-transform">→</span>
    </a>
  {/if}

  <!-- Dados do perfil (somente leitura) -->
  <div class="card card-body mb-6">
    <h2 class="font-semibold text-gray-800 dark:text-gray-200 mb-4">Perfil</h2>
    <dl class="space-y-3 text-sm">
      <div class="flex justify-between">
        <dt class="text-gray-500 dark:text-gray-400">Nome</dt>
        <dd class="font-medium text-gray-900 dark:text-gray-100">{$currentPlayer?.name ?? '—'}</dd>
      </div>
      <div class="flex justify-between items-center">
        <dt class="text-gray-500 dark:text-gray-400">Apelido</dt>
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
            placeholder="Como te chamam?"
            maxlength="50"
            disabled={savingNickname}
            autofocus />
          <button type="submit" class="btn-primary btn-sm shrink-0" disabled={savingNickname}>
            {savingNickname ? 'Salvando…' : 'Salvar'}
          </button>
          <button type="button" class="btn-secondary btn-sm shrink-0" onclick={() => { editingNickname = false; nickname = $currentPlayer?.nickname ?? ''; }}>
            Cancelar
          </button>
        </form>
      {/if}
      <div class="flex justify-between">
        <dt class="text-gray-500 dark:text-gray-400">WhatsApp</dt>
        <dd class="font-mono text-gray-700 dark:text-gray-300">{$currentPlayer?.whatsapp ?? '—'}</dd>
      </div>
      <div class="flex justify-between">
        <dt class="text-gray-500 dark:text-gray-400">Perfil</dt>
        <dd>
          <span class="badge {$currentPlayer?.role === 'admin' ? 'badge-blue' : 'badge-gray'}">
            {$currentPlayer?.role === 'admin' ? 'Admin' : 'Jogador'}
          </span>
        </dd>
      </div>
    </dl>
  </div>

  <!-- Plano atual (apenas não-admins) -->
  {#if !$isAdmin && sub}
    <div class="card card-body mb-6">
      <h2 class="font-semibold text-gray-800 dark:text-gray-200 mb-4">Seu Plano</h2>

      <!-- Badge do plano -->
      <div class="flex items-center gap-2 mb-4">
        <span class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-sm font-semibold bg-primary-100 dark:bg-primary-900/30 text-primary-700 dark:text-primary-300">
          <span class="w-2 h-2 rounded-full bg-primary-500"></span>
          {sub.plan.charAt(0).toUpperCase() + sub.plan.slice(1)}
        </span>
      </div>

      <div class="space-y-4">
        <!-- Grupos -->
        {#if sub.groups_limit !== null}
          {@const pct = Math.min(100, Math.round((sub.groups_used / sub.groups_limit) * 100))}
          <div>
            <div class="flex items-center justify-between mb-1.5">
              <span class="text-sm text-gray-600 dark:text-gray-400">Grupos</span>
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

        <!-- Membros por grupo -->
        {#if sub.members_limit !== null}
          <div class="flex items-center justify-between">
            <span class="text-sm text-gray-600 dark:text-gray-400">Membros por grupo</span>
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">até {sub.members_limit}</span>
          </div>
        {/if}
      </div>
    </div>
  {/if}

  <!-- Notificações push -->
  {#if pushSupported}
    <div class="card card-body mb-6">
      <h2 class="font-semibold text-gray-800 dark:text-gray-200 mb-1 flex items-center gap-2">
        <Bell size={16} class="text-primary-600" /> Notificações
      </h2>
      <p class="text-xs text-gray-400 dark:text-gray-500 mb-4">
        Receba avisos de novas partidas, convites e lembretes diretamente no seu dispositivo.
      </p>
      {#if pushPermission === 'denied'}
        <p class="text-xs text-red-500 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg px-3 py-2">
          Notificações bloqueadas no navegador. Altere nas configurações do seu dispositivo para habilitá-las.
        </p>
      {:else if pushSubscribed}
        <div class="flex items-center justify-between">
          <span class="text-sm text-green-600 dark:text-green-400 flex items-center gap-1.5">
            <Bell size={14} /> Ativas
          </span>
          <button class="btn-sm btn-secondary flex items-center gap-1" onclick={disablePush} disabled={pushLoading}>
            <BellOff size={14} /> {pushLoading ? 'Aguarde…' : 'Desativar'}
          </button>
        </div>
      {:else}
        <button class="btn-primary w-full justify-center" onclick={enablePush} disabled={pushLoading}>
          <Bell size={15} /> {pushLoading ? 'Ativando…' : 'Ativar notificações'}
        </button>
      {/if}
    </div>
  {/if}

  <!-- Troca de senha -->
  <div class="card card-body">
    <h2 class="font-semibold text-gray-800 mb-4 flex items-center gap-2">
      <KeyRound size={16} class="text-primary-600" /> Alterar Senha
    </h2>

    <form onsubmit={(e) => { e.preventDefault(); submit(); }} class="space-y-4">
      <div class="form-group">
        <label class="label" for="current-pw">Senha atual</label>
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
      </div>

      <div class="form-group">
        <label class="label" for="new-pw">Nova senha</label>
        <div class="relative">
          <input id="new-pw" class="input pr-10"
            type={showNew ? 'text' : 'password'}
            bind:value={newPw} placeholder="Mínimo 6 caracteres" required minlength="6" autocomplete="new-password"
            disabled={saving} />
          <button type="button" onclick={() => showNew = !showNew}
            class="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600">
            {#if showNew}<EyeOff size={15} />{:else}<Eye size={15} />{/if}
          </button>
        </div>
      </div>

      <div class="form-group">
        <label class="label" for="confirm-pw">Confirmar nova senha</label>
        <input id="confirm-pw" class="input"
          type="password" bind:value={confirmPw}
          placeholder="Repita a nova senha" required autocomplete="new-password"
          disabled={saving} />
      </div>

      {#if validationError}
        <div class="alert-error text-sm">{validationError}</div>
      {/if}

      <button type="submit" class="btn-primary w-full justify-center py-2.5"
        disabled={saving || !!validationError || !currentPw || !newPw || !confirmPw}>
        {saving ? 'Salvando…' : 'Alterar senha'}
      </button>
    </form>
  </div>
</main>
</PageBackground>
