# Malayalam subtitle feature (development branch only)

## What it does
- /subtitles <movie title> [year] searches a separate MongoDB collection named MalayalamSubtitles.
- The bot's /start message includes a Malayalam Subtitles section button.
- Movie result cards gain a Malayalam Subtitle button only when a matching subtitle exists.
- Clicking a subtitle file sends the stored Telegram document. The existing force-subscription membership check is applied before sending.
- .srt, .ass, and .vtt documents with a clear film title and four-digit release year are supported.

## Before enabling
1. Create a **separate** Telegram channel for Malayalam subtitle files.
2. Add @AutoFilterMGbot as an administrator capable of receiving channel posts.
3. Obtain its numeric -100... Telegram channel ID.
4. **After approval to merge and deploy**, configure SUBTITLE_CHANNEL_ID in Render to that ID.
   Do not use any channel listed in FILE_CHANNELS.
5. Upload a document named like `The.Descent.2005.Malayalam.srt`.
6. Test /subtitles The Descent 2005 and the subtitle button on a matching movie card.
7. Test .ass and .vtt, duplicates, a movie with no subtitles, and a blocked user.

## Constraints and caveats
- Existing movie files, indexing checkpoints, MongoDB movie collections, and sleep scheduling remain unchanged.
- No subtitle migration or automatic backfill is provided. Only newly received channel posts are indexed.
- Title matching uses normalized English/Romanized titles plus year, not an external movie database.
- The feature doesn't translate or author subtitles. Only upload files you are permitted to distribute.
- A separate MongoDB collection is created when this code is deployed; not in the live bot now.
- Do not merge/deploy until approved and CI is checked.

## Technical tests
`go test ./internal/subtitles`
`go test ./internal/core ./internal/database/mongo -run '^$'`

## Release-aware subtitle ranking

When a user opens the Malayalam Subtitle button from a movie card, the bot
compares subtitle filenames with the video filenames in that particular movie
card. It recognizes the following tags when explicitly present:

- Source: BluRay, WEB-DL, WEBRip, HDTV, DVDRip
- Resolution: 480p, 720p, 1080p, 2160p
- Release group: explicit final `-GROUP` suffix
- FPS: explicit `23.976fps`, `24fps`, `25fps`, `29.970fps`, etc.

Subtitle choices are sorted as **Likely release match**, **Possible release
match**, **Release unverified**, and **Different release**. Conflicting options
remain visible with a warning rather than being silently hidden.

For example:
- Video: `The.Descent.2005.1080p.BluRay-GROUP.mkv`
- Subtitle: `The.Descent.2005.Malayalam.1080p.BluRay-GROUP.srt`

If only `/subtitles The Descent 2005` is entered without selecting a movie
card, the bot has no specific video version to compare. All results correctly
show **Release unverified**.

Subtitle metadata is written to the independent `MalayalamSubtitles`
collection for new channel posts; older records can still be parsed from their
stored filenames. No existing movie files are rewritten.

**Caution:** Release-name matching cannot prove synchronization, correct
dialogue or subtitle language. FPS, source labels and release groups are
optional and must not be guessed. Subtitle authors may use inconsistent names.
Actual timing validation would require inspecting the video/subtitle streams
or letting the user report compatibility.

## Real-world filename compatibility (2026-11-23 screenshots)

- `War.Dogs.2016.720p.BluRay.x264-[YTS.AG].mp4`: source BluRay,
  resolution 720p, known release label YTS.
- `Never.Surrender.2026.720p.WEB-DL.@Team_Goat_Bot.mkv`:
  source WEB-DL, resolution 720p, uploader @Team_Goat_Bot;
  the uploader handle is **not** a proof of subtitle compatibility.
- `Transformers.2.2009.IMAX.1080p.BRRip` and other 720p/1080p
  BluRay variants: BRRip, BDRip and BluRay are related disc-source
  families, but synchronization is **not guaranteed**. They are ranked
  as possible matches rather than claimed to be identical.
- HDRip is not automatically grouped with BluRay/BRRip/BDRip. An HDRip
  source could be from a different master.
- `Transformers 2 (2009)` and `Transformers: Revenge of the Fallen
  (2009)` are alternate names for the same film, but the current
  exact normalized-title matching does not automatically resolve
  those aliases. Such files require a consistent title or an
  explicit alias feature before reliable linking.

The screenshot listing several Transformers video releases does not
include a corresponding SRT file or timing comparison. A subtitle
offered by a separate bot or webpage does not establish that all
versions are in sync. For a validated result, test opening the
actual .srt and movie in a player and checking timestamps at the
beginning, middle and end.

## Malayalam Sub video caption tagging (development branch)

In your **existing movie storage channel**, upload a video and write this caption:

`Malayalam Sub`

The caption detector also recognizes `Malayalam subtitle`,
`Malayalam Subs`, and `Malayalam subtitles` (case-insensitive).
To avoid accidental tagging, it excludes explicitly negative labels such as
`No Malayalam subtitles available`.

The new video-post handler uses the **original Telegram document filename** to
recognize .mkv, .mp4, .avi, .mov, .webm, .m4v, .ts, and .wmv uploads, plus
native Telegram video media. It does not tag SRT, ASS, VTT or arbitrary ZIP
documents.

The existing movie record remains untouched. A separate
`MalayalamCaptionTags` collection contains only the original
`file_unique_id` and a boolean manual-suitability marker; repeated uploads
are idempotent.

Within each movie card users can choose:

- **✅ Malayalam Sub Available** — exact movie file IDs manually tagged
  by a storage-channel caption
- **🎬 Other Files** — movie files not so tagged
- **🎬 All Files** — the existing unchanged movie results

The SRT compatibility finder also lists these under a **manual-caption** heading,
separate from **audio-verified** synchronization matches.

**Important limitations**:
- These are **manual declarations**, not audio synchronization evidence.
- New channel posts are indexed immediately. **Edits to captions on
  existing channel video posts are also recognized**, if Telegram delivers
  `edited_channel_post` updates to the bot. Older captions that are never
  edited still require a separate historical backfill, which is not included.
- Tags are keyed to the Telegram file's immutable unique ID; multiple
  source posts of the same Telegram file reuse one tag.
- No changes are made to movie filenames, file IDs, the original movie
  collection, existing commands or the overnight schedule.
- A deployment of the development branch would create the separate tag
  collection when a new caption-tagged upload is received.

For a previously uploaded movie, edit its channel caption after deployment to include `Malayalam Sub`; the edited-channel-post handler records the tag. Removing this phrase in an edit removes its manual tag. This never rewrites the existing movie file record. Verify that your Telegram bot receives `edited_channel_post` updates.
