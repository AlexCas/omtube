# Proposal: Metadata Enrichment (Fase 4)

## Intent

Fase 3 feeds the enrichment layer the **raw** YouTube title as the track name and the
channel as the artist. On music-video (MV) sources these are noisy
(`"Artist - Song (Official Music Video)"`, uploader `"ArtistVEVO"`), so lrclib `/api/get`
frequently misses and artwork is always a video frame, not the release cover. This phase
normalizes the query, adds a same-provider lyrics fallback, and sources real album art —
lifting the two headline Fase 3 features at once, with no new runtime dependency.

## Scope

### In Scope
- New pure-Go `internal/metadata` normalizer: derive clean `(artist, title)` from raw
  `(Title, Uploader)` for **outbound queries only**.
- Wire normalized strings into the lyrics query path; add lrclib `/api/search` fallback on
  an `/api/get` miss (same provider, same normalized query).
- Album cover art via MusicBrainz recording/release search → Cover Art Archive front cover,
  keyed off normalized `(artist, title)`; stdlib HTTP only; descriptive User-Agent; ~1 req/s
  throttle; cache positive **and** negative results; YouTube thumbnail remains the fallback.
- Config toggles for each new behavior; off ⇒ exact Fase 3 behavior.

### Out of Scope
- Genius / Musixmatch / any token-gated or scraping lyrics adapter (ToS/auth risk — deferred).
- iTunes cover-art source.
- Mutating stored `Title`/`Uploader` or library rows (history/favorites/playlists stay raw).
- Native kitty/sixel pixel render and `teatest` golden coverage (carried-over Fase 3 debt).

## Capabilities

> Contract for sdd-spec. Researched against `openspec/specs/`.

### New Capabilities
- `metadata`: pure, deterministic normalizer producing clean `{artist, title}` for lyrics/
  artwork queries (split `Artist - Title`; strip `(Official Video)`/`[MV]`/`(Lyrics)`/
  `(Audio)`/`feat.`/VEVO-`- Topic` channel noise; collapse whitespace). Query-only; never
  mutates stored data. Table-testable.

### Modified Capabilities
- `lyrics`: query lrclib with normalized `(artist, title)`; on `/api/get` miss, retry via
  lrclib `/api/search` before declaring "no lyrics". Caching/sync/highlight unchanged.
- `artwork`: source real release cover art (MusicBrainz → Cover Art Archive) keyed off
  normalized metadata, cached (incl. negative), throttled ~1 req/s; YouTube thumbnail is the
  guaranteed fallback on miss/offline. (Folded here, not a new capability: the artwork spec
  already owns "fetch and display artwork for the current track"; this changes its source.)
- `tui-shell`: **None** — lyrics/artwork panels already exist; no UI surface change required.

## Approach

`internal/metadata.Normalize(search.Result) (artist, title string)` is the shared, dependency-
free foundation. The lyrics command (`ui/messages.go`) calls it before `lyrics.Fetch`; the
lyrics service adds a `/api/search` step on a `/api/get` miss using the same `httpDoer`/`baseURL`
seam. A cover-art resolver (MB+CAA, stdlib `net/http`, throttled, cached by normalized key /
videoID) plugs into the `artworkAdapter` source resolution in `main.go`, preferring a resolved
cover, else the cached/remote YouTube thumbnail. All three behaviors are config toggles that
degrade to Fase 3 when off, offline, or on no-match.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/metadata` | New | Pure normalizer + table tests |
| `internal/lyrics/lyrics.go` | Modified | `/api/search` fallback on `/api/get` miss |
| `internal/ui/messages.go` | Modified | Normalize before `lyrics.Fetch` query |
| `internal/artwork` | Modified | MB→CAA cover resolver, throttle |
| `main.go` (`artworkAdapter`) | Modified | Prefer resolved cover, fall back to thumbnail |
| `internal/cache` | Modified | Cover/negative-result cache |
| `internal/config` | Modified | New toggles (defaults preserve Fase 3) |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Heuristic mis-split of unusual titles | Med | Table tests; raw query as secondary attempt |
| MB rate-limit / blocking | Med | ~1 req/s throttle; descriptive UA; cache neg results |
| Wrong-match cover art | Med | Confidence + duration cross-check; thumbnail fallback |
| Render debt bounds visible artwork gain | Low | Out of scope; chafa path still improves |

## Rollback Plan

Each behavior is an opt-in toggle; set to off (or default-off) restores exact Fase 3 behavior.
No schema-breaking persistence; cover cache lives in XDG cache and is safe to delete. Revert =
flip toggles + drop new package/cache; no library DB impact.

## Dependencies

- Fase 3 (`add-media-enrichment`) completed (lyrics/artwork/cache seams in place).
- No new Go dependency — MusicBrainz + Cover Art Archive over stdlib `net/http`; pure Go / no
  cgo / single static binary preserved.

## Success Criteria

- [ ] Dirty MV title (`"Artist - Song (Official Music Video)"`, uploader `"ArtistVEVO"`)
  normalizes to clean `artist + title`.
- [ ] lrclib `/api/search` returns lyrics after an `/api/get` miss on the normalized query.
- [ ] MB+CAA front cover is fetched, cached (positive & negative), and rendered; miss/offline
  falls back to the YouTube thumbnail.
- [ ] Normalization never mutates stored `search.Result` or library rows (history/favorites/
  playlists still show the original YouTube title).
- [ ] All new features toggled off ⇒ behavior identical to Fase 3.
- [ ] `CGO_ENABLED=0` static build still succeeds with no new dependency.
