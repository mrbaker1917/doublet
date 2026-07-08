![Status of tests on this REPO](https://github.com/mrbaker1917/doublet/actions/workflows/ci.yml/badge.svg)

# Doublet (in Go)
NB: currently deployed at [Doublet](https://doublet.fly.dev/).
A CLI and WEB word game where you transform one word into another by changing one letter at a time. Each intermediate step must also be a valid dictionary word.

## Rules

- Start and target words must be the same length.
- Each move must change exactly one letter.
- Every move must produce a valid word.
- You must stay within the allowed maximum changes.

## Run CLI

```bash
go run ./cmd/cli
```

You can provide flags instead of prompts:

```bash
go run ./cmd/cli -start cat -end dog -difficulty hard -solve
```

## Web API

Start the HTTP server (run from the project root so dictionary files resolve):

```bash
go run ./cmd/web
```

Then visit `http://localhost:8080` to play in the browser.

## Security Measures

Game state is kept in memory with bounds to limit resource use:

- `-max-games` (default `2000`) — evicts the least recently used game when full
- `-game-ttl` (default `24h`) — removes inactive games after this duration; access refreshes the timer

A background cleanup runs every 5 minutes to drop expired games.

`POST /api/games` is protected against CPU exhaustion:

- `-max-concurrent-bfs` (default `4`) — caps simultaneous path searches; excess waits up to `-bfs-wait` (default `5s`) then returns HTTP 503
- `-path-cache-size` (default `4096`) — caches shortest paths for repeated start/end pairs

All JSON API routes are rate limited per IP (HTTP 429) using `-api-rate-window` (default `1m`):

- `-create-rate-limit` (default `20`) — `POST /api/games`
- `-move-rate-limit` (default `120`) — `POST /api/games/{id}/move`
- `-read-rate-limit` (default `180`) — `GET /api/suggestions` and `GET /api/games/{id}`

JSON API bodies are capped at `-max-request-body` bytes (default `8192`); larger requests get HTTP 413.

All responses include security headers: a strict Content-Security-Policy (same-origin scripts, styles, and API calls), clickjacking protection, `nosniff`, referrer and permissions policies, and HSTS on HTTPS (including behind Fly’s TLS terminator via `X-Forwarded-Proto`).

The UI supports suggested doublets, custom start/target words, difficulty selection, move history, win/lose feedback, hints, restart, and an **Expert mode** toggle (top right).

- **Default:** common dictionary (`words-common.txt`) — 3–5 letter words without rare letters, suited for casual play.
- **Expert mode:** full dictionary (`words-large.txt`) — allows obscure bridge words and longer vocabulary.

Expert mode is chosen when a game starts and stays fixed for that game. The preference is saved in the browser.

## API endpoints:

- `POST /api/games` — start a game (`start`, `end`, `difficulty`, optional `max`, optional `expert`)
- `GET /api/games/{id}` — fetch game state
- `POST /api/games/{id}/move` — submit a move (`word`)
- `GET /api/suggestions` — random easy/medium/hard doublet pairs; add `?expert=true` for expert pools

To regenerate suggestion pools after editing seed files in `internal/game/suggestiondata/`:

```bash
go run ./cmd/seedpairs
go run ./cmd/seedpairs -dict words-common.txt -pool common
```

To rebuild the common dictionary from `words-large.txt`:

```bash
go run ./cmd/buildcommon
go run ./cmd/seedpairs -dict words-common.txt -pool common
```

Both steps are required: `buildcommon` updates the playable dictionary; `seedpairs` regenerates the embedded suggestion chips. Words in `common-excluded.txt` are removed from the dictionary and filtered out of common suggestion start/target pairs when you regenerate.

## Dictionary Options for CLI:

Use a built-in preset:

- `-lexicon small` uses `words.txt`.
- `-lexicon common` uses `words-common.txt` (3–5 letter casual words).
- `-lexicon large` uses `words-large.txt` (default).

Use your own dictionary file:

```bash
go run ./cmd/cli -dict mywords.txt -start cold -end warm -difficulty medium -solve
```

When `-dict` is provided, it overrides `-lexicon`.

## Difficulty Options for CLI:

- `easy`: max changes = shortest path + scaled slack (`+1` for 1-step, `+2` for 2–4 steps, `+3` for 5+)
- `medium`: max changes = shortest path + scaled slack (`+0` for 1-step, `+1` for 2–4 steps, `+2` for 5+)
- `hard`: max changes = shortest path (no slack)
- `custom`: requires `-max`; must be at least the shortest path length and at most `shortest + 10` (capped at 100). Preset difficulties ignore any `max` sent in the API.

Examples:

```bash
go run ./cmd/cli -lexicon small -start cat -end dog -difficulty easy -solve
go run ./cmd/cli -lexicon small -start cat -end dog -difficulty custom -max 3 -solve
```

## Interactive Commands

During gameplay:

- `/hint` shows the next step on the shortest path.
- `/restart` starts the same game over from start word
- `/solve` reveals the full shortest path.
- `/quit` exits the game.
- these are now also available as buttons on the web app
- on web app, also available: `Give up?` button when you are really stuck.

NB: currently deployed at [Doublet](https://doublet.fly.dev/).