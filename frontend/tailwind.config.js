/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      fontFamily: {
        sans: [
          '"Segoe UI Variable"',
          '"Segoe UI"',
          '"Microsoft YaHei UI"',
          '"PingFang SC"',
          '"Noto Sans CJK SC"',
          'system-ui',
          '-apple-system',
          'sans-serif',
        ],
        mono: [
          '"SF Mono"',
          '"JetBrains Mono"',
          '"Cascadia Code"',
          'ui-monospace',
          'Menlo',
          'Consolas',
          'monospace',
        ],
      },
      animation: {
        'fade-in': 'fadeIn 0.18s ease-out',
        'dock-in': 'dockIn 0.28s cubic-bezier(0.16, 1, 0.3, 1)',
        'drawer-in': 'drawerIn 0.26s cubic-bezier(0.16, 1, 0.3, 1)',
        'modal-in': 'modalIn 0.22s cubic-bezier(0.16, 1, 0.3, 1)',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        dockIn: {
          '0%': { opacity: '0', transform: 'translate(-50%, 18px) scale(0.96)' },
          '100%': { opacity: '1', transform: 'translate(-50%, 0) scale(1)' },
        },
        drawerIn: {
          '0%': { opacity: '0', transform: 'translateX(20px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' },
        },
        modalIn: {
          '0%': { opacity: '0', transform: 'scale(0.97) translateY(6px)' },
          '100%': { opacity: '1', transform: 'scale(1) translateY(0)' },
        },
      },
    },
  },
  plugins: [],
};
