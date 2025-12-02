import { ref, watch } from 'vue'

export type ThemeMode = 'dark' | 'light'
export type LanguageCode = 'zh-CN' | 'en-US'

export interface AppSettings {
  theme: ThemeMode
  language: LanguageCode
  dataPath: string
  datasetExportPath: string
  modelOutputPath: string
}

const STORAGE_KEY = 'easymark_settings'

const defaultSettings: AppSettings = {
  theme: 'dark',
  language: 'zh-CN',
  dataPath: 'D:\\EasyMark\\Data',
  datasetExportPath: 'D:\\EasyMark\\Data\\database_out',
  modelOutputPath: 'D:\\EasyMark\\Data\\model_out'
}

const settingsRef = ref<AppSettings>(loadInitial())

function loadInitial(): AppSettings {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return { ...defaultSettings }
    const parsed = JSON.parse(raw) as Partial<AppSettings>
    return {
      ...defaultSettings,
      ...parsed
    }
  } catch {
    return { ...defaultSettings }
  }
}

watch(
  settingsRef,
  (value) => {
    try {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(value))
    } catch {
      // ignore persistence errors for now
    }
  },
  { deep: true }
)

export function useSettings() {
  const setTheme = (theme: ThemeMode) => {
    settingsRef.value.theme = theme
  }

  const setLanguage = (language: LanguageCode) => {
    settingsRef.value.language = language
  }

  const updatePaths = (paths: Partial<Pick<AppSettings, 'dataPath' | 'datasetExportPath' | 'modelOutputPath'>>) => {
    settingsRef.value = {
      ...settingsRef.value,
      ...paths
    }
  }

  return {
    settings: settingsRef,
    setTheme,
    setLanguage,
    updatePaths
  }
}
