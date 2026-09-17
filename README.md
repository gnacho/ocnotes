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

### The easy way (OpenCloud App Store)

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

### Build from source

```bash
cd extension
pnpm install && pnpm build
```

Copy `dist/` to the OpenCloud apps folder as `notes/`.

### Backend (ocapps)

The backend lives in the [ocapps](https://github.com/gnacho/ocapps) repo.
See its `deploy/README.md` for the full installation guide (systemd,
environment variables, reverse proxy snippets).

Quick reference for the proxy:

```nginx
location /index.php/apps/notes/ {
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_pass http://127.0.0.1:8096;
}
```

## Store package / release zip

Releases are built automatically by GitHub Actions when a tag `notes-v*` is pushed.
The workflow installs dependencies, builds the extension, creates the zip with the
official layout (`notes/manifest.json` + `notes/js/remoteEntry-*.mjs`), and attaches it
to the GitHub release.

## License

AGPL-3.0. See [LICENSE](LICENSE).
