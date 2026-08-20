<script lang="ts">
  import { page } from '$app/stores';
  import { matches as matchesApi, matchVideos, groups as groupsApi, ApiError } from '$lib/api';
  import type { MatchDetail, MatchVideosResponse, MatchVideoItem } from '$lib/api';
  import { currentPlayer, isAdmin, isLoggedIn } from '$lib/stores/auth';
  import { toastSuccess, toastError } from '$lib/stores/toast';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import MatchBannerCard from '$lib/components/MatchBannerCard.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import AvatarImage from '$lib/components/AvatarImage.svelte';
  import { Clapperboard, Plus, Trash2, Loader2 } from 'lucide-svelte';
  import { t } from '$lib/i18n';
  import { playerDisplayName } from '$lib/utils';

  const matchHash = $page.params.hash ?? '';

  const MAX_SIZE_BYTES = 150 * 1024 * 1024;
  const MAX_DURATION_SECONDS = 61; // margem sobre 60s; o worker (ffprobe) é a autoridade

  let match = $state<MatchDetail | null>(null);
  let data = $state<MatchVideosResponse | null>(null);
  let loading = $state(true);
  let isGroupAdmin = $state(false);

  let fileInput = $state<HTMLInputElement | null>(null);
  let uploading = $state(false);
  let uploadPct = $state(0);

  let confirmDeleteOpen = $state(false);
  let deleteTarget = $state<MatchVideoItem | null>(null);
  let deleting = $state(false);

  // Autoplay do vídeo visível (mudo) e pausa dos demais + src lazy
  let observer: IntersectionObserver | null = null;

  async function refresh() {
    data = await matchVideos.listPublic(matchHash);
  }

  $effect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [m, v] = await Promise.all([
          matchesApi.getByHash(matchHash),
          matchVideos.listPublic(matchHash),
        ]);
        if (cancelled) return;
        match = m;
        data = v;
      } catch {
        if (!cancelled) { match = null; data = null; }
      }
      if (!cancelled) loading = false;
    })();
    return () => { cancelled = true; };
  });

  $effect(() => {
    const player = $currentPlayer;
    const m = match;
    if (!player || !m) return;
    if (player.role === 'admin') { isGroupAdmin = true; return; }
    groupsApi.get(m.group_id)
      .then(g => { isGroupAdmin = g.members.some(mb => mb.player.id === player.id && mb.role === 'admin'); })
      .catch(() => { isGroupAdmin = false; });
  });

  // Enquanto houver upload próprio em processamento, atualiza a lista a cada 5s
  let hasInFlight = $derived(
    (data?.videos ?? []).some(v =>
      v.uploader?.id === $currentPlayer?.id &&
      (v.status === 'uploaded' || v.status === 'processing')
    )
  );

  $effect(() => {
    if (!hasInFlight) return;
    const id = setInterval(() => { refresh().catch(() => {}); }, 5000);
    return () => clearInterval(id);
  });

  function videoDuration(file: File): Promise<number> {
    return new Promise((resolve, reject) => {
      const url = URL.createObjectURL(file);
      const el = document.createElement('video');
      el.preload = 'metadata';
      el.onloadedmetadata = () => { URL.revokeObjectURL(url); resolve(el.duration); };
      el.onerror = () => { URL.revokeObjectURL(url); reject(new Error('metadata')); };
      el.src = url;
    });
  }

  async function onFileChosen(e: Event) {
    const file = (e.target as HTMLInputElement).files?.[0];
    if (!file || !match) return;

    if (file.size > MAX_SIZE_BYTES) {
      toastError($t('match.videos.too_large'));
      if (fileInput) fileInput.value = '';
      return;
    }
    // Validação advisory de duração — se o metadata não carregar, deixa passar
    try {
      const dur = await videoDuration(file);
      if (isFinite(dur) && dur > MAX_DURATION_SECONDS) {
        toastError($t('match.videos.too_long'));
        if (fileInput) fileInput.value = '';
        return;
      }
    } catch { /* worker valida via ffprobe */ }

    uploading = true;
    uploadPct = 0;
    try {
      const ticket = await matchVideos.createUpload(match.id, {
        size_bytes: file.size,
        content_type: file.type || 'video/mp4',
      });
      await matchVideos.uploadToR2(ticket.upload_url, file, (pct) => { uploadPct = pct; });
      await matchVideos.confirm(match.id, ticket.video_id);
      toastSuccess($t('match.videos.upload_sent'));
      await refresh();
    } catch (err) {
      if (err instanceof ApiError && err.message === 'VIDEO_LIMIT_REACHED') {
        toastError($t('match.videos.limit_reached'));
      } else {
        toastError($t('match.videos.upload_error'));
      }
    } finally {
      uploading = false;
      if (fileInput) fileInput.value = '';
    }
  }

  function askDelete(v: MatchVideoItem) {
    deleteTarget = v;
    confirmDeleteOpen = true;
  }

  async function doDelete() {
    if (!deleteTarget) return;
    deleting = true;
    try {
      await matchVideos.delete(deleteTarget.id);
      toastSuccess($t('match.videos.deleted'));
      await refresh();
    } catch {
      toastError($t('match.videos.delete_error'));
    } finally {
      deleting = false;
      deleteTarget = null;
    }
  }

  function canDelete(v: MatchVideoItem): boolean {
    if (!$isLoggedIn) return false;
    return v.uploader?.id === $currentPlayer?.id || isGroupAdmin || $isAdmin;
  }

  // IntersectionObserver: src lazy + autoplay mudo do item visível
  function observeVideo(el: HTMLVideoElement) {
    if (!observer) {
      observer = new IntersectionObserver(
        (entries) => {
          for (const entry of entries) {
            const video = entry.target as HTMLVideoElement;
            if (entry.isIntersecting) {
              if (!video.src && video.dataset.src) video.src = video.dataset.src;
              if (entry.intersectionRatio >= 0.6) {
                video.play().catch(() => {});
              }
            } else {
              video.pause();
            }
          }
        },
        { threshold: [0, 0.6] }
      );
    }
    observer.observe(el);
    return { destroy: () => observer?.unobserve(el) };
  }

  $effect(() => () => { observer?.disconnect(); });

  let readyVideos = $derived((data?.videos ?? []).filter(v => v.status === 'ready'));
  let inFlightVideos = $derived((data?.videos ?? []).filter(v => v.status !== 'ready'));
  let canUploadNow = $derived(!!data && data.can_upload && data.count < data.max_videos);
</script>

<svelte:head><title>{$t('match.videos.title')} — rachao.app</title></svelte:head>

<PageBackground>
  {#if loading}
    <div class="flex items-center justify-center min-h-screen">
      <div class="w-8 h-8 border-4 border-primary-500 border-t-transparent rounded-full animate-spin"></div>
    </div>
  {:else if match && data}
    <main class="relative z-10 max-w-lg mx-auto px-4 py-6">
      <MatchBannerCard {match} />

      <div class="flex items-center justify-between mt-4 mb-3">
        <h2 class="text-sm font-semibold text-white flex items-center gap-2">
          <Clapperboard size={16} class="text-primary-400" /> {$t('match.videos.title')}
          <span class="text-white/50 font-normal">({data.count}/{data.max_videos})</span>
        </h2>
        {#if canUploadNow}
          <button
            onclick={() => fileInput?.click()}
            disabled={uploading}
            class="btn btn-primary btn-sm gap-1.5 disabled:opacity-60">
            {#if uploading}
              <Loader2 size={14} class="animate-spin" /> {uploadPct}%
            {:else}
              <Plus size={14} /> {$t('match.videos.upload')}
            {/if}
          </button>
        {/if}
      </div>

      <input
        bind:this={fileInput}
        type="file"
        accept="video/mp4,video/quicktime,video/webm,video/*"
        capture="environment"
        class="hidden"
        onchange={onFileChosen} />

      {#if uploading}
        <div class="card p-3 mb-3">
          <p class="text-xs text-gray-600 dark:text-gray-300 mb-2">{$t('match.videos.uploading')}</p>
          <div class="h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
            <div class="h-full bg-primary-500 rounded-full transition-all" style="width: {uploadPct}%"></div>
          </div>
        </div>
      {/if}

      <!-- Vídeos do próprio usuário ainda em processamento / com falha -->
      {#each inFlightVideos as v (v.id)}
        <div class="card p-3 mb-3 flex items-center gap-3">
          {#if v.status === 'failed'}
            <span class="text-lg">⚠️</span>
            <p class="text-xs text-red-500 flex-1">{$t('match.videos.failed')}</p>
          {:else}
            <Loader2 size={16} class="animate-spin text-primary-500 shrink-0" />
            <p class="text-xs text-gray-600 dark:text-gray-300 flex-1">{$t('match.videos.processing')}</p>
          {/if}
          {#if canDelete(v)}
            <button onclick={() => askDelete(v)} class="btn btn-ghost btn-sm gap-1 text-red-500">
              <Trash2 size={14} /> {$t('match.videos.delete')}
            </button>
          {/if}
        </div>
      {/each}

      <!-- Feed vertical -->
      {#if readyVideos.length === 0 && inFlightVideos.length === 0}
        <div class="card p-8 text-center">
          <p class="text-3xl mb-2">🎬</p>
          <p class="text-sm text-gray-600 dark:text-gray-300">{$t('match.videos.empty')}</p>
          {#if canUploadNow}
            <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">{$t('match.videos.empty_cta')}</p>
          {/if}
        </div>
      {:else}
        <div class="snap-y snap-mandatory space-y-4">
          {#each readyVideos as v (v.id)}
            <div class="card overflow-hidden snap-start">
              <div class="relative bg-black aspect-[9/16] max-h-[70dvh] w-full">
                <!-- svelte-ignore a11y_media_has_caption -->
                <video
                  use:observeVideo
                  data-src={v.video_url}
                  poster={v.poster_url}
                  playsinline
                  muted
                  loop
                  controls
                  preload="none"
                  class="absolute inset-0 w-full h-full object-contain"></video>
              </div>
              <div class="px-3 py-2 flex items-center gap-2">
                {#if v.uploader}
                  <AvatarImage name={v.uploader.name} avatarUrl={v.uploader.avatar_url} size={24} />
                  <span class="text-xs text-gray-600 dark:text-gray-300 flex-1 truncate">
                    {playerDisplayName(v.uploader.name, v.uploader.nickname)}
                  </span>
                {:else}
                  <span class="flex-1"></span>
                {/if}
                {#if v.duration_seconds}
                  <span class="text-[11px] text-gray-400">{Math.round(v.duration_seconds)}s</span>
                {/if}
                {#if canDelete(v)}
                  <button onclick={() => askDelete(v)} class="btn btn-ghost btn-sm gap-1 text-red-500">
                    <Trash2 size={14} /> {$t('match.videos.delete')}
                  </button>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </main>
  {:else}
    <main class="relative z-10 max-w-lg mx-auto px-4 py-16 text-center">
      <p class="text-white/70">{$t('match.not_found_title')}</p>
    </main>
  {/if}
</PageBackground>

<ConfirmDialog
  bind:open={confirmDeleteOpen}
  message={$t('match.videos.confirm_delete')}
  confirmLabel={$t('match.videos.delete')}
  danger={true}
  onConfirm={doDelete} />
