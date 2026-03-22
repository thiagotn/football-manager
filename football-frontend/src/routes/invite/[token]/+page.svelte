<script lang="ts">
  import { page } from '$app/stores';
  import { invites, ApiError } from '$lib/api';
  import { authStore } from '$lib/stores/auth';
  import { goto } from '$app/navigation';
  import { UserPlus, CheckCircle, Eye, EyeOff, ArrowLeft } from 'lucide-svelte';
  import PhoneInput from '$lib/components/PhoneInput.svelte';
  import { t } from '$lib/i18n';

  const token = $page.params.token;

  type Step = 'whatsapp' | 'login' | 'register';

  let info: { group_name: string; expires_at: string } | null = $state(null);
  let errorReason = $state<'expired' | 'used' | 'not_found' | null>(null);
  let loading = $state(true);

  let step = $state<Step>('whatsapp');
  let whatsapp = $state('');
  let firstName = $state('');           // nome do usuário existente (para boas-vindas)

  let form = $state({ name: '', nickname: '', password: '' });
  let showPw = $state(false);
  let checking = $state(false);        // aguardando resposta do check
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
        if (!cancelled) {
          if (e instanceof ApiError) {
            if (e.message === 'Convite expirado') errorReason = 'expired';
            else if (e.message === 'Convite já utilizado') errorReason = 'used';
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

  async function checkWhatsapp() {
    error = '';
    checking = true;
    try {
      const result = await invites.checkWhatsapp(token, whatsapp);
      if (result.exists) {
        firstName = result.first_name ?? '';
        step = 'login';
      } else {
        step = 'register';
      }
    } catch (e) {
      error = e instanceof ApiError ? e.message : $t('invite.verify_whatsapp_error');
    }
    checking = false;
  }

  async function accept() {
    error = '';
    submitting = true;
    try {
      const payload: { whatsapp: string; password: string; name?: string; nickname?: string } = {
        whatsapp,
        password: form.password,
      };
      if (step === 'register') {
        payload.name = form.name;
        if (form.nickname) payload.nickname = form.nickname;
      }
      const res = await invites.accept(token, payload);
      authStore.login(res.access_token, res);
      done = true;
      setTimeout(() => goto('/'), 2000);
    } catch (e) {
      error = e instanceof ApiError ? e.message : $t('invite.accept_error');
    }
    submitting = false;
  }

  function back() {
    error = '';
    form = { name: '', nickname: '', password: '' };
    step = 'whatsapp';
  }

  function fmtExpiry(s: string) {
    return new Date(s).toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' });
  }
</script>

<svelte:head><title>Convite — rachao.app</title></svelte:head>

<div class="min-h-screen bg-gradient-to-br from-primary-700 to-primary-900 flex items-center justify-center p-4">
  <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-2xl w-full max-w-sm p-8">
    <div class="text-center mb-6">
      <div class="text-4xl mb-2">⚽</div>
      <h1 class="text-xl font-bold text-gray-900 dark:text-gray-100">{$t('invite.title')}</h1>
    </div>

    {#if loading}
      <div class="animate-pulse space-y-3">
        <div class="h-4 bg-gray-200 rounded"></div>
        <div class="h-4 bg-gray-200 rounded w-2/3"></div>
      </div>

    {:else if errorReason === 'expired'}
      <div class="alert-error text-center">
        <p class="font-semibold">{$t('invite.expired_title')}</p>
        <p class="mt-1 text-xs">{$t('invite.expired_desc')}</p>
      </div>

    {:else if errorReason === 'used'}
      <div class="alert-error text-center">
        <p class="font-semibold">{$t('invite.used_title')}</p>
        <p class="mt-1 text-xs">{$t('invite.used_desc')}</p>
      </div>

    {:else if errorReason === 'not_found'}
      <div class="alert-error text-center">
        <p class="font-semibold">{$t('invite.invalid_title')}</p>
        <p class="mt-1 text-xs">{$t('invite.invalid_desc')}</p>
      </div>

    {:else if done}
      <div class="text-center py-4">
        <CheckCircle size={48} class="text-green-500 mx-auto mb-3" />
        <p class="font-semibold text-gray-900 dark:text-gray-100">{$t('invite.joined')}</p>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">{$t('invite.redirecting')}</p>
      </div>

    {:else if info}
      <div class="alert-info mb-5 text-center">
        <p class="font-semibold">{$t('invite.invited_to')}</p>
        <p class="text-lg font-bold text-blue-800 mt-0.5">{info.group_name}</p>
        <p class="text-xs mt-1 text-blue-600">{$t('invite.expires_at').replace('{time}', fmtExpiry(info.expires_at))}</p>
      </div>

      {#if error}
        <div class="alert-error mb-4">{error}</div>
      {/if}

      <!-- ETAPA 1: WhatsApp -->
      {#if step === 'whatsapp'}
        <form onsubmit={(e) => { e.preventDefault(); checkWhatsapp(); }} class="space-y-4">
          <div class="form-group">
            <label class="label" for="wa">{$t('invite.whatsapp_label')}</label>
            <PhoneInput id="wa" bind:value={whatsapp} placeholder="11999990000" required />
            <p class="text-xs text-gray-400 mt-1">{$t('invite.whatsapp_hint')}</p>
          </div>
          <button type="submit" class="btn-primary w-full justify-center py-2.5" disabled={checking}>
            {checking ? $t('invite.checking') : $t('invite.continue')}
          </button>
        </form>

      <!-- ETAPA 2a: Usuário existente — só senha -->
      {:else if step === 'login'}
        <form onsubmit={(e) => { e.preventDefault(); accept(); }} class="space-y-4">
          <div class="alert-info text-sm">
            {#if firstName}
              {$t('invite.existing_user_with_name').replace('{name}', firstName)}
            {:else}
              {$t('invite.existing_user')}
            {/if}
            {$t('invite.enter_group')}
          </div>
          <div class="form-group">
            <label class="label" for="wa-ro">WhatsApp</label>
            <input id="wa-ro" class="input bg-gray-50 dark:bg-gray-700 text-gray-500 dark:text-gray-400" value={whatsapp} readonly />
          </div>
          <div class="form-group">
            <label class="label" for="pw">{$t('invite.password_label')}</label>
            <div class="relative">
              <input id="pw" class="input pr-10" type={showPw ? 'text' : 'password'}
                bind:value={form.password} placeholder={$t('invite.password_placeholder')} required />
              <button type="button" onclick={() => showPw = !showPw}
                class="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600">
                {#if showPw}<EyeOff size={16} />{:else}<Eye size={16} />{/if}
              </button>
            </div>
          </div>
          <button type="submit" class="btn-primary w-full justify-center py-2.5" disabled={submitting}>
            <UserPlus size={16} />
            {submitting ? $t('invite.joining') : $t('invite.join_group')}
          </button>
          <button type="button" onclick={back}
            class="w-full flex items-center justify-center gap-1 text-xs text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-400 mt-1">
            <ArrowLeft size={13} /> {$t('invite.use_other_number')}
          </button>
        </form>

      <!-- ETAPA 2b: Novo usuário — cadastro completo -->
      {:else if step === 'register'}
        <form onsubmit={(e) => { e.preventDefault(); accept(); }} class="space-y-3">
          <p class="text-sm text-gray-500 dark:text-gray-400 -mt-1">{$t('invite.fill_data')}</p>
          <div class="bg-primary-50 dark:bg-primary-900/20 rounded-xl p-3 text-xs text-primary-700 dark:text-primary-300 space-y-1">
            <p class="font-semibold text-primary-800 dark:text-primary-200 mb-1.5">{$t('invite.benefits_title')}</p>
            <p>{$t('invite.benefit_1')}</p>
            <p>{$t('invite.benefit_2')}</p>
            <p>{$t('invite.benefit_3')}</p>
            <p>{$t('invite.benefit_4')}</p>
          </div>
          <div class="form-group">
            <label class="label" for="name">{$t('invite.full_name_label')}</label>
            <input id="name" class="input" bind:value={form.name} placeholder="Seu nome" required minlength="2" />
          </div>
          <div class="form-group">
            <label class="label" for="nick">{$t('invite.nickname_label')}</label>
            <input id="nick" class="input" bind:value={form.nickname} placeholder="Como te chamam no campo?" />
          </div>
          <div class="form-group">
            <label class="label" for="wa-ro2">WhatsApp</label>
            <input id="wa-ro2" class="input bg-gray-50 dark:bg-gray-700 text-gray-500 dark:text-gray-400" value={whatsapp} readonly />
          </div>
          <div class="form-group">
            <label class="label" for="pw2">{$t('invite.create_password_label')}</label>
            <div class="relative">
              <input id="pw2" class="input pr-10" type={showPw ? 'text' : 'password'}
                bind:value={form.password} placeholder={$t('invite.create_password_placeholder')} required minlength="6" />
              <button type="button" onclick={() => showPw = !showPw}
                class="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600">
                {#if showPw}<EyeOff size={16} />{:else}<Eye size={16} />{/if}
              </button>
            </div>
          </div>
          <button type="submit" class="btn-primary w-full justify-center py-2.5 mt-1" disabled={submitting}>
            <UserPlus size={16} />
            {submitting ? $t('invite.creating') : $t('invite.create_and_join')}
          </button>
          <button type="button" onclick={back}
            class="w-full flex items-center justify-center gap-1 text-xs text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-400">
            <ArrowLeft size={13} /> {$t('invite.use_other_number')}
          </button>
        </form>
      {/if}
    {/if}
  </div>
</div>
