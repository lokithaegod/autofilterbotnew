# Automatic SRT → video compatibility finder (development branch)

## Commands and input

- Send a `.srt` document directly to @AutoFilterMGbot in a private chat.
- Reply `/findcompatible` to a `.srt` document.
- Open the existing Malayalam subtitle section, find a subtitle, and tap
  **Find compatible video files**.

This module uses the existing Telegram file IDs and the pre-existing MongoDB
file records, without changing their schema or inserting copies. The new
MongoDB read-only method queries **each** existing file collection (not the
normal 500-record search limit), deduplicating file unique IDs.

## Four separate result sections

1. **Audio-aligned / high confidence**: a readable local video, a parseable
   SRT, FFprobe, FFsubsync, a positive alignment score, at least 12 subtitle
   cues over five minutes, and <=0.35s maximum adjustment with <=0.20s
   opening-to-ending drift. Speech-activity checks are statistical evidence,
   not guarantees that a translation or cut is correct.
2. **Needs synchronization**: audio analysis detects plausible offset/drift.
   The module NEVER modifies the original SRT or video.
3. **Incompatible**: explicit conflicting movie title/year/cut, or subtitle
   timestamps exceed an examined video's runtime.
4. **Other files — unverified**: all plausible candidates without audio
   evidence, including legacy document rows with lost extensions.

Old file rows frequently have **extensions stripped during the bot's indexing
pipeline**. The old schema cannot distinguish every document-movie from every
document-subtitle, and does not store FPS, duration or an accessible path.
Do not imply that an unverified row is definitely a video. A new metadata
migration could fix this in future but is intentionally NOT included.

## Requirements before any live rollout

- Existing Telegram Bot API permits downloading selected SRTs up to 20 MB;
  this module uses a stricter **4 MB** limit.
- Videos stored only as Telegram `file_id` values are **not available** for
  FFprobe/FFsubsync analysis. Standard bot `getFile` cannot download large
  videos, and MongoDB records do not expose raw audio bytes.
- An authorized operator must provide existing local media paths, for example
  via a mounted read-only volume and `COMPAT_MEDIA_MANIFEST` pointing to a JSON
  file mapping *existing file unique IDs* to accessible local video paths.
  Example: `{"AgAD...unique":"<absolute local media path>"}`.
  No new media file copies or downloads are created by this feature.
- The analysis host must have **python3, ffprobe, and ffsubsync** installed.
  The current production Dockerfile does not install them. Do not modify
  Render settings, GitHub secrets, or Dockerfile until the user approves.
- Analysis is deliberately bounded to **12 audio candidates per request** to
  protect the existing Render service; all other matching rows appear in the
  unverified section, with counts, rather than being called compatible.
  This is not yet a complete automatic verification of thousands of movies.

## Performance and security

Only indexed files matching normalized title/year are fetched from MongoDB,
and the selected SRT is downloaded once into a private temporary file. The
original subtitle file remains unchanged. FFsubsync writes its trial output
only in a temporary directory. Subprocesses have hard timeouts. No remote
video URL supplied by users is opened, no MongoDB records are rewritten,
and existing movie index and sleep workflows are not touched.

## Tests

`go test ./internal/compatfinder`

`python3 -m unittest discover -s scripts -p test_compatibility_probe.py -v`

`go test ./internal/core ./internal/database/mongo -run '^$'`

The first two have self-contained unit tests; the final app compilation
requires GitHub Actions/network access to Go dependencies. Full media tests
require actual accessible video + SRT sample files, which are not present in
this development environment.

## Unresolved scope

- Resolution alias matching (`Transformers 2` vs
  `Transformers: Revenge of the Fallen`) needs explicit title alias data.
- Duration/FPS and optional hashes can be extracted only from accessible video
  bytes, never from the existing Telegram ID alone.
- A stable media-access integration is required before the finder can truly
  verify *every* video across thousands of records.
- A verified audio timing result is high-confidence algorithmic evidence, not
  an absolute guarantee of synchronization at every dialogue line.

**This feature must not be merged or deployed until separately approved.**
