import { defineConfig } from 'vite'
import tailwindcss from '@tailwindcss/vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { mockServer } from './plugins/mock-server'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    tailwindcss(),
    mockServer(),
    svelte()
  ],
})
