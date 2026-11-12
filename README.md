# ocnotes

Markdown notes for OpenCloud. It implements the Nextcloud Notes API (v1 and v1.4), so
clients like [Iotas](https://github.com/nerzhul/iotas) work against it, and it ships
as a web extension that lives inside the OpenCloud UI.

> **This is beta software.** It is in active development, APIs and behavior
> may change, and you may hit bugs. Point it at your test server first and
> keep a backup of the data directory.

## What works today

**Backend** (part of [ocapps](https://github.com/gnacho/ocapps), a single static Go binary, SQLite, no external services):

- Nextcloud Notes API v1/v1.4: create, read, update, delete notes
- **Multi-user**: each OpenCloud account sees and edits only its own notes. No configuration needed.
- Categories, favorites and search across titles and content
- Attachments with signed URLs
- Spanish/English UI negotiated per user

**Web extension** (Vue 3, installed like any OpenCloud web app):

- Markdown editor with live preview
- Notes list with category filter and favorites
- Search across titles and content
- Create, edit, delete and categorize from the UI

## Architecture

OpenCloud web extensions are frontend only, so the notes engine lives in a
companion service. Since 2026, the backend is part of `ocapps`, a unified
server that also covers News and ocphotos:

```
┌─────────────────────────────────────────────────────────────┐
│ OpenCloud Web                                               │
│   web-app-notes (Vue 3 extension, in the app switcher)      │
│        │  same-origin REST, session auth                    │
└────────┼────────────────────────────────────────────────────┘
         │  /index.php/apps/notes/api/v1/   (reverse proxy)
         ▼
    ocapps backend (Go, single binary)
    • Notes API v1.4       • Markdown storage (SQLite)
    • Multi-user scoping   • OpenCloud Graph auth
    • Attachments          • Search
```

## Install

Notes is **two pieces** and both are required:

1. The **backend** (`ocapps`), a Go service that serves the Notes API. Without it
   the web app opens and every request fails with `Request failed with status code 404`.
2. The **web extension** (the `notes-X.Y.Z.zip` release asset), which is the UI
   inside OpenCloud.

Installing only the extension is the most common mistake, so start with the
backend.

### 1. Backend (`ocapps`), required

It runs next to OpenCloud on the same server, listens on port 8096 and
authenticates against OpenCloud's Graph API. The one-command installer sets up
the service user, the binary, the environment file, the systemd unit and the
extension, with `--apps notes` to skip the News and Photos ones:

```bash
curl -fsSL https://raw.githubusercontent.com/gnacho/ocapps/main/install.sh \
  | sudo sh -s -- --url https://cloud.example.com --apps notes
```

It does not touch your reverse proxy, so these two locations are on you.
OpenCloud does not forward them itself.

```nginx
location /index.php/apps/notes/ {
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_pass http://127.0.0.1:8096;
}

location = /ocs/v2.php/cloud/capabilities {
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_pass http://127.0.0.1:8096;
}
```

OpenCloud also has to see the extension folder: `WEB_ASSET_APPS_PATH` must point
to it (`/etc/opencloud/web/assets/apps` on a standard install).

Full guide, Docker Compose deployment and Caddy/Traefik equivalents:
[`ocapps/deploy/README.md`](https://github.com/gnacho/ocapps/blob/main/deploy/README.md).

Check the backend before blaming the proxy:

```bash
systemctl is-active opencloud-apps
curl -i http://127.0.0.1:8096/healthz
```

### 2. Web extension

#### Easy way (`ocapps-ext`)

The extension ships as a release asset and can be installed and updated from the
server with [`ocapps-ext`](https://github.com/gnacho/ocapps/blob/main/deploy/README.md):

```bash
ocapps-ext install notes     # first install: download, apps.yaml entry, restart, verify
ocapps-ext update            # update every app that came from a release
ocapps-ext update --dry-run  # show what would change, touch nothing
ocapps-ext rollback notes    # restore the previous directory
```

It picks the newest release whose asset matches the app, replaces the folder
under `/etc/opencloud/web/assets/apps`, restarts OpenCloud once and checks that
the app appears in `/config.json` with a reachable entry point. If that check
fails it restores the previous build on its own. Directories built by hand are
marked `local:<date>` and are never overwritten unless you pass `--force`.

#### Manual install

Download the latest release zip from the [releases page](https://github.com/gnacho/ocnotes/releases):

```bash
# Download notes-X.Y.Z.zip and extract to your OpenCloud apps folder
# (commonly /var/lib/opencloud/web/assets/apps or /etc/opencloud/web/assets/apps)
```

Add to `/etc/opencloud/apps.yaml`:

```yaml
notes:
  config: {}
```

Restart OpenCloud. The Notes app appears in the app switcher.

#### Build from source

```bash
cd extension
pnpm install && pnpm build
```

Copy `dist/` to the OpenCloud apps folder as `notes/`.

## Mobile app (Android)

Notes has a native Android client: [ocnotes-android](https://github.com/gnacho/ocnotes-android)
(Kotlin + Jetpack Compose, AGPL-3.0). It talks to the same Notes API v1.4 served by
`ocapps`, so it works with this backend and with any Nextcloud Notes server.

- Download the signed APK from the [latest release](https://github.com/gnacho/ocnotes-android/releases/latest).
- In the app, enter your OpenCloud server URL, your user name and an **app token**
  (OpenCloud settings, App tokens, or `opencloud auth-app create`). A password will
  not work.
- The server path is built automatically as `/index.php/apps/notes/api/v1/`, so the
  server URL must be the base one (`https://cloud.example.com`).
- Offline-first: notes are cached locally and pending changes are pushed when the
  connection is back. Signing out deletes the local cache, so copy anything unsaved
  before doing it.

## Store package / release zip

Releases are built automatically by GitHub Actions when a tag `notes-v*` is pushed.
The workflow installs dependencies, builds the extension, creates the zip with the
official layout (`notes/manifest.json` + `notes/js/remoteEntry-*.mjs`), and attaches it
to the GitHub release.

## License

AGPL-3.0. See [LICENSE](LICENSE).
