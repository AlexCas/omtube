# Delta for tui-shell

## ADDED Requirements

### Requirement: Search Results Modal

The system MUST present search results in a full-screen `modeResults` takeover
modal that is entered ONLY after a multi-result search completes. The modal MUST
own the entire viewport while active and MUST be the sole surface where results
are browsed and acted upon. Results-mode key hints MUST be shown ONLY while the
modal is active (D4).

#### Scenario: Open modal on multi-result search

- GIVEN the TUI is in normal or search mode
- WHEN a search returns multiple results
- THEN the `modeResults` modal opens full-screen with the results
- AND the main view is hidden until the modal is closed

#### Scenario: Dismiss with Esc

- GIVEN the results modal is active
- WHEN the user presses Esc
- THEN the modal closes and returns to the main view without enqueuing anything

#### Scenario: Enqueue with Enter

- GIVEN the results modal is active with a selection
- WHEN the user presses Enter
- THEN the selected track is added to the queue and the modal returns to the main view

#### Scenario: Navigate results

- GIVEN the results modal is active
- WHEN the user presses Up/Down or `j`/`k`
- THEN the selection moves through the results list

#### Scenario: Add selection to a playlist

- GIVEN the results modal is active with a selection
- WHEN the user presses `a` and chooses a playlist
- THEN the track is added to that playlist and the flow returns to the results modal (not the library)

#### Scenario: Toggle favorite on selection

- GIVEN the results modal is active with a selection
- WHEN the user presses `f`
- THEN the favorite status of the selected track is toggled and the modal remains active

#### Scenario: Results hints visible only in modal

- GIVEN the TUI is in the main view (no modal active)
- WHEN the help line is displayed
- THEN the results-mode key hints are not shown
- AND those hints only appear while `modeResults` is active

## MODIFIED Requirements

### Requirement: Results and Queue Panels

The system MUST display search results in a full-screen `modeResults` takeover
modal entered only after a multi-result search, and MUST display the playback
queue as an inline full-size panel always visible in the main view, highlighting
the current track. The results panel MUST NOT be drawn in the main view. Entering
search mode (`/`) MUST present a fresh search input and MUST NOT reopen the previous
results; submitting a non-empty query MUST run a new search and rebuild the results
modal from scratch, while submitting an empty query MUST start no search. There is
no persistent "reopen last results" affordance (D2).
(Previously: results and the queue were drawn side-by-side as always-visible panels in the main view.)

#### Scenario: Enqueue from results

- GIVEN results are visible in the `modeResults` modal
- WHEN the user selects a result and confirms
- THEN the track is added to the queue (and played if the queue was empty)

#### Scenario: Queue always visible inline

- GIVEN the TUI is in the main view
- WHEN no modal is active
- THEN the queue is shown as a full-size inline panel
- AND the results panel is not drawn in the main view

#### Scenario: Opening search does not reopen previous results

- GIVEN results from a previous search existed
- WHEN the user presses `/` to enter search mode
- THEN a fresh search input is shown and the previous results are not reopened
- AND submitting an empty query starts no search
- AND submitting a non-empty query runs a new search and rebuilds the results modal

### Requirement: Add by URL Input Mode

The TUI MUST provide a mode to paste a YouTube video URL. On submit, it MUST resolve the
URL, append the resolved track to the queue, and present that track so the user can add
it to an existing playlist via the existing add-to-playlist picker. A single track
resolved from a URL MUST NOT open the `modeResults` modal; the TUI MUST stay in the main
view (D1). Resolution MUST be non-blocking and surface a readable error on failure.
(Previously: did not constrain modal behavior for a single URL-resolved track.)

#### Scenario: Paste a video URL

- GIVEN the user opens the "add by URL" mode
- WHEN they paste a video URL and submit it
- THEN the resolved track is enqueued and displayed as a selectable result
- AND the TUI remains in the main view without opening the results modal

#### Scenario: Add the URL track to a playlist

- GIVEN a track freshly resolved from a URL is displayed
- WHEN the user presses the add-to-playlist action
- THEN the existing playlist picker opens for that track

#### Scenario: Invalid URL feedback

- GIVEN the user submits a URL that cannot be resolved
- WHEN resolution fails
- THEN a readable error is displayed and the TUI remains operational
