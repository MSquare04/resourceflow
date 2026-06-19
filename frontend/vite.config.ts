import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  root: ".",
  plugins: [react()],
  build: {
    rollupOptions: {
      input: "index.html",
    },
  },
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: "http://localhost:18080",
        changeOrigin: true,
      },
    },
  },
});
