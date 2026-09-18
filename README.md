# gmail-planner

Go CLI that reads Gmail (and optionally Google Calendar), builds an OpenAI family digest, and sends it by email. This document focuses on **configuring Google Cloud and OAuth** so the app has the API access it needs.

## Command-line flags

All flags are defined in [`src/cmd/main.go`](src/cmd/main.go). Run from `src/` (e.g. `go run ./cmd …`) or pass the same arguments to the built binary / Docker image.

| Flag | Type | Default | Description |
| ---- | ---- | ------- | ----------- |
| `-digest` | bool | `false` | **Digest mode:** list Gmail messages (using `-n` and optionally `-q`), optionally fetch Google Calendar events for `GMAIL_CALENDAR_ID` over the next `DIGEST_WEEKS`, summarize with OpenAI, then send a multipart HTML digest to `DIGEST_TO_EMAIL`. Requires digest-related variables in `.env` (see [`src/.env-example`](src/.env-example)). |
| `-print-summary` | bool | `false` | Only meaningful **with `-digest`:** after the digest email is sent, print the plain-text body to stdout. |
| `-q` | string | `""` | **Only in digest mode:** Gmail search query. When non-empty, overrides `GMAIL_DIGEST_QUERY` for that run. Syntax matches [Gmail search operators](https://support.google.com/mail/answer/7190) (e.g. `newer_than:14d`). When empty, the digest uses `GMAIL_DIGEST_QUERY` from `.env`; if that is empty too, messages are listed from **INBOX** with no extra search filter (still capped by `-n`). If `GMAIL_DIGEST_LABELS` is set, each label runs as a **separate** list (base query + one `label:` term per label, up to `-n` messages per label); results are merged, deduplicated by message ID, then summarized together with label attribution. |
| `-n` | int | `10` | Maximum number of inbox messages to fetch (`1`–`500`). Applies to **default mode** (JSON dump) and **`-digest`**. |

**Default mode** (no `-digest`): fetches up to `-n` messages from the inbox (no `-q`), writes a JSON array of message summaries to stdout, then exits.

**Digest mode** (`-digest`): fetches inbox messages using the base query and optional per-label searches (see `-q` / `GMAIL_DIGEST_LABELS`), fetches upcoming events from `GMAIL_CALENDAR_ID` when set (window = now → +`DIGEST_WEEKS`), runs summarization (emails + calendar under a **Kalender** section), sends email, then optionally prints the text digest if `-print-summary` is set.

---

## What you need in Google Cloud

1. A **Google Cloud project** [https://console.cloud.google.com/](https://console.cloud.google.com/)
2. **Gmail API** and **Google Calendar API** enabled for that project
3. An **OAuth consent screen** with the Gmail and Calendar scopes your app uses
4. An **OAuth 2.0 Client ID** of type **Desktop app**
5. The **authorized redirect URI** matching your app (default: `http://127.0.0.1:8765/`)

The app uses these OAuth scopes (see [Gmail API scopes](https://developers.google.com/gmail/api/auth/scopes) and [Calendar API scopes](https://developers.google.com/calendar/api/auth)):

| Scope | Why |
| ----- | --- |
| `https://www.googleapis.com/auth/gmail.readonly` | List and read messages |
| `https://www.googleapis.com/auth/gmail.send` | Send the digest email (`-digest` mode) |
| `https://www.googleapis.com/auth/calendar.readonly` | Read events from `GMAIL_CALENDAR_ID` for the digest |

If you only ever run the JSON inbox dump and never `-digest`, you could in theory use read-only Gmail only—but the codebase requests **all three** scopes so digest send and calendar work without a separate build.

---

## Step-by-step: Google Cloud Console

Sign in at [Google Cloud Console](https://console.cloud.google.com/) with the Google account that will **own** the OAuth client (often the same account whose Gmail you read).

### 1. Create or pick a project

- Top bar: **project selector** → **New project** (or choose an existing project).
- Note the **project number** if you need it for support links.

### 2. Enable the Gmail API and Google Calendar API

- **APIs & services** → **Library**
- Search for **Gmail API** → **Enable**
- Search for **Google Calendar API** → **Enable**

If either step is skipped, you may see errors like `SERVICE_DISABLED` / `accessNotConfigured` when calling that API.

### 3. Configure the OAuth consent screen

- **APIs & services** → **OAuth consent screen**
- Choose **External** (typical for personal / family use) unless you are on Google Workspace and use **Internal**.
- Fill **App name**, **User support email**, **Developer contact**
- On **Scopes** (or **Edit app** → **Scopes** / **Add or remove scopes**):

  - **Add scopes** → filter for **Gmail** / **Calendar**
  - Add at least:
    - `.../auth/gmail.readonly`
    - `.../auth/gmail.send`
    - `.../auth/calendar.readonly`
  - Save

- **Test users**: while the app is in **Testing** publishing status, only listed Google accounts can complete sign-in. Add every family member account that will run OAuth, or **Publish** the app (stricter verification may apply for sensitive scopes at scale).

### 4. Create OAuth client credentials

- **APIs & services** → **Credentials** → **Create credentials** → **OAuth client ID**
- If prompted, configure the consent screen first (step 3).
- Application type: **Desktop app**
- Name it (e.g. `gmail-planner-desktop`)
- Create → copy **Client ID** and **Client secret** into your `.env` as `GMAIL_CLIENT_ID` and `GMAIL_CLIENT_SECRET` (see `src/.env-example`).

### 5. Authorized redirect URI (Desktop flow)

This app runs a local HTTP listener for the OAuth callback.

- Default redirect: `http://127.0.0.1:8765/` (including trailing slash if you use that in code and in Cloud Console—**they must match exactly**).
- For **Desktop** clients, Google often lets you use `http://localhost` patterns; this project defaults to `127.0.0.1:8765`. In **Credentials** → your **OAuth 2.0 Client ID** → **Authorized redirect URIs**, add:

  `http://127.0.0.1:8765/`

If you change `GMAIL_OAUTH_REDIRECT_URL` in `.env`, add that exact URI here too.

### 6. First run and token file

From `src/` (see `src/cmd/main.go`):

```bash
go run ./cmd
```

The first run opens a browser URL in the logs; after you consent, the app saves a token file (default `gmail-token.json` next to the working directory, or `GMAIL_TOKEN_PATH`).

### 7. After you add or change scopes

If you already had a token and then add **`gmail.send`**, **`calendar.readonly`**, or change scopes on the consent screen:

1. Delete the saved token file (`gmail-token.json` or your `GMAIL_TOKEN_PATH`).
2. Run the app again and complete the browser consent flow so the new refresh token includes the new scopes.

---

## OpenAI digest mode (`-digest`)

Digest mode sends email **through your Gmail account** to `DIGEST_TO_EMAIL` and calls **OpenAI** with message text (and calendar events when configured). You need:

- `OPENAI_API_KEY` and `DIGEST_TO_EMAIL` in `.env` (see `src/.env-example`)
- Optional: `GMAIL_DIGEST_LABELS` — comma-separated Gmail label names; digest mode runs **one Gmail list per label** (each uses `GMAIL_DIGEST_QUERY` / `-q` as base plus that label, capped by `-n` **per search**), merges unique messages, then summarizes in one pass with label references
- Optional: `GMAIL_CALENDAR_ID` — Google Calendar ID for the shared family calendar (often the owner's email, e.g. `tedochjohanna@gmail.com`). When set, digest mode lists events from **now** through **`DIGEST_WEEKS`** (default 1 week) and includes them under **Kalender**. When unset, calendar fetch is skipped with a warning and the digest continues with email only.
- Gmail send + Calendar readonly scopes enabled and a fresh token (steps above)

Example:

```bash
cd src
go run ./cmd -digest -n 25
go run ./cmd -digest -n 25 -print-summary
go run ./cmd -digest -n 25 -q "newer_than:7d"
```

See [Command-line flags](#command-line-flags) for all options.

---

## Docker and Docker Compose

From the **repository root** (where `docker-compose.yml` lives):

```bash
docker compose down -v
docker compose build
```

- The image is defined in [`docker/Dockerfile`](docker/Dockerfile); Compose uses the **repository root** as build context. Context exclusions are in the repo-root [`.dockerignore`](.dockerignore) (Docker reads that path relative to the context, not from inside `docker/`).
- Put a **`.env`** next to `docker-compose.yml` (Compose loads it when present; the file is optional for `docker compose build`). Copy from [`src/.env-example`](src/.env-example), save as `.env` in the repo root, and fill in values before `docker compose run`.
- Compose mounts **`./src` → `/app/src`** so the container uses the same **`src/gmail-token.json`** as when you run `go run ./cmd` from `src/` (see `GMAIL_TOKEN_PATH` in `docker-compose.yml`).

Run digest (example):

```bash
docker compose run --rm gmail-planner -digest -n 25
```

Other flags work the same way, e.g. `-print-summary`, `-q "newer_than:7d"`.

**OAuth inside Docker:** The callback server listens on the host from `GMAIL_OAUTH_REDIRECT_URL` (default `http://127.0.0.1:8765/`). In a normal container that binds **loopback inside the container**, a browser on your machine usually **cannot** complete the redirect. **Recommended:** run `go run ./cmd` (or the built binary) **on the host** once to obtain `src/gmail-token.json`; Compose reuses that path via the `./src` volume mount. On **Linux**, you can use `network_mode: host` on the service if you need the browser flow to hit the process inside Docker (not portable to Docker Desktop on Windows/macOS the same way).

The image default `CMD` is `-digest`; `docker compose run` arguments override that behavior as usual.

---

## Troubleshooting

| Symptom | What to check |
| ------- | ------------- |
| `SERVICE_DISABLED` / Gmail or Calendar API error link | Enable **Gmail API** and **Google Calendar API** for the correct project (same project as the OAuth client). |
| `redirect_uri_mismatch` | Redirect URI in Google Cloud must match `GMAIL_OAUTH_REDIRECT_URL` / default exactly. |
| `access_denied` / consent screen | Add the Google account as a **test user**, or publish the app. |
| Send fails after code changes | Delete token file and re-consent; confirm **`gmail.send`** is on the consent screen and in your OAuth client’s allowed scopes. |
| Calendar list fails / insufficient scopes | Delete token file and re-consent; confirm **`calendar.readonly`** is on the consent screen. Confirm `GMAIL_CALENDAR_ID` is the shared calendar’s ID (often the owner’s email). |

---

## Repo layout

- `src/cmd` — CLI entrypoint  
- `src/internal/gmail` — Gmail OAuth, list/detail, send  
- `src/internal/calendar` — Google Calendar event list  
- `src/internal/digest` — OpenAI summarization  
- `src/internal/config` — env loading  
- `docker/` — `Dockerfile` for the container image  

Copy `src/.env-example` to `src/.env` and fill in secrets; `.env` is gitignored.
