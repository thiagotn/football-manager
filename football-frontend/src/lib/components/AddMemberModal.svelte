<script lang="ts">
  import { groups as groupsApi, ApiError } from '$lib/api';
  import type { LookupPlayerInfo, GroupMember } from '$lib/api';
  import Modal from '$lib/components/Modal.svelte';
  import PhoneInput from '$lib/components/PhoneInput.svelte';
  import StarRating from '$lib/components/StarRating.svelte';
  import AvatarImage from '$lib/components/AvatarImage.svelte';
  import { t } from '$lib/i18n';

  let {
    open = $bindable(false),
    groupId,
    onAdded,
  }: {
    open?: boolean;
    groupId: string;
    onAdded?: (member: GroupMember, isNew: boolean) => void;
  } = $props();

  type Step = 'phone' | 'confirm_existing' | 'create_new' | 'success';

  let step = $state<Step>('phone');
  let phone = $state('');
  let searching = $state(false);
  let saving = $state(false);
  let errorMsg = $state('');

  // Found player (existing account)
  let foundPlayer = $state<LookupPlayerInfo | null>(null);

  // Form fields
  let name = $state('');
  let nickname = $state('');
  let skillStars = $state(2);
  let isGoalkeeper = $state(false);

  // Success state
  let successName = $state('');
  let successIsNew = $state(false);
  let successPhone = $state('');

  function reset() {
    step = 'phone';
    phone = '';
    searching = false;
    saving = false;
    errorMsg = '';
    foundPlayer = null;
    name = '';
    nickname = '';
    skillStars = 2;
    isGoalkeeper = false;
    successName = '';
    successIsNew = false;
    successPhone = '';
  }

  function handleClose() {
    reset();
    open = false;
  }

  function formatPhoneDisplay(e164: string): string {
    // Remove +55 prefix and format as (XX) XXXXX-XXXX for BR numbers
    if (e164.startsWith('+55') && e164.length >= 13) {
      const local = e164.slice(3);
      if (local.length === 11) {
        return `(${local.slice(0,2)}) ${local.slice(2,7)}-${local.slice(7)}`;
      }
    }
    return e164;
  }

  async function searchPlayer() {
    if (!phone) return;
    errorMsg = '';
    searching = true;
    try {
      const res = await groupsApi.lookupMember(groupId, phone);
      if (res.status === 'already_member') {
        errorMsg = $t('groups.add_manual.already_member');
      } else if (res.status === 'found') {
        foundPlayer = res.player ?? null;
        step = 'confirm_existing';
      } else {
        step = 'create_new';
      }
    } catch (e) {
      errorMsg = e instanceof ApiError ? e.message : $t('groups.add_manual.error_generic');
    }
    searching = false;
  }

  async function confirmAdd() {
    errorMsg = '';
    saving = true;
    try {
      const res = await groupsApi.addMemberByPhone(groupId, {
        whatsapp: phone,
        skill_stars: skillStars,
        is_goalkeeper: isGoalkeeper,
      });
      successName = res.member.player.name;
      successIsNew = res.is_new;
      successPhone = formatPhoneDisplay(phone);
      step = 'success';
      onAdded?.(res.member, res.is_new);
    } catch (e) {
      if (e instanceof ApiError) {
        if (e.status === 403 && e.message === 'PLAN_LIMIT_EXCEEDED') {
          errorMsg = $t('groups.add_manual.error_limit');
        } else if (e.status === 409) {
          errorMsg = $t('groups.add_manual.error_conflict');
        } else {
          errorMsg = e.message;
        }
      } else {
        errorMsg = $t('groups.add_manual.error_generic');
      }
    }
    saving = false;
  }

  async function createAndAdd() {
    if (!name.trim()) return;
    errorMsg = '';
    saving = true;
    try {
      const res = await groupsApi.addMemberByPhone(groupId, {
        whatsapp: phone,
        name: name.trim(),
        nickname: nickname.trim() || undefined,
        skill_stars: skillStars,
        is_goalkeeper: isGoalkeeper,
      });
      successName = res.member.player.name;
      successIsNew = res.is_new;
      successPhone = formatPhoneDisplay(phone);
      step = 'success';
      onAdded?.(res.member, res.is_new);
    } catch (e) {
      if (e instanceof ApiError) {
        if (e.status === 403 && e.message === 'PLAN_LIMIT_EXCEEDED') {
          errorMsg = $t('groups.add_manual.error_limit');
        } else if (e.status === 409) {
          errorMsg = $t('groups.add_manual.error_conflict');
        } else {
          errorMsg = e.message;
        }
      } else {
        errorMsg = $t('groups.add_manual.error_generic');
      }
    }
    saving = false;
  }

  function stepTitle(): string {
    if (step === 'confirm_existing') return $t('groups.add_manual.found_title');
    if (step === 'create_new') return $t('groups.add_manual.new_title');
    return $t('groups.add_manual.title');
  }
</script>

<Modal bind:open title={stepTitle()} onClose={handleClose}>
  <!-- Step 1: Phone search -->
  {#if step === 'phone'}
    <form onsubmit={(e) => { e.preventDefault(); searchPlayer(); }} class="space-y-4">
      <div class="form-group">
        <label class="label" for="add-phone">{$t('groups.add_manual.step1_label')}</label>
        <PhoneInput id="add-phone" bind:value={phone} required />
      </div>
      {#if errorMsg}
        <p class="text-sm text-red-500">{errorMsg}</p>
      {/if}
      <div class="flex justify-end">
        <button type="submit" class="btn-primary" disabled={searching || !phone}>
          {searching ? '...' : $t('groups.add_manual.search_btn')}
        </button>
      </div>
    </form>

  <!-- Step 2-A: Confirm existing player -->
  {:else if step === 'confirm_existing'}
    <div class="space-y-4">
      {#if foundPlayer}
        <div class="flex items-center gap-3 p-3 rounded-lg bg-gray-800/50">
          <AvatarImage name={foundPlayer.name} avatarUrl={foundPlayer.avatar_url} size={48} />
          <div>
            <p class="font-semibold text-gray-100">{foundPlayer.name}</p>
            {#if foundPlayer.nickname}
              <p class="text-sm text-gray-400">{foundPlayer.nickname}</p>
            {/if}
          </div>
        </div>
      {/if}
      <p class="text-sm text-gray-400">{$t('groups.add_manual.found_desc')}</p>

      <!-- Stars + Goalkeeper -->
      <div class="space-y-3">
        <div class="flex items-center gap-3">
          <span class="text-sm text-gray-400 w-20">{$t('groups.add_manual.stars_label')}</span>
          <StarRating bind:rating={skillStars} size={20} />
        </div>
        <div class="flex items-center gap-3">
          <span class="text-sm text-gray-400 w-20">{$t('groups.add_manual.goalkeeper_label')}</span>
          <button
            type="button"
            onclick={() => isGoalkeeper = !isGoalkeeper}
            class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors {isGoalkeeper ? 'bg-primary-500' : 'bg-gray-600'}"
            role="switch"
            aria-checked={isGoalkeeper}
          >
            <span class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform {isGoalkeeper ? 'translate-x-6' : 'translate-x-1'}" />
          </button>
        </div>
      </div>

      {#if errorMsg}
        <p class="text-sm text-red-500">{errorMsg}</p>
      {/if}
      <div class="flex gap-3 justify-between">
        <button class="btn-secondary btn-sm" onclick={() => { step = 'phone'; errorMsg = ''; }}>
          ← {$t('groups.add_manual.back_btn')}
        </button>
        <button class="btn-primary" onclick={confirmAdd} disabled={saving}>
          {saving ? '...' : $t('groups.add_manual.confirm_btn')}
        </button>
      </div>
    </div>

  <!-- Step 2-B: Create new player -->
  {:else if step === 'create_new'}
    <form onsubmit={(e) => { e.preventDefault(); createAndAdd(); }} class="space-y-4">
      <!-- Phone read-only -->
      <div class="form-group">
        <p class="label">{$t('groups.add_manual.phone_readonly_label')}</p>
        <p class="input bg-gray-800/50 text-gray-300 cursor-default">{formatPhoneDisplay(phone)}</p>
      </div>

      <!-- Name -->
      <div class="form-group">
        <label class="label" for="new-name">{$t('groups.add_manual.name_label')} *</label>
        <input
          id="new-name"
          class="input"
          type="text"
          bind:value={name}
          placeholder={$t('groups.add_manual.name_placeholder')}
          minlength="2"
          maxlength="100"
          required
        />
      </div>

      <!-- Nickname -->
      <div class="form-group">
        <label class="label" for="new-nickname">{$t('groups.add_manual.nickname_label')}</label>
        <input
          id="new-nickname"
          class="input"
          type="text"
          bind:value={nickname}
          placeholder={$t('groups.add_manual.nickname_placeholder')}
          maxlength="50"
        />
      </div>

      <!-- Stars + Goalkeeper -->
      <div class="space-y-3">
        <div class="flex items-center gap-3">
          <span class="text-sm text-gray-400 w-20">{$t('groups.add_manual.stars_label')}</span>
          <StarRating bind:rating={skillStars} size={20} />
        </div>
        <div class="flex items-center gap-3">
          <span class="text-sm text-gray-400 w-20">{$t('groups.add_manual.goalkeeper_label')}</span>
          <button
            type="button"
            onclick={() => isGoalkeeper = !isGoalkeeper}
            class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors {isGoalkeeper ? 'bg-primary-500' : 'bg-gray-600'}"
            role="switch"
            aria-checked={isGoalkeeper}
          >
            <span class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform {isGoalkeeper ? 'translate-x-6' : 'translate-x-1'}" />
          </button>
        </div>
      </div>

      {#if errorMsg}
        <p class="text-sm text-red-500">{errorMsg}</p>
      {/if}
      <div class="flex gap-3 justify-between">
        <button type="button" class="btn-secondary btn-sm" onclick={() => { step = 'phone'; errorMsg = ''; }}>
          ← {$t('groups.add_manual.back_btn')}
        </button>
        <button type="submit" class="btn-primary" disabled={saving || !name.trim()}>
          {saving ? '...' : $t('groups.add_manual.create_btn')}
        </button>
      </div>
    </form>

  <!-- Success -->
  {:else if step === 'success'}
    <div class="space-y-4">
      <div class="p-4 rounded-lg bg-green-900/30 border border-green-700/40 text-sm text-green-300 space-y-2">
        <p class="font-semibold">✅ {$t('groups.add_manual.success_added').replace('{name}', successName)}</p>
        {#if successIsNew}
          <p class="text-green-200/80 text-xs mt-2">
            {$t('groups.add_manual.success_new_hint').replace('{phone}', successPhone)}
          </p>
        {/if}
      </div>
      <div class="flex justify-end">
        <button class="btn-primary" onclick={handleClose}>{$t('groups.add_manual.close_btn')}</button>
      </div>
    </div>
  {/if}
</Modal>
