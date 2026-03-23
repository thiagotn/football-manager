<script lang="ts">
  import { goto } from '$app/navigation';
  import { groups as groupsApi } from '$lib/api';
  import { toastSuccess, toastError } from '$lib/stores/toast';
  import { t } from '$lib/i18n';
  import { TIMEZONE_OPTIONS, TIMEZONE_GROUPS } from '$lib/timezones';

  let name = $state('');
  let description = $state('');
  let isPublic = $state(true);
  let voteOpenDelay = $state(20);
  let voteDuration = $state(24);
  let timezone = $state('America/Sao_Paulo');
  let loading = $state(false);
  let error = $state('');

  async function handleCreate() {
    error = '';
    loading = true;
    try {
      const g = await groupsApi.create({
        name,
        description: description || undefined,
        is_public: isPublic,
        vote_open_delay_minutes: voteOpenDelay,
        vote_duration_hours: voteDuration,
        timezone,
      });
      toastSuccess($t('new_group.success'));
      goto(`/groups/${g.id}`);
    } catch (e: any) {
      error = e.message ?? 'Erro ao criar grupo';
    } finally {
      loading = false;
    }
  }
</script>

<svelte:head><title>Novo Grupo — rachao.app</title></svelte:head>

<main class="max-w-xl mx-auto px-4 py-8">
  <div class="mb-6">
    <a href="/groups" class="text-sm text-gray-500 hover:text-gray-700">{$t('new_group.back')}</a>
    <h1 class="text-2xl font-bold text-gray-900 mt-2">{$t('new_group.title')}</h1>
  </div>

  <div class="card p-6">
    <form onsubmit={(e) => { e.preventDefault(); handleCreate(); }} class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">{$t('new_group.name_label')}</label>
        <input
          type="text"
          bind:value={name}
          class="input"
          placeholder={$t('new_group.name_placeholder')}
          required
          maxlength="100"
        />
      </div>
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">{$t('new_group.desc_label')}</label>
        <textarea
          bind:value={description}
          class="input resize-none"
          rows="3"
          placeholder={$t('new_group.desc_placeholder')}
          maxlength="500"
        ></textarea>
      </div>

      <div class="border-t border-gray-100 pt-4">
        <label class="block text-sm font-medium text-gray-700 mb-1">{$t('new_group.timezone_label')}</label>
        <p class="text-xs text-gray-500 mb-2">{$t('new_group.timezone_desc')}</p>
        <select bind:value={timezone} class="input">
          {#each TIMEZONE_GROUPS as group}
            <optgroup label={group}>
              {#each TIMEZONE_OPTIONS.filter(tz => tz.group === group) as tz}
                <option value={tz.value}>{tz.label} ({tz.offset})</option>
              {/each}
            </optgroup>
          {/each}
        </select>
      </div>

      <div class="border-t border-gray-100 pt-4">
        <p class="text-sm font-medium text-gray-700 mb-3">{$t('new_group.visibility')}</p>
        <div class="flex items-center justify-between p-3 rounded-xl border border-gray-200 bg-gray-50">
          <div>
            <p class="text-sm font-medium text-gray-700">{$t('new_group.public_title')}</p>
            <p class="text-xs text-gray-500 mt-0.5">
              {#if isPublic}
                {$t('new_group.public_desc')}
              {:else}
                {$t('new_group.private_desc')}
              {/if}
            </p>
          </div>
          <button
            type="button"
            onclick={() => isPublic = !isPublic}
            class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors {isPublic ? 'bg-primary-600' : 'bg-gray-300'}"
            role="switch"
            aria-checked={isPublic}>
            <span class="inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform {isPublic ? 'translate-x-6' : 'translate-x-1'}"></span>
          </button>
        </div>
      </div>

      <div class="border-t border-gray-100 pt-4">
        <p class="text-sm font-medium text-gray-700 mb-3">{$t('new_group.vote_settings')}</p>
        <div class="space-y-3">
          <div>
            <label class="block text-sm text-gray-600 mb-1">{$t('new_group.vote_delay_label')}</label>
            <select bind:value={voteOpenDelay} class="input">
              <option value={0}>{$t('new_group.vote_immediate')}</option>
              <option value={10}>{$t('new_group.vote_10min')}</option>
              <option value={20}>{$t('new_group.vote_20min')}</option>
              <option value={30}>{$t('new_group.vote_30min')}</option>
              <option value={60}>{$t('new_group.vote_1h')}</option>
            </select>
          </div>
          <div>
            <label class="block text-sm text-gray-600 mb-1">{$t('new_group.vote_duration_label')}</label>
            <select bind:value={voteDuration} class="input">
              <option value={2}>{$t('new_group.vote_2h')}</option>
              <option value={4}>{$t('new_group.vote_4h')}</option>
              <option value={6}>{$t('new_group.vote_6h')}</option>
              <option value={12}>{$t('new_group.vote_12h')}</option>
              <option value={24}>{$t('new_group.vote_24h')}</option>
              <option value={48}>{$t('new_group.vote_48h')}</option>
              <option value={72}>{$t('new_group.vote_72h')}</option>
            </select>
          </div>
        </div>
      </div>

      {#if error}
        <div class="bg-red-50 border border-red-200 text-red-700 text-sm px-3 py-2 rounded-lg">{error}</div>
      {/if}

      <div class="flex gap-3 pt-2">
        <a href="/groups" class="btn-secondary flex-1 justify-center">{$t('new_group.cancel')}</a>
        <button type="submit" class="btn-primary flex-1" disabled={loading || !name.trim()}>
          {loading ? $t('new_group.creating') : $t('new_group.create')}
        </button>
      </div>
    </form>
  </div>
</main>
