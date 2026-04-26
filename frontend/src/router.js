import { createRouter, createWebHistory } from "vue-router";
import HomeView from "./views/HomeView.vue";
import ChatView from "./views/ChatView.vue";

// Lightweight two-route SPA: the sidebar is always on screen, so the router
// only controls what lives in the main content panel. Direct chat URLs are
// fully navigable (refresh, back/forward, shareable).
const routes = [
  { path: "/", name: "home", component: HomeView },
  { path: "/chat/:id", name: "chat", component: ChatView, props: true },
  { path: "/:pathMatch(.*)*", redirect: "/" },
];

export default createRouter({
  history: createWebHistory(),
  routes,
});
