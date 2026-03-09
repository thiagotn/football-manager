<script lang="ts">
  import { auth, ApiError } from '$lib/api';
  import { authStore } from '$lib/stores/auth';
  import { goto } from '$app/navigation';
  import { toastError } from '$lib/stores/toast';
  import { Eye, EyeOff, UserPlus } from 'lucide-svelte';

  let name = $state('');
  let nickname = $state('');
  let whatsapp = $state('');
  let password = $state('');
  let confirmPassword = $state('');
  let loading = $state(false);
  let showPw = $state(false);
  let error = $state('');

  async function handleRegister() {
    error = '';
    if (password !== confirmPassword) {
      error = 'As senhas não coincidem';
      return;
    }
    loading = true;
    try {
      const res = await auth.register({ name, whatsapp, password, nickname: nickname || undefined });
      authStore.login(res.access_token, res);
      goto('/');
    } catch (e) {
      if (e instanceof ApiError) {
        error = e.status === 409 ? 'Este WhatsApp já está cadastrado.' : e.message;
      } else {
        error = 'Erro ao conectar';
      }
    } finally {
      loading = false;
    }
  }
</script>

<svelte:head><title>Cadastro gratuito — rachao.app</title></svelte:head>

<div class="min-h-screen flex items-center justify-center p-4 relative bg-primary-900"
  style="background-image: url('/background-login.png'); background-size: cover; background-position: center;">
  <div class="absolute inset-0 bg-primary-900/65"></div>
  <div class="relative z-10 bg-white dark:bg-gray-800 rounded-2xl shadow-2xl w-full max-w-sm p-8">
    <div class="text-center mb-8">
      <img src="/logo.png" alt="rachao.app" width="320" height="174" class="w-56 block mx-auto mb-1" />
      <div class="flex items-center justify-center gap-2 mb-1">
        <span class="text-xs font-bold bg-green-100 text-green-700 px-1.5 py-0.5 rounded-full">Grátis</span>
      </div>
      <p class="text-sm text-gray-500 dark:text-gray-400">Crie sua conta e organize seus rachões</p>
    </div>

    {#if error}
      <div class="alert-error mb-4">{error}</div>
    {/if}

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

      <div class="form-group">
        <label class="label" for="whatsapp">WhatsApp *</label>
        <input id="whatsapp" class="input" type="tel" bind:value={whatsapp}
          placeholder="11999990000" required autocomplete="tel" />
        <p class="text-xs text-gray-400 mt-1">Somente números, com DDD</p>
      </div>

      <div class="form-group">
        <label class="label" for="password">Senha *</label>
        <div class="relative">
          <input id="password" class="input pr-10"
            type={showPw ? 'text' : 'password'}
            bind:value={password} placeholder="Mínimo 6 caracteres" required
            minlength="6" autocomplete="new-password" />
          <button type="button" onclick={() => showPw = !showPw}
            class="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600">
            {#if showPw}<EyeOff size={16} />{:else}<Eye size={16} />{/if}
          </button>
        </div>
      </div>

      <div class="form-group">
        <label class="label" for="confirm-password">Confirmar senha *</label>
        <input id="confirm-password" class="input"
          type={showPw ? 'text' : 'password'}
          bind:value={confirmPassword} placeholder="Repita a senha" required
          autocomplete="new-password" />
      </div>

      <button type="submit" class="btn-primary w-full justify-center py-2.5" disabled={loading}>
        <UserPlus size={16} />
        {loading ? 'Criando conta…' : 'Criar conta grátis'}
      </button>
    </form>

    <p class="text-xs text-gray-400 dark:text-gray-500 text-center mt-6">
      Já tem conta? <a href="/login" class="text-primary-600 hover:underline">Entrar</a>
    </p>
  </div>
</div>
