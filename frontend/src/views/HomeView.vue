<template>
  <div v-if="hasHomeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    />
    <!-- The custom homepage remains an administrator-controlled override. -->
    <div v-else v-html="homeContent" />
  </div>

  <div
    v-else
    class="immersive-home dark"
    :data-testid="compactHomeEnabled ? 'compact-home' : 'default-home'"
  >
    <header class="home-header">
      <nav class="home-nav">
        <router-link to="/home" class="home-brand" :aria-label="siteName">
          <img :src="siteLogo || '/logo.svg'" alt="Logo" />
          <span>{{ siteName }}</span>
        </router-link>
        <div class="home-actions">
          <LocaleSwitcher />
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="nav-link"
            :aria-label="t('home.viewDocs')"
          >
            <Icon name="book" size="md" /><span class="hidden sm:inline">{{
              t('home.docs')
            }}</span>
          </a>
          <router-link
            v-if="showModelPlazaEntry"
            to="/model-plaza"
            class="nav-link"
            :aria-label="t('nav.modelPlaza')"
          >
            <Icon name="grid" size="md" /><span class="hidden sm:inline">{{
              t('nav.modelPlaza')
            }}</span>
          </router-link>
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="nav-login"
          >
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
            <Icon name="arrowRight" size="sm" />
          </router-link>
        </div>
      </nav>
    </header>

    <main>
      <NeuralFlowHero>
        <p class="hero-eyebrow"><span aria-hidden="true" />AI API GATEWAY</p>
        <h1>{{ siteName }}</h1>
        <p class="hero-subtitle">{{ siteSubtitle }}</p>
        <router-link
          :to="isAuthenticated ? dashboardPath : '/login'"
          class="hero-cta"
        >
          {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
          <Icon name="arrowRight" size="md" :stroke-width="2" />
        </router-link>
      </NeuralFlowHero>
    </main>

    <footer class="home-footer">&copy; {{ currentYear }} {{ siteName }}</footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import NeuralFlowHero from '@/components/home/NeuralFlowHero.vue'
import { sanitizeUrl } from '@/utils/url'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'

const { t } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()

// Site settings - directly from appStore (already initialized from injected config)
const siteName = computed(
  () =>
    appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API',
)
const siteLogo = computed(() =>
  sanitizeUrl(
    appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '',
    { allowRelative: true, allowDataUrl: true },
  ),
)
const siteSubtitle = computed(
  () =>
    appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform',
)
const docUrl = computed(() =>
  sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''),
)
const homeContent = computed(
  () => appStore.cachedPublicSettings?.home_content || '',
)
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)
const compactHomeEnabled = computed(
  () => appStore.cachedPublicSettings?.compact_home_enabled === true,
)
const modelPlazaEnabled = computed(() =>
  isFeatureFlagEnabled(FeatureFlags.modelPlaza),
)

// Check if homeContent is a URL (for iframe display)
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

// Auth state
const isAuthenticated = computed(() => authStore.isAuthenticated)
const modelPlazaRequiresAuth = computed(
  () => appStore.cachedPublicSettings?.model_plaza_require_auth === true,
)
const showModelPlazaEntry = computed(
  () =>
    modelPlazaEnabled.value &&
    (isAuthenticated.value || !modelPlazaRequiresAuth.value),
)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() =>
  isAdmin.value ? '/admin/dashboard' : '/dashboard',
)

// Current year for footer
const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  // Check auth state
  authStore.checkAuth()

  // Ensure public settings are loaded (will use cache if already loaded from injected config)
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>

<style scoped>
.immersive-home {
  --home-ink: #020617;
  --home-accent: #ff8a52;
  position: relative;
  isolation: isolate;
  min-height: 100svh;
  background: var(--home-ink);
  color: #f8fafc;
}
.home-header {
  position: absolute;
  z-index: 10;
  inset: 0 0 auto;
  padding: max(20px, env(safe-area-inset-top)) clamp(20px, 5vw, 96px) 20px;
  background: linear-gradient(var(--home-ink), transparent);
}
.home-nav,
.home-actions,
.home-brand,
.nav-link,
.nav-login {
  display: flex;
  align-items: center;
}
.home-nav {
  justify-content: space-between;
  gap: 16px;
}
.home-brand {
  gap: 12px;
  min-width: 0;
  font-size: 17px;
  font-weight: 600;
  letter-spacing: -0.03em;
}
.home-brand img {
  width: 36px;
  height: 36px;
  object-fit: contain;
  flex-shrink: 0;
}
.home-brand span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 240px;
}
.home-actions {
  gap: 12px;
  flex-shrink: 0;
}
.nav-link {
  gap: 8px;
  min-width: 44px;
  min-height: 44px;
  justify-content: center;
  font-size: 13px;
  color: #cbd5e1;
}
.nav-login {
  gap: 20px;
  min-height: 44px;
  padding: 0 18px;
  border: 1px solid #ffffff35;
  border-radius: 999px;
  font-size: 13px;
  color: #f8fafc;
  background: #ffffff06;
}
.nav-link,
.nav-login {
  transition:
    color 180ms,
    background 180ms;
}
.nav-link:hover,
.nav-login:hover {
  color: var(--home-accent);
  background: #ffffff0a;
}
.hero-eyebrow {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 28px;
  font-size: 12px;
  letter-spacing: 0.28em;
  color: #ffad80;
}
.hero-eyebrow span {
  width: 32px;
  height: 1px;
  background: var(--home-accent);
  box-shadow: 0 0 12px #ff8a5270;
}
h1 {
  margin: 0 0 28px;
  font-size: clamp(56px, 7.6vw, 144px);
  font-weight: 650;
  letter-spacing: -0.065em;
  line-height: 1.06;
  overflow-wrap: anywhere;
  color: #fff8f3;
}
.hero-subtitle {
  max-width: 440px;
  margin-bottom: 40px;
  font-size: clamp(16px, 1.3vw, 21px);
  line-height: 1.8;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  color: #a7b2c6;
}
.hero-cta {
  display: inline-flex;
  align-items: center;
  justify-content: space-between;
  gap: 36px;
  min-height: 56px;
  padding: 0 28px;
  border: 1px solid #ffb48680;
  border-radius: 999px;
  background: linear-gradient(120deg, #ffb27c, #ff8044);
  box-shadow: 0 0 40px #ff6b351a;
  color: #201008;
  font-size: 15px;
  font-weight: 650;
  transition:
    box-shadow 200ms,
    transform 200ms;
}
.hero-cta:hover {
  box-shadow: 0 0 56px #ff6b3540;
  transform: translateY(-2px);
}
a:focus-visible {
  outline: 2px solid var(--home-accent);
  outline-offset: 5px;
}
.home-footer {
  position: absolute;
  z-index: 2;
  left: clamp(20px, 5vw, 96px);
  bottom: max(28px, env(safe-area-inset-bottom));
  max-width: calc(100% - 110px);
  font-size: 12px;
  letter-spacing: 0.04em;
  color: #8290a8;
  overflow-wrap: anywhere;
  pointer-events: none;
}
@media (max-width: 767px) {
  .home-brand span {
    display: none;
  }
  .home-actions {
    gap: 4px;
  }
  .nav-login {
    padding: 0 14px;
    gap: 8px;
  }
  .hero-eyebrow {
    margin-bottom: 20px;
    font-size: 11px;
  }
  h1 {
    font-size: clamp(50px, 13vw, 86px);
    margin-bottom: 20px;
  }
  .hero-subtitle {
    max-width: 320px;
    font-size: 16px;
    margin-bottom: 28px;
  }
  .hero-cta {
    min-height: 52px;
    padding: 0 24px;
  }
}
@media (prefers-reduced-motion: reduce) {
  .hero-cta {
    transition: none;
  }
  .hero-cta:hover {
    transform: none;
  }
}
</style>
