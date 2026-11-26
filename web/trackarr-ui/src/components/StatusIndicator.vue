<script setup>
defineProps({ online: { type: Boolean, required: true } })
</script>

<template>
  <span class="indicator" :class="online ? 'online' : 'offline'" role="img" :aria-label="online ? 'Online' : 'Offline'" />
</template>

<style scoped>
.indicator {
  display: inline-block;
  width: 0.75rem;
  height: 0.75rem;
  margin-inline-end: 0.75rem;
  border-radius: 50%;
  background: currentColor;
  animation: pulse 1.5s ease-out infinite;
}

.online {
  color: #4caf50;
}

.offline {
  color: #f44336;
}

@keyframes pulse {
  from {
    box-shadow: 0 0 0 0 color-mix(in srgb, currentColor 60%, transparent);
  }
  to {
    box-shadow: 0 0 0 0.6rem transparent;
  }
}

@media (prefers-reduced-motion: reduce) {
  .indicator {
    animation: none;
  }
}
</style>
