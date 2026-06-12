# Tasks: Metadata Enrichment (Fase 4)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~520-600 (WU1 ~210, WU2 ~330) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (WU1) -> PR 2 (WU2) |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending (team decision) |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Normalizer + lyrics normalized query + `/api/search` fallback | PR 1 | Independent; ~210 lines incl. tests. Under budget alone. |
| 2 | MB->CAA cover resolver + cover cache + config toggle + adapter wiring | PR 2 | Depends on WU1 normalizer. ~330 lines incl. tests. Under budget alone, but combined > 400. |

WU1 and WU2 are each individually under 400 lines; combined they exceed it. Ship as two chained/stacked PRs.

## Phase 1: Metadata Normalizer (WU1)

Satisfies: `metadata` spec — "Normalize Query Metadata" (all 3 scenarios), "Query-Only, Non-Mutating".

- [x] 1.1 Create `internal/metadata/metadata.go` with `func Normalize(r search.Result) (artist, title string)`: split on `" - "`/`" – "` (first separator); when no split, derive artist from `Uploader` stripping `VEVO`/`- Topic`/`Official`.
- [x] 1.2 In `metadata.go`, strip bracket/paren tags (`(Official [Music] Video)`, `[MV]`, `(Lyrics)`, `(Lyric Video)`, `(Audio)`, `(Visualizer)`, `(HD)`, year tags) and drop `feat.`/`ft.` segments; collapse leading/trailing/repeated whitespace. Pure, no I/O; never reads/writes `r`.
- [x] 1.3 Create `internal/metadata/metadata_test.go`: table tests covering split, suffix/feat strip, channel-derived artist (VEVO/`- Topic`), whitespace collapse, and a case asserting the input `search.Result` is unchanged (non-mutating).

## Phase 2: Lyrics Normalized Query + Search Fallback (WU1)

Satisfies: `lyrics` spec — "Fetch Lyrics": "Normalized query used", "Search fallback after get miss".

- [x] 2.1 In `internal/ui/messages.go` `fetchLyricsCmd`, call `metadata.Normalize(track)` and pass the normalized `(artist, title)` to `l.Fetch` instead of raw `track.Title`/`track.Uploader` (gated by existing `lyrics.enabled`; keep raw `track.ID`).
- [x] 2.2 In `internal/lyrics/lyrics.go`, add `fetchSearch(ctx, title, artist, dur)` hitting `baseURL+"/api/search"` via the same `httpDoer`/UA; parse the result array, pick best candidate (artist+title match, duration tie-break), return body/synced/ok. Gate with a `searchFallback bool` field on `Service` (default true).
- [x] 2.3 In `Fetch`, after `fetchRemote` (`/api/get`) returns `ok=false` and `searchFallback` is on, call `fetchSearch` before returning empty `Lyrics{}`. Preserve existing cache-first and store-on-hit behavior.
- [x] 2.4 In `internal/lyrics/lyrics_test.go`, add a test reusing the existing httptest server + `s.baseURL` fake: `/api/get` returns 404, `/api/search` returns a candidate -> lyrics resolved. Add a test with `searchFallback=false` asserting no `/api/search` call. Confirm existing 404-for-all tests still yield empty.
- [x] 2.5 Build/test gate (WU1): `CGO_ENABLED=0 go build ./...` and `go test ./internal/metadata/... ./internal/lyrics/... ./internal/ui/...` pass.

## Phase 3: Cover Resolver + Cache (WU2)

Satisfies: `artwork` spec — "Render Current Track Artwork": "Real cover resolved", "Cover lookup cached", "Thumbnail fallback on miss or offline".

- [x] 3.1 Create `internal/artwork/cover.go`: `CoverResolver` interface `Resolve(ctx, artist, title string, durationSec int) (path string, ok bool)`; `mbCoverResolver` querying MusicBrainz recording search -> release MBID -> `coverartarchive.org/release/<mbid>/front-500`. Never returns error (ok=false on miss/offline).
- [x] 3.2 In `cover.go`, add a shared `time.Ticker`/mutex limiter (~1 req/s) and descriptive `User-Agent` (reuse the lyrics UA string). Use stdlib `net/http` only — no new module dependency.
- [x] 3.3 In `cover.go`, add file cache under `CacheHome/covers/`: image at `covers/<sha1(artist|title)>.<ext>`, negative marker `covers/<key>.miss` (empty file). Resolve order: cache hit -> path; `.miss` -> ok=false; else fetch then write image or `.miss`.
- [x] 3.4 In `cover.go`, apply duration cross-check: accept MB recording only if its `length` (ms) is within ±7 s of `durationSec`; if MB omits length, accept top-ranked result but write `.miss` on CAA 404.
- [x] 3.5 In `internal/cache/cache.go`, extend `Evict`/`Clear` (and `dropEntry` as needed) to also remove the `covers/` subdir contents so cover eviction piggybacks on the existing cache lifecycle.
- [x] 3.6 Create `internal/artwork/cover_test.go`: httptest MB+CAA servers + temp `covers/` dir. Cover-resolved case; cache-hit case asserting a single network call on the second `Resolve`; negative `.miss` case; duration-mismatch rejection; offline (dial error) -> ok=false.

## Phase 4: Config Toggle + Adapter Wiring (WU2)

Satisfies: `artwork` spec — "Render Current Track Artwork" (toggle/fallback precedence); proposal success criteria (toggles off => Fase 3 parity).

- [x] 4.1 In `internal/config/config.go`, add `LyricsSearchFallback bool` (default `lyrics.search_fallback` = `true`) and `ArtworkCoverArt bool` (default `artwork.cover_art` = `false`); set Viper defaults and populate the returned `Config`.
- [x] 4.2 In `main.go`, when `cfg.LyricsSearchFallback` is set, propagate it into `lyrics.New` (constructor arg or setter) so the service's `searchFallback` reflects config.
- [x] 4.3 In `main.go` `artworkAdapter`, add a `cover CoverResolver` field; construct `mbCoverResolver` only when `cfg.ArtworkCoverArt` is true (else nil = thumbnail-only, exact Fase 3). In `Render`, normalize the track and try `cover.Resolve` first; on ok use the cover path, else fall back to cached thumb then remote ytimg URL.
- [x] 4.4 Build/test gate (WU2): `CGO_ENABLED=0 go build ./...` and `go test ./...` pass; confirm `artwork.cover_art=false` keeps artwork byte-identical to Fase 3 and a new module dependency was NOT added (`go.mod` unchanged).
