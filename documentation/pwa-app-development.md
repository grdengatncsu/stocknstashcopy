# PWA App Development Guide

This document is for contributors building the Stock 'n Stash user
application. It explains the agreed architecture, the proposed frontend tool
stack, and an implementation order that keeps online, offline, and security
behavior consistent.

The detailed data models and API behavior are defined in
[`pi-api-contract.md`](pi-api-contract.md). If an example in this guide
conflicts with that contract, follow the contract and update this guide.

## Current Status

The repository defines the target app architecture, but an `app/` project has
not yet been added. The architecture in the **Confirmed** column below is
already recorded in the API contract. Tooling labeled **Proposed** is a
recommended baseline and should be confirmed by the app team before the first
scaffold is committed.

## System Context

```text
Python edge pipeline
        |
        | local HTTP/JSON
        v
Go service on the Pi ---- SQLite and synchronization outbox
        |
        | authenticated synchronization
        v
Supabase: PostgreSQL + Auth + RLS + Realtime
        ^
        | HTTPS
        |
React / TypeScript PWA ---- IndexedDB offline state and operation queue
```

The production PWA communicates with Supabase. Direct communication with the
Pi is reserved for same-network development and demonstrations. The browser
must never receive a privileged database or device credential.

## Software Stack

| Layer | Technology | Status | Purpose |
| --- | --- | --- | --- |
| UI | React | Confirmed | Component-based inventory and review screens |
| Language | TypeScript | Confirmed | Type checking for API records and component inputs |
| App format | Progressive Web App | Confirmed | Installable web application with an offline shell |
| Cloud data | Supabase PostgreSQL | Confirmed | Shared household inventory and pending results |
| Authentication | Supabase Auth | Confirmed | User sessions and household identity |
| Authorization | PostgreSQL Row Level Security | Confirmed | Restrict records to authorized household members |
| Live updates | Supabase Realtime | Confirmed | Receive accepted changes while online |
| Offline storage | IndexedDB | Confirmed | Cache household data and pending operations |
| Local device API | Go HTTP/JSON service | Confirmed | Same-network Pi development and scan ingestion |
| Runtime | Current Node.js LTS | Proposed | Local frontend tooling and package scripts |
| Package manager | npm | Proposed | Dependency and script management |
| Build tool | Vite | Proposed | React/TypeScript development and production builds |
| PWA tooling | `vite-plugin-pwa` / Workbox | Proposed | Manifest generation and service-worker lifecycle |
| Routing | React Router | Proposed | Client-side navigation between app screens |
| Server-state helpers | TanStack Query | Proposed | Loading, error, cache, and retry behavior |
| IndexedDB wrapper | Dexie | Proposed | Typed IndexedDB tables and transactions |
| Unit/component tests | Vitest + React Testing Library | Proposed | Fast logic and accessible UI tests |
| Browser tests | Playwright | Proposed | Installation, offline, synchronization, and user flows |

Do not add competing libraries for the same responsibility without discussing
the choice in the pull request. Once the app team confirms the proposed rows,
change their status to **Confirmed**.

## Required Product Screens

The first complete app should provide:

1. **Sign in / join household** — authenticate and join the correct shared
   household without exposing a device credential.
2. **Inventory** — list active inventory records and show quantity, UPC, and
   expiration date when known.
3. **Add item** — manually add an inventory item.
4. **Edit item** — change supported fields using the record's current version.
5. **Pending review** — accept or correct uncertain recognition results.
6. **Synchronization status** — show `offline`, `pending`, `syncing`, `synced`,
   or `failed`, including the last successful synchronization time.
7. **Settings / device pairing** — manage the household session and the
   one-time product pairing flow when that backend behavior is implemented.

Normal inventory views must hide records with a non-null `deleted_at` value.
Pending results stay visible until they are resolved or deleted.

## Before Creating the App

App contributors need:

- Git configured as described in [`CONTRIBUTING.md`](../CONTRIBUTING.md)
- A current Node.js LTS release
- npm, which is included with Node.js
- Access to the team's development Supabase project
- The approved app stack choices listed at the end of this document

Verify local tools:

```bash
node --version
npm --version
git --version
```

## One-Time Scaffold

Only the contributor assigned to create the initial app should run this. If an
`app/` directory exists, use the existing project instead.

After Vite and npm are confirmed:

```bash
npm create vite@latest app -- --template react-ts
cd app
npm install
```

Install only dependencies that the team has confirmed. A likely baseline is:

```bash
npm install @supabase/supabase-js react-router-dom
npm install @tanstack/react-query dexie
npm install -D vite-plugin-pwa vitest @testing-library/react
npm install -D @testing-library/jest-dom @testing-library/user-event playwright
```

The initial scaffold pull request should contain the build configuration,
formatting and linting rules, empty app shell, test setup, PWA manifest, and
contributor commands. It should not also implement every product screen.

## Proposed App Organization

```text
app/
├── public/
│   └── icons/                 PWA icons and non-generated public assets
├── src/
│   ├── app/                   Router, providers, and application shell
│   ├── components/            Reusable presentational components
│   ├── features/
│   │   ├── auth/              Sign-in and household access
│   │   ├── inventory/         Inventory list, add, edit, and delete
│   │   ├── pending/           Recognition review and correction
│   │   └── sync/              Queue processing and status display
│   ├── lib/
│   │   ├── supabase.ts        Configured browser client
│   │   ├── database.ts        IndexedDB schema and helpers
│   │   └── connectivity.ts    Online/offline signals
│   ├── types/                 Shared TypeScript domain types
│   ├── test/                  Test setup and shared fixtures
│   └── main.tsx               Browser entry point
├── .env.example               Variable names with non-secret examples
├── package.json               Scripts and exact dependencies
├── tsconfig.json              TypeScript configuration
└── vite.config.ts             Build, test, and PWA configuration
```

Organize code by product feature rather than placing every component, API call,
and test into one large folder. Keep network and storage operations outside UI
components so they can be tested independently.

## Environment Variables

The browser client needs the Supabase project URL and a publishable key. With
Vite, local values normally live in `app/.env.local`:

```dotenv
VITE_SUPABASE_URL=https://example.supabase.co
VITE_SUPABASE_PUBLISHABLE_KEY=replace-with-development-publishable-key
```

Commit an `.env.example` containing names and placeholders, but never commit
`.env.local` or real credentials. Add all local environment files to
`.gitignore` before development begins.

Only the Supabase publishable browser key belongs in the PWA. Never place a
Supabase secret key, service-role key, database password, Pi device credential,
product pairing code, or household join code in source code or committed files.

## Use the Shared Contract as the Type Source

Start with TypeScript types that match `pi-api-contract.md`. Important domain
objects include:

- `InventoryItem`
- `PendingResult`
- `ScanReport` and `ScanReportItem`
- `SynchronizationOperation`
- API error and list-response envelopes
- Synchronization state

Preserve distinctions in the contract:

- Optional unknown values are `null`, not empty strings.
- IDs are UUID strings except the separately defined scan and association IDs.
- Timestamps are RFC 3339/ISO 8601 UTC strings.
- `version` is required for optimistic concurrency.
- `deleted_at` is a tombstone; deletion is not the same as losing a record.

Once the Supabase schema exists, generate TypeScript database types from the
schema and keep domain types aligned with them. Do not maintain several
different hand-written shapes for the same record.

## Supabase Client and Security Boundary

Create one configured Supabase browser client in `src/lib/supabase.ts` and
import it where needed. Do not create a new client inside every React component.

Every exposed table must have appropriate grants and Row Level Security enabled.
Policies must restrict reads and writes to the signed-in user's household. A
hidden button or route guard is not authorization; database policy is the
security boundary.

The app should:

- Restore and refresh the Supabase Auth session.
- Block household screens until the session is known.
- Handle expired or revoked sessions without discarding unsynchronized local
  operations.
- Unsubscribe from Realtime channels when the user signs out or changes
  households.
- Never log tokens, join codes, or sensitive record contents unnecessarily.

## PWA Requirements

A PWA is more than a responsive website. The app must include:

- A web app manifest linked from the HTML document
- App name, short name, theme colors, display mode, start URL, and icons
- HTTPS in production
- A registered service worker with an intentional update strategy
- An offline application shell
- Usable layouts and touch targets on supported phones
- Visible feedback when the browser is offline or data is unsynchronized

Example manifest fields:

```json
{
  "name": "Stock 'n Stash",
  "short_name": "Stock 'n Stash",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#ffffff",
  "theme_color": "#1f6f4a",
  "icons": [
    {
      "src": "/icons/icon-192.png",
      "sizes": "192x192",
      "type": "image/png"
    },
    {
      "src": "/icons/icon-512.png",
      "sizes": "512x512",
      "type": "image/png"
    }
  ]
}
```

The final name, colors, and icon artwork require product-owner approval.

### Service-worker caching rules

Use the service worker to cache versioned static app files so the interface can
open offline. Do not treat authenticated API responses as ordinary static files
or place all server data into Cache Storage.

- Cache the built HTML, JavaScript, CSS, fonts, and app icons.
- Store structured inventory, pending results, and operations in IndexedDB.
- Provide an offline navigation fallback to the app shell.
- Use bounded cache names and remove old caches during service-worker updates.
- Show an update prompt or use another documented update policy; do not silently
  leave users on incompatible application code.
- Test first load, repeat load, offline reload, and service-worker upgrade.

## Offline Data and Synchronization

Do not use the browser's `online` event as proof that a request will succeed.
Treat every network operation as fallible and preserve it until the server
acknowledges it.

### Local data

IndexedDB should contain at least:

- The latest inventory records for the active household
- The latest pending-recognition records
- An outbox of unsynchronized operations
- Synchronization metadata and the last successful cursor/time

Keep authentication tokens in the storage managed by the approved Supabase
client. Do not copy tokens into application database records.

### Local-first mutation sequence

For an addition, edit, resolution, or deletion:

1. Validate the user's input.
2. Create a unique `operation_id`.
3. Record the expected server `version` when editing an existing record.
4. Write the optimistic record change and outbox operation in one IndexedDB
   transaction.
5. Update the interface immediately and label the change `pending`.
6. Attempt synchronization when connectivity is available.
7. Remove the outbox entry only after the server acknowledges that operation.
8. Replace local data with the accepted server record and version.

If the app closes after step 4, the operation must still exist after restart.
This is the core offline durability requirement.

### Retry and conflict behavior

- Retrying the same `operation_id` must not apply the operation twice.
- Process the queue in a deterministic order.
- Use bounded backoff for temporary failures instead of a tight retry loop.
- A `409 Conflict` means the expected version is stale. Fetch the current
  record and ask the user to retry or resolve the conflicting values.
- Never choose a winner based only on a device clock.
- Keep tombstones long enough for offline clients to learn about deletions.
- Realtime updates refresh accepted cloud state but do not erase the local
  outbox.

## Suggested Implementation Order

Keep pull requests small enough for another teammate to understand and test.

1. **Scaffold and quality checks** — React/TypeScript build, lint, tests, PWA
   manifest, service-worker registration, and empty responsive shell.
2. **Domain types and fixtures** — implement contract types and test fixtures
   before building screens around guessed data.
3. **Supabase connection and Auth** — session provider, sign-in flow, protected
   routes, household selection, and tested RLS policies.
4. **Read-only inventory** — fetch, cache, render, and handle empty/loading/error
   states.
5. **Pending review** — show uncertain results and submit corrections.
6. **Inventory mutations** — add, edit, quantity change, and tombstone delete.
7. **IndexedDB cache and outbox** — make reads and mutations survive reloads.
8. **Synchronization worker** — acknowledgement, retry, version conflict, and
   status reporting.
9. **Realtime updates** — merge accepted remote changes with cached state while
   preserving unsynchronized operations.
10. **Install and offline hardening** — icons, install behavior, service-worker
    updates, browser testing, accessibility, and deployment.

## Testing Expectations

### Unit tests

Test pure logic such as:

- API-to-domain conversion
- Input validation
- Queue ordering and retry decisions
- Version-conflict handling
- Synchronization status calculation

### Component tests

Test visible behavior using user interactions:

- Loading, empty, success, and error states
- Inventory additions and edits
- Pending-result correction
- Offline and pending indicators
- Keyboard operation, labels, and focus behavior

### Integration and browser tests

Test complete workflows against controlled test services:

- Sign in and household isolation
- Initial synchronization
- Make changes offline, reload, reconnect, and synchronize once
- Receive a Realtime update
- Resolve a stale-version conflict
- Install the PWA and launch it in standalone mode
- Upgrade from one service-worker version to another

Do not use a production household or production credentials in automated tests.

### Required package scripts

The scaffold should provide stable commands similar to:

```bash
npm run dev
npm run lint
npm run typecheck
npm run test
npm run build
npm run test:e2e
```

The exact tools may change, but contributors should not need to remember hidden
commands to verify their work.

## Accessibility and Mobile Usability

- Use semantic HTML before adding ARIA attributes.
- Associate every form control with a visible label.
- Keep focus visible and return it sensibly after dialogs close.
- Support keyboard-only operation.
- Do not communicate synchronization or validation state by color alone.
- Use touch targets and layouts suitable for the smallest supported phone.
- Test with browser zoom and increased text size.
- Confirm destructive operations before applying them.

## Definition of Done for an App Change

An app task is complete when:

- The behavior matches `pi-api-contract.md`.
- Online, offline, loading, empty, and failure states are considered.
- Household authorization is enforced in RLS, not only in the UI.
- No privileged credential is included in browser code.
- Relevant unit, component, and browser tests pass.
- The production build completes.
- The feature is usable on the supported phone viewport and by keyboard.
- Documentation and `.env.example` are updated when configuration changes.
- The pull request explains manual and automated verification.

## Stack Decisions Still Needed from the App Team

Before the scaffold is finalized, record these choices here:

- Node.js major version and whether it is pinned in a version file
- npm, pnpm, or another package manager
- Vite or a different React build framework
- PWA/service-worker integration library
- Router
- Styling approach and component library, if any
- IndexedDB wrapper
- Unit, component, and end-to-end test tools
- Hosting provider and deployment environments
- Supported browsers, phone operating systems, and minimum versions
- Authentication method: password, magic link, OTP, or another approved flow
- Final app name, colors, icons, and install-screen assets

Once the team provides these decisions, replace the proposed entries rather
than adding parallel alternatives.

## Official References

- [MDN: Making PWAs installable](https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Guides/Making_PWAs_installable)
- [Supabase: Use Supabase with React](https://supabase.com/docs/guides/getting-started/quickstarts/reactjs)
- [Supabase Auth](https://supabase.com/docs/guides/auth)
- [Supabase: Securing your data](https://supabase.com/docs/guides/database/secure-data)
- [Supabase TypeScript support](https://supabase.com/docs/reference/javascript/typescript-support)
