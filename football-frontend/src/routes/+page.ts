import { redirect } from '@sveltejs/kit';

export const ssr = false;

export function load() {
  if (typeof window !== 'undefined') {
    const token = localStorage.getItem('token');
    if (!token) {
      redirect(302, '/lp');
    }
  }
}
