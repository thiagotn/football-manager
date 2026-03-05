<script lang="ts">
  import { auth as authApi, players as playersApi, ApiError } from '$lib/api';
  import { authStore, currentPlayer } from '$lib/stores/auth';
  import { toastSuccess, toastError } from '$lib/stores/toast';
  import { goto } from '$app/navigation';
  import { Eye, EyeOff, KeyRound, Pencil } from 'lucide-svelte';

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

<main class="max-w-lg mx-auto px-4 py-8">
  <div class="mb-6">
    <h1 class="text-2xl font-bold text-gray-900 dark:text-gray-100">Minha Conta</h1>
    <p class="text-sm text-gray-500 dark:text-gray-400 mt-0.5">Informações do seu perfil e segurança</p>
  </div>

  {#if $currentPlayer?.must_change_password}
    <div class="bg-amber-50 border border-amber-300 text-amber-800 rounded-lg px-4 py-3 mb-6 text-sm font-medium">
      ⚠️ Sua senha foi redefinida pelo administrador. Por favor, defina uma nova senha abaixo antes de continuar.
    </div>
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
