# Feature additions for AutoFilterMG

This branch preserves existing file indexing, /index, /settings and search. No new secrets are required.

## Welcome text (admin only)
- `/setwelcome Welcome to my file library!` — sets plain-text welcome body (up to 3000 characters). You can include `{mention}` as a template placeholder.
- `/resetwelcome` — restore the default welcome.
- `/setwelcomebutton Help | https://t.me/YourSupportChannel` — append a URL button to the welcome message. First addition retains existing default buttons.
- `/resetwelcomebuttons` — restore default buttons.

## Existing admin features
- `/settings` → **File AutoDelete**: configure deletion of sent files in minutes.
- `/settings` → **Auto Delete**: deletion of search results in minutes.
- `/settings` → **Force Sub**: add required channels, where the bot has adequate admin invite and membership permissions. Membership checks use existing code.
- Existing admin commands include `/index`, `/logs`, `/broadcast`, `/delete`, `/deleteall`.

## New quality/language filtering
- Perform a normal file search in a group.
- Tap **🎬 Quality / Language**.
- Choose **All / 480p / 720p / 1080p / 4K** and **All / Malayalam / Tamil / Hindi / English**.
- Filtered results are paginated and can be restored using **Original results**.

Only filenames already included in the initial search results are inspected (up to the configured result limit); old database records are not modified. A filename such as `Movie.1080p.Malayalam.mkv` will be detected. Unknown or unlabeled versions will not be assigned a language. This does not inspect internal media tracks, translate, or change the video itself. If the database has no indexed matching files, the filter shows no matches.

## Deployment
GitHub branch: `feature/search-filters-welcome-controls`. Test builds with `go test ./...` and `go build ./...`. Review and merge the pull request only after tests pass. The Render service has auto deploy disabled, so merging does not automatically update your currently running bot. In Render, manually deploy after reviewing, then test /start, /settings, indexing and search.

## Security
Use Render environment variables for secrets. Your MongoDB Atlas access list currently includes a temporary broad network rule from previous debugging: replace it with restricted ingress when possible and rotate the database password.
