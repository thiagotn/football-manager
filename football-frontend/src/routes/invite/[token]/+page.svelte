<script lang="ts">
  import { page } from '$app/stores';
  import { invites, ApiError } from '$lib/api';
  import { authStore } from '$lib/stores/auth';
  import { goto } from '$app/navigation';
  import { UserPlus, CheckCircle, Eye, EyeOff } from 'lucide-svelte';

  const token = $page.params.token;

  let info: { group_name: string; expires_at: string } | null = $state(null);
  let expired = $state(false);
  let loading = $state(true);

  let form = $state({ name: '', nickname: '', whatsapp: '', password: '' });
  let showPw = $state(false);
  let submitting = $state(false);
  let done = $state(false);
  let error = $state('');

  $effect(() => {
    let cancelled = false;
    (async () => {
      try {
        const i = await invites.getInfo(token);
        if (!cancelled) info = i;
      } catch (e) {
        if (!cancelled) expired = true;
      }
      if (!cancelled) loading = false;
    })();
    return () => { cancelled = true; };
  });

  async function accept() {
    error = '';
    submitting = true;
    try {
      const res = await invites.accept(token, {
        name: form.name,
        nickname: form.nickname || undefined,
        whatsapp: form.whatsapp,
        password: form.password,
      });
      authStore.login(res.access_token, res);
      done = true;
      setTimeout(() => goto('/'), 2000);
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Erro ao aceitar convite';
    }
    submitting = false;
  }

  function fmtExpiry(s: string) {
    return new Date(s).toLocaleString('pt-BR', { hour: '2-digit', minute: '2-digit' });
  }
</script>

<svelte:head><title>Convite — rachao.app</title></svelte:head>

<div class="min-h-screen bg-gradient-to-br from-primary-700 to-primary-900 flex items-center justify-center p-4">
  <div class="bg-white rounded-2xl shadow-2xl w-full max-w-sm p-8">
    <div class="text-center mb-6">
      <div class="text-4xl mb-2">⚽</div>
      <h1 class="text-xl font-bold text-gray-900">Convite para Grupo</h1>
    </div>

    {#if loading}
      <div class="animate-pulse space-y-3">
        <div class="h-4 bg-gray-200 rounded"></div>
        <div class="h-4 bg-gray-200 rounded w-2/3"></div>
      </div>

    {:else if expired}
      <div class="alert-error text-center">
        <p class="font-semibold">Convite inválido ou expirado</p>
        <p class="mt-1 text-xs">Solicite um novo convite ao administrador.</p>
      </div>

    {:else if done}
      <div class="text-center py-4">
        <CheckCircle size={48} class="text-green-500 mx-auto mb-3" />
        <p class="font-semibold text-gray-900">Você entrou no grupo!</p>
        <p class="text-sm text-gray-500 mt-1">Redirecionando…</p>
      </div>

    {:else if info}
      <div class="alert-info mb-5 text-center">
        <p class="font-semibold">Você foi convidado para:</p>
        <p class="text-lg font-bold text-blue-800 mt-0.5">{info.group_name}</p>
        <p class="text-xs mt-1 text-blue-600">Expira às {fmtExpiry(info.expires_at)}</p>
      </div>

      {#if error}
        <div class="alert-error mb-4">{error}</div>
      {/if}

      <form onsubmit={(e) => { e.preventDefault(); accept(); }} class="space-y-3">
        <div class="form-group">
          <label class="label" for="name">Nome completo *</label>
          <input id="name" class="input" bind:value={form.name} placeholder="Seu nome" required minlength="2" />
        </div>
        <div class="form-group">
          <label class="label" for="nick">Apelido</label>
          <input id="nick" class="input" bind:value={form.nickname} placeholder="Como te chamam no campo?" />
        </div>
        <div class="form-group">
          <label class="label" for="wa">WhatsApp *</label>
          <input id="wa" class="input" type="tel" bind:value={form.whatsapp} placeholder="11999990000" required />
        </div>
        <div class="form-group">
          <label class="label" for="pw">Criar senha *</label>
          <div class="relative">
            <input id="pw" class="input pr-10" type={showPw ? 'text' : 'password'} bind:value={form.password}
              placeholder="Mínimo 6 caracteres" required minlength="6" />
            <button type="button" onclick={() => showPw = !showPw}
              class="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600">
              {#if showPw}<EyeOff size={16} />{:else}<Eye size={16} />{/if}
            </button>
          </div>
        </div>

        <button type="submit" class="btn-primary w-full justify-center py-2.5 mt-2" disabled={submitting}>
          <UserPlus size={16} />
          {submitting ? 'Entrando…' : 'Entrar no Grupo'}
        </button>
      </form>
    {/if}
  </div>
</div>
