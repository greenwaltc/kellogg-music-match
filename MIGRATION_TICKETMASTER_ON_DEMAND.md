# Migration Guide: On‑Demand Ticketmaster Events

This guide describes how to migrate Kellogg Music Match from a Chicago‑only, pre‑synced Ticketmaster subset to an on‑demand, full‑fidelity Ticketmaster search that supports any event type and geography, while preserving user event associations and simplifying data lifecycle.

Date: 2025‑10‑17
Owner: greenwaltc
Branch seed: feature/any-ticketmaster-event (recommended)

## Goals

- Remove the Ticketmaster “daily + startup” ingest job that caches a limited Chicago concerts subset.
- Expose the full Ticketmaster Discovery API query surface to users in the Events UI, with pass‑through filtering and pagination.
- Only persist events that have at least one user association; delete events immediately when the last association is removed.
- Unify on‑demand Ticketmaster results with locally associated events that also match the user’s filters (to attach association metadata before returning to the UI).
- Provide a simple endpoint that lists all events with associations (no Ticketmaster call) for the Matches page.

## High‑level architecture changes

- Before: background sync job fetched Chicago concerts and cached them in Postgres; UI browsed that cache.
- After: UI submits filters → Backend validates and calls Ticketmaster → Results unified with locally associated events → Response to UI. DB only stores events with associations.
- Keep a daily task to delete past events and to purge orphaned events with zero associations.

---

## Phase 0 — Preparation (Feature flag + contracts)

1) Feature flag/config
- Add a backend config flag to enable the on‑demand mode (default off initially):
  - Env: `TICKETMASTER_ON_DEMAND=true`
  - Wire into `backend/config/config.go` and plumb into relevant services.
  - Add Pulumi stack config defaults.

2) OpenAPI additions (non‑breaking)
- Add new endpoints alongside existing ones:
  - GET `/events/search` — pass‑through Ticketmaster filters (see below) with server‑enforced limits.
  - GET `/events/associated` — return all locally associated events (optional filters like date range, segment, city for convenience), no Ticketmaster call.
  - POST `/events/{eventId}/association` — set association state: INTERESTED | GOING | LFG | NONE.
  - DELETE `/events/{eventId}/association` — remove association (equivalent to NONE).
- Back‑compat (temporary): keep existing “concerts” endpoints as aliases; mark deprecated in OpenAPI. Example: `/concerts/{id}/interest` → `/events/{id}/association`.

3) Ticketmaster filter surface (contract)
- On `/events/search`, accept a safe subset of Ticketmaster Discovery API params (validate + whitelist):
  - `keyword`, `segmentName`, `classificationName`, `countryCode`, `stateCode`, `city`,
  - `latlong` + `radius` (miles),
  - `startDateTime`, `endDateTime` (ISO 8601),
  - `sort` (e.g., `date,asc`),
  - `size` (page size, max 50), `page` (zero‑based),
  - Optional `includeAssociated=true|false` (merge local association overlays; default true).

4) Database readiness
- Ensure event tables capture the minimal snapshot necessary to render/identify an event and link associations:
  - `events` (id PK, `source` enum, `external_id` unique, `name`, `venue`, `city`, `state`, `country`, `start_utc`, `url`, `raw_json` JSONB, `created_at`, `updated_at`).
  - `user_event_associations` (`user_id`, `event_id`, `status`, `created_at`, `updated_at`), PK (`user_id`, `event_id`).
- Indexes:
  - `events(external_id)`, `events(start_utc)`, `user_event_associations(event_id)`, `user_event_associations(user_id)`.
- Data rules:
  - Insert an `events` row only when first association is created.
  - On the last association removal → delete the `events` row (app logic or DB trigger; start with app logic).

---

## Phase 1 — Backend endpoints and services

1) Implement `/events/search`
- Contract:
  - Inputs: whitelisted TM filters above, `size<=50`, `page<=1000` guardrails.
  - Behavior: build a TM Discovery API request, pass credentials from config, handle rate limits/backoff.
  - Unification: if `includeAssociated` is true, fetch locally associated `events` that match the filter (by geo/city/date/segment, as applicable), merge with TM results by `external_id`, and attach association overlays: counts, current user’s status, and lists (within size cap).
  - Output: normalized event DTOs with association metadata: `{ id, externalId, name, venue, location, startUtc, url, association: { myStatus, interestedCount, goingCount, lfgCount, recentUsers? } }`.
- Edge cases:
  - De‑dupe events present in both sources by `external_id`.
  - If TM fails, still return associated events that match filters when possible; include an error hint so UI can show a partial results banner.

2) Implement `/events/associated`
- Returns all events with one or more associations (optionally filterable by date range, segment/type) with their association overlays.
- No Ticketmaster call.
- Pagination supported (default sort by `start_utc asc`).

3) Association lifecycle endpoints
- Replace/alias old interest endpoints:
  - POST `/events/{eventId}/association`: body `{ status: INTERESTED|GOING|LFG|NONE }`.
  - On first association to a non‑persisted TM event:
    - Create `events` row from a minimal TM event snapshot (second TM call not required; use snapshot from UI or search results payload if available via hidden field; otherwise one lookup call by ID).
  - On association removal leading to zero associations:
    - Delete the `events` row.
- Keep `/concerts/*` routes as proxies until UI fully migrates.

4) Keep cleanup job; remove ingest job
- Remove the daily ingest/sync that caches Chicago concerts.
- Keep a daily cleanup job to delete:
  - Events whose `start_utc` < now() − grace period.
  - Any events with zero associations (belt‑and‑suspenders if app logic missed a race).

5) Guardrails & resilience
- Rate limiting: keep per‑user request caps to `/events/search`; surface `Retry‑After`.
- Backoff on Ticketmaster 429; log and surface partial results.
- Optional in‑memory short‑TTL cache keyed by filter hash (e.g., 30–60s) to soften bursts.

---

## Phase 2 — UI: Events page rework

1) Build a full Ticketmaster filter form
- Inputs: keyword, segment/type, classification, location (city/state/country or lat/long + radius), date range, sort, page size.
- Persist state in URL query and (optionally) localStorage; deep‑linkable and shareable.
- Debounce inputs; submit on Enter or Search button; show query summary chips.

2) Wire to new endpoints
- On submit → call `/events/search` with mapped filters; show paginated results with infinite scroll.
- Overlay association info on cards (my status + counts).
- Association toggles on each result call POST `/events/{id}/association` (create local event row if needed).

3) Matches page shortcut
- Add an “All associated events” link (or tile) on Matches that goes to a route using `/events/associated` with no TM call.

4) Remove Chicago‑specific copy
- Update empty‑state and help text to reflect global events, not only Chicago concerts.

---

## Phase 3 — Data lifecycle & consistency

1) Persist on first association
- When user sets status on a TM event that doesn’t exist in `events`, store minimal snapshot (avoid full denormalization, but keep enough for cards).

2) Delete on last dissociation
- When last user clears status, delete the `events` row (ensure no FK constraints prevent this).
- Handle races with a transaction or a post‑commit recheck.

3) Cleanup job refinements
- Retain cleanup that removes past events and any orphans.
- Add monitoring around deletions to ensure no unexpected churn.

---

## Phase 4 — Decommission & cleanup

1) Toggle the feature flag on progressively (stacks or % of users).
2) Update UI to remove deprecated Chicago ingest assumptions.
3) Deprecate and later remove `/concerts/*` endpoints once all traffic uses `/events/*`.
4) Drop any Chicago‑only indexes/data that are no longer used.

---

## API design details

- `/events/search` (GET)
  - Query params: `keyword`, `segmentName`, `classificationName`, `countryCode`, `stateCode`, `city`, `latlong`, `radius`, `startDateTime`, `endDateTime`, `sort`, `size`, `page`, `includeAssociated`.
  - Returns: `{ page, size, total (TM only), items: EventDTO[] }` where `EventDTO` has association overlay.

- `/events/associated` (GET)
  - Query params: optional `startDateTime`, `endDateTime`, `segmentName`, `city`, `size`, `page`.
  - Returns: `{ page, size, total (local), items: EventDTO[] }`.

- `/events/{id}/association` (POST/DELETE)
  - Body: `{ status }` for POST; none for DELETE.
  - Side‑effects: create or delete `events` row depending on association count.

- Response normalization
  - Normalize TM events into a consistent DTO (IDs, dates, venues) to merge easily with local rows.

---

## Backend implementation notes

- Add a new service (e.g., `EventSearchService`) that:
  - Validates filters (whitelist, size caps, date bounds) and builds a TM API request.
  - Calls a `TicketmasterClient` wrapper that centralizes auth, rate‑limit handling, retries, and logging.
  - Fetches local associated events matching filters; merges by `external_id`.
  - Shapes unified DTOs including association summaries (counts + current user status).

- Repository additions in `backend/business/database.go` and `db/sqlc`:
  - `GetAssociatedEvents(filters, page, size)`
  - `UpsertEventFromSnapshot(snapshot)`
  - `GetEventByExternalID(externalId)`
  - `DeleteEventIfNoAssociations(eventId)`

- Keep existing interest endpoints temporarily; implement new association endpoints that re‑use the same business logic with a more generic path name.

---

## UI implementation notes

- Create/expand `EventsComponent` with a filter form mirroring TM options.
- Update `ApiBaseService`/`ConfigService` with new endpoints.
- Ensure cards display association info (my status + counts) and act on toggles without full page reload.
- Add an “Associated events” tab or quick filter that calls `/events/associated`.

---

## Testing strategy

- Backend unit tests:
  - Filter validation and TM request construction (happy + invalid cases).
  - Merge logic: dedupe, overlay associations, partial TM failure.
  - Association lifecycle: create on first association, delete on last.
  - Cleanup job: deletes past events + orphans.
  - Back‑compat aliases still work while flagged on.

- Backend integration tests (optional DB + TM mocked):
  - Paginated queries; date/geo filters; rate limit/backoff handling.

- UI tests:
  - Filter form wiring; URL state; debounce.
  - Infinite scroll; association toggles; associated events view.

---

## Rollout & operations

- Rollout plan:
  - Enable `TICKETMASTER_ON_DEMAND` in dev; validate with ngrok.
  - QA: large geo/radius queries; high traffic times; rate limit edge cases.
  - Enable in staging; watch error budgets and TM 429/5xx.
  - Gradually enable in prod (flag on) and remove old sync job.

- Monitoring:
  - Log TM request counts, 429s/5xx, latency.
  - Track `/events/search` and `/events/associated` QPS and error rates.

- Rollback:
  - Disable flag → UI keeps working with existing Chicago cache (until code removed).

---

## Migration checklist

- [ ] Add feature flag and config plumbing.
- [ ] Extend OpenAPI with `/events/search`, `/events/associated`, `/events/{id}/association`.
- [ ] Implement `TicketmasterClient` and `EventSearchService` with merge logic.
- [ ] Add/confirm DB schema + indexes (events + user_event_associations minimal snapshot).
- [ ] Implement association lifecycle (create on first, delete on last).
- [ ] Keep cleanup job; remove ingest job behind flag.
- [ ] Implement UI filter form + pagination + association overlays.
- [ ] Add Matches shortcut to `/events/associated`.
- [ ] Tests (backend + UI) updated/added.
- [ ] Progressive rollout; monitor; deprecate old endpoints.

---

## Notes & gotchas

- Pagination with merged sources:
  - Use TM paging primarily; fetch local associated events and merge/dedupe; if merge order is important, define a deterministic sort (e.g., by start time then ID).
  - Consider a two‑pane approach (Associated, All) if merged pagination feels awkward.

- Rate limits:
  - Respect TM rate limits; surface `Retry‑After`; short‑TTL cache popular queries.

- Security:
  - Strictly validate/whitelist filters; clamp size/page; sanitize strings.

- Back‑compat:
  - Maintain `/concerts/*` until clients fully migrate.

---

Happy shipping! 🚀