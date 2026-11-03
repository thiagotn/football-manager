/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{html,js,svelte,ts}'],
  darkMode: 'class',
  theme: {
    extend: {
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
        cond: ['"Barlow Condensed"', 'Inter', 'sans-serif'],
      },
      colors: {
        // Paletas do card-escudo (handoff design_handoff_player_crest_card).
        // `rachao` é o verde institucional do escudo — diferente de `primary`.
        rachao: {
          200: '#8dd7a6',
          300: '#52c179',
          400: '#2aab58',
          500: '#1f9a4d',
          600: '#168641',
          700: '#106834',
          800: '#0d5028',
          900: '#07331a',
        },
        gold: {
          300: '#f7e2ad',
          400: '#f5d585',
          500: '#e0b95c',
          shadow: '#8a6c2a',
          edge: '#c9a34d',
        },
        primary: {
          50:  '#f0fdf4',
          100: '#dcfce7',
          200: '#bbf7d0',
          300: '#86efac',
          400: '#4ade80',
          500: '#22c55e',
          600: '#16a34a',
          700: '#15803d',
          800: '#166534',
          900: '#14532d',
        },
      },
    },
  },
  plugins: [],
};
