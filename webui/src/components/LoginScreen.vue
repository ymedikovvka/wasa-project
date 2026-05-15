<script setup>
import { computed, ref } from "vue";

const props = defineProps({
  // Render a "signing in…" state while the parent is awaiting the login call.
  loading: { type: Boolean, default: false },
  serverError: { type: String, default: "" },
});
const emit = defineEmits(["login", "clear-error"]);
const username = ref("");
const error = ref("");
const displayError = computed(() => error.value || props.serverError);

function clearError() {
  error.value = "";
  emit("clear-error");
}

function submit() {
  // Keep the client-side check in lock-step with the backend rule
  // (service/database/database.go :: validUserName) and the OpenAPI spec:
  // trimmed length between 3 and 16 characters, no character-class restriction.
  const value = username.value.trim();
  if (value.length < 3 || value.length > 16) {
    error.value = "Username must be 3 to 16 characters.";
    return;
  }
  error.value = "";
  emit("login", value);
}
</script>

<template>
  <section class="entry-screen">
    <form class="entry-card" @submit.prevent="submit">
      <div class="entry-brand">
        <div class="brand-mark">W</div>
        <div>
          <p class="eyebrow">WASAText</p>
          <h1>Sign in</h1>
        </div>
      </div>
      <p class="entry-help">
        Use a username to continue. New names are created automatically.
      </p>
      <div class="entry-row">
        <input
          v-model="username"
          autocomplete="off"
          maxlength="16"
          placeholder="Medi"
          :disabled="loading"
          :aria-invalid="Boolean(displayError)"
          @input="clearError"
        />
        <button class="send-button" :disabled="loading">{{ loading ? "…" : "Enter" }}</button>
      </div>
      <p v-if="displayError" class="form-error">{{ displayError }}</p>
    </form>
  </section>
</template>
