<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="homeContent" class="min-h-screen">
    <!-- iframe mode -->
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- HTML mode - SECURITY: homeContent is admin-only setting, XSS risk is acceptable -->
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Default Home Page -->
  <div
    v-else
    class="relative flex min-h-screen flex-col overflow-hidden bg-[#faf7f0] dark:bg-dark-950"
  >
    <!-- Background Decorations: 1 warm glow + grid -->
    <div class="pointer-events-none absolute inset-0 overflow-hidden">
      <div
        class="absolute -left-40 -top-40 h-96 w-96 rounded-full bg-primary-400/25 blur-3xl"
      ></div>
      <div
        class="absolute inset-0 bg-[linear-gradient(rgba(255,107,53,0.03)_1px,transparent_1px),linear-gradient(90deg,rgba(255,107,53,0.03)_1px,transparent_1px)] bg-[size:64px_64px]"
      ></div>
    </div>

    <!-- Header -->
    <header class="relative z-20 px-6 py-5 md:px-8">
      <nav class="mx-auto flex max-w-6xl items-center justify-between">
        <!-- Logo + brand -->
        <div class="flex items-center gap-2.5">
          <div class="h-9 w-9 overflow-hidden rounded-[10px]">
            <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-cover" />
          </div>
          <span class="text-sm font-bold tracking-tight text-gray-900 dark:text-white">
            {{ siteName }} <span class="text-gray-400 dark:text-dark-500">· 2026</span>
          </span>
        </div>

        <!-- Nav Actions -->
        <div class="flex items-center gap-3">
          <!-- Language Switcher -->
          <LocaleSwitcher />

          <!-- Doc Link -->
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="rounded-lg p-2 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>

          <!-- Theme Toggle -->
          <button
            @click="toggleTheme"
            class="rounded-lg p-2 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>

          <!-- Login / Dashboard Button -->
          <router-link
            v-if="isAuthenticated"
            :to="dashboardPath"
            class="inline-flex items-center gap-1.5 rounded-full bg-gray-900 py-1 pl-1 pr-2.5 transition-colors hover:bg-gray-800 dark:bg-gray-800 dark:hover:bg-gray-700"
          >
            <span
              class="flex h-5 w-5 items-center justify-center rounded-full bg-gradient-to-br from-primary-400 to-primary-600 text-[10px] font-semibold text-white"
            >
              {{ userInitial }}
            </span>
            <span class="text-xs font-medium text-white">{{ t('home.dashboard') }}</span>
            <svg
              class="h-3 w-3 text-gray-400"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M4.5 19.5l15-15m0 0H8.25m11.25 0v11.25"
              />
            </svg>
          </router-link>
          <router-link
            v-else
            to="/login"
            class="inline-flex items-center rounded-full bg-gray-900 px-3 py-1 text-xs font-medium text-white transition-colors hover:bg-gray-800 dark:bg-gray-800 dark:hover:bg-gray-700"
          >
            {{ t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <!-- Main Content (Magazine + Bento Hybrid) -->
    <main class="relative z-10 flex-1 px-6 py-8 md:px-8 md:py-12">
      <div class="mx-auto max-w-6xl">
        <!-- ====== HERO · Magazine Typography ====== -->
        <section class="py-6 md:py-10">
          <div class="inline-block border-b border-gray-900 pb-1 text-[10px] font-semibold uppercase tracking-[0.2em] text-gray-600 dark:border-white dark:text-dark-300">
            ISSUE · 01 · AI API GATEWAY
          </div>
          <h1 class="mt-5 text-5xl font-black leading-[0.9] tracking-tight text-gray-900 dark:text-white md:text-7xl lg:text-[84px]">
            {{ siteName }}<span class="bg-gradient-to-r from-[#ff6b35] to-[#ff9166] bg-clip-text text-transparent">.</span>
          </h1>
          <div class="mt-6 grid grid-cols-1 items-end gap-6 md:grid-cols-[2fr_1fr] md:gap-10">
            <p class="max-w-xl text-sm leading-relaxed text-gray-600 dark:text-dark-300 md:text-base">
              {{ siteSubtitle }}
            </p>
            <div class="flex flex-wrap gap-2 md:justify-end">
              <router-link
                :to="isAuthenticated ? dashboardPath : '/login'"
                class="inline-flex items-center gap-2 rounded-lg bg-gray-900 px-5 py-2.5 text-sm font-bold text-white transition-colors hover:bg-black dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
              >
                {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
                <Icon name="arrowRight" size="sm" :stroke-width="2.5" />
              </router-link>
              <a
                v-if="docUrl"
                :href="docUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="inline-flex items-center gap-2 rounded-lg border border-gray-300 px-5 py-2.5 text-sm font-semibold text-gray-900 transition-colors hover:bg-gray-100 dark:border-dark-600 dark:text-white dark:hover:bg-dark-800"
              >
                {{ t('home.docs') }}
              </a>
            </div>
          </div>
        </section>

        <!-- Divider -->
        <div class="my-8 h-px bg-gray-900 dark:bg-white md:my-10"></div>

        <!-- ====== BENTO data strip ====== -->
        <section class="grid grid-cols-2 gap-3 md:grid-cols-4">
          <!-- Tile 1: MODELS -->
          <div class="rounded-xl border border-[#ebe6dc] bg-white/70 p-4 backdrop-blur-sm dark:border-dark-700 dark:bg-dark-800/60">
            <div class="text-[10px] font-semibold uppercase tracking-widest text-gray-500 dark:text-dark-400">MODELS</div>
            <div class="mt-1 text-3xl font-black text-gray-900 dark:text-white md:text-4xl">4+</div>
            <div class="mt-1 text-[10px] text-gray-500 dark:text-dark-400">Claude · GPT · Gemini · ...</div>
          </div>
          <!-- Tile 2: STATUS (filled orange) -->
          <div class="rounded-xl bg-[#ff6b35] p-4 text-[#0f0f10]">
            <div class="text-[10px] font-semibold uppercase tracking-widest opacity-80">STATUS</div>
            <div class="mt-1 text-xl font-black md:text-2xl">●●● Online</div>
            <div class="mt-1 text-[10px] opacity-75">API 网关 · 正常运行</div>
          </div>
          <!-- Tile 3: CAPACITY -->
          <div class="rounded-xl border border-[#ebe6dc] bg-white/70 p-4 backdrop-blur-sm dark:border-dark-700 dark:bg-dark-800/60">
            <div class="text-[10px] font-semibold uppercase tracking-widest text-gray-500 dark:text-dark-400">MAX CAPACITY</div>
            <div class="mt-1 text-3xl font-black text-gray-900 dark:text-white md:text-4xl">
              20<span class="text-gray-400 dark:text-dark-500">×</span>
            </div>
            <div class="mt-1 text-[10px] text-gray-500 dark:text-dark-400">Max 配额加成</div>
          </div>
          <!-- Tile 4: code snippet -->
          <div class="rounded-xl bg-[#0f0f10] p-4 font-mono text-[11px] leading-relaxed text-gray-300">
            <div class="text-[9px] uppercase tracking-widest text-gray-500">// 快速开始</div>
            <div class="mt-1 text-purple-300">curl</div>
            <div class="truncate text-blue-300">api.pubwhere.cn</div>
            <div class="font-bold text-green-400">→ 200 OK</div>
          </div>
        </section>

        <!-- Divider -->
        <div class="my-8 h-px bg-gray-900 dark:bg-white md:my-10"></div>

        <!-- ====== Numbered features (Magazine) ====== -->
        <section class="grid grid-cols-1 gap-8 md:grid-cols-3 md:gap-10">
          <div>
            <div class="text-4xl font-black text-[#ff6b35] md:text-5xl">01</div>
            <div class="mt-2 text-base font-bold text-gray-900 dark:text-white md:text-lg">
              {{ t('home.features.unifiedGateway') }}
            </div>
            <div class="mt-1 text-xs leading-relaxed text-gray-500 dark:text-dark-400 md:text-sm">
              {{ t('home.features.unifiedGatewayDesc') }}
            </div>
          </div>
          <div>
            <div class="text-4xl font-black text-[#ff6b35] md:text-5xl">02</div>
            <div class="mt-2 text-base font-bold text-gray-900 dark:text-white md:text-lg">
              {{ t('home.features.multiAccount') }}
            </div>
            <div class="mt-1 text-xs leading-relaxed text-gray-500 dark:text-dark-400 md:text-sm">
              {{ t('home.features.multiAccountDesc') }}
            </div>
          </div>
          <div>
            <div class="text-4xl font-black text-[#ff6b35] md:text-5xl">03</div>
            <div class="mt-2 text-base font-bold text-gray-900 dark:text-white md:text-lg">
              {{ t('home.features.balanceQuota') }}
            </div>
            <div class="mt-1 text-xs leading-relaxed text-gray-500 dark:text-dark-400 md:text-sm">
              {{ t('home.features.balanceQuotaDesc') }}
            </div>
          </div>
        </section>

        <!-- Divider -->
        <div class="my-8 h-px bg-gray-900 dark:bg-white md:my-10"></div>

        <!-- ====== Provider text strip ====== -->
        <section class="flex flex-wrap items-center gap-x-4 gap-y-2 text-sm md:gap-x-6">
          <div class="text-[10px] font-bold uppercase tracking-[0.2em] text-gray-600 dark:text-dark-300">
            POWERED BY —
          </div>
          <span class="font-bold text-gray-900 dark:text-white">{{ t('home.providers.claude') }}</span>
          <span class="text-gray-300 dark:text-dark-600">·</span>
          <span class="font-bold text-gray-900 dark:text-white">GPT</span>
          <span class="text-gray-300 dark:text-dark-600">·</span>
          <span class="font-bold text-gray-900 dark:text-white">{{ t('home.providers.gemini') }}</span>
          <span class="text-gray-300 dark:text-dark-600">·</span>
          <span class="font-bold text-gray-900 dark:text-white">{{ t('home.providers.antigravity') }}</span>
          <span class="ml-auto text-xs text-gray-500 dark:text-dark-400">+ {{ t('home.providers.soon') }}</span>
        </section>
      </div>
    </main>

    <!-- Footer -->
    <footer class="relative z-10 border-t border-gray-200/50 px-6 py-8 dark:border-dark-800/50">
      <div
        class="mx-auto flex max-w-6xl flex-col items-center justify-center gap-4 text-center sm:flex-row sm:text-left"
      >
        <p class="text-sm text-gray-500 dark:text-dark-400">
          &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
        </p>
        <div v-if="docUrl" class="flex items-center gap-4">
          <a
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="text-sm text-gray-500 transition-colors hover:text-gray-700 dark:text-dark-400 dark:hover:text-white"
          >
            {{ t('home.docs') }}
          </a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()

// Site settings - directly from appStore (already initialized from injected config)
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform')
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')

// Check if homeContent is a URL (for iframe display)
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

// Theme
const isDark = ref(document.documentElement.classList.contains('dark'))

// Auth state
const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
const userInitial = computed(() => {
  const user = authStore.user
  if (!user || !user.email) return ''
  return user.email.charAt(0).toUpperCase()
})

// Current year for footer
const currentYear = computed(() => new Date().getFullYear())

// Toggle theme
function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

// Initialize theme
function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (
    savedTheme === 'dark' ||
    (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  ) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

onMounted(() => {
  initTheme()

  // Check auth state
  authStore.checkAuth()

  // Ensure public settings are loaded (will use cache if already loaded from injected config)
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>

<style scoped>
/* Terminal Container */
.terminal-container {
  position: relative;
  display: inline-block;
}

/* Terminal Window */
.terminal-window {
  width: 420px;
  background: linear-gradient(145deg, #1e293b 0%, #0f172a 100%);
  border-radius: 14px;
  box-shadow:
    0 25px 50px -12px rgba(0, 0, 0, 0.4),
    0 0 0 1px rgba(255, 255, 255, 0.1),
    inset 0 1px 0 rgba(255, 255, 255, 0.1);
  overflow: hidden;
  transform: perspective(1000px) rotateX(2deg) rotateY(-2deg);
  transition: transform 0.3s ease;
}

.terminal-window:hover {
  transform: perspective(1000px) rotateX(0deg) rotateY(0deg) translateY(-4px);
}

/* Terminal Header */
.terminal-header {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  background: rgba(30, 41, 59, 0.8);
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.terminal-buttons {
  display: flex;
  gap: 8px;
}

.terminal-buttons span {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.btn-close {
  background: #ef4444;
}
.btn-minimize {
  background: #eab308;
}
.btn-maximize {
  background: #22c55e;
}

.terminal-title {
  flex: 1;
  text-align: center;
  font-size: 12px;
  font-family: ui-monospace, monospace;
  color: #64748b;
  margin-right: 52px;
}

/* Terminal Body */
.terminal-body {
  padding: 20px 24px;
  font-family: ui-monospace, 'Fira Code', monospace;
  font-size: 14px;
  line-height: 2;
}

.code-line {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  opacity: 0;
  animation: line-appear 0.5s ease forwards;
}

.line-1 {
  animation-delay: 0.3s;
}
.line-2 {
  animation-delay: 1s;
}
.line-3 {
  animation-delay: 1.8s;
}
.line-4 {
  animation-delay: 2.5s;
}

@keyframes line-appear {
  from {
    opacity: 0;
    transform: translateY(5px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.code-prompt {
  color: #22c55e;
  font-weight: bold;
}
.code-cmd {
  color: #38bdf8;
}
.code-flag {
  color: #a78bfa;
}
.code-url {
  color: #ff6b35;
}
.code-comment {
  color: #64748b;
  font-style: italic;
}
.code-success {
  color: #22c55e;
  background: rgba(34, 197, 94, 0.15);
  padding: 2px 8px;
  border-radius: 4px;
  font-weight: 600;
}
.code-response {
  color: #fbbf24;
}

/* Blinking Cursor */
.cursor {
  display: inline-block;
  width: 8px;
  height: 16px;
  background: #22c55e;
  animation: blink 1s step-end infinite;
}

@keyframes blink {
  0%,
  50% {
    opacity: 1;
  }
  51%,
  100% {
    opacity: 0;
  }
}

/* Dark mode adjustments */
:deep(.dark) .terminal-window {
  box-shadow:
    0 25px 50px -12px rgba(0, 0, 0, 0.6),
    0 0 0 1px rgba(255, 107, 53, 0.2),
    0 0 40px rgba(255, 107, 53, 0.1),
    inset 0 1px 0 rgba(255, 255, 255, 0.1);
}
</style>
