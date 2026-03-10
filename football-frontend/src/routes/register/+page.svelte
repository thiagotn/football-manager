<script lang="ts">
  import { auth, ApiError } from '$lib/api';
  import { authStore } from '$lib/stores/auth';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { toastError } from '$lib/stores/toast';
  import { Eye, EyeOff, UserPlus } from 'lucide-svelte';
  import { getPlan } from '$lib/plans';

  const planKey = $derived($page.url.searchParams.get('plan') ?? 'free');
  const plan = $derived(getPlan(planKey));

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
    <div class="text-center mb-6">
      <img src="/logo.png" alt="rachao.app" width="320" height="174" class="w-56 block mx-auto mb-4" />

      <!-- Banner do plano selecionado -->
      <div class="bg-primary-50 dark:bg-primary-900/20 border border-primary-200 dark:border-primary-800 rounded-xl px-4 py-3 text-left mb-2">
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs font-semibold text-primary-700 dark:text-primary-300 uppercase tracking-wide">Plano selecionado</span>
          <span class="text-sm font-bold text-primary-700 dark:text-primary-300">
            {plan.price_monthly === null ? 'Grátis' : `R$ ${plan.price_monthly.toFixed(2).replace('.', ',')}/mês`}
          </span>
        </div>
        <p class="text-sm font-semibold text-gray-800 dark:text-gray-200 mb-1.5">{plan.name}</p>
        <ul class="space-y-0.5 mb-2">
          {#each plan.highlights.filter(h => !h.toLowerCase().includes('votação')) as item}
            <li class="text-xs text-gray-600 dark:text-gray-400 flex items-start gap-1.5">
              <span class="text-primary-500 shrink-0">✓</span>{item}
            </li>
          {/each}
        </ul>
        <!-- Voting highlight -->
        <div class="bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg px-2.5 py-2 flex items-start gap-2">
          <span class="text-base leading-none shrink-0">🏆</span>
          <div>
            <p class="text-xs font-semibold text-amber-800 dark:text-amber-300">Votação pós-partida inclusa</p>
            <p class="text-xs text-amber-700 dark:text-amber-400 mt-0.5">Top 5 da pelada + Decepção do jogo — abre automaticamente após cada partida.</p>
          </div>
        </div>
      </div>
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
