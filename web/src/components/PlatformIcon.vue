<template>
  <span
    class="platform-icon"
    :class="sizeClass"
    :style="{ background: iconBg }"
    :title="label"
    aria-hidden="true"
  >
    <img
      v-if="iconSrc"
      :src="iconSrc"
      :alt="label"
      class="platform-icon-img"
      :class="{ 'platform-icon-img-tight': iconTight }"
      draggable="false"
    />
    <span v-else class="platform-icon-fallback">{{ fallbackIcon }}</span>
  </span>
</template>

<script setup>
import { computed } from 'vue'
import { getPlatformMeta } from '@/lib/platforms'

const props = defineProps({
  platform: { type: Object, default: null },
  platformId: { type: String, default: '' },
  size: { type: String, default: 'md' },
})

const meta = computed(() => props.platform || getPlatformMeta(props.platformId) || {})
const label = computed(() => meta.value.label || props.platformId || '平台')
const iconSrc = computed(() => meta.value.iconSrc || '')
const iconBg = computed(() => meta.value.iconBg || 'var(--app-surface-muted)')
const iconTight = computed(() => meta.value.iconTight === true)
const fallbackIcon = computed(() => meta.value.icon || label.value.slice(0, 1))

const sizeClass = computed(() => ({
  xs: 'platform-icon-xs',
  sm: 'platform-icon-sm',
  md: 'platform-icon-md',
  lg: 'platform-icon-lg',
}[props.size] || 'platform-icon-md'))
</script>

<style scoped>
.platform-icon {
  @apply inline-flex shrink-0 items-center justify-center overflow-hidden border;
  border-color: var(--app-border-soft);
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.05);
}

.platform-icon-xs {
  @apply h-5 w-5 rounded-md p-0.5;
}

.platform-icon-sm {
  @apply h-7 w-7 rounded-lg p-1;
}

.platform-icon-md {
  @apply h-9 w-9 rounded-lg p-1.5;
}

.platform-icon-lg {
  @apply h-12 w-12 rounded-2xl p-2;
}

.platform-icon-img {
  @apply block h-full w-full object-contain;
}

.platform-icon-img-tight {
  @apply scale-110;
}

.platform-icon-fallback {
  @apply text-xs font-semibold;
  color: var(--app-text-secondary);
}
</style>
