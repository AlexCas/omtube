# Design: Metadata Enrichment (Fase 4)

## Technical Approach

A new pure `internal/metadata.Normalize` cleans `(artist, title)` for outbound queries
only. The lyrics command normalizes before `Fetch`; the lyrics service gains an
`/api/search` step on an `/api/get` miss over the existing `httpDoer`/`baseURL` seam. A
new `internal/artwork` cover resolver (MusicBrainz → Cover Art Archive, stdlib HTTP,
~1 req/s, cached positive+negative) feeds `artworkAdapter`, which prefers a resolved cover
then falls back to the existing thumbnail. Every behavior is a config toggle defaulting to
exact Fase 3 parity. Implements specs `metadata`, `lyrics`, `artwork`.

## Architecture Decisions

| # | Decision | Choice | Rejected | Rationale |
|---|----------|--------|----------|-----------|
| 1 | Normalizer API | `metadata.Normalize(search.Result) (artist, title string)`; called in `ui/messages.go` `fetchLyricsCmd` and inside the cover resolver | Mutate `search.Result`; normalize inside lyrics service | Pure/table-testable, query-only (spec "Query-Only"); both consumers reuse one seam; UI keeps raw fields for display |
| 2 | Lyrics fallback | After `/api/get` miss, call `s.fetchSearch(ctx, ...)` hitting `baseURL+"/api/search"` via same `httpDoer`; pick best candidate (artist+title match, duration tie-break) | New provider/interface now; separate client | Same-provider, no new dep/ToS; reuses `baseURL` so existing httptest tests stay green (404-for-all server still yields empty) |
| 3 | Cover resolver | New `artwork.CoverResolver` interface; `mbCoverResolver` (MB recording search → release MBID → `coverartarchive.org/release/<mbid>/front`); composed in `artworkAdapter`: resolved cover → cached thumb → remote thumbnail | Resolve inside `Backend.Render`; iTunes | Keeps `Backend` pure; adapter already owns source selection; MB+CAA is auth-free stdlib; iTunes out of scope |
| 4 | Throttle | Single shared `time.Ticker`/mutex limiter (~1 req/s) in resolver; descriptive `User-Agent` (reuse lyrics UA string) | `golang.org/x/time/rate` | No new dep; MB etiquette satisfied |
| 5 | Cover cache | File cache under `CacheHome/covers/`; image at `covers/<sha1(artist|title)>.<ext>`; negative marker `covers/<key>.miss` (empty file). Resolver: hit→path; `.miss`→fall back; else fetch | DB table; key by videoID | Reuses XDG cache, safe to delete (rollback); content-addressed by normalized key dedupes across videoIDs; no schema change |
| 6 | Match confidence (duration cross-check) | Accept MB release only if MB recording `length` (ms) is within ±N s (N≈7) of `track.Duration`; if MB omits length, accept top-ranked result but record `.miss` on CAA 404 | Trust MB score blindly; require exact length | Mitigates wrong-match covers per spec/risk; thumbnail fallback bounds damage |
| 7 | `LyricsProvider` interface | YAGNI — do NOT introduce now | Add interface seam | Only lrclib ships; Genius/Musixmatch out of scope; in-provider `/api/search` needs no abstraction. Note: revisit if a 2nd provider lands |

## Data Flow

```
track ─→ metadata.Normalize ─→ (artist,title)
                                  │
       lyrics path ──────────────┼──→ /api/get ──miss──→ /api/search ──→ Lyrics
                                  │
       artwork path ─→ CoverResolver ─→ covers/ cache ──miss──→ MB ─→ CAA front
                                  │            │                         │
                                  └→ adapter: cover? → cached thumb? → ytimg thumbnail → Backend.Render
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/metadata/metadata.go` | Create | `Normalize`, suffix/feat strip, channel cleanup (pure) |
| `internal/metadata/metadata_test.go` | Create | Table tests over dirty MV titles/channels |
| `internal/artwork/cover.go` | Create | `CoverResolver`, `mbCoverResolver`, throttle, file cache, duration check |
| `internal/artwork/cover_test.go` | Create | httptest MB/CAA, cache hit, negative, duration mismatch |
| `internal/lyrics/lyrics.go` | Modify | `/api/search` fallback after `/api/get` miss |
| `internal/lyrics/lyrics_test.go` | Modify | Search-fallback-after-get-miss test |
| `internal/ui/messages.go` | Modify | `Normalize` before `l.Fetch` in `fetchLyricsCmd` |
| `internal/config/config.go` | Modify | New toggles + defaults |
| `main.go` | Modify | Wire resolver into `artworkAdapter` |

## Interfaces / Contracts

```go
// internal/metadata
func Normalize(r search.Result) (artist, title string)

// internal/artwork
type CoverResolver interface {
    // Resolve returns a local cover image path, ok=false on miss/offline (never errors).
    Resolve(ctx context.Context, artist, title string, durationSec int) (path string, ok bool)
}
// artworkAdapter gains: cover CoverResolver (nil ⇒ thumbnail-only, Fase 3 parity)
```

## Config Toggles (defaults = Fase 3 parity)

| Key | Default | Effect |
|-----|---------|--------|
| `lyrics.search_fallback` | `true` | `/api/search` retry; off ⇒ get-only |
| `artwork.cover_art` | `false` | MB+CAA cover; off ⇒ thumbnail only (exact Fase 3) |

Normalization for lyrics is unconditional (query-only, no behavior toggle) but gated by
`lyrics.enabled`. `artwork.cover_art` defaults **off** to guarantee byte-identical Fase 3
artwork until explicitly enabled.

## Testing Strategy

| Layer | What | Approach |
|-------|------|----------|
| Unit | `Normalize` split/strip/channel cases | Table tests, no I/O |
| Unit | Lyrics search fallback | httptest: `/api/get` 404 → `/api/search` hit; verify existing tests still pass |
| Unit | Cover resolve / cache hit / negative `.miss` / duration mismatch / offline | httptest MB+CAA + temp `covers/` dir; assert single network call on second resolve |
| Integration | Adapter precedence cover→thumb→remote | Fake resolver + fake backend |

## Migration / Rollout

No migration. Cover cache is a new XDG subdir, safe to delete. Rollback = toggles off
(`cover_art` already default-off) + remove `covers/`; no library DB impact.

## Open Questions — Resolved (human review gate)

- [x] Duration tolerance: **fixed ±7 s** (not configurable for the MVP).
- [x] CAA front-cover size: **`/front-500`** (500 px; lighter and sufficient for chafa rendering).
- [x] `covers/` eviction: **piggyback on the existing cache `Evict`/`Clear`** now (consistent with the rest of the cache).
