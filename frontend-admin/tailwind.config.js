/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js}'],
  theme: {
    extend: {
      colors: {
        bg: '#07090C',
        panel: '#10161C',
        panel2: '#161E26',
        line: '#24303A',
        ink: '#E7EEF4',
        mute: '#8B9AA8',
        amber: '#E8B84A',
        cyan: '#2EE6D6',
        danger: '#FF6B4A',
        ok: '#7CFFB2',
      },
      fontFamily: {
        display: ['Oxanium', 'sans-serif'],
        sans: ['"Noto Sans SC"', 'sans-serif'],
        mono: ['"IBM Plex Mono"', 'monospace'],
      },
    },
  },
  plugins: [],
}
