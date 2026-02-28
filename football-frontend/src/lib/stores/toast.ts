import { writable } from 'svelte/store';

type Toast = { id: number; type: 'success' | 'error' | 'info'; message: string };
const { subscribe, update } = writable<Toast[]>([]);
let next = 0;

export const toasts = { subscribe };

export function toast(type: Toast['type'], message: string, duration = 4000) {
  const id = ++next;
  update(ts => [...ts, { id, type, message }]);
  setTimeout(() => update(ts => ts.filter(t => t.id !== id)), duration);
}

export const toastSuccess = (msg: string) => toast('success', msg);
export const toastError = (msg: string) => toast('error', msg);
export const toastInfo = (msg: string) => toast('info', msg);
