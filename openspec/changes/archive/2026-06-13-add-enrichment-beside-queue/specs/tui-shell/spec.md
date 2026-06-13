# tui-shell — delta for add-enrichment-beside-queue

## MODIFIED Requirements

### Requirement: Results and Queue Panels

The system MUST display search results in a full-screen `modeResults` takeover
modal entered only after a multi-result search, and MUST display the playback
queue as an inline full-size panel always visible in the main view, highlighting
the current track. The results panel MUST NOT be drawn in the main view. Entering
search mode (`/`) MUST present a fresh search input and MUST NOT reopen the previous
results; submitting a non-empty query MUST run a new search and rebuild the results
modal from scratch, while submitting an empty query MUST start no search. There is
no persistent "reopen last results" affordance. When the lyrics and/or artwork
enrichment services are active, their panels MUST be drawn in the same horizontal
row as the queue (queue | lyrics | cover) rather than stacked below it; when both
enrichment services are off, the row MUST contain only the queue.

#### Scenario: Enrichment panels beside the queue

- GIVEN the TUI is in the main view with the lyrics and/or artwork services active
- WHEN the main view is rendered
- THEN the lyrics and cover panels are drawn in the same horizontal row as the queue (queue | lyrics | cover)
- AND when both enrichment services are off the row contains only the queue
