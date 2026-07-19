/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ['Inter', 'sans-serif'],
        sora: ['Sora', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
      colors: {
        canvas: '#F7F8FA',
        surface: '#FFFFFF',
        'ink-primary': '#101828',
        'ink-muted': '#667085',
        'accent-primary': '#0F9D8C',
        'accent-warm': '#F59E0B',
        'accent-danger': '#E5484D',
      },
      boxShadow: {
        'soft': '0 1px 2px rgba(16,24,40,0.04), 0 4px 12px rgba(16,24,40,0.04)',
      },
      ringWidth: {
        '3': '3px',
      }
    },
  },
  plugins: [],
}
