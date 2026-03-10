<script lang="ts">
  import { auth, ApiError } from '$lib/api';
  import { authStore } from '$lib/stores/auth';
  import { goto } from '$app/navigation';
  import { toastError } from '$lib/stores/toast';
  import { Eye, EyeOff, LogIn } from 'lucide-svelte';

  let whatsapp = $state('');
  let password = $state('');
  let loading = $state(false);
  let showPw = $state(false);
  let error = $state('');

  async function handleLogin() {
    error = '';
    loading = true;
    try {
      const res = await auth.login(whatsapp, password);
      authStore.login(res.access_token, res);
      goto(res.must_change_password ? '/profile' : '/');
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Erro ao conectar';
    } finally {
      loading = false;
    }
  }
</script>

<svelte:head><title>Login — rachao.app</title></svelte:head>

<div class="min-h-screen flex items-center justify-center p-4 relative bg-primary-900"
  style="background-image: url('/background-login.png'); background-size: cover; background-position: center;">
  <div class="absolute inset-0 bg-primary-900/65"></div>
  <div class="relative z-10 bg-white dark:bg-gray-800 rounded-2xl shadow-2xl w-full max-w-sm p-8">
    <div class="text-center mb-8">
      <img src="/logo.png" alt="rachao.app" width="320" height="174" class="w-56 block mx-auto mb-1" />
      <div class="flex items-center justify-center gap-2 mb-1">
        <span class="text-xs font-bold bg-yellow-400 text-yellow-900 px-1.5 py-0.5 rounded-full">Beta</span>
      </div>
      <p class="text-sm text-gray-500 dark:text-gray-400">Gestão de grupos e partidas</p>
    </div>

    {#if error}
      <div class="alert-error mb-4">{error}</div>
    {/if}

    <form onsubmit={(e) => { e.preventDefault(); handleLogin(); }} class="space-y-4">
      <div class="form-group">
        <label class="label" for="whatsapp">WhatsApp</label>
        <input id="whatsapp" class="input" type="tel" bind:value={whatsapp}
          placeholder="11999990000" required autocomplete="username" />
      </div>

      <div class="form-group">
        <label class="label" for="password">Senha</label>
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
      </div>

      <button type="submit" class="btn-primary w-full justify-center py-2.5" disabled={loading}>
        <LogIn size={16} />
        {loading ? 'Entrando…' : 'Entrar'}
      </button>
    </form>

    <p class="text-xs text-gray-400 dark:text-gray-500 text-center mt-6">
      Não tem conta? <a href="/register" class="text-primary-600 hover:underline">Cadastre-se grátis</a>
    </p>
    <div class="flex justify-center gap-4 mt-4 text-xs text-gray-300 dark:text-gray-600">
      <a href="/terms" class="hover:text-gray-500 transition-colors">Termos de Uso</a>
      <a href="/privacy" class="hover:text-gray-500 transition-colors">Privacidade</a>
    </div>
  </div>
</div>

