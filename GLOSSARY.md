# Glossary

**Tracker**: a private torrent site whose IRC announce channel trackarr watches. Defined by an autodl tracker definition plus the user's tracker config (`config.TrackerInstance`).

**Announce**: one IRC line (or several, for multi-line patterns) describing a new upload. The processor parses it into a Release.

**Release**: a parsed announce (`release.Release`, `config.ReleaseInfo`): name, torrent URL, size, category and so on. It is what gets filtered and pushed.

**Enrichment**: filling in what an announce lacks (name, category, size, file list). Owned by `Release.Enrich`. Policy: tracker API first, then the torrent file (**bencode**) if enabled, otherwise the Release is dropped. The result records its source: API, bencode, or none.

**Tracker API**: a tracker's own lookup endpoint (PTP, BTN, MTV), reached through a `tracker.Interface` adapter. Shares the lookup plumbing in `utils/tracker/lookup.go`.

**Bencode**: decoding the downloaded `.torrent` file to read its name, size and file list. Also the Release's fallback source when the Tracker API can't help.

**PVR**: a Sonarr/Radarr-style download manager that receives Releases (`config.PvrConfig`). Each PVR has ignore, accept and delay expressions.

**Pusher**: the seam that delivers a Release to a PVR. Default adapter: the PVR's v3 `release/push` HTTP endpoint.

**Publisher**: the seam that records a pushed Release in the database and broadcasts it on the websocket `releases` topic.

**Proxied torrent URL**: the `/api/torrent` URL on trackarr handed to a PVR instead of the tracker's, so cookies and second-sweep file expressions can be applied.

**Second sweep**: re-evaluating a PVR's ignore expressions against the torrent file contents when the PVR downloads it through the proxied URL.
