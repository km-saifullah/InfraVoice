/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,jsx}'],
  theme: {
    extend: {
      colors: {
        ink: {
          950: '#0F1216',
          900: '#171B21',
          850: '#1C212A',
          800: '#232832',
          700: '#2E3440',
        },
        mist: {
          500: '#6B7486',
          400: '#8A93A3',
          200: '#C3C9D1',
          100: '#E7E9ED',
        },
        signal: {
          DEFAULT: '#22D3C7',
          600: '#1AA89F',
          dim: '#163B38',
        },
        amber: {
          DEFAULT: '#F2A93B',
          dim: '#3A2E15',
        },
        danger: {
          DEFAULT: '#E5484D',
          dim: '#3A1A1C',
        },
        success: {
          DEFAULT: '#34D399',
          dim: '#123528',
        },
      },
      fontFamily: {
        display: ['"Space Grotesk"', 'sans-serif'],
        sans: ['Inter', 'sans-serif'],
        mono: ['"IBM Plex Mono"', 'monospace'],
      },
    },
  },
  plugins: [],
}
