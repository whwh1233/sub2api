<template>
  <AppLayout>
    <section class="mx-auto flex min-h-[calc(100dvh-7rem)] max-w-7xl flex-col gap-5">
      <header class="space-y-4">
        <div class="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
          <div>
            <p class="text-sm font-semibold text-primary-600 dark:text-primary-400">CloseAI</p>
            <h1 class="mt-1 text-3xl font-bold tracking-normal text-gray-950 dark:text-white">
              {{ t('community.title') }}
            </h1>
          </div>
          <p class="max-w-2xl text-base leading-7 text-gray-600 dark:text-dark-300">
            {{ t('community.description') }}
          </p>
        </div>

        <div class="grid gap-3 lg:grid-cols-[1.05fr_1fr]">
          <article
            data-test="community-notice-intro"
            class="flex gap-4 rounded-2xl border border-primary-100 bg-primary-50/80 p-4 dark:border-primary-500/20 dark:bg-primary-500/10"
          >
            <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-white text-primary-600 shadow-sm dark:bg-primary-500/15 dark:text-primary-300">
              <Icon name="users" size="lg" />
            </div>
            <div>
              <h2 class="text-base font-semibold text-gray-950 dark:text-white">
                {{ t('community.introTitle') }}
              </h2>
              <p class="mt-1 text-sm leading-6 text-gray-700 dark:text-dark-200">
                {{ t('community.introText') }}
              </p>
            </div>
          </article>

          <article
            data-test="community-notice-tip"
            class="flex gap-4 rounded-2xl border border-amber-200 bg-amber-50 p-4 dark:border-amber-500/25 dark:bg-amber-500/10"
          >
            <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-white text-amber-700 shadow-sm dark:bg-amber-500/15 dark:text-amber-300">
              <Icon name="exclamationTriangle" size="lg" />
            </div>
            <div>
              <h2 class="text-base font-semibold text-gray-950 dark:text-white">
                {{ t('community.tipTitle') }}
              </h2>
              <p class="mt-1 text-sm leading-6 text-gray-700 dark:text-dark-200">
                {{ t('community.tipText') }}
              </p>
            </div>
          </article>
        </div>
      </header>

      <div data-test="community-qr-grid" class="grid flex-1 gap-5 lg:grid-cols-3">
        <figure
          v-for="channel in communityChannels"
          :key="channel.key"
          data-test="community-qr-card"
          class="card flex min-h-[520px] flex-col overflow-hidden p-4 md:p-5 lg:min-h-0"
        >
          <div class="flex items-center justify-between gap-3 pb-4">
            <div>
              <h2 class="text-lg font-semibold text-gray-950 dark:text-white">
                {{ t(channel.titleKey) }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
                {{ t(channel.captionKey) }}
              </p>
            </div>
            <span class="shrink-0 rounded-full bg-gray-100 px-3 py-1 text-xs font-semibold text-gray-700 dark:bg-dark-700 dark:text-dark-200">
              {{ t('community.scanLabel') }}
            </span>
          </div>

          <div class="flex min-h-[380px] flex-1 items-center justify-center rounded-2xl border border-gray-100 bg-gray-50 p-3 dark:border-dark-700 dark:bg-white">
            <img
              :src="channel.qr"
              :alt="channel.alt"
              class="h-full max-h-[72vh] w-full max-w-[520px] rounded-xl object-contain"
              loading="lazy"
            />
          </div>

          <figcaption class="pt-4 text-center">
            <p class="text-sm font-medium leading-6 text-gray-700 dark:text-dark-200">
              {{ t(channel.hintKey) }}
            </p>
          </figcaption>
        </figure>
      </div>
    </section>
  </AppLayout>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import wechatQr from '@/assets/community-wechat-qr.png'
import qqQr from '@/assets/community-qq-qr.jpg'
import telegramQr from '@/assets/community-telegram-qr.png'

const { t } = useI18n()

const communityChannels = [
  {
    key: 'wechat',
    titleKey: 'community.channels.wechat.title',
    captionKey: 'community.channels.wechat.caption',
    hintKey: 'community.channels.wechat.hint',
    qr: wechatQr,
    alt: 'CloseAI 天使用户服务群二维码',
  },
  {
    key: 'qq',
    titleKey: 'community.channels.qq.title',
    captionKey: 'community.channels.qq.caption',
    hintKey: 'community.channels.qq.hint',
    qr: qqQr,
    alt: 'CloseAI 使用答疑群二维码',
  },
  {
    key: 'telegram',
    titleKey: 'community.channels.telegram.title',
    captionKey: 'community.channels.telegram.caption',
    hintKey: 'community.channels.telegram.hint',
    qr: telegramQr,
    alt: 'CloseAI Telegram 交流群二维码',
  },
] as const
</script>
