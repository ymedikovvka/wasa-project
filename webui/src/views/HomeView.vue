<script setup>
// Shown on "/" when no conversation is open. Kept deliberately lightweight so
// the app shell renders a useful splash panel instead of a blank area while
// the sidebar still lets the user pick or start a chat.
import { useAuth } from "../composables/useAuth";

const { currentUser } = useAuth();
</script>

<template>
  <section class="empty-state">
    <div class="empty-state-inner">
      <div class="brand-mark xl">W</div>
      <h2>Welcome{{ currentUser?.name ? `, ${currentUser.name}` : "" }}</h2>
      <p>Pick a chat on the left, search for someone in People, or start a new group.</p>
    </div>
  </section>
</template>
