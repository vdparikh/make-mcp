import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ command }) => {
  const proxyTarget = process.env.DEV_API_PROXY_TARGET?.trim()
  if (command === 'serve' && !proxyTarget) {
    throw new Error(
      'DEV_API_PROXY_TARGET is not set. Source env.example.sh (e.g. set -a && source ../env.example.sh && set +a) or export DEV_API_PROXY_TARGET to match config/config.yaml server address.'
    )
  }

  return {
    plugins: [react()],
    server: {
      port: 3000,
      ...(command === 'serve' && proxyTarget
        ? {
            proxy: {
              '/api': {
                target: proxyTarget,
                changeOrigin: true,
              },
            },
          }
        : {}),
    },
  }
})
