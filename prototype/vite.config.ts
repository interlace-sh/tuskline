import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { viteSingleFile } from "vite-plugin-singlefile";

export default defineConfig(({ mode }) => {
  const single = mode === "singlefile";
  return {
    base: "./",
    plugins: single ? [react(), viteSingleFile()] : [react()],
    build: {
      outDir: single ? "dist-single" : "dist",
      emptyOutDir: true,
      assetsInlineLimit: single ? 100000000 : 4096,
      chunkSizeWarningLimit: 3000,
    },
  };
});
