/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          50:  '#f0ecfb',
          100: '#ddd5f6',
          200: '#bbabed',
          300: '#9981e4',
          400: '#7757db',
          500: '#513CC8',
          600: '#4230a0',
          700: '#322478',
          800: '#211850',
          900: '#110c28',
        },
      },
    },
  },
  plugins: [],
}
