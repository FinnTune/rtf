import react from '@vitejs/plugin-react'
import type { ProxyOptions } from 'vite'
import { defineConfig } from 'vitest/config'

const backend: ProxyOptions = {
  target: 'https://localhost:8443',
  changeOrigin: true,
  secure: false, // the backend's TLS cert is a locally-generated self-signed cert
  configure: (proxy) => {
    // The backend's CSRF check and WebSocket-upgrade check are both a
    // strict Origin allowlist (websocket/handlers.go's checkOrigin, default
    // "https://localhost:8443"). The browser's real Origin here is the Vite
    // dev server (http://localhost:5173), which never matches — local dev
    // only works if this proxy presents the Origin the backend actually
    // expects. http-proxy fires these as two separate events: proxyReq for
    // regular HTTP requests, proxyReqWs for the WebSocket upgrade request
    // specifically — missing the second one leaves /ws rejected with 403
    // even once regular API calls work.
    proxy.on('proxyReq', (proxyReq) => {
      proxyReq.setHeader('Origin', 'https://localhost:8443')
    })
    proxy.on('proxyReqWs', (proxyReq) => {
      proxyReq.setHeader('Origin', 'https://localhost:8443')
    })
  },
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
