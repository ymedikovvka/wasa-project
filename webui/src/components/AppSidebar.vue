<script setup>
defineProps({
  currentUser: { type: Object, default: null },
  conversations: { type: Array, default: () => [] },
  activeConversation: { type: Object, default: null },
  people: { type: Array, default: () => [] },
  peopleQuery: { type: String, default: "" },
  chatQuery: { type: String, default: "" },
  loading: { type: Boolean, default: false },
});

const emit = defineEmits([
  "search-people",
  "search-chats",
  "open-conversation",
  "start-conversation",
  "new-chat",
  "new-group",
  "settings",
  "refresh",
]);

function avatarText(name) {
  return String(name || "W").slice(0, 1).toUpperCase();
}

function preview(conversation) {
  if (conversation.latestMessageIsPhoto) return "Photo message";
  return conversation.latestMessagePreview || "No messages yet";
}

// The WASAText spec requires each conversation in the list to show the date
// and time of its latest message. Same-day messages show the clock time;
// older ones show the day/month.
function latestTime(conversation) {
  if (!conversation.latestMessageAt) return "";
  const when = new Date(conversation.latestMessageAt);
  if (Number.isNaN(when.getTime())) return "";
  const now = new Date();
  if (when.toDateString() === now.toDateString()) {
    return when.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  }
  return when.toLocaleDateString([], { day: "2-digit", month: "2-digit" });
}
</script>

<template>
  <aside class="sidebar">
    <section class="identity-panel">
      <div>
        <p class="eyebrow">WASAText</p>
        <h1>Messages</h1>
      </div>
      <button class="icon-button ghost" type="button" @click="$emit('settings')">Settings</button>
    </section>

    <section class="tools-panel">
      <div class="section-title">
        <span>People</span>
        <button class="icon-button" type="button" :disabled="loading" @click="$emit('refresh')">{{ loading ? "Syncing…" : "Sync" }}</button>
      </div>
      <div class="input-row">
        <input :value="peopleQuery" placeholder="Search users" @input="$emit('search-people', $event.target.value)" />
        <button class="icon-button primary" type="button" @click="$emit('search-people', peopleQuery)">Go</button>
      </div>
      <div class="search-results">
        <p v-if="!peopleQuery" class="list-empty">Search by name to start a chat</p>
        <p v-else-if="people.length === 0" class="list-empty">No people found</p>
        <template v-else>
          <button
            v-for="person in people"
            :key="person.id"
            class="person-row"
            type="button"
            @click="$emit('start-conversation', person.id)"
          >
            <span class="avatar" :style="person.photo ? { backgroundImage: `url(data:image/png;base64,${person.photo})` } : {}">
              <span v-if="!person.photo">{{ avatarText(person.name) }}</span>
            </span>
            <span>
              <strong>{{ person.name }}</strong>
              <small>Available to chat</small>
            </span>
            <span class="mini-button">Chat</span>
          </button>
        </template>
      </div>
    </section>

    <section class="tools-panel group-shortcut">
      <div class="section-title">
        <span>Groups</span>
        <button class="icon-button accent" type="button" @click="$emit('new-group')">+</button>
      </div>
    </section>

    <section class="conversation-panel">
      <div class="section-title">
        <span>Chats</span>
        <span class="count">{{ conversations.length }}</span>
      </div>
      <div class="chat-search">
        <input :value="chatQuery" placeholder="Search chats" @input="$emit('search-chats', $event.target.value)" />
      </div>
      <div class="chat-filters">
        <button class="filter-chip active" type="button">All</button>
        <button class="filter-chip" type="button" @click="$emit('new-chat')">+</button>
      </div>
      <div class="conversation-list">
        <p v-if="conversations.length === 0" class="list-empty">No chats yet</p>
        <template v-else>
          <button
            v-for="conversation in conversations"
            :key="conversation.id"
            :class="['conversation-row', { active: activeConversation && activeConversation.id === conversation.id }]"
            type="button"
            @click="$emit('open-conversation', conversation.id)"
          >
            <span class="avatar" :style="conversation.photo ? { backgroundImage: `url(data:image/png;base64,${conversation.photo})` } : {}">
              <span v-if="!conversation.photo">{{ avatarText(conversation.name) }}</span>
            </span>
            <span class="conversation-info">
              <strong>{{ conversation.name }}</strong>
              <small>{{ preview(conversation) }}</small>
            </span>
            <span class="conversation-meta">
              <span v-if="latestTime(conversation)" class="conversation-time">{{ latestTime(conversation) }}</span>
              <span v-if="conversation.type === 'group'" class="conversation-tag">Group</span>
            </span>
          </button>
        </template>
      </div>
    </section>
  </aside>
</template>
