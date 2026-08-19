import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  build: {
    target: 'esnext',
    manifest: true,
    rollupOptions: {
      output: {
        entryFileNames: 'js/remoteEntry-[hash].mjs',
        chunkFileNames: 'js/[name]-[hash].mjs',
        assetFileNames: (info) => {
          if (/\.(gif|jpe?g|png|svg|webp)$/i.test(info.name)) {
            return 'assets/[name][extname]'
          }
          return 'assets/[name]-[hash][extname]'
        },
      },
    },
  },
})
