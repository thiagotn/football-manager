<script lang="ts">
  import '../app.css';
  import '@fontsource/lilita-one/400.css';
  import '@fontsource/nunito/400.css';
  import '@fontsource/nunito/700.css';
  import '@fontsource/nunito/800.css';
  import { onMount, type Snippet } from 'svelte';
  import { initLocale } from '$lib/i18n';

  let { children }: { children: Snippet } = $props();

  onMount(initLocale);
</script>

{@render children()}
