import { writable } from 'svelte/store';
import { browser } from '$app/environment';

function createThemeStore() {
  const { subscribe, set } = writable<'light' | 'dark'>('dark');

  return {
    subscribe,
    init() {
      if (!browser) return;
      const saved = localStorage.getItem('theme') as 'light' | 'dark' | null;
      const theme: 'light' | 'dark' = saved ?? 'dark';
      set(theme);
      document.documentElement.classList.toggle('dark', theme === 'dark');
    },
    toggle() {
      const isDark = document.documentElement.classList.toggle('dark');
      const theme: 'light' | 'dark' = isDark ? 'dark' : 'light';
      set(theme);
      localStorage.setItem('theme', theme);
    },
  };
}

export const themeStore = createThemeStore();
