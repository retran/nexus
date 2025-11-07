import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import path from "path";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    // Development server configuration for Traefik reverse proxy
    // Production: will use different HMR settings for actual domain
    host: true,
    port: 3000,
    allowedHosts: [
      "frontend",
      "nexus.local",
      ".nexus.local",
      "localhost",
    ],
    hmr: {
      clientPort: 80,
      protocol: "ws",
      host: "nexus.local",  // Dev: .local domain; Prod: actual domain
    },
    watch: {
      usePolling: true,
    },
  },
});
