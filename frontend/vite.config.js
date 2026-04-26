import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      "/session": "http://localhost:3000",
      "/users": "http://localhost:3000",
      "/conversations": "http://localhost:3000",
      "/messages": "http://localhost:3000",
      "/groups": "http://localhost:3000",
      "/liveness": "http://localhost:3000",
    },
  },
});
