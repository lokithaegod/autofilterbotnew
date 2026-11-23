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
