import { ref, watch } from "vue";
import { api, setCurrentIdentifier } from "../api";

// Module-scoped reactive state so every component that imports `useAuth`
// shares the same logged-in user without going through Vuex / Pinia. The
// identifier list is persisted in localStorage so reloading the page keeps
// the "switch account" affordance working.
const identities = ref(JSON.parse(localStorage.getItem("wasatext.identities") || "[]"));
const currentUser = ref(JSON.parse(localStorage.getItem("wasatext.currentUser") || "null"));
const loading = ref(false);
const error = ref("");

watch(
  currentUser,
  (user) => {
    setCurrentIdentifier(user?.identifier || "");
    localStorage.setItem("wasatext.currentUser", JSON.stringify(user));
  },
  { immediate: true },
);

watch(
  identities,
  (value) => {
    localStorage.setItem("wasatext.identities", JSON.stringify(value));
  },
  { deep: true },
);

function rememberIdentity(user) {
  const identity = {
    identifier: user.id || user.identifier,
    name: user.name,
    photo: user.photo || null,
  };
  const index = identities.value.findIndex((item) => item.identifier === identity.identifier);
  if (index >= 0) identities.value[index] = identity;
  else identities.value.push(identity);
  currentUser.value = identity;
}

async function login(name) {
  loading.value = true;
  error.value = "";
  try {
    const session = await api("post", "/session", { name });
    rememberIdentity({ id: session.identifier, name });
  } catch (err) {
    error.value = err.response?.data?.error || err.message;
    throw err;
  } finally {
    loading.value = false;
  }
}

async function updateName(newName) {
  loading.value = true;
  error.value = "";
  try {
    const updated = await api("put", "/users/me/name", { name: newName });
    rememberIdentity(updated);
    return updated;
  } catch (err) {
    error.value = err.response?.data?.error || err.message;
    throw err;
  } finally {
    loading.value = false;
  }
}

async function updatePhoto(photoBase64) {
  loading.value = true;
  error.value = "";
  try {
    const updated = await api("put", "/users/me/photo", { photo: photoBase64 });
    rememberIdentity(updated);
    return updated;
  } catch (err) {
    error.value = err.response?.data?.error || err.message;
    throw err;
  } finally {
    loading.value = false;
  }
}

function switchTo(identity) {
  currentUser.value = identity;
}

function logout() {
  currentUser.value = null;
}

export function useAuth() {
  return {
    currentUser,
    identities,
    loading,
    error,
    login,
    rememberIdentity,
    switchTo,
    updateName,
    updatePhoto,
    logout,
  };
}
