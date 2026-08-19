import { defineWebApplication } from '@opencloud-eu/web-pkg'
import { urlJoin } from '@opencloud-eu/web-client'
import translations from './l10n/translations.json'

export default defineWebApplication({
  setup(args) {
    const appInfo = {
      id: 'notes',
      name: () => translations.es['Notes'] || 'Notes',
      icon: 'sticky-note',
      color: '#ee7318',
    }
    const routes = [
      {
        path: '/',
        name: 'notes-root',
        component: () => import('./views/NotesApp.vue'),
        meta: { authContext: 'user' as const, title: translations.es['Notes'] || 'Notes' },
      },
    ]
    const extensions = ({ applicationConfig }) => []
    return { appInfo, routes, extensions: extensions(args), translations }
  },
})
