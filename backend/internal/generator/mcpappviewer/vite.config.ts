import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { viteSingleFile } from "vite-plugin-singlefile";
import path from "node:path";

export default defineConfig({
  plugins: [react(), viteSingleFile()],
  build: {
    outDir: path.resolve(__dirname, "../embed"),
    emptyOutDir: false,
    rollupOptions: {
      input: path.resolve(__dirname, "index.html"),
    },
  },
});
