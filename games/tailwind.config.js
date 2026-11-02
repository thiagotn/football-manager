/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{html,js,svelte,ts}'],
  theme: {
    extend: {
      fontFamily: {
        sans: ['Nunito', 'system-ui', '-apple-system', 'Segoe UI', 'sans-serif'],
        display: ['"Lilita One"', '"Arial Rounded MT Bold"', 'system-ui', 'sans-serif'],
      },
      colors: {
        // Paleta do Enche Aí! — lanchonete à noite (ver docs/design/enche-ai/prototipo.html)
        ink: { DEFAULT: '#0f1622', 2: '#172233', 3: '#22314a' },
        wood: { DEFAULT: '#3b2519', dark: '#1f130c' },
        tile: { DEFAULT: '#183a69', light: '#2f68ad' },
        mustard: { DEFAULT: '#ffc93c', ink: '#3a2a00' },
        ketchup: '#e8473b',
        paper: { DEFAULT: '#fbf4e4', ink: '#3b2a20', muted: '#8a7563' },
      },
    },
  },
  plugins: [],
};
