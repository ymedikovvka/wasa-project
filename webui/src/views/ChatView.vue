<script setup>
// Renders the active conversation for `/chat/:id`. The view is route-driven
// (no parent callback): it watches `route.params.id` and hydrates the
// shared composable state, so browser back/forward and direct-URL loads
// both work out of the box.
import { watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useAuth } from "../composables/useAuth";
import { useConversations } from "../composables/useConversations";
import { pickAndEncodePhoto, PhotoCancelled } from "../photo";
import ChatPanel from "../components/ChatPanel.vue";

const route = useRoute();
const router = useRouter();

const { currentUser } = useAuth();
const {
  conversations,
  activeConversation,
  loadingActive,
  openConversation,
  sendText,
  sendPhoto,
  addReaction,
  removeReaction,
  deleteMessage,
  forwardMessage,
  regularMessage,
} = useConversations();

async function hydrate(id) {
  if (!id) return;
  try {
    await openConversation(id);
  } catch {
    // Unknown / forbidden conversation — fall back to the home view.
    router.replace("/");
  }
}

watch(
  () => route.params.id,
  (id) => hydrate(id),
  { immediate: true },
);

function onSendMessage({ text, replyToMessageId }) {
  sendText(text, replyToMessageId);
}

async function onSendPhoto() {
  try {
    const photo = await pickAndEncodePhoto();
    await sendPhoto(photo);
  } catch (err) {
    if (err instanceof PhotoCancelled) return;
    // Surface the reason so the photo-picker never silently no-ops. Most
    // common causes: server rejected the payload (HTTP 4xx returned by
    // `sendPhoto`) or the browser couldn't decode the selected file.
    const message = err.response?.data?.error || err.message || "failed to send photo";
    window.alert(`Could not send photo: ${message}`);
  }
}

function onReact({ message, reaction }) {
  addReaction(message, reaction);
}

function onClearReaction(message) {
  removeReaction(message);
}

function onDelete(message) {
  deleteMessage(message);
}

function onForward({ message, conversationId }) {
  forwardMessage(message, conversationId);
}

function onRegular({ message, conversationId }) {
  regularMessage(message, conversationId);
}

defineEmits(["group-info"]);
</script>

<template>
  <div v-if="loadingActive && !activeConversation" class="empty-state">
    <div class="empty-state-inner">
      <p>Loading conversation…</p>
    </div>
  </div>
  <ChatPanel
    v-else
    :conversation="activeConversation"
    :current-user="currentUser"
    :conversations="conversations"
    @send-message="onSendMessage"
    @send-photo="onSendPhoto"
    @react="onReact"
    @clear-reaction="onClearReaction"
    @delete-message="onDelete"
    @forward-message="onForward"
    @regular-message="onRegular"
    @group-info="$emit('group-info')"
  />
</template>
