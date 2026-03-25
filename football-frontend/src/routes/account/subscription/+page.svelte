<script lang="ts">
  import { PUBLIC_LEGAL_CONTACT_EMAIL } from '$env/static/public';
  import { goto } from '$app/navigation';
  import { isLoggedIn, isAdmin } from '$lib/stores/auth';
  import { subscriptions as subsApi, ApiError } from '$lib/api';
  import type { SubscriptionInfo } from '$lib/api';
  import { PLANS, formatCents } from '$lib/plans';
  import { billingEnabled } from '$lib/billing';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import { CreditCard, Zap, Check, ExternalLink } from 'lucide-svelte';
  import { t } from '$lib/i18n';

  if (!billingEnabled) goto('/');

  let sub = $state<SubscriptionInfo | null>(null);
  let loading = $state(true);
  let cycle = $state<'monthly' | 'yearly'>('monthly');
  let selectedPlan = $state<'basic' | 'pro'>('basic');
  let checkoutLoading = $state(false);
  let error = $state('');

  $effect(() => {
    if (!$isLoggedIn) { goto('/login'); return; }
    if ($isAdmin) { goto('/'); return; }
    subsApi.me()
      .then(d => { sub = d; loading = false; })
      .catch(() => { error = $t('account.load_error'); loading = false; });
  });

  function formatDate(iso: string | null): string {
    if (!iso) return '—';
    return new Date(iso).toLocaleDateString('pt-BR', { day: '2-digit', month: 'long', year: 'numeric' });
  }

  function statusLabel(status: string): { label: string; cls: string } {
    return ({
      active:   { label: $t('account.status_active'),    cls: 'badge-green' },
      past_due: { label: $t('account.status_past_due'),  cls: 'badge-yellow' },
      canceled: { label: $t('account.status_canceled'),  cls: 'badge-gray'  },
      free:     { label: $t('account.status_free'),      cls: 'badge-gray'  },
    } as Record<string, { label: string; cls: string }>)[status] ?? { label: status, cls: 'badge-gray' };
  }

  function price(planKey: string, c: 'monthly' | 'yearly'): string {
    const p = PLANS[planKey as keyof typeof PLANS];
    if (!p || p.price_monthly === null) return $t('account.status_free');
    const cents = c === 'yearly' ? (p.price_yearly ?? p.price_monthly * 10) : p.price_monthly;
    return `${formatCents(cents)}${c === 'yearly' ? $t('plans.per_year') : $t('plans.per_month')}`;
  }

  async function startCheckout() {
    checkoutLoading = true;
    try {
      const { checkout_url } = await subsApi.createCheckout(selectedPlan, cycle);
      window.location.href = checkout_url;
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Erro ao iniciar checkout. Tente novamente.';
      checkoutLoading = false;
    }
  }
</script>

<svelte:head><title>Assinatura — rachao.app</title></svelte:head>

<PageBackground>
  <main class="relative z-10 max-w-4xl mx-auto px-4 py-8">

    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-white flex items-center gap-2">
          <CreditCard size={24} class="text-primary-400" /> {$t('account.title')}
        </h1>
        <p class="text-sm text-white/60 mt-0.5">{$t('account.subtitle')}</p>
      </div>
    </div>

    {#if loading}
      <div class="space-y-4">
        {#each [1, 2] as _}
          <div class="card animate-pulse h-28 bg-gray-100 dark:bg-gray-800"></div>
        {/each}
      </div>

    {:else if error}
      <div class="card card-body text-red-500 text-center">{error}</div>

    {:else if sub}
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 items-start">

        <!-- Coluna esquerda: plano atual -->
        <div class="space-y-4">
          <div class="card card-body">
            <h2 class="font-semibold text-gray-800 dark:text-gray-200 mb-4">{$t('account.current_plan')}</h2>
            <div class="flex items-center gap-2 mb-4">
              <span class="text-2xl font-black text-gray-900 dark:text-gray-100 capitalize">
                {$t(PLANS[sub.plan as keyof typeof PLANS]?.name ?? sub.plan)}
              </span>
              <span class="badge {statusLabel(sub.status ?? sub.plan).cls}">
                {statusLabel(sub.status ?? sub.plan).label}
              </span>
            </div>

            <dl class="space-y-2 text-sm">
              {#if sub.current_period_end}
                <div class="flex justify-between">
                  <dt class="text-gray-500 dark:text-gray-400">{$t('account.renews_at')}</dt>
                  <dd class="font-medium text-gray-800 dark:text-gray-200">{formatDate(sub.current_period_end)}</dd>
                </div>
              {/if}
              {#if sub.grace_period_end}
                <div class="flex justify-between">
                  <dt class="text-gray-500 dark:text-gray-400">{$t('account.grace_period')}</dt>
                  <dd class="font-medium text-amber-600 dark:text-amber-400">{formatDate(sub.grace_period_end)}</dd>
                </div>
              {/if}
              <div class="flex justify-between">
                <dt class="text-gray-500 dark:text-gray-400">{$t('account.groups')}</dt>
                <dd class="font-medium text-gray-800 dark:text-gray-200">
                  {sub.groups_used} {sub.groups_limit !== null ? `de ${sub.groups_limit}` : $t('account.groups_unlimited')}
                </dd>
              </div>
              {#if sub.members_limit !== null}
                <div class="flex justify-between">
                  <dt class="text-gray-500 dark:text-gray-400">{$t('account.members')}</dt>
                  <dd class="font-medium text-gray-800 dark:text-gray-200">{$t('account.members_up_to').replace('{n}', String(sub.members_limit))}</dd>
                </div>
              {/if}
            </dl>

            {#if sub.gateway_sub_id}
              <p class="text-xs text-gray-400 dark:text-gray-500 mt-4 pt-3 border-t border-gray-100 dark:border-gray-700">
                {#if $t('account.cancel_contact').includes('{email}')}
                  {@html $t('account.cancel_contact').replace('{email}', `<a href="mailto:${PUBLIC_LEGAL_CONTACT_EMAIL}" class="underline">${PUBLIC_LEGAL_CONTACT_EMAIL}</a>`)}
                {:else}
                  Para cancelar, entre em contato pelo e-mail <a href="mailto:{PUBLIC_LEGAL_CONTACT_EMAIL}" class="underline">{PUBLIC_LEGAL_CONTACT_EMAIL}</a> ou acesse o portal do Stripe.
                {/if}
              </p>
            {/if}
          </div>
        </div>

        <!-- Coluna direita: upgrade (apenas se estiver no free) -->
        {#if sub.plan === 'free'}
          <div class="card card-body">
            <h2 class="font-semibold text-gray-800 dark:text-gray-200 mb-4">{$t('account.upgrade_title')}</h2>

            <!-- Ciclo de cobrança -->
            <div class="flex gap-2 mb-4">
              <button
                class="flex-1 py-2 rounded-lg text-sm font-medium border transition-colors {cycle === 'monthly' ? 'bg-primary-50 dark:bg-primary-900/20 border-primary-400 text-primary-700 dark:text-primary-300' : 'border-gray-200 dark:border-gray-700 text-gray-600 dark:text-gray-400 hover:border-gray-300'}"
                onclick={() => cycle = 'monthly'}>
                {$t('account.monthly')}
              </button>
              <button
                class="flex-1 py-2 rounded-lg text-sm font-medium border transition-colors {cycle === 'yearly' ? 'bg-primary-50 dark:bg-primary-900/20 border-primary-400 text-primary-700 dark:text-primary-300' : 'border-gray-200 dark:border-gray-700 text-gray-600 dark:text-gray-400 hover:border-gray-300'}"
                onclick={() => cycle = 'yearly'}>
                {$t('account.yearly')} <span class="text-xs text-green-600 dark:text-green-400">{$t('account.yearly_saving')}</span>
              </button>
            </div>

            <!-- Seleção do plano -->
            <div class="space-y-2 mb-4">
              {#each (['basic', 'pro'] as const) as pk}
                {@const plan = PLANS[pk]}
                <button
                  class="w-full flex items-center gap-3 p-3 rounded-xl border-2 transition-colors text-left {selectedPlan === pk ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20' : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'}"
                  onclick={() => selectedPlan = pk}>
                  <span class="w-5 h-5 rounded-full border-2 flex items-center justify-center shrink-0 {selectedPlan === pk ? 'border-primary-500 bg-primary-500' : 'border-gray-300 dark:border-gray-600'}">
                    {#if selectedPlan === pk}<Check size={12} class="text-white" />{/if}
                  </span>
                  <span class="flex-1 min-w-0">
                    <span class="font-semibold text-gray-900 dark:text-gray-100 text-sm">{$t(plan.name)}</span>
                    <span class="text-xs text-gray-400 dark:text-gray-500 ml-2">{$t('account.members_hint').replace('{groups}', String(plan.groups)).replace('{members}', plan.members === -1 ? $t('account.members_unlimited') : String(plan.members))}</span>
                  </span>
                  <span class="text-sm font-bold text-primary-600 dark:text-primary-400 shrink-0">{price(pk, cycle)}</span>
                </button>
              {/each}
            </div>

            {#if error}
              <div class="text-sm text-red-500 mb-3">{error}</div>
            {/if}

            <button
              class="btn-primary w-full justify-center"
              onclick={startCheckout}
              disabled={checkoutLoading}>
              {#if checkoutLoading}
                <Zap size={15} class="animate-pulse" /> {$t('account.checkout_redirecting')}
              {:else}
                <ExternalLink size={15} /> {$t('account.checkout_btn')}
              {/if}
            </button>

            <p class="text-xs text-gray-400 dark:text-gray-500 text-center mt-3">
              {$t('account.checkout_secure')}
            </p>
          </div>
        {/if}

      </div>
    {/if}

  </main>
</PageBackground>
