import {
  defineWebApplication,
  ApplicationSetupOptions,
  Extension,
  AppMenuItemExtension
} from '@opencloud-eu/web-pkg'
import { urlJoin } from '@opencloud-eu/web-client'
import sdkStyles from '@opencloud-eu/extension-sdk/tailwind.css?inline'
import styles from './styles.css?inline'

function injectStyles() {
  for (const [id, css] of [
    ['ocnotes-sdk-styles', sdkStyles],
    ['ocnotes-styles', styles],
  ] as const) {
    if (document.getElementById(id)) continue
    const el = document.createElement('style')
    el.id = id
    el.textContent = css
    document.head.appendChild(el)
  }
}

injectStyles()
import { RouteRecordRaw } from 'vue-router'
import { computed } from 'vue'
import { useGettext } from 'vue3-gettext'
import translations from './l10n/translations.json'

export default defineWebApplication({
  setup(args) {
    const { $gettext } = useGettext()

    const appInfo = {
      id: 'notes',
      name: $gettext('Notes'),
      icon: 'sticky-note',
      color: '#ee7318'
    }

    // UNA sola ruta por path (issue #001): el host monta bajo /notes.
    const routes: RouteRecordRaw[] = [
      {
        path: '/',
        name: 'notes-root',
        component: () => import('./views/NotesApp.vue'),
        meta: {
          authContext: 'user' as const,
          title: $gettext('Notes')
        }
      }
    ]

    const extensions = ({ applicationConfig }: ApplicationSetupOptions) => {
      return computed<Extension[]>(() => {
        const menuItems: AppMenuItemExtension[] = [
          {
            id: `app.${appInfo.id}.menuItem`,
            type: 'appMenuItem',
            label: () => appInfo.name,
            color: appInfo.color,
            icon: appInfo.icon,
            path: urlJoin(appInfo.id)
          }
        ]
        return [...menuItems]
      })
    }

    return {
      appInfo,
      routes,
      extensions: extensions(args),
      translations
    }
  }
})
