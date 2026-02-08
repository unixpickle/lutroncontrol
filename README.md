# lutroncontrol

A small web UI + HTTP API for controlling Lutron devices over the Internet.

## Running

The server connects to your Lutron devices over the Internet using your online username and password.
As a result, it can run from anywhere, not just on the same network as the Lutron hub itself.

Set credentials and start the server from the repo root directory:

```bash
export LUTRON_USERNAME="your@email"
export LUTRON_PASSWORD="your-password"
go run ./lutroncontrol -asset-dir lutroncontrol/assets -addr :8080
```

Open the UI at `http://localhost:8080/`.

### CLI flags

- `-addr` (default `:8080`): address to listen on.
- `-asset-dir` (default `assets`): directory with the UI assets.
- `-save-path` (default `state.json`): path to the cached broker state.
- `-secret` (default empty): if set, serve everything under `/<secret>/`.
  - Example: `-secret somesecret` → UI at `http://localhost:8080/somesecret/`
  - If not set, routes are served at `/`.

## HTTP API

All endpoints are `GET` and return JSON. If `-secret` is set, prefix all paths with `/<secret>`.

### Device/State

- `GET /devices`
  - Returns the current list of devices, including zones, levels, and buttons.
- `GET /clear_cache`
  - Clears cached programming model data and returns `{ "data": true }`.

### Device control

- `GET /command/set_level?type=<CommandType>&zone=<zoneId>&level=<0-100>`
  - Sends a level command to a zone.
  - `type` options:
    - `GoToDimmedLevel` (recommended for dimmers)
    - `GoToSwitchedLevel` (recommended for wall switches; use `level=0` or `100`)
    - `GoToLevel` (legacy)
    - `Raise`, `Lower`, `Stop` (for shades; `level` ignored)

- `GET /command/press_and_release?button=<buttonId>`
  - Press-and-release for a physical or virtual button.

- `GET /command/all_off`
  - Turns off all lights (skips shades). Returns `{ "data": true }`.

### Scenes

- `GET /scenes`
  - Returns the list of virtual buttons (scenes), including `IsProgrammed`.

- `GET /scene/activate?scene=<sceneId>`
  - Activates a scene by virtual button ID.

- `GET /scene/activate_by_name?name=<sceneName>`
  - Activates a programmed scene by name (case-insensitive).
  - Returns `{ "data": false }` if not found.

## UI

The UI is a single-page dashboard that:

- Groups devices by room.
- Shows per-device controls:
  - Dimmers: slider + on/off.
  - Switches: on/off.
  - Shades: raise/stop/lower + open percentage.
  - Pico/button devices: button actions.
- Provides a Scenes panel with programmed scenes.
- Includes an All Off button.
- Auto-refreshes periodically.

## Notes

- The server caches programming model/preset data in `state.json` to speed up subsequent loads.
- If you change credentials or want a full refresh, delete `state.json` before restarting.
