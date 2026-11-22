# Optional AdsGram bot ads — DISABLED BY DEFAULT

This code is prepared for development review only. **No live advertisements or rewards are enabled by this change.**

## Provider and API

The implementation targets the **AdsGram.ai Bot Ad Block API**, not the unrelated adsgram.store pilot.
Official docs: https://docs.adsgram.ai/bots/block-integration and
https://docs.adsgram.ai/bots/create-block

The bot requests an ad from `https://api.adsgram.ai/advbot` after eligible successful private-chat file searches.
The response is displayed as HTML with two URL buttons in a Telegram
`ProtectContent: true` sponsored message. No ad request is made when disabled,
and no ad is displayed in public groups.

## Safeguards

* The bot's normal movie search and file results always work independently of ads.
* Delivery only after 10 successful searches, at most once per 24 hours per user.
* MongoDB `AdsGramPacing` stores only an HMAC-pseudonymized user key,
  a small search counter, last ad opportunity timestamp and 90-day TTL.
  It does not record movie titles, filenames, clicked links, or raw Telegram IDs.
* Provider requests have a timeout; HTTP 204/errors are skipped silently.
* Ad short links must be HTTPS URLs on `api.adsgram.ai/short/`.
* All sponsored messages have a clear label and are protected from forwarding.
* Credentials are read only from server-side environment variables.

## Activation — REQUIRES SEPARATE USER APPROVAL

The following variables are **not set in this development change**:

| Variable | Purpose |
| --- | --- |
| `ADSGRAM_ENABLED` | Must equal `true`; defaults to disabled |
| `ADSGRAM_OWNER_APPROVED` | Second explicit consent gate, also must equal `true` |
| `ADSGRAM_BLOCK_ID` | AdsGram Bot Ad Block numeric ID, with or without `bot-` prefix |
| `ADSGRAM_TOKEN` | Private AdsGram account token (Render secret, never GitHub) |
| `ADSGRAM_PACING_SALT` | Private random salt for pseudonymous MongoDB records |

**Do not set these in production until the user separately authorizes live ads.**

## Reward URL and unresolved provider authentication

AdsGram requires a Reward URL when registering a bot block.
The draft endpoint is `https://autofilterbotnew-ioi4.onrender.com/ads/reward`.
When advertising is disabled it returns HTTP 404.
When advertising is enabled it returns HTTP 501 and **never issues a reward**.

This behavior is intentional: the documented bot flow does not provide a
cryptographic signature or trusted verification of arbitrary reward callbacks.
A GET containing a user ID must NOT grant rewards. Before publishing
a block or enabling paid ads, decide a **real optional reward**, confirm the
exact provider callback behavior with AdsGram, implement tamper-resistant,
replay-safe verification and idempotent accounting, and run an end-to-end
test with a moderated test block. Do not claim that the Reward URL is ready
for AdsGram approval yet.

AdsGram may reject bots that distribute content without necessary rights.
The publisher must check applicable policies, moderation and copyright
before applying for production approval.

## Development tests

`go test ./internal/core/... ./internal/database/mongo/...`
`go build ./...`

Tests use a local HTTP mock; never request a real sponsored impression.
