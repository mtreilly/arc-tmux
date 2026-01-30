# arc-tmux

Native tmux control surface for sessions, windows, and panes. Designed for automation and agent workflows.

## Quick start

```
arc-tmux sessions
arc-tmux panes --session dev --window 2
arc-tmux inspect --pane=dev:2.0
arc-tmux follow --pane=dev:2.0 --lines 200
arc-tmux locate --field command node
```

## Output formats

All inventory-style commands support `--output table|json|yaml|quiet`.

### sessions --output json

JSON shape:

```json
[
  {
    "name": "dev",
    "windows": 3,
    "attached": 1,
    "created_at": "2025-01-29T10:12:15Z",
    "activity_at": "2025-01-29T10:15:42Z"
  }
]
```

### panes --output json

Filters: `--command`, `--title`, `--path`, `--session`, `--window`.
Use `--fuzzy` for fuzzy matching.

JSON shape:

```json
[
  {
    "session": "dev",
    "window_index": 2,
    "window_name": "api",
    "window_active": true,
    "pane_index": 0,
    "pane_id": "%5",
    "formatted_id": "dev:2.0",
    "active": true,
    "command": "bash",
    "title": "build",
    "path": "/Users/me/project",
    "pid": 1234,
    "activity_at": "2025-01-29T10:15:42Z"
  }
]
```

### inspect --output json

JSON shape:

```json
{
  "pane": {
    "session": "dev",
    "window_index": 2,
    "window_name": "api",
    "window_active": true,
    "pane_index": 0,
    "pane_id": "%5",
    "active": true,
    "command": "bash",
    "title": "build",
    "path": "/Users/me/project",
    "pid": 1234,
    "activity_at": "2025-01-29T10:15:42Z"
  },
  "process_tree": [
    { "pid": 1234, "ppid": 1, "command": "bash", "depth": 0 },
    { "pid": 2345, "ppid": 1234, "command": "node server.js", "depth": 1 }
  ]
}
```

### follow --output json

Streams NDJSON events (one object per line):

```json
{ "time": "2025-01-29T10:15:42.123Z", "line": "Starting server..." }
```

Use `--from-start` to emit the full buffer first. `--lines` controls the capture size (0 for full).

### locate --output json

Same shape as `panes --output json`, filtered by query and field.
Use `--fuzzy` for fuzzy matching or `--regex` for regex matching.

### Pane selectors

Commands that accept `--pane` also support selectors:

- `@current` uses the current pane when inside tmux.
- `@active` uses the active pane across all sessions.
- `@name` uses a saved alias (see `alias` below).

Session selectors (for `--session`) support `@current` and `@managed`.

### Aliases

Create and use pane aliases for quick targeting:

```
arc-tmux alias set api --pane=@current
arc-tmux send "npm test" --pane=@api
arc-tmux alias list
```

## Agent workflows

- Run a command and capture output:
  - `arc-tmux run "make test" --pane=dev:2.0 --timeout 300 --output json`
- Stream new output only:
  - `arc-tmux follow --pane=dev:2.0 --lines 200`
- Full buffer then follow:
  - `arc-tmux follow --pane=dev:2.0 --from-start`
- Locate panes by command/title/path:
  - `arc-tmux locate --field command node`

## Integration tests

Integration tests require tmux and an isolated socket:

```
ARC_TMUX_IT=1 go test ./...
```
