# Exploration: Metadata Enrichment (Fase 4)

## Problem Statement

Fase 3 (media-enrichment, archived 2026-06-12) shipped download cache, lyrics
(lrclib), terminal artwork, and Discord presence. In real use against YouTube
*music-video* (MV) sources, three quality gaps surfaced — all rooted in the same
cause: the enrichment layer treats the **raw** YouTube video title as the track
name and the **channel/`Uploader`** as the artist.

1. **Dirty metadata → poor lyrics hit-rate.** lrclib `/api/get` is queried with
   `track_name = search.Result.Title` and `artist_name = search.Result.Uploader`
   (see `internal/ui/messages.go:88` → `lyrics.Service.Fetch`). For an MV these are
   noisy: title is `"Artist - Song (Official Music Video)"` and uploader is
   `"ArtistVEVO"` or a label channel. lrclib does exact-ish matching on a clean
   `track_name`/`artist_name`, so these queries frequently miss when a clean query
   would hit. No normalization happens anywhere today.

2. **Single lyrics source.** Only lrclib is consulted. On a miss, `Fetch` returns an
   empty `Lyrics{}` (graceful, but no second chance). `lyrics.Service` was *designed*
   for a fallback — it carries a configurable `baseURL` and an injectable `httpDoer`
   (`internal/lyrics/lyrics.go:28-42`) — but no second provider is wired.

3. **Artwork is a video frame, not the album cover.** Artwork is always the YouTube
   thumbnail: the cached `--write-thumbnail` file when present, else
   `https://i.ytimg.com/vi/<id>/hqdefault.jpg` (`main.go:141-150`,
   `cache.Service.ThumbPath` at `internal/cache/cache.go:123`). For an MV that
   thumbnail is a still frame from the video, not the release cover art.

Why it matters: lyrics and artwork are the headline Fase 3 features; on the most
common real-world input (MV links) they silently underperform. Fixing the metadata
that feeds them lifts both at once.

## Current State / Code Map

| Area | Where | Today's behavior |
|------|-------|------------------|
| Track identity | `internal/search/search.go:7` `Result{ID,Title,Uploader,Duration}` | Raw YouTube fields; no artist/title split, no clean name. `Title`/`Uploader` flow unchanged into history, favorites, playlists, cache (`upsertTrackQuery`). |
| Lyrics query | `internal/ui/messages.go:88` → `lyrics.Service.Fetch(ctx, id, Title, Uploader, dur)` | Passes raw title/uploader straight to lrclib `track_name`/`artist_name` (`lyrics.go:113-126`). |
| Lyrics source | `internal/lyrics/lyrics.go` | lrclib only. `baseURL` const-initialized to `https://lrclib.net`; `httpDoer` injectable. DB cache via `LyricsRepo` keyed by `videoID`. Miss ⇒ empty `Lyrics{}`, `err=nil`. |
| Artwork source | `main.go:136` `artworkAdapter.Render` + `internal/artwork/artwork.go` | `src = cached thumb || i.ytimg hqdefault.jpg`; `Backend.Render` runs `chafa` (kitty/sixel return placeholder — carried-over debt W1/Known Issues). YouTube thumbnail only. |
| Config toggles | `internal/config/config.go` | `cache.enabled`, `lyrics.enabled`, `artwork.enabled`, `presence.*`. No metadata/normalization keys. |

### Carried-over debt from Fase 3 (do not re-solve here, but be aware)
- `teatest` golden-frame UI coverage still missing (W3).
- Native kitty/sixel pixel render still placeholder-only; only `chafa` renders. This
  bounds how much "real cover art" visibly improves on non-chafa terminals.

## Constraints (locked)
- Pure Go / **no cgo**; single static binary (`CGO_ENABLED=0`). Any new dep must hold this.
- Every enrichment feature is an opt-in config toggle and degrades gracefully (offline,
  API down, no match). SponsorBlock stays out of scope.
- Stack: Go + Bubble Tea + mpv IPC + yt-dlp + Viper + Zap + modernc sqlite.

---

## Gap 1 — Metadata normalization

**Goal:** derive a clean `(artist, title)` from raw `(Title, Uploader)` before lyrics
and artwork lookups.

### Approaches
1. **Heuristic string normalizer (pure Go, in-repo).** New `internal/metadata` package:
   split on the first `" - "`/`" – "` into artist/title; strip parenthetical/bracketed
   suffixes (`(Official Video)`, `(Official Music Video)`, `[MV]`, `(Lyrics)`,
   `(Audio)`, `(Lyric Video)`, `(Visualizer)`, `(HD)`, year tags); strip leading/trailing
   `feat.`/`ft.` segments (capture them but drop from the lyrics query); collapse
   whitespace; derive artist from channel by stripping `VEVO`/`- Topic`/`Official`
   when no `" - "` split exists.
   - Pros: zero deps; offline; deterministic and table-testable; keeps static binary;
     directly raises lrclib hit-rate; reusable by artwork.
   - Cons: heuristics never catch every channel naming convention; needs a maintained
     suffix list; non-Latin / unusual titles may mis-split.
   - Effort: Low–Medium.

2. **External metadata API (e.g. MusicBrainz search) to canonicalize.** Query MB with
   the raw title and let it return the best recording match.
   - Pros: authoritative artist/title/release; reuses the same MB call needed for cover art.
   - Cons: network dependency for *every* track; MB rate limit (~1 req/s, requires
     User-Agent); still needs a clean-ish query to match, so you need (1) first anyway;
     fails offline.
   - Effort: Medium (and overlaps Gap 3).

3. **Do nothing / accept raw.** Status quo.
   - Pros: zero work. Cons: the actual problem persists.

**Recommendation:** **Option 1** as the foundation, feeding cleaned strings into the
lyrics query and (optionally) the MB lookup of Gap 3. MB canonicalization (option 2)
is a *consumer* of the normalizer, not a replacement for it. New capability:
`metadata`.

---

## Gap 2 — Lyrics fallback source

**Goal:** when lrclib misses, try a second source before giving up. Architecture
already supports it (`baseURL` + `httpDoer`).

### Candidate sources
- **lrclib alt endpoint (`/api/search`) before declaring a miss.** lrclib also exposes a
  fuzzy `/api/search?q=` that returns candidates; today only the strict `/api/get` is
  used. Trying `/api/search` on a `/api/get` miss is a *same-provider* fallback with
  **no new ToS/auth/dep risk** and pairs naturally with the normalized query from Gap 1.
- **Genius.** Large catalog, but: requires an API token (OAuth/client token), the API
  returns song *metadata + a URL*, **not** the lyric text (lyrics are HTML-scraped —
  against Genius ToS), and never synced. High licensing/ToS risk; needs user-supplied token.
- **Musixmatch.** Has synced lyrics but a commercial API: API key required, free tier
  returns only a 30% lyric excerpt, strict ToS. High risk; not a clean fit.
- **NetEase / QQ music etc.** Region-locked, undocumented, ToS-gray. Reject.

### Approaches
1. **In-provider fallback: `/api/get` → `/api/search` on lrclib, both with the
   normalized query.** Pure Go, no new dep, no auth, same ToS we already accept.
   - Pros: lowest risk; biggest hit-rate win comes from Gap 1 feeding both calls;
     keeps synced-lyrics capability; trivial to toggle.
   - Cons: still bounded by lrclib's catalog.
   - Effort: Low.

2. **Pluggable second provider behind an interface (`LyricsProvider`), ship lrclib
   only, leave Genius/Musixmatch as opt-in, token-gated, disabled-by-default adapters.**
   - Pros: clean extension point; documents the licensing posture; user can opt in.
   - Cons: real value only if a usable second provider exists — and the strong
     candidates carry ToS/scraping risk we should not ship enabled.
   - Effort: Medium.

3. **Ship a scraping fallback (Genius/Musixmatch).** Reject — ToS/licensing risk,
   fragile HTML parsing, no synced lyrics.

**Recommendation:** **Option 1** (lrclib `/api/search` fallback on the normalized
query) as the safe, high-yield MVP, optionally structured behind a minimal
`LyricsProvider` interface (Option 2's seam) so a future token-gated provider can be
added without re-architecting. **Decision for human:** do we ship *any* third-party
(Genius/Musixmatch) adapter now, or defer all of them given the ToS/auth risk?
Recommendation: defer.

---

## Gap 3 — Real album artwork

**Goal:** prefer real release cover art over the YouTube video frame, keyed off the
normalized `(artist, title)`, with the existing thumbnail as fallback.

### Approach: MusicBrainz → Cover Art Archive (CAA)
Flow: `metadata.Normalize(track)` → MB recording/release search
(`https://musicbrainz.org/ws/2/recording?query=...&fmt=json`) → pick best release →
fetch cover from `https://coverartarchive.org/release/<mbid>/front` → cache the image
file locally → `Backend.Render` it. Both APIs are free, no auth, pure HTTP (stdlib —
no new dep, preserves static binary).

**Constraints/risks specific to MB+CAA:**
- MB requires a descriptive **User-Agent** (we already set one for lrclib) and asks for
  **~1 req/s** rate limiting — must add a simple client-side throttle.
- Network call per *uncached* track; must cache the resolved cover (and ideally the
  negative result) — reuse the cache layer / a new `artwork_cache` keyed by `videoID`
  or by `mbid`.
- CAA may have no front cover for a release → fall back to YouTube thumbnail.
- Matching is only as good as Gap 1's normalization; wrong match ⇒ wrong cover (so
  prefer high-confidence matches; consider duration cross-check against MB length).

### Approaches
1. **MB+CAA lookup, cached, with YouTube thumbnail fallback; opt-in toggle.**
   - Pros: real cover art; free; no auth; stdlib HTTP (no dep); degrades to today's behavior.
   - Cons: rate-limit handling + caching to build; mismatch risk; another network path
     to make graceful; visible benefit partly bounded by chafa-only render debt.
   - Effort: Medium.

2. **iTunes Search API for cover art** (`itunes.apple.com/search?entity=song`).
   - Pros: simple JSON, returns artwork URLs, no auth, generous; good hit-rate for
     mainstream music; can sidestep MB rate-limit friction.
   - Cons: Apple ToS (intended for affiliate/search use), lower coverage for niche/non-Western;
     artwork sizes via URL munging. Worth listing as an alternative/secondary art source.
   - Effort: Low–Medium.

3. **Keep YouTube thumbnail only.** Status quo — rejected (it's the problem).

**Recommendation:** **Option 1 (MB + Cover Art Archive)** as primary, behind a toggle,
with the **YouTube thumbnail as the guaranteed fallback** and per-track caching +
~1 req/s throttle. iTunes Search (Option 2) is a reasonable *alternative or secondary*
source — flag as a human decision. **Decision for human:** MB+CAA vs iTunes vs both
for the first slice.

---

## Affected Capabilities
- **NEW `metadata`** — title/artist normalization (the shared foundation).
- **MODIFIED `lyrics`** — query with normalized strings; add lrclib `/api/search`
  fallback; optional `LyricsProvider` seam.
- **MODIFIED `artwork`** — source real cover art (MB+CAA) keyed off normalized metadata,
  thumbnail fallback; new artwork/cover cache.
- Possibly **MODIFIED `tui-shell`** only if we surface "matched as Artist — Title" or a
  source indicator; likely not required for MVP.

## Recommended MVP First Slice (Fase 4)
1. New `internal/metadata` package: pure-Go `Normalize(search.Result) (artist, title string)`
   with table tests over real dirty MV titles/channels. **(highest leverage, zero risk)**
2. Wire normalized `(artist, title)` into the lyrics query path (`ui/messages.go` →
   `lyrics.Fetch`), and add the lrclib `/api/search` fallback on a `/api/get` miss.
3. Add MB→CAA cover-art lookup (toggle `artwork.cover_art` / similar), cached per track,
   ~1 req/s throttle, YouTube thumbnail as fallback.

Slice 1+2 are low-risk and deliver the lyrics hit-rate win largely on their own. Slice 3
is the larger/network-heavier piece and a natural second PR if the work exceeds the
400-line review budget.

## Risks & Open Questions
- **Normalization is heuristic** — a maintained suffix/channel list will need tuning;
  some titles will still mis-split. Mitigate with table tests + the raw query as a
  secondary attempt.
- **MB rate limit / etiquette** — must throttle (~1 req/s) and send a descriptive
  User-Agent or risk being blocked; need negative-result caching to avoid re-hitting.
- **Wrong-match cover art** — confidence threshold + duration cross-check; always keep
  thumbnail fallback.
- **Render debt bounds the artwork payoff** — on non-chafa terminals kitty/sixel still
  renders a placeholder (Fase 3 W1). Real cover art only *looks* better where render works.
- **No-cgo/static-binary** — all proposed sources use stdlib `net/http`; no new deps
  required for the recommended path (rich-go remains the only third-party runtime dep).

### Decisions the human should make before propose
1. Lyrics fallback: lrclib `/api/search` only (recommended), or also ship a token-gated
   Genius/Musixmatch adapter (ToS/auth risk; recommend defer)?
2. Cover-art source: MusicBrainz+Cover Art Archive (recommended), iTunes Search, or both?
3. Should normalization mutate the stored `Title`/`Uploader` (affecting history/favorites/
   library display), or stay query-only (used solely for lyrics/artwork lookups, leaving
   stored data raw)? Recommend **query-only** for the MVP to avoid touching the library.

## Ready for Proposal
**Yes.** The problem is well shaped and grounds in existing code seams (`baseURL`/
`httpDoer` in lyrics, `artworkAdapter` source resolution, `search.Result`). The three
human decisions above should be resolved at the start of the propose phase.
