<script lang="ts">
  import { page } from '$app/stores';
  import { goto, afterNavigate } from '$app/navigation';
  import { matches as matchesApi, matchVideos, groups as groupsApi, ApiError } from '$lib/api';
  import type { MatchDetail, MatchVideosResponse, MatchVideoItem, VideoLiker } from '$lib/api';
  import { currentPlayer, isAdmin, isLoggedIn } from '$lib/stores/auth';
  import { toastSuccess, toastError } from '$lib/stores/toast';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import Modal from '$lib/components/Modal.svelte';
  import AvatarImage from '$lib/components/AvatarImage.svelte';
  import { ArrowLeft, Plus, Trash2, Loader2, Heart, Volume2, VolumeX, Share2, Eye, ChevronUp } from 'lucide-svelte';
  import { t } from '$lib/i18n';
  import { playerDisplayName, nativeShare } from '$lib/utils';
  import { tick } from 'svelte';

  const matchHash = $page.params.hash ?? '';

  const MAX_VIDEO_BYTES = 150 * 1024 * 1024;
  const MAX_IMAGE_BYTES = 25 * 1024 * 1024;
  const MAX_DURATION_SECONDS = 61; // margem sobre 60s; o worker (ffprobe) é a autoridade

  let match = $state<MatchDetail | null>(null);
  let data = $state<MatchVideosResponse | null>(null);
  let loading = $state(true);
  let isGroupAdmin = $state(false);

  // ── Som ────────────────────────────────────────────────────────────────────
  // Estratégia: tentar autoplay COM som (fluxo fluido; botões físicos de volume
  // só controlam mídia não-muda). Se o navegador bloquear, cai para mudo e o
  // PRIMEIRO toque/swipe no feed reativa o som — sem depender do ícone.
  const soundPref = typeof sessionStorage !== 'undefined' ? sessionStorage.getItem('feed_sound') : null;
  let wantSound = $state(soundPref !== 'off'); // escolha do usuário (persistida na sessão)
  let muted = $state(soundPref === 'off');     // estado efetivo dos <video>
  let autoMuted = $state(false);               // mudo por bloqueio do navegador, não por escolha
  let soundHint = $state(false);
  let lastAutoUnmuteAt = 0;
  let visibleVideo: HTMLVideoElement | null = null;

  function tryPlay(video: HTMLVideoElement) {
    video.muted = muted;
    video.play().catch(() => {
      if (!muted) {
        // Autoplay com som bloqueado — toca mudo e espera a primeira interação
        muted = true;
        autoMuted = true;
        soundHint = true;
        video.muted = true;
        video.play().catch(() => {});
      }
    });
  }

  function playVisible() {
    if (visibleVideo) tryPlay(visibleVideo);
  }

  function enableSound() {
    muted = false;
    autoMuted = false;
    soundHint = false;
    lastAutoUnmuteAt = Date.now();
    playVisible();
  }

  // Qualquer interação (toque, swipe, tecla) conta como gesto para o navegador:
  // se estamos mudos só por bloqueio, aproveita para religar o som.
  function onFirstInteraction() {
    if (autoMuted && wantSound) enableSound();
  }

  function toggleMute() {
    if (muted) {
      wantSound = true;
      enableSound();
    } else {
      muted = true;
      autoMuted = false;
      soundHint = false;
      wantSound = false;
    }
    try { sessionStorage.setItem('feed_sound', muted ? 'off' : 'on'); } catch { /* sessão privada */ }
  }

  let fileInput = $state<HTMLInputElement | null>(null);
  let uploading = $state(false);
  let uploadPct = $state(0);

  let confirmDeleteOpen = $state(false);
  let deleteTarget = $state<MatchVideoItem | null>(null);

  let likersOpen = $state(false);
  let likersLoading = $state(false);
  let likers = $state<VideoLiker[]>([]);
  let likePending = $state<Record<string, boolean>>({});

  // Voltar sem empilhar entrada no histórico: se chegamos aqui navegando
  // dentro do app, history.back() (senão o botão voltar do browser fica
  // quicando entre a partida e o feed); em deep-link, replaceState.
  let cameFromApp = false;
  afterNavigate(({ from }) => {
    if (from?.url) cameFromApp = true;
  });

  function goBack() {
    if (cameFromApp) {
      history.back();
    } else {
      goto(`/match/${matchHash}`, { replaceState: true });
    }
  }

  // Autoplay do vídeo visível (mudo por padrão) + src lazy
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
        await scrollToDeepLinkedItem();
      } catch {
        if (!cancelled) { match = null; data = null; }
      }
      if (!cancelled) loading = false;
    })();
    return () => { cancelled = true; };
  });

  // Deep link ?item=<id>: posiciona o feed no item compartilhado
  async function scrollToDeepLinkedItem() {
    const itemId = $page.url.searchParams.get('item');
    if (!itemId) return;
    await tick();
    document.getElementById(`feed-item-${itemId}`)?.scrollIntoView({ behavior: 'instant', block: 'start' });
  }

  async function shareFeed() {
    if (!match) return;
    const url = `${window.location.origin}/match/${matchHash}/feed`;
    await nativeShare({
      title: $t('match.videos.title'),
      text: $t('match.videos.share_text').replace('{group}', match.group_name),
      url,
    });
  }

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

    const isImage = file.type.startsWith('image/');
    if (file.size > (isImage ? MAX_IMAGE_BYTES : MAX_VIDEO_BYTES)) {
      toastError($t(isImage ? 'match.videos.too_large_image' : 'match.videos.too_large'));
      if (fileInput) fileInput.value = '';
      return;
    }
    if (!isImage) {
      try {
        const dur = await videoDuration(file);
        if (isFinite(dur) && dur > MAX_DURATION_SECONDS) {
          toastError($t('match.videos.too_long'));
          if (fileInput) fileInput.value = '';
          return;
        }
      } catch { /* worker valida via ffprobe */ }
    }

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
    try {
      await matchVideos.delete(deleteTarget.id);
      toastSuccess($t('match.videos.deleted'));
      await refresh();
    } catch {
      toastError($t('match.videos.delete_error'));
    } finally {
      deleteTarget = null;
    }
  }

  function canDelete(v: MatchVideoItem): boolean {
    if (!$isLoggedIn) return false;
    return v.uploader?.id === $currentPlayer?.id || isGroupAdmin || $isAdmin;
  }

  async function toggleLike(v: MatchVideoItem) {
    if (!$isLoggedIn) {
      toastError($t('match.videos.login_to_like'));
      return;
    }
    if (likePending[v.id] || !data) return;
    likePending = { ...likePending, [v.id]: true };
    const wasLiked = v.liked_by_me;
    // Otimista
    data = {
      ...data,
      videos: data.videos.map(item => item.id === v.id
        ? { ...item, liked_by_me: !wasLiked, like_count: item.like_count + (wasLiked ? -1 : 1) }
        : item),
    };
    try {
      const res = wasLiked ? await matchVideos.unlike(v.id) : await matchVideos.like(v.id);
      data = {
        ...data,
        videos: data.videos.map(item => item.id === v.id
          ? { ...item, liked_by_me: res.liked_by_me, like_count: res.like_count }
          : item),
      };
    } catch {
      // Reverte
      data = {
        ...data,
        videos: data.videos.map(item => item.id === v.id
          ? { ...item, liked_by_me: wasLiked, like_count: item.like_count + (wasLiked ? 1 : -1) }
          : item),
      };
      toastError($t('match.videos.like_error'));
    } finally {
      likePending = { ...likePending, [v.id]: false };
    }
  }

  async function openLikers(v: MatchVideoItem) {
    likersOpen = true;
    likersLoading = true;
    likers = [];
    try {
      const res = await matchVideos.listLikes(v.id);
      likers = res.likers;
    } catch {
      toastError($t('match.videos.like_error'));
    } finally {
      likersLoading = false;
    }
  }

  // ── Slide ativo: progresso (estilo TikTok), views e dica de swipe ─────────
  const IMAGE_DURATION_MS = 6000; // tempo de exibição de uma foto
  let activeId = $state<string | null>(null);
  let progress = $state(0);        // 0..1 do item ativo
  let completedOnce = $state(false);
  const viewed = new Set<string>(); // dedupe de views nesta visita
  let imageTimer: ReturnType<typeof setInterval> | null = null;
  let slideObserver: IntersectionObserver | null = null;

  function stopImageTimer() {
    if (imageTimer) { clearInterval(imageTimer); imageTimer = null; }
  }

  function startImageTimer() {
    stopImageTimer();
    imageTimer = setInterval(() => {
      progress = Math.min(1, progress + 50 / IMAGE_DURATION_MS);
      if (progress >= 1) { completedOnce = true; stopImageTimer(); }
    }, 50);
  }

  function setActive(id: string) {
    if (activeId === id) return;
    activeId = id;
    progress = 0;
    completedOnce = false;
    stopImageTimer();
    const item = (data?.videos ?? []).find(x => x.id === id);
    if (!item || item.status !== 'ready') return;
    if (!viewed.has(id)) {
      viewed.add(id);
      matchVideos.registerView(id).catch(() => {});
    }
    if (item.media_type === 'image') startImageTimer();
  }

  function observeSlide(el: HTMLElement, id: string) {
    if (!slideObserver) {
      slideObserver = new IntersectionObserver(
        (entries) => {
          for (const entry of entries) {
            if (entry.intersectionRatio >= 0.6) {
              const slideId = (entry.target as HTMLElement).dataset.itemId;
              if (slideId) setActive(slideId);
            }
          }
        },
        { threshold: [0.6] }
      );
    }
    el.dataset.itemId = id;
    slideObserver.observe(el);
    return { destroy: () => slideObserver?.unobserve(el) };
  }

  function onVideoTime(e: Event, id: string) {
    if (id !== activeId) return;
    const video = e.currentTarget as HTMLVideoElement;
    if (!video.duration || !isFinite(video.duration)) return;
    const p = video.currentTime / video.duration;
    // Loop reiniciou (currentTime voltou ao começo) → já assistiu inteiro
    if (p < progress - 0.5 || p >= 0.97) completedOnce = true;
    progress = p;
  }

  function isLastItem(id: string): boolean {
    const list = data?.videos ?? [];
    return list.length === 0 || list[list.length - 1].id === id;
  }

  $effect(() => () => { slideObserver?.disconnect(); stopImageTimer(); });

  // IntersectionObserver: src lazy + autoplay do slide visível, pausa dos demais
  function observeVideo(el: HTMLVideoElement) {
    if (!observer) {
      observer = new IntersectionObserver(
        (entries) => {
          for (const entry of entries) {
            const video = entry.target as HTMLVideoElement;
            if (entry.isIntersecting) {
              if (!video.src && video.dataset.src) video.src = video.dataset.src;
              if (entry.intersectionRatio >= 0.6) {
                visibleVideo = video;
                tryPlay(video);
              }
            } else {
              if (visibleVideo === video) visibleVideo = null;
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

  function togglePlay(e: Event) {
    // O toque que acabou de religar o som não deve também pausar o vídeo
    if (Date.now() - lastAutoUnmuteAt < 500) return;
    const video = e.currentTarget as HTMLVideoElement;
    if (video.paused) video.play().catch(() => {});
    else video.pause();
  }

  let feedVideos = $derived(data?.videos ?? []);
  let canUploadNow = $derived(!!data && data.can_upload && data.count < data.max_videos);
</script>

<svelte:head><title>{$t('match.videos.title')} — rachao.app</title></svelte:head>

{#if loading}
  <div class="fixed inset-0 z-50 bg-black flex items-center justify-center">
    <div class="w-8 h-8 border-4 border-primary-500 border-t-transparent rounded-full animate-spin"></div>
  </div>
{:else if match && data}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="fixed inset-0 z-50 bg-black" onpointerdown={onFirstInteraction} ontouchend={onFirstInteraction}>
    <!-- Feed vertical fullscreen com snap (swipe up/down) -->
    <div class="h-full overflow-y-scroll snap-y snap-mandatory overscroll-contain" style="scrollbar-width: none;">
      {#if feedVideos.length === 0}
        <section class="relative h-full w-full snap-start flex flex-col items-center justify-center text-center px-8">
          <p class="text-5xl mb-3">🎬</p>
          <p class="text-white/90 font-medium">{$t('match.videos.empty')}</p>
          {#if canUploadNow}
            <p class="text-white/50 text-sm mt-1">{$t('match.videos.empty_cta')}</p>
            <button
              onclick={() => fileInput?.click()}
              class="btn btn-primary mt-6 gap-1.5">
              <Plus size={16} /> {$t('match.videos.upload')}
            </button>
          {/if}
        </section>
      {/if}

      {#each feedVideos as v (v.id)}
        <section id="feed-item-{v.id}" use:observeSlide={v.id} class="relative h-full w-full snap-start snap-always">
          {#if v.status === 'ready' && v.media_type === 'image'}
            <img
              src={v.poster_url}
              alt=""
              loading="lazy"
              class="absolute inset-0 w-full h-full object-contain" />
          {:else if v.status === 'ready'}
            <!-- svelte-ignore a11y_media_has_caption -->
            <video
              use:observeVideo
              data-src={v.video_url}
              poster={v.poster_url}
              playsinline
              loop
              {muted}
              preload="none"
              onclick={togglePlay}
              ontimeupdate={(e) => onVideoTime(e, v.id)}
              class="absolute inset-0 w-full h-full object-contain"></video>
          {:else}
            <div class="absolute inset-0 flex flex-col items-center justify-center text-center px-8">
              {#if v.status === 'failed'}
                <p class="text-4xl mb-3">⚠️</p>
                <p class="text-red-400 text-sm">{$t('match.videos.failed')}</p>
              {:else}
                <Loader2 size={32} class="animate-spin text-primary-400 mb-3" />
                <p class="text-white/70 text-sm">{$t('match.videos.processing')}</p>
              {/if}
            </div>
          {/if}

          <!-- Gradiente inferior + uploader -->
          <div class="absolute inset-x-0 bottom-0 pb-6 pt-16 px-4 bg-gradient-to-t from-black/70 to-transparent pointer-events-none">
            {#if v.uploader}
              <div class="flex items-center gap-2">
                <AvatarImage name={v.uploader.name} avatarUrl={v.uploader.avatar_url} size={32} />
                <span class="text-white text-sm font-medium drop-shadow">
                  {playerDisplayName(v.uploader.name, v.uploader.nickname)}
                </span>
                {#if v.media_type === 'image'}
                  <span class="text-white/60 text-xs">· 📷</span>
                {:else if v.duration_seconds}
                  <span class="text-white/60 text-xs">· {Math.round(v.duration_seconds)}s</span>
                {/if}
              </div>
            {/if}
          </div>

          <!-- Rail de ações (direita) -->
          <div class="absolute right-3 bottom-24 flex flex-col items-center gap-5">
            {#if v.status === 'ready'}
              <div class="flex flex-col items-center pointer-events-none" aria-hidden="true">
                <div class="p-2"><Eye size={26} class="text-white drop-shadow" /></div>
                <span class="text-white text-xs font-semibold drop-shadow -mt-1">{v.view_count}</span>
              </div>
            {/if}
            <div class="flex flex-col items-center">
              <button
                onclick={() => toggleLike(v)}
                disabled={likePending[v.id]}
                class="p-2 rounded-full bg-black/30 backdrop-blur-sm active:scale-90 transition-transform disabled:opacity-60"
                aria-label={v.liked_by_me ? $t('match.videos.unlike') : $t('match.videos.like')}>
                <Heart size={28} class={v.liked_by_me ? 'text-red-500 fill-red-500' : 'text-white'} />
              </button>
              <button
                onclick={() => openLikers(v)}
                class="text-white text-xs font-semibold mt-1 drop-shadow"
                aria-label={$t('match.videos.likes_title')}>
                {v.like_count}
              </button>
            </div>
            {#if canDelete(v)}
              <button
                onclick={() => askDelete(v)}
                class="p-2 rounded-full bg-black/30 backdrop-blur-sm text-white active:scale-90 transition-transform"
                aria-label={$t('match.videos.delete')}>
                <Trash2 size={22} />
              </button>
            {/if}
          </div>

          {#if activeId === v.id && v.status === 'ready'}
            <!-- Dica de swipe após completar o item (se houver próximo) -->
            {#if completedOnce && !isLastItem(v.id)}
              <div class="absolute bottom-10 left-1/2 -translate-x-1/2 flex flex-col items-center text-white/90 pointer-events-none animate-bounce drop-shadow">
                <ChevronUp size={24} />
                <span class="text-xs font-medium">{$t('match.videos.swipe_next')}</span>
              </div>
            {/if}
            <!-- Linha de progresso (vídeo acompanha a reprodução; foto tem timer) -->
            <div class="absolute bottom-0 inset-x-0 h-1 bg-white/20 z-10">
              <div
                class="h-full bg-white/90 transition-[width] duration-200 ease-linear"
                style="width: {Math.round(progress * 1000) / 10}%"></div>
            </div>
          {/if}
        </section>
      {/each}
    </div>

    <!-- Barra superior -->
    <div class="absolute inset-x-0 top-0 pt-3 pb-8 px-3 bg-gradient-to-b from-black/70 to-transparent flex items-center gap-3 pointer-events-none">
      <button
        onclick={goBack}
        class="p-2 rounded-full bg-black/30 backdrop-blur-sm text-white pointer-events-auto"
        aria-label={$t('match.videos.back')}>
        <ArrowLeft size={20} />
      </button>
      <div class="flex-1 min-w-0">
        <p class="text-white text-sm font-semibold truncate drop-shadow">🎬 {$t('match.videos.title')}</p>
        <p class="text-white/60 text-xs">{data.count}/{data.max_videos}</p>
      </div>
      <button
        onclick={shareFeed}
        class="p-2 rounded-full bg-black/30 backdrop-blur-sm text-white pointer-events-auto"
        aria-label={$t('match.videos.share')}>
        <Share2 size={20} />
      </button>
      <button
        onclick={toggleMute}
        class="p-2 rounded-full bg-black/30 backdrop-blur-sm text-white pointer-events-auto"
        aria-label={muted ? $t('match.videos.unmute') : $t('match.videos.mute')}>
        {#if muted}<VolumeX size={20} />{:else}<Volume2 size={20} />{/if}
      </button>
      {#if canUploadNow}
        <button
          onclick={() => fileInput?.click()}
          disabled={uploading}
          class="p-2 rounded-full bg-primary-600 text-white pointer-events-auto disabled:opacity-60"
          aria-label={$t('match.videos.upload')}>
          {#if uploading}<Loader2 size={20} class="animate-spin" />{:else}<Plus size={20} />{/if}
        </button>
      {/if}
    </div>

    {#if uploading}
      <div class="absolute inset-x-0 top-0 h-1 bg-white/20">
        <div class="h-full bg-primary-500 transition-all" style="width: {uploadPct}%"></div>
      </div>
    {/if}

    {#if soundHint}
      <button
        onclick={enableSound}
        class="absolute bottom-28 left-1/2 -translate-x-1/2 px-4 py-2 rounded-full bg-black/60 backdrop-blur-sm text-white text-xs font-medium flex items-center gap-1.5">
        <VolumeX size={14} /> {$t('match.videos.tap_for_sound')}
      </button>
    {/if}

    <input
      bind:this={fileInput}
      type="file"
      accept="video/mp4,video/quicktime,video/webm,video/*,image/jpeg,image/png,image/webp"
      class="hidden"
      onchange={onFileChosen} />

    <ConfirmDialog
      bind:open={confirmDeleteOpen}
      message={$t('match.videos.confirm_delete')}
      confirmLabel={$t('match.videos.delete')}
      danger={true}
      onConfirm={doDelete} />

    <Modal bind:open={likersOpen} title={$t('match.videos.likes_title')}>
      {#if likersLoading}
        <div class="flex justify-center py-6">
          <div class="w-6 h-6 border-4 border-primary-500 border-t-transparent rounded-full animate-spin"></div>
        </div>
      {:else if likers.length === 0}
        <p class="text-sm text-gray-500 dark:text-gray-400 text-center py-4">{$t('match.videos.no_likes')}</p>
      {:else}
        <ul class="space-y-3">
          {#each likers as liker (liker.id)}
            <li class="flex items-center gap-3">
              <AvatarImage name={liker.name} avatarUrl={liker.avatar_url} size={32} />
              <span class="text-sm text-gray-800 dark:text-gray-100">
                {playerDisplayName(liker.name, liker.nickname)}
              </span>
            </li>
          {/each}
        </ul>
      {/if}
    </Modal>
  </div>
{:else}
  <div class="fixed inset-0 z-50 bg-black flex flex-col items-center justify-center px-8 text-center">
    <p class="text-white/70">{$t('match.not_found_title')}</p>
    <button onclick={goBack} class="btn btn-primary mt-4">
      {$t('match.videos.back')}
    </button>
  </div>
{/if}
