<script lang="ts">
  import { goto } from '$app/navigation';
  import { authStore, isChatEnabled } from '$lib/stores/auth';
  import { toastError } from '$lib/stores/toast';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import { MessageCircle, Send, RotateCcw } from 'lucide-svelte';
  import { t } from '$lib/i18n';

  const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8000/api/v1';

  type Message = { role: 'user' | 'assistant'; content: string; streaming?: boolean };

  let messages = $state<Message[]>([]);
  let input = $state('');
  let streaming = $state(false);
  let messagesEl = $state<HTMLElement | null>(null);

  // Guard: redirect if chat not enabled (once auth finishes loading)
  $effect(() => {
    if ($authStore.loading) return;
    if (!$isChatEnabled) {
      goto('/dashboard', { replaceState: true });
    }
  });

  function scrollToBottom() {
    if (messagesEl) {
      messagesEl.scrollTop = messagesEl.scrollHeight;
    }
  }

  async function send() {
    const text = input.trim();
    if (!text || streaming) return;

    input = '';
    messages = [...messages, { role: 'user', content: text }];
    messages = [...messages, { role: 'assistant', content: '', streaming: true }];
    streaming = true;

    setTimeout(scrollToBottom, 0);

    const token = typeof localStorage !== 'undefined' ? localStorage.getItem('token') : null;

    try {
      const res = await fetch(`${API_BASE}/chat`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({
          messages: messages
            .filter(m => !m.streaming)
            .slice(0, -1) // exclude the empty assistant placeholder
            .concat({ role: 'user', content: text })
            .map(m => ({ role: m.role, content: m.content })),
        }),
      });

      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        if (res.status === 429) {
          messages = messages.slice(0, -1);
          toastError($t('chat.error_rate_limit'));
        } else {
          messages = messages.slice(0, -1).concat({ role: 'assistant', content: body.detail ?? $t('chat.error_generic') });
        }
        streaming = false;
        return;
      }

      const reader = res.body!.getReader();
      const decoder = new TextDecoder();
      let buffer = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() ?? '';

        for (const line of lines) {
          if (!line.startsWith('data: ')) continue;
          const payload = line.slice(6).trim();
          if (payload === '[DONE]') break;

          try {
            const data = JSON.parse(payload);
            if (data.text) {
              messages = messages.map((m, i) =>
                i === messages.length - 1
                  ? { ...m, content: m.content + data.text }
                  : m
              );
              scrollToBottom();
            } else if (data.error) {
              messages = messages.map((m, i) =>
                i === messages.length - 1
                  ? { ...m, content: data.error, streaming: false }
                  : m
              );
            }
          } catch {
            // ignore parse errors
          }
        }
      }
    } catch {
      messages = messages.slice(0, -1).concat({ role: 'assistant', content: $t('chat.error_generic') });
    }

    messages = messages.map((m, i) =>
      i === messages.length - 1 ? { ...m, streaming: false } : m
    );
    streaming = false;
    setTimeout(scrollToBottom, 0);
  }

  function newConversation() {
    messages = [];
    input = '';
    streaming = false;
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      send();
    }
  }

  function renderMarkdown(text: string): string {
    return text
      .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
      .replace(/\*\*(.+?)\*\*/gs, '<strong>$1</strong>')
      .replace(/\*(.+?)\*/gs, '<em>$1</em>')
      .replace(/\n\n/g, '</p><p class="mt-2">')
      .replace(/\n/g, '<br>')
      .replace(/^/, '<p>').replace(/$/, '</p>');
  }
</script>

<svelte:head><title>{$t('chat.title')} | rachao.app</title></svelte:head>

<PageBackground>
  <main class="relative z-10 max-w-3xl mx-auto px-4 py-8 flex flex-col h-[calc(100vh-4rem)]">

    <!-- Header -->
    <div class="flex items-center justify-between mb-4 shrink-0 bg-black/40 backdrop-blur-sm rounded-2xl px-4 py-3">
      <div>
        <h1 class="text-2xl font-bold text-white flex items-center gap-2">
          <MessageCircle size={24} class="text-primary-400" /> {$t('chat.title')}
        </h1>
        <p class="text-sm text-white/60 mt-0.5">{$t('chat.subtitle')}</p>
      </div>
      {#if messages.length > 0}
        <button
          onclick={newConversation}
          class="btn-sm btn-ghost text-white/70 hover:text-white flex items-center gap-1.5"
        >
          <RotateCcw size={14} /> {$t('chat.new_conversation')}
        </button>
      {/if}
    </div>

    <!-- Messages area -->
    <div
      bind:this={messagesEl}
      class="flex-1 overflow-y-auto space-y-4 py-4 px-3 my-2 bg-black/40 backdrop-blur-sm rounded-2xl"
    >
      {#if messages.length === 0}
        <div class="flex flex-col items-center justify-center h-full text-center gap-3 text-white/40 select-none">
          <MessageCircle size={48} class="text-white/20" />
          <p class="text-sm">{$t('chat.subtitle')}</p>
        </div>
      {/if}

      {#each messages as msg}
        <div class="flex {msg.role === 'user' ? 'justify-end' : 'justify-start'}">
          <div class="max-w-[85%] {msg.role === 'user'
            ? 'bg-primary-600 text-white rounded-2xl rounded-br-sm'
            : 'bg-white/20 text-white rounded-2xl rounded-bl-sm'} px-4 py-2.5 text-sm leading-relaxed">
            {#if msg.streaming && !msg.content}
              <span class="inline-flex gap-1 items-center text-white/50">
                <span class="animate-bounce [animation-delay:0ms]">·</span>
                <span class="animate-bounce [animation-delay:150ms]">·</span>
                <span class="animate-bounce [animation-delay:300ms]">·</span>
              </span>
            {:else if msg.role === 'assistant'}
              {@html renderMarkdown(msg.content)}
            {:else}
              {msg.content}
            {/if}
          </div>
        </div>
      {/each}
    </div>

    <!-- Input -->
    <div class="shrink-0 pt-1">
      <div class="flex gap-2 items-end bg-black/50 backdrop-blur-sm rounded-2xl p-2">
        <textarea
          bind:value={input}
          onkeydown={onKeydown}
          disabled={streaming}
          placeholder={$t('chat.placeholder')}
          rows={1}
          class="flex-1 bg-transparent text-white placeholder-white/40 text-sm resize-none outline-none px-2 py-1.5 max-h-32 disabled:opacity-50"
          style="field-sizing: content;"
        ></textarea>
        <button
          onclick={send}
          disabled={streaming || !input.trim()}
          class="btn-primary btn-sm shrink-0 flex items-center gap-1.5 disabled:opacity-40"
        >
          <Send size={14} /> {$t('chat.send')}
        </button>
      </div>
    </div>

  </main>
</PageBackground>
