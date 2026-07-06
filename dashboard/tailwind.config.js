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
      },
      colors: {
        brand: {
          50: '#f4f6f8',
          100: '#e4e8ec',
          200: '#cbd4dc',
          300: '#a3b4c1',
          400: '#738da0',
          500: '#567184',
          600: '#465b6a',
          700: '#3a4a57',
          800: '#33404b',
          900: '#2d3640',
          950: '#1d242a',
        },
        emerald: {
          50: '#ecfdf5',
          500: '#10b981',
          600: '#059669',
        },
        amber: {
          50: '#fffbeb',
          500: '#f59e0b',
          600: '#d97706',
        },
        rose: {
          50: '#fff1f2',
          500: '#f43f5e',
          600: '#e11d48',
        }
      }
    },
  },
  plugins: [],
}
