<script setup>
import { computed, nextTick, ref, watch } from "vue";
import { supportedReactions, normalizeReaction } from "../reactions";

const props = defineProps({
  conversation: { type: Object, default: null },
  currentUser: { type: Object, default: null },
  conversations: { type: Array, default: () => [] },
});

const emit = defineEmits([
  "send-message",
  "send-photo",
  "reply",
  "clear-reply",
  "react",
  "clear-reaction",
  "delete-message",
  "forward-message",
  "regular-message",
  "group-info",
]);

const text = ref("");
const replyTarget = ref(null);
const forwardTarget = ref(null);
const pickerTarget = ref(null);
const messageList = ref(null);

const subtitle = computed(() => {
  if (!props.conversation) return "Choose a user, start a chat, or create a group.";
  if (props.conversation.type === "group") return `${props.conversation.members.length} members`;
  return "Direct message";
});

const orderedMessages = computed(() => {
  return [...(props.conversation?.messages || [])].sort((left, right) => {
    return new Date(left.timestamp).getTime() - new Date(right.timestamp).getTime();
  });
});

function scrollToLatest() {
  nextTick(() => {
    if (!messageList.value) return;
    messageList.value.scrollTop = messageList.value.scrollHeight;
  });
}

watch(
  () => [props.conversation?.id, props.conversation?.messages?.length],
  scrollToLatest,
  { immediate: true },
);

function avatarText(name) {
  return String(name || "W").slice(0, 1).toUpperCase();
}

function isMine(message) {
  return props.currentUser && message.sender.id === props.currentUser.identifier;
}

function submit() {
  const value = text.value.trim();
  if (!value) return;
  emit("send-message", { text: value, replyToMessageId: replyTarget.value?.id });
  text.value = "";
  replyTarget.value = null;
}

function selectReply(message) {
  replyTarget.value = message;
  emit("reply", message);
}

function clearReply() {
  replyTarget.value = null;
  emit("clear-reply");
}

function chooseForward(message, conversationId) {
  emit("forward-message", { message, conversationId });
  forwardTarget.value = null;
}

function sendRegular(message, conversationId) {
  emit("regular-message", { message, conversationId });
  forwardTarget.value = null;
}
</script>

<template>
  <main class="chat-area">
    <header class="chat-header">
      <div class="chat-title-wrap">
        <span class="avatar large" :style="conversation?.photo ? { backgroundImage: `url(data:image/png;base64,${conversation.photo})` } : {}">
          <span v-if="!conversation?.photo">{{ avatarText(conversation?.name) }}</span>
        </span>
        <div>
          <h2>{{ conversation?.name || "Choose a conversation" }}</h2>
          <p>{{ subtitle }}</p>
        </div>
      </div>
      <button v-if="conversation?.type === 'group'" class="tool-button" type="button" @click="$emit('group-info')">
        Group info
      </button>
    </header>

    <section v-if="!conversation" class="empty-state">
      <div class="empty-visual"><span></span><span></span><span></span></div>
      <p class="eyebrow">Ready when you are</p>
      <h2>Your conversations live here</h2>
      <p>Choose a chat, search for someone, or create a group to start messaging.</p>
    </section>

    <section v-else ref="messageList" class="message-list">
      <article
        v-for="message in orderedMessages"
        :key="message.id"
        :class="['message', { mine: isMine(message) }]"
      >
        <div class="message-meta">{{ message.sender.name }}</div>
        <div :class="['bubble', { 'photo-bubble': message.type === 'photo' }]">
          <div v-if="message.replyTo" class="reply-box">
            {{ message.replyTo.senderName }}: {{ message.replyTo.preview }}
          </div>
          <div v-if="message.isForwarded" class="forwarded-label">
            Forwarded<span v-if="message.forwardedFrom"> from {{ message.forwardedFrom }}</span>
          </div>
          <img v-if="message.type === 'photo'" class="photo-message" :src="`data:image/png;base64,${message.photo}`" alt="Photo message" />
          <div v-else class="message-text">{{ message.text }}</div>
          <div class="bubble-footer">
            <span>{{ new Date(message.timestamp).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }) }}</span>
            <span v-if="isMine(message)" class="delivery-status">{{ message.deliveryStatus || "received" }}</span>
          </div>
        </div>

        <div class="reaction-strip">
          <span v-for="reaction in message.reactions" :key="`${message.id}-${reaction.user.id}`" class="mini-button">
            <span class="reaction-user">{{ reaction.user.name }}</span>
            <span class="reaction-emoji">{{ normalizeReaction(reaction.emoticon) }}</span>
          </span>
        </div>

        <div class="quick-reaction-bar">
          <button
            v-for="reaction in supportedReactions.slice(0, 5)"
            :key="reaction.value"
            class="quick-reaction-button"
            type="button"
            :title="reaction.name"
            @click="$emit('react', { message, reaction: reaction.value })"
          >
            {{ reaction.value }}
          </button>
          <button class="quick-reaction-button add-reaction" type="button" @click="pickerTarget = message">+</button>
        </div>

        <div class="message-tools">
          <button class="mini-button" type="button" @click="selectReply(message)">Reply</button>
          <button class="mini-button" type="button" @click="forwardTarget = message">Forward</button>
          <button class="mini-button" type="button" @click="$emit('clear-reaction', message)">Clear reaction</button>
          <button v-if="isMine(message)" class="mini-button danger" type="button" @click="$emit('delete-message', message)">Delete</button>
        </div>
      </article>
    </section>

    <form v-if="conversation" class="composer" @submit.prevent="submit">
      <div v-if="replyTarget" class="reply-preview">
        <div>
          <span>Replying to</span>
          <strong>{{ replyTarget.sender.name }}: {{ replyTarget.text || 'Photo message' }}</strong>
        </div>
        <button class="icon-button" type="button" @click="clearReply">X</button>
      </div>
      <div class="composer-row">
        <button class="icon-button" type="button" @click="$emit('send-photo')">+</button>
        <input v-model="text" autocomplete="off" placeholder="Write a message" />
        <button class="send-button">Send</button>
      </div>
    </form>

    <div v-if="pickerTarget" class="floating-picker">
      <button
        v-for="reaction in supportedReactions"
        :key="reaction.value"
        type="button"
        @click="$emit('react', { message: pickerTarget, reaction: reaction.value }); pickerTarget = null"
      >
        {{ reaction.value }}
      </button>
    </div>

    <div v-if="forwardTarget" class="modal-backdrop" @click.self="forwardTarget = null">
      <section class="modal">
        <header class="modal-header">
          <h3>Forward message</h3>
          <button class="icon-button" type="button" @click="forwardTarget = null">X</button>
        </header>
        <div class="modal-body">
          <p class="modal-empty">Choose how this message should be sent.</p>
          <div class="modal-list">
            <div v-for="conversationItem in conversations" :key="conversationItem.id" class="forward-destination">
              <span class="avatar">{{ avatarText(conversationItem.name) }}</span>
              <div>
                <strong>{{ conversationItem.name }}</strong>
                <small>{{ conversationItem.type === "group" ? "Group" : "Direct message" }}</small>
                <div class="forward-actions">
                  <button class="mini-button" type="button" @click="chooseForward(forwardTarget, conversationItem.id)">Forward</button>
                  <button class="mini-button" type="button" @click="sendRegular(forwardTarget, conversationItem.id)">Send regular</button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>
  </main>
</template>
