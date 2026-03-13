<script lang="ts">
  import { goto } from '$app/navigation';
  import { billingEnabled } from '$lib/billing';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import { CheckCircle } from 'lucide-svelte';

  if (!billingEnabled) goto('/');
</script>

<svelte:head><title>Assinatura confirmada — rachao.app</title></svelte:head>

<PageBackground>
  <main class="relative z-10 max-w-md mx-auto px-4 py-16 text-center">
    <div class="card card-body flex flex-col items-center gap-4">
      <CheckCircle size={56} class="text-green-500" />
      <h1 class="text-2xl font-bold text-gray-900 dark:text-gray-100">Assinatura confirmada!</h1>
      <p class="text-gray-500 dark:text-gray-400 text-sm">
        Seu plano foi ativado com sucesso. Aproveite todos os recursos disponíveis.
      </p>
      <div class="flex flex-col gap-2 w-full mt-2">
        <a href="/" class="btn-primary w-full justify-center">Ir para o Dashboard</a>
        <a href="/account/subscription" class="btn-secondary w-full justify-center">Ver minha assinatura</a>
      </div>
    </div>
  </main>
</PageBackground>
