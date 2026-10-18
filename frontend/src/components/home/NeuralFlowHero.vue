<template>
  <section
    ref="root"
    class="neural-hero"
    data-testid="neural-flow-hero"
    @pointermove="followPointer"
    @pointerleave="clearPointer"
    @pointercancel="clearPointer"
  >
    <div class="neural-content"><slot /></div>
    <canvas ref="canvas" class="neural-canvas" aria-hidden="true" />
    <button
      v-if="ready"
      type="button"
      class="neural-control"
      :aria-label="playing ? t('home.pauseAnimation') : t('home.playAnimation')"
      :title="playing ? t('home.pauseAnimation') : t('home.playAnimation')"
      @click="togglePlayback"
    >
      <span aria-hidden="true">{{ playing ? 'Ⅱ' : '▶' }}</span>
    </button>
  </section>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { createNeuralFlow } from './neuralFlow'

const { t } = useI18n()
const root = ref<HTMLElement>()
const canvas = ref<HTMLCanvasElement>()
const ready = ref(false)
const playing = ref(false)
let renderer: ReturnType<typeof createNeuralFlow> = null
let frame = 0
let last = 0
let elapsed = 0
let visible = true
let cleanup = () => {}

function stop() {
  cancelAnimationFrame(frame)
  frame = 0
  last = 0
}

function tick(now: number) {
  frame = 0
  if (!playing.value || !visible || document.hidden) return
  if (last) elapsed += Math.min(0.05, (now - last) / 1000)
  last = now
  renderer?.render(elapsed)
  frame = requestAnimationFrame(tick)
}

function start() {
  if (!frame && ready.value && playing.value && visible && !document.hidden) {
    last = 0
    frame = requestAnimationFrame(tick)
  }
}

function togglePlayback() {
  playing.value = !playing.value
  if (playing.value) start()
  else stop()
}

function followPointer(event: PointerEvent) {
  if (!playing.value || event.pointerType === 'touch' || !canvas.value) return
  const bounds = canvas.value.getBoundingClientRect()
  renderer?.movePointer(event.clientX - bounds.left, event.clientY - bounds.top)
}

function clearPointer() {
  renderer?.clearPointer()
}

onMounted(() => {
  if (!canvas.value || !root.value) return
  renderer = createNeuralFlow(canvas.value)
  if (!renderer) return
  ready.value = true
  const motion = window.matchMedia('(prefers-reduced-motion: reduce)')
  playing.value = !motion.matches
  const resize = new ResizeObserver(() => renderer?.resize())
  const intersection = new IntersectionObserver(([entry]) => {
    visible = entry.isIntersecting
    if (visible) start()
    else stop()
  })
  const onVisibility = () => {
    if (document.hidden) {
      clearPointer()
      stop()
    } else start()
  }
  const onMotion = () => {
    if (motion.matches) {
      clearPointer()
      playing.value = false
      stop()
    }
  }
  resize.observe(canvas.value)
  intersection.observe(root.value)
  document.addEventListener('visibilitychange', onVisibility)
  motion.addEventListener('change', onMotion)
  renderer.resize()
  start()
  cleanup = () => {
    resize.disconnect()
    intersection.disconnect()
    document.removeEventListener('visibilitychange', onVisibility)
    motion.removeEventListener('change', onMotion)
  }
})

onBeforeUnmount(() => {
  stop()
  cleanup()
  renderer?.dispose()
  renderer = null
})
</script>

<style scoped>
.neural-hero {
  position: relative;
  isolation: isolate;
  display: flex;
  align-items: center;
  overflow: hidden;
  min-width: 0;
  min-height: 100svh;
  background: #020617;
  color: white;
}
.neural-content {
  position: relative;
  z-index: 2;
  width: 52%;
  padding: 140px 0 140px clamp(20px, 7vw, 140px);
  animation: content-enter 450ms ease-out both;
}
.neural-canvas {
  position: absolute;
  inset: 0;
  display: block;
  width: 100%;
  height: 100%;
  pointer-events: none;
}
.neural-hero::after {
  content: '';
  position: absolute;
  inset: 0;
  z-index: 1;
  pointer-events: none;
  background:
    linear-gradient(90deg, #02061775 0%, transparent 60%),
    linear-gradient(
      0deg,
      #020617 0%,
      transparent 18%,
      transparent 88%,
      #02061755 100%
    );
}
.neural-control {
  position: absolute;
  z-index: 3;
  right: clamp(20px, 3vw, 48px);
  bottom: max(20px, env(safe-area-inset-bottom));
  display: grid;
  width: 44px;
  height: 44px;
  place-items: center;
  border: 1px solid #ff8a5240;
  border-radius: 50%;
  background: #0f172a80;
  color: #ffad80;
  transition: background 150ms;
  cursor: pointer;
}
.neural-control:hover {
  background: #1e293b;
}
.neural-control:focus-visible {
  outline: 2px solid #ffad80;
  outline-offset: 3px;
}
@keyframes content-enter {
  from {
    opacity: 0;
    transform: translateY(12px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
@media (max-width: 767px) {
  .neural-hero {
    align-items: flex-start;
    min-height: max(100svh, 760px);
  }
  .neural-content {
    width: 100%;
    padding: 140px 24px 360px;
  }
  .neural-hero::after {
    background: linear-gradient(
      180deg,
      #020617 0%,
      #020617e8 30%,
      transparent 65%,
      #02061780 100%
    );
  }
}
@media (prefers-reduced-motion: reduce) {
  .neural-content {
    animation: none;
  }
}
</style>
