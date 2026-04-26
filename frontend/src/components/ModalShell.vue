<script setup>
defineProps({
  title: { type: String, required: true },
});

defineEmits(["close"]);
</script>

<template>
  <div class="modal-backdrop" @click.self="$emit('close')">
    <section class="modal">
      <header class="modal-header">
        <h3>{{ title }}</h3>
        <button class="icon-button" type="button" aria-label="Close" @click="$emit('close')">X</button>
      </header>
      <div class="modal-body">
        <slot />
      </div>
    </section>
  </div>
</template>
