<template>
  <div class="relative min-h-screen overflow-hidden bg-[#faf7f0] dark:bg-dark-950">
    <!-- Decorative: single orange glow top-left + subtle grid -->
    <div class="pointer-events-none absolute -left-40 -top-40 h-96 w-96 rounded-full bg-primary-400/25 blur-3xl"></div>
    <div
      class="pointer-events-none absolute inset-0 bg-[linear-gradient(rgba(255,107,53,0.03)_1px,transparent_1px),linear-gradient(90deg,rgba(255,107,53,0.03)_1px,transparent_1px)] bg-[size:64px_64px]"
    ></div>

    <!-- Layout: stacked on mobile, split on md+ -->
    <div class="relative z-10 flex min-h-screen w-full flex-col md:flex-row">
      <!-- ===== Left: Brand Panel ===== -->
      <div class="flex flex-[1.3] flex-col justify-between gap-10 px-8 py-10 md:px-14 md:py-14 lg:px-20 lg:py-20">
        <!-- Top: Logo + name -->
        <template v-if="settingsLoaded">
          <div class="flex items-center gap-3">
            <div class="h-9 w-9 overflow-hidden rounded-xl shadow-sm">
              <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-cover" />
            </div>
            <span class="text-base font-bold text-gray-900 dark:text-white">{{ siteName }}</span>
          </div>

          <!-- Middle: Hero tagline -->
          <div class="max-w-xl">
            <h1 class="text-4xl font-extrabold leading-[0.95] tracking-tight text-gray-900 dark:text-white md:text-5xl lg:text-6xl">
              {{ siteName }}<span class="bg-gradient-to-r from-[#ff6b35] to-[#ff9166] bg-clip-text text-transparent">.</span>
            </h1>
            <p v-if="siteSubtitle" class="mt-5 max-w-md text-sm text-gray-600 dark:text-gray-400 md:text-base">
              {{ siteSubtitle }}
            </p>
          </div>

          <!-- Bottom: Copyright -->
          <div class="text-xs text-gray-400 dark:text-dark-500">
            &copy; {{ currentYear }} {{ siteName }}. All rights reserved.
          </div>
        </template>
      </div>

      <!-- ===== Right: Form Panel ===== -->
      <div class="flex flex-1 items-center justify-center px-6 pb-10 md:px-10 md:py-14 lg:px-14">
        <div class="w-full max-w-md rounded-2xl border border-[#ebe6dc] bg-white p-8 shadow-[0_8px_24px_rgba(0,0,0,0.04)] dark:border-dark-700 dark:bg-dark-900 dark:shadow-[0_8px_24px_rgba(0,0,0,0.3)]">
          <slot />
          <div class="mt-6 text-center text-sm">
            <slot name="footer" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

const appStore = useAppStore()

const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'Subscription to API Conversion Platform')
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
.text-gradient {
  @apply bg-gradient-to-r from-primary-600 to-primary-500 bg-clip-text text-transparent;
}
</style>
