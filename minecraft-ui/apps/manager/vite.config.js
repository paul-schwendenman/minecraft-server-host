import { defineConfig } from 'vite'
import tailwindcss from '@tailwindcss/vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { svelteTesting } from '@testing-library/svelte/vite'
import { VitePWA } from 'vite-plugin-pwa'
import { mockServer } from './src/plugins/mock-server'

// https://vite.dev/config/
export default defineConfig(({ command, mode }) => {
  const isDev = command === 'serve' || mode === 'development'

  return {
    plugins: [
      tailwindcss(),
      ...(isDev ? [mockServer()] : []),
      svelte(),
      svelteTesting(),
      VitePWA({
        registerType: 'autoUpdate',
        includeAssets: ['favicon.png', 'robots.txt'],
        manifest: {
          name: 'Minecraft Server Manager',
          short_name: 'MinecraftServerManager',
          description: 'UI for controlling the launch of a Minecraft server',
          display: 'standalone',
          orientation: 'portrait-primary',
          start_url: '/',
          background_color: '#18181b',
          theme_color: '#18181b',
          icons: [
            {
              src: '/favicon.png',
              sizes: '32x32',
              type: 'image/png',
            },
            {
              src: '/icons/icon-96.png',
              sizes: '96x96',
              type: 'image/png',
              purpose: 'maskable any',
            },
            {
              src: '/icons/icon-96.svg',
              sizes: '96x96',
              type: 'image/svg+xml',
              purpose: 'maskable any',
            },
            {
              src: '/icons/icon-192.png',
              sizes: '192x192',
              type: 'image/png',
              purpose: 'maskable any',
            },
            {
              src: '/icons/icon-192.svg',
              sizes: '192x192',
              type: 'image/svg+xml',
              purpose: 'maskable any',
            },
            {
              src: '/icons/icon-512.png',
              sizes: '512x512',
              type: 'image/png',
              purpose: 'maskable any',
            },
            {
              src: '/icons/icon-512.svg',
              sizes: '512x512',
              type: 'image/svg+xml',
              purpose: 'maskable any',
            },
          ],
        },
      })
    ],
    test: {
      globals: true,
      environment: 'jsdom',
      setupFiles: './src/setupTests.js',
      include: ['{src,test}/**/*.{test,spec}.{js,ts}'],
      coverage: {
        reporter: ['text', 'html'],
      },
    },
  }
})
