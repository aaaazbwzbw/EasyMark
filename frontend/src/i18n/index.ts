import { createI18n } from 'vue-i18n'
import zhCN from '../locales/zh-CN'
import enUS from '../locales/en-US'
import type { App } from 'vue'

export const SUPPORTED_LOCALES = ['zh-CN', 'en-US'] as const
export type SupportedLocale = (typeof SUPPORTED_LOCALES)[number]

export const messages = {
  'zh-CN': zhCN,
  'en-US': enUS
}

export function createAppI18n(locale: SupportedLocale) {
  return createI18n({
    legacy: false,
    locale,
    fallbackLocale: 'en-US',
    messages
  })
}

export function setupI18n(app: App, locale: SupportedLocale) {
  const i18n = createAppI18n(locale)
  app.use(i18n)
  return i18n
}
