import { writeFileSync } from "node:fs";
import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import wails from "@wailsio/runtime/plugins/vite";

// main.go embeds frontend/dist; keep the committed placeholder after Vite
// empties the folder, so a build leaves the working tree clean.
const keepDist = {
  name: "keep-dist-placeholder",
  closeBundle() {
    writeFileSync("dist/.gitkeep", "");
  },
};

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
    // DEVBRIDGE=http://127.0.0.1:9300: the dev screens page reaches
    // overlay/scripts/devbridge.mjs at /devbridge (the page's CSP allows
    // only its own origin)
    proxy: process.env.DEVBRIDGE
      ? { "/devbridge": { target: process.env.DEVBRIDGE, rewrite: (p) => p.replace(/^\/devbridge/, "") } }
      : undefined,
  },
  plugins: [svelte(), wails("./bindings"), keepDist],
});
