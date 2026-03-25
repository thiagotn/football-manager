<script lang="ts">
  import { goto } from '$app/navigation';
  import { isLoggedIn, isAdmin } from '$lib/stores/auth';
  import { subscriptions as subsApi, ApiError } from '$lib/api';
  import type { SubscriptionInfo } from '$lib/api';
  import { PLANS, PLAN_ORDER, formatCents } from '$lib/plans';
  import { billingEnabled } from '$lib/billing';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import { CreditCard, Check, Zap } from 'lucide-svelte';
  import { t } from '$lib/i18n';

  if (!billingEnabled) goto('/');

  let sub = $state<SubscriptionInfo | null>(null);
  let cycle = $state<'monthly' | 'yearly'>('monthly');
  let loading = $state(false);
  let checkoutLoading = $state<string | null>(null); // plan key sendo processado

  $effect(() => {
    if (!$isLoggedIn || $isAdmin) return;
    subsApi.me().then(d => { sub = d; }).catch(() => {});
  });

  function price(planKey: string): string {
    const p = PLANS[planKey as keyof typeof PLANS];
    if (!p || p.price_monthly === null) return $t('plans.free');
    const cents = cycle === 'yearly'
      ? (p.price_yearly ?? p.price_monthly * 10)
      : p.price_monthly;
    return formatCents(cents);
  }

  function priceSuffix(): string {
    return cycle === 'yearly' ? $t('plans.per_year') : $t('plans.per_month');
  }

  function yearlySaving(planKey: string): string | null {
    const p = PLANS[planKey as keyof typeof PLANS];
    if (!p || !p.price_monthly || !p.price_yearly) return null;
    const fullYear = p.price_monthly * 12;
    const saved = fullYear - p.price_yearly;
    if (saved <= 0) return null;
    return formatCents(saved);
  }

  async function handleSelect(planKey: string) {
    if (planKey === 'free') return;
    const plan = PLANS[planKey as keyof typeof PLANS];
    if (!plan?.available) return;

    if (!$isLoggedIn) {
      goto(`/register?plan=${planKey}`);
      return;
    }

    if (sub?.plan === planKey) return;

    checkoutLoading = planKey;
    try {
      const { checkout_url } = await subsApi.createCheckout(planKey, cycle);
      window.location.href = checkout_url;
    } catch (e) {
      alert(e instanceof ApiError ? e.message : 'Erro ao iniciar checkout. Tente novamente.');
      checkoutLoading = null;
    }
  }
</script>

<svelte:head><title>Planos — rachao.app</title></svelte:head>

<PageBackground>
  <main class="relative z-10 max-w-4xl mx-auto px-4 py-8">

    <!-- Cabeçalho -->
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-white flex items-center gap-2">
          <CreditCard size={24} class="text-primary-400" /> {$t('plans.title')}
        </h1>
        <p class="text-sm text-white/60 mt-0.5">{$t('plans.subtitle')}</p>
      </div>
    </div>

    <!-- Toggle mensal/anual -->
    <div class="flex justify-center mb-8">
      <div class="inline-flex items-center bg-white/10 rounded-full p-1 gap-1">
        <button
          class="px-4 py-1.5 rounded-full text-sm font-medium transition-colors {cycle === 'monthly' ? 'bg-white text-gray-900' : 'text-white/70 hover:text-white'}"
          onclick={() => cycle = 'monthly'}>
          {$t('plans.monthly')}
        </button>
        <button
          class="px-4 py-1.5 rounded-full text-sm font-medium transition-colors {cycle === 'yearly' ? 'bg-white text-gray-900' : 'text-white/70 hover:text-white'}"
          onclick={() => cycle = 'yearly'}>
          {$t('plans.yearly')}
          <span class="ml-1 text-xs font-semibold text-green-400">{$t('plans.yearly_saving')}</span>
        </button>
      </div>
    </div>

    <!-- Cards de plano -->
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
      {#each PLAN_ORDER as planKey}
        {@const plan = PLANS[planKey]}
        {@const isCurrent = sub?.plan === planKey}
        {@const saving = cycle === 'yearly' ? yearlySaving(planKey) : null}
        <div class="card card-body flex flex-col relative {isCurrent ? 'ring-2 ring-primary-500' : ''} {planKey === 'basic' ? 'sm:scale-[1.02]' : ''}">

          {#if isCurrent}
            <span class="absolute -top-3 left-1/2 -translate-x-1/2 bg-primary-500 text-white text-xs font-bold px-3 py-0.5 rounded-full whitespace-nowrap">
              {$t('plans.current_plan')}
            </span>
          {/if}
          {#if planKey === 'basic' && !isCurrent}
            <span class="absolute -top-3 left-1/2 -translate-x-1/2 bg-amber-400 text-gray-900 text-xs font-bold px-3 py-0.5 rounded-full whitespace-nowrap">
              {$t('plans.most_popular')}
            </span>
          {/if}

          <h2 class="text-lg font-bold text-gray-900 dark:text-gray-100">{$t(plan.name)}</h2>

          <div class="mt-2 mb-4">
            {#if plan.price_monthly === null}
              <span class="text-3xl font-black text-gray-900 dark:text-gray-100">{$t('plans.free')}</span>
            {:else}
              <span class="text-3xl font-black text-gray-900 dark:text-gray-100">{price(planKey)}</span>
              <span class="text-sm text-gray-400 dark:text-gray-500">{priceSuffix()}</span>
              {#if saving}
                <p class="text-xs text-green-600 dark:text-green-400 mt-0.5">{$t('plans.yearly_saving_hint')}</p>
              {/if}
            {/if}
          </div>

          <ul class="space-y-2 flex-1 mb-6">
            {#each plan.highlights as item}
              <li class="flex items-start gap-2 text-sm text-gray-600 dark:text-gray-400">
                <Check size={14} class="text-primary-500 shrink-0 mt-0.5" />
                {$t(item)}
              </li>
            {/each}
          </ul>

          {#if planKey === 'free'}
            <button class="btn-secondary w-full justify-center" disabled>
              {isCurrent ? $t('plans.current_btn') : $t('plans.free_btn')}
            </button>
          {:else if isCurrent}
            <button class="btn-secondary w-full justify-center" disabled>{$t('plans.current_btn')}</button>
          {:else}
            <button
              class="btn-primary w-full justify-center"
              onclick={() => handleSelect(planKey)}
              disabled={checkoutLoading === planKey}>
              {#if checkoutLoading === planKey}
                <Zap size={15} class="animate-pulse" /> {$t('plans.subscribe_loading')}
              {:else}
                <Zap size={15} /> {$t('plans.subscribe').replace('{name}', $t(plan.name))}
              {/if}
            </button>
          {/if}
        </div>
      {/each}
    </div>

    <p class="text-center text-sm text-white/80 mt-8 bg-white/10 rounded-xl px-4 py-3 max-w-sm mx-auto">
      {$t('plans.security_note')}
    </p>

  </main>
</PageBackground>
