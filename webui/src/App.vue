<script setup>
// Application shell: mounts the LoginScreen when unauthenticated, otherwise
// draws the sidebar and delegates the main content panel to the router. All
// domain state lives in composables (useAuth, useConversations) so views can
// read/mutate it without prop-drilling.
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api } from "./api";
import { useAuth } from "./composables/useAuth";
import { useConversations } from "./composables/useConversations";
import { pickAndEncodePhoto, PhotoCancelled } from "./photo";
import LoginScreen from "./components/LoginScreen.vue";
import AppSidebar from "./components/AppSidebar.vue";
import ModalShell from "./components/ModalShell.vue";

const route = useRoute();
const router = useRouter();

const {
  currentUser,
  identities,
  loading: authLoading,
  login,
  rememberIdentity,
  switchTo,
  updateName,
  updatePhoto,
  logout: authLogout,
} = useAuth();

const {
  conversations,
  activeConversation,
  loadingList,
  loadingActive,
  loadConversations,
  openConversation,
  clearActive,
  startPrivate,
} = useConversations();

const people = ref([]);
const peopleQuery = ref("");
const peopleLoading = ref(false);
const chatQuery = ref("");
const toast = ref("");
const loginError = ref("");
const modal = ref(null);
const selectedMembers = ref(new Set());
const groupName = ref("");
const renameValue = ref("");
const newAccountName = ref("");
const modalBusy = ref(false);

const visibleConversations = computed(() => {
  const query = chatQuery.value.trim().toLowerCase();
  let list = conversations.value;
  if (query) {
    list = list.filter((conversation) => {
      const preview = conversation.latestMessageIsPhoto
        ? "photo message"
        : conversation.latestMessagePreview || "";
      return `${conversation.name} ${preview}`.toLowerCase().includes(query);
    });
  }
  // Reverse-chronological order (most recent activity first), as the
  // WASAText specification requires for the conversation list.
  return [...list].sort((left, right) => {
    const leftTime = left.latestMessageAt ? new Date(left.latestMessageAt).getTime() : 0;
    const rightTime = right.latestMessageAt ? new Date(right.latestMessageAt).getTime() : 0;
    return rightTime - leftTime;
  });
});

const anyLoading = computed(
  () => authLoading.value || loadingList.value || loadingActive.value || peopleLoading.value || modalBusy.value,
);

onMounted(async () => {
  if (currentUser.value) {
    await refreshEverything();
  }
});

// Keep the sidebar in sync when the active route changes away from a chat
// (e.g. after logout or after leaving a group pushes us back to "/").
watch(
  () => route.name,
  (name) => {
    if (name !== "chat") clearActive();
  },
);

function notify(message) {
  toast.value = message;
  window.setTimeout(() => {
    if (toast.value === message) toast.value = "";
  }, 2400);
}

async function refreshEverything() {
  try {
    await Promise.all([loadConversations(), searchPeople(peopleQuery.value)]);
    if (route.params.id) {
      await openConversation(route.params.id);
    }
  } catch (err) {
    notify(err.response?.data?.error || err.message);
  }
}

async function handleLogin(name) {
  loginError.value = "";
  try {
    await login(name);
    await refreshEverything();
    notify(`Using ${name}`);
  } catch (err) {
    loginError.value = err.response?.data?.error || err.message;
  }
}

async function searchPeople(query) {
  peopleQuery.value = query;
  if (!currentUser.value || !query.trim()) {
    people.value = [];
    return;
  }
  peopleLoading.value = true;
  try {
    const payload = await api("get", `/users?name=${encodeURIComponent(query)}`);
    people.value = (payload.users || []).filter((person) => person.id !== currentUser.value.identifier);
  } catch (err) {
    notify(err.response?.data?.error || err.message);
  } finally {
    peopleLoading.value = false;
  }
}

async function startConversation(userId) {
  try {
    const conversation = await startPrivate(userId);
    router.push(`/chat/${conversation.id}`);
  } catch (err) {
    notify(err.response?.data?.error || err.message);
  }
}

function navigateTo(id) {
  router.push(`/chat/${id}`);
}

async function showCreateGroup() {
  selectedMembers.value = new Set();
  groupName.value = "";
  peopleLoading.value = true;
  try {
    const payload = await api("get", "/users?name=");
    people.value = (payload.users || []).filter((person) => person.id !== currentUser.value.identifier);
    modal.value = "group";
  } catch (err) {
    notify(err.response?.data?.error || err.message);
  } finally {
    peopleLoading.value = false;
  }
}

function toggleMember(id) {
  const next = new Set(selectedMembers.value);
  if (next.has(id)) next.delete(id);
  else next.add(id);
  selectedMembers.value = next;
}

async function createGroup() {
  if (!groupName.value.trim() || selectedMembers.value.size === 0) {
    notify("Choose a group name and at least one member");
    return;
  }
  modalBusy.value = true;
  try {
    const group = await api("post", "/groups", {
      name: groupName.value.trim(),
      memberIds: Array.from(selectedMembers.value),
    });
    modal.value = null;
    await loadConversations();
    router.push(`/chat/${group.id}`);
  } catch (err) {
    notify(err.response?.data?.error || err.message);
  } finally {
    modalBusy.value = false;
  }
}

function showGroupInfo() {
  modal.value = "group-info";
}

async function showAddMembers() {
  peopleLoading.value = true;
  try {
    const payload = await api("get", "/users?name=");
    const memberIds = new Set((activeConversation.value?.members || []).map((member) => member.id));
    people.value = (payload.users || []).filter((person) => !memberIds.has(person.id));
    modal.value = "add-members";
  } catch (err) {
    notify(err.response?.data?.error || err.message);
  } finally {
    peopleLoading.value = false;
  }
}

function renameGroup() {
  renameValue.value = activeConversation.value.name;
  modal.value = "rename-group";
}

async function saveGroupName() {
  modalBusy.value = true;
  try {
    const updated = await api("put", `/groups/${activeConversation.value.id}/name`, {
      name: renameValue.value.trim(),
    });
    modal.value = null;
    activeConversation.value = updated;
    await loadConversations();
  } catch (err) {
    notify(err.response?.data?.error || err.message);
  } finally {
    modalBusy.value = false;
  }
}

async function setGroupPhoto() {
  let photo;
  try {
    photo = await pickAndEncodePhoto();
  } catch (err) {
    if (err instanceof PhotoCancelled) return;
    notify(err.message || "could not read image");
    return;
  }
  modalBusy.value = true;
  try {
    activeConversation.value = await api("put", `/groups/${activeConversation.value.id}/photo`, {
      photo,
    });
    await loadConversations();
    notify("Group photo updated");
  } catch (err) {
    notify(err.response?.data?.error || err.message);
  } finally {
    modalBusy.value = false;
  }
}

async function addMember(id) {
  modalBusy.value = true;
  try {
    activeConversation.value = await api("put", `/groups/${activeConversation.value.id}/members/${id}`);
    notify("Member added");
  } catch (err) {
    notify(err.response?.data?.error || err.message);
  } finally {
    modalBusy.value = false;
  }
}

async function leaveGroup() {
  modalBusy.value = true;
  try {
    await api("delete", `/groups/${activeConversation.value.id}/members/me`);
    modal.value = null;
    clearActive();
    await loadConversations();
    router.push("/");
  } catch (err) {
    notify(err.response?.data?.error || err.message);
  } finally {
    modalBusy.value = false;
  }
}

function showSettings() {
  modal.value = "settings";
}

function renameCurrentUser() {
  renameValue.value = currentUser.value.name;
  modal.value = "rename-user";
}

async function saveUserName() {
  modalBusy.value = true;
  try {
    await updateName(renameValue.value.trim());
    modal.value = "settings";
    await refreshEverything();
  } catch (err) {
    notify(err.response?.data?.error || err.message);
  } finally {
    modalBusy.value = false;
  }
}

async function setUserPhoto() {
  let photo;
  try {
    photo = await pickAndEncodePhoto();
  } catch (err) {
    if (err instanceof PhotoCancelled) return;
    notify(err.message || "could not read image");
    return;
  }
  modalBusy.value = true;
  try {
    await updatePhoto(photo);
    await refreshEverything();
    notify("Profile photo updated");
  } catch (err) {
    notify(err.response?.data?.error || err.message);
  } finally {
    modalBusy.value = false;
  }
}

async function switchAccount(identity) {
  switchTo(identity);
  modal.value = null;
  await refreshEverything();
}

async function addAccount() {
  const name = newAccountName.value.trim();
  newAccountName.value = "";
  modal.value = null;
  await handleLogin(name);
}

function logout() {
  authLogout();
  clearActive();
  conversations.value = [];
  modal.value = null;
  router.push("/");
}
</script>

<template>
  <LoginScreen
    v-if="!currentUser"
    :loading="authLoading"
    :server-error="loginError"
    @clear-error="loginError = ''"
    @login="handleLogin"
  />

  <div v-else class="app-shell">
    <AppSidebar
      :current-user="currentUser"
      :conversations="visibleConversations"
      :active-conversation="activeConversation"
      :people="people"
      :people-query="peopleQuery"
      :chat-query="chatQuery"
      :loading="loadingList || peopleLoading"
      @search-people="searchPeople"
      @search-chats="chatQuery = $event"
      @open-conversation="navigateTo"
      @start-conversation="startConversation"
      @new-chat="modal = 'new-chat'"
      @new-group="showCreateGroup"
      @settings="showSettings"
      @refresh="refreshEverything"
    />
    <router-view @group-info="showGroupInfo" />
  </div>

  <ModalShell v-if="modal === 'new-chat'" title="New chat" @close="modal = null">
    <p class="modal-empty">Use People search to start a direct chat, or create a group.</p>
    <button class="send-button" type="button" @click="modal = null; showCreateGroup()">New group</button>
  </ModalShell>

  <ModalShell v-if="modal === 'group'" title="New group" @close="modal = null">
    <input v-model="groupName" class="modal-main-input" minlength="3" maxlength="30" placeholder="Group name" />
    <p class="modal-count">{{ selectedMembers.size }} selected</p>
    <div class="modal-member-list">
      <button
        v-for="person in people"
        :key="person.id"
        :class="['person-row', { selected: selectedMembers.has(person.id) }]"
        type="button"
        @click="toggleMember(person.id)"
      >
        <span class="avatar">{{ person.name.slice(0, 1).toUpperCase() }}</span>
        <span><strong>{{ person.name }}</strong><small>Available to add</small></span>
        <span class="mini-button">{{ selectedMembers.has(person.id) ? "Added" : "Add" }}</span>
      </button>
    </div>
    <button class="send-button" type="button" :disabled="modalBusy" @click="createGroup">
      {{ modalBusy ? "Creating…" : "Create" }}
    </button>
  </ModalShell>

  <ModalShell v-if="modal === 'group-info' && activeConversation" title="Group info" @close="modal = null">
    <section class="group-info-hero">
      <span class="avatar xl" :style="activeConversation.photo ? { backgroundImage: `url(data:image/png;base64,${activeConversation.photo})` } : {}">
        <span v-if="!activeConversation.photo">{{ activeConversation.name.slice(0, 1).toUpperCase() }}</span>
      </span>
      <h2>{{ activeConversation.name }}</h2>
      <p>{{ activeConversation.members.length }} members</p>
    </section>
    <button class="send-button" type="button" @click="showAddMembers">Add members</button>
    <div class="settings-group">
      <button class="settings-row" type="button" @click="renameGroup">Rename group</button>
      <button class="settings-row" type="button" @click="setGroupPhoto">Change photo</button>
    </div>
    <p class="modal-count">Members</p>
    <div class="group-member-list">
      <div v-for="member in activeConversation.members" :key="member.id" class="group-member-row">
        <span class="avatar">{{ member.name.slice(0, 1).toUpperCase() }}</span>
        <strong>{{ member.name }}</strong>
      </div>
    </div>
    <div class="settings-group danger-zone">
      <button class="settings-row" type="button" :disabled="modalBusy" @click="leaveGroup">
        {{ modalBusy ? "Leaving…" : "Leave group" }}
      </button>
    </div>
  </ModalShell>

  <ModalShell v-if="modal === 'add-members' && activeConversation" title="Add members" @close="modal = 'group-info'">
    <p v-if="people.length === 0" class="modal-empty">No available users to add.</p>
    <div v-else class="modal-member-list">
      <button v-for="person in people" :key="person.id" class="person-row" type="button" @click="addMember(person.id); modal = 'group-info'">
        <span class="avatar">{{ person.name.slice(0, 1).toUpperCase() }}</span>
        <span><strong>{{ person.name }}</strong><small>Available to add</small></span>
        <span class="mini-button">Add</span>
      </button>
    </div>
  </ModalShell>

  <ModalShell v-if="modal === 'rename-group'" title="Rename group" @close="modal = null">
    <input v-model="renameValue" class="modal-main-input" minlength="3" maxlength="30" />
    <button class="send-button" type="button" :disabled="modalBusy" @click="saveGroupName">
      {{ modalBusy ? "Saving…" : "Save" }}
    </button>
  </ModalShell>

  <ModalShell v-if="modal === 'settings'" title="Settings" @close="modal = null">
    <section class="account-profile">
      <span class="avatar xl" :style="currentUser.photo ? { backgroundImage: `url(data:image/png;base64,${currentUser.photo})` } : {}">
        <span v-if="!currentUser.photo">{{ currentUser.name.slice(0, 1).toUpperCase() }}</span>
      </span>
      <h2>{{ currentUser.name }}</h2>
      <p>Current profile</p>
    </section>
    <div class="settings-group">
      <button class="settings-row" type="button" @click="renameCurrentUser">Profile name</button>
      <button class="settings-row" type="button" @click="setUserPhoto">Profile photo</button>
      <button class="settings-row" type="button" @click="modal = 'switch-account'">Switch or add account</button>
      <button class="settings-row" type="button" @click="logout">Logout</button>
    </div>
  </ModalShell>

  <ModalShell v-if="modal === 'rename-user'" title="Rename profile" @close="modal = 'settings'">
    <input v-model="renameValue" class="modal-main-input" minlength="3" maxlength="16" />
    <button class="send-button" type="button" :disabled="modalBusy" @click="saveUserName">
      {{ modalBusy ? "Saving…" : "Save" }}
    </button>
  </ModalShell>

  <ModalShell v-if="modal === 'switch-account'" title="Choose account" @close="modal = null">
    <button
      v-for="identity in identities"
      :key="identity.identifier"
      class="person-row"
      type="button"
      @click="switchAccount(identity)"
    >
      <span class="avatar">{{ identity.name.slice(0, 1).toUpperCase() }}</span>
      <span><strong>{{ identity.name }}</strong><small>Saved profile</small></span>
    </button>
    <form class="modal-form" @submit.prevent="addAccount">
      <input v-model="newAccountName" class="modal-main-input" minlength="3" maxlength="16" placeholder="Username" />
      <button class="send-button" :disabled="authLoading">
        {{ authLoading ? "…" : "Enter" }}
      </button>
    </form>
  </ModalShell>

  <div v-if="toast" class="toast">{{ toast }}</div>
  <div v-if="anyLoading" class="loading-bar" aria-hidden="true"></div>
</template>
