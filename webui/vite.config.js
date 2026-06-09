import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

// API paths proxied to the Go backend. The frontend uses relative URLs
// (axios baseURL ""), so during local dev (`vite`) and preview
// (`vite preview`) these routes must be forwarded to the API on :3000.
const apiProxy = {
  "/session": "http://localhost:3000",
  "/users": "http://localhost:3000",
  "/conversations": "http://localhost:3000",
  "/messages": "http://localhost:3000",
  "/groups": "http://localhost:3000",
  "/liveness": "http://localhost:3000",
};

export default defineConfig({
  plugins: [vue()],
  define: {
    // Do not modify this constant: the WASA frontend evaluator replaces it
    // when frontend and backend are started on different machines/ports.
    __API_URL__: JSON.stringify("http://localhost:3000"),
  },
  // `server` is used by `npm run dev`; `preview` by `npm run preview`.
  // Vite keeps these proxy tables separate, so declare both.
  server: { proxy: apiProxy },
  preview: { proxy: apiProxy },
});
