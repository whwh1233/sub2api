<template>
  <AppLayout>
    <section class="mx-auto max-w-5xl space-y-6">
      <header class="page-header">
        <p class="text-sm font-medium text-primary-600 dark:text-primary-400">CloseAI</p>
        <h1 class="page-title">{{ t('community.title') }}</h1>
        <p class="page-description max-w-2xl">{{ t('community.description') }}</p>
      </header>

      <div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_360px]">
        <div class="space-y-6">
          <article class="card p-6 md:p-8">
            <div class="mb-5 flex h-12 w-12 items-center justify-center rounded-xl bg-primary-50 text-primary-600 dark:bg-primary-500/10 dark:text-primary-300">
              <Icon name="users" size="lg" />
            </div>
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
              {{ t('community.introTitle') }}
            </h2>
            <p class="mt-3 max-w-2xl text-base leading-7 text-gray-600 dark:text-dark-300">
              {{ t('community.introText') }}
            </p>
          </article>

          <article class="card border-amber-100 bg-amber-50/70 p-6 shadow-sm dark:border-amber-500/20 dark:bg-amber-500/10 md:p-8">
            <div class="mb-5 flex h-12 w-12 items-center justify-center rounded-xl bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300">
              <Icon name="exclamationTriangle" size="lg" />
            </div>
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
              {{ t('community.tipTitle') }}
            </h2>
            <p class="mt-3 max-w-2xl text-base leading-7 text-gray-700 dark:text-dark-200">
              {{ t('community.tipText') }}
            </p>
          </article>
        </div>

        <aside class="card p-4 md:p-5">
          <div class="rounded-xl border border-gray-100 bg-white p-3 shadow-sm dark:border-dark-700 dark:bg-white">
            <img
              :src="communityQr"
              alt="CloseAI 天使用户服务群二维码"
              class="aspect-square w-full rounded-lg object-contain"
              loading="lazy"
            />
          </div>
          <p class="mt-4 text-center text-sm font-medium text-gray-900 dark:text-white">
            {{ t('community.qrCaption') }}
          </p>
          <p class="mt-2 text-center text-xs leading-5 text-gray-500 dark:text-dark-400">
            {{ t('community.scanHint') }}
          </p>
        </aside>
      </div>
    </section>
  </AppLayout>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import communityQr from '@/assets/community-wechat-qr.png'

const { t } = useI18n()
</script>
