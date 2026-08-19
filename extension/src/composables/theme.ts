import { computed } from 'vue'
import { useThemeStore } from '@opencloud-eu/web-pkg'

export function useIsDark() {
  try {
    const store = useThemeStore()
    return computed(() => store.currentTheme?.isDark ?? false)
  } catch {
    return computed(() => window.matchMedia('(prefers-color-scheme: dark)').matches)
  }
}
