import react from '@vitejs/plugin-react'
import { defineConfig } from 'vitest/config'

const backend = {
  target: 'https://localhost:8443',
  changeOrigin: true,
  secure: false, // the backend's TLS cert is a locally-generated self-signed cert
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    // Lets `npm run dev` talk to the real Go backend for API/WebSocket
    // calls while Vite serves the React app with HMR. The backend's own
    // CSP/CORS posture is same-origin-only, so this proxy — not browser
    // CORS — is what makes local dev work at all.
    proxy: {
      '/checkLogin': backend,
      '/login': backend,
      '/register': backend,
      '/logout': backend,
      '/getAllPosts': backend,
      '/getPostsByAuthor': backend,
      '/getPost': backend,
      '/getCategories': backend,
      '/getPostCategories': backend,
      '/getPostsByCategory': backend,
      '/searchPosts': backend,
      '/addPost': backend,
      '/editPost': backend,
      '/deletePost': backend,
      '/addcomment': backend,
      '/editComment': backend,
      '/deleteComment': backend,
      '/comments': backend,
      '/ws': { ...backend, ws: true },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/setupTests.ts'],
  },
})
