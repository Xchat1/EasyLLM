<template>
  <span
    class="codex-icon"
    :class="sizeClass"
    :style="{ background: iconBg }"
    :title="label"
    aria-hidden="true"
  >
    <img
      v-if="iconSrc"
      :src="iconSrc"
      :alt="label"
      class="codex-icon-img"
      :class="{ 'codex-icon-img-tight': iconTight }"
      draggable="false"
    />
    <span v-else class="codex-icon-fallback">{{ fallbackIcon }}</span>
  </span>
</template>

<script setup>
import { computed } from 'vue'
import { getCodexRouteMeta } from '@/lib/codexRoutes'

const props = defineProps({
  item: { type: Object, default: null },
  itemId: { type: String, default: '' },
  size: { type: String, default: 'md' },
})

const meta = computed(() => props.item || getCodexRouteMeta(props.itemId) || {})
const label = computed(() => meta.value.label || props.itemId || 'Codex')
const iconSrc = computed(() => meta.value.iconSrc || '')
const iconBg = computed(() => meta.value.iconBg || 'var(--app-surface-muted)')
const iconTight = computed(() => meta.value.iconTight === true)
const fallbackIcon = computed(() => meta.value.icon || label.value.slice(0, 1))

const sizeClass = computed(() => ({
  xs: 'codex-icon-xs',
  sm: 'codex-icon-sm',
  md: 'codex-icon-md',
  lg: 'codex-icon-lg',
}[props.size] || 'codex-icon-md'))
</script>

<style scoped>
.codex-icon {
  @apply inline-flex shrink-0 items-center justify-center overflow-hidden border;
  border-color: var(--app-border-soft);
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.05);
}

.codex-icon-xs {
  @apply h-5 w-5 rounded-md p-0.5;
}

.codex-icon-sm {
  @apply h-7 w-7 rounded-lg p-1;
}

.codex-icon-md {
  @apply h-9 w-9 rounded-lg p-1.5;
}

.codex-icon-lg {
  @apply h-12 w-12 rounded-2xl p-2;
}

.codex-icon-img {
  @apply block h-full w-full object-contain;
}

.codex-icon-img-tight {
  @apply scale-110;
}

.codex-icon-fallback {
  @apply text-xs font-semibold;
  color: var(--app-text-secondary);
}
</style>
