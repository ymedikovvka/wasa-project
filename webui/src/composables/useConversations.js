import { ref } from "vue";
import { api } from "../api";
import { normalizeReaction } from "../reactions";

// Module-scoped reactive state: list + active conversation are shared across
// every component that calls useConversations(), so a route-level view can
// drive the sidebar state without prop-drilling through App.vue.
const conversations = ref([]);
const activeConversation = ref(null);
const loadingList = ref(false);
const loadingActive = ref(false);
const sending = ref(false);
const error = ref("");

function runWith(flag, fn) {
  // Small helper that flips a boolean while an async operation is in flight,
  // captures failures into the shared `error` ref, and still rethrows so
  // callers can react (e.g. keep a modal open).
  return async (...args) => {
    flag.value = true;
    error.value = "";
    try {
      return await fn(...args);
    } catch (err) {
      error.value = err.response?.data?.error || err.message;
      throw err;
    } finally {
      flag.value = false;
    }
  };
}

const loadConversations = runWith(loadingList, async () => {
  const payload = await api("get", "/conversations");
  conversations.value = payload.conversations || [];
});

const openConversation = runWith(loadingActive, async (id) => {
  activeConversation.value = await api("get", `/conversations/${id}`);
  return activeConversation.value;
});

function clearActive() {
  activeConversation.value = null;
}

const startPrivate = runWith(sending, async (userId) => {
  const conversation = await api("post", "/conversations", { userId });
  await loadConversations();
  await openConversation(conversation.id);
  return conversation;
});

const sendText = runWith(sending, async (text, replyToMessageId) => {
  if (!activeConversation.value) return;
  const body = { type: "text", text };
  if (replyToMessageId) body.replyToMessageId = replyToMessageId;
  await api("post", `/conversations/${activeConversation.value.id}/messages`, body);
  await openConversation(activeConversation.value.id);
  await loadConversations();
});

const sendPhoto = runWith(sending, async (photoBase64) => {
  if (!activeConversation.value) return;
  await api("post", `/conversations/${activeConversation.value.id}/messages`, {
    type: "photo",
    photo: photoBase64,
  });
  await openConversation(activeConversation.value.id);
  await loadConversations();
});

const addReaction = runWith(sending, async (message, reaction) => {
  const normalized = normalizeReaction(reaction);
  await api("put", `/messages/${message.id}/comments/me`, { emoticon: normalized });
  if (activeConversation.value) await openConversation(activeConversation.value.id);
});

const removeReaction = runWith(sending, async (message) => {
  await api("delete", `/messages/${message.id}/comments/me`);
  if (activeConversation.value) await openConversation(activeConversation.value.id);
});

const deleteMessage = runWith(sending, async (message) => {
  await api("delete", `/messages/${message.id}`);
  if (activeConversation.value) await openConversation(activeConversation.value.id);
  await loadConversations();
});

const forwardMessage = runWith(sending, async (message, conversationId) => {
  await api("post", `/messages/${message.id}/forward`, { conversationId });
  await loadConversations();
});

const regularMessage = runWith(sending, async (message, conversationId) => {
  const body =
    message.type === "photo"
      ? { type: "photo", photo: message.photo }
      : { type: "text", text: message.text || "" };
  await api("post", `/conversations/${conversationId}/messages`, body);
  await loadConversations();
});

export function useConversations() {
  return {
    conversations,
    activeConversation,
    loadingList,
    loadingActive,
    sending,
    error,
    loadConversations,
    openConversation,
    clearActive,
    startPrivate,
    sendText,
    sendPhoto,
    addReaction,
    removeReaction,
    deleteMessage,
    forwardMessage,
    regularMessage,
  };
}
